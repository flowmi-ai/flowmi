package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"time"

	"github.com/go-resty/resty/v2"
)

var restyClient = resty.New().SetTimeout(30 * time.Second).SetResponseBodyLimit(1 << 20)

// PlaceholderRedirectURI is a fixed redirect URI used in the login flow.
// No real callback server is needed since the CLI calls the login API directly.
const PlaceholderRedirectURI = "http://127.0.0.1:12345/callback"

// TokenResponse holds the tokens returned by the authorization server.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// LoginRequest is the JSON body sent to the /oauth2/login endpoint.
type LoginRequest struct {
	Email               string `json:"email"`
	Password            string `json:"password"`
	ClientID            string `json:"client_id"`
	RedirectURI         string `json:"redirect_uri"`
	ResponseType        string `json:"response_type"`
	CodeChallenge       string `json:"code_challenge"`
	CodeChallengeMethod string `json:"code_challenge_method"`
	State               string `json:"state"`
}

// LoginResponse is the JSON response from the /oauth2/login endpoint.
type LoginResponse struct {
	Code        string `json:"code"`
	RedirectURI string `json:"redirect_uri"`
	State       string `json:"state"`
}

// GeneratePKCE creates a code_verifier and its S256 code_challenge.
func GeneratePKCE() (verifier, challenge string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("generating random bytes: %w", err)
	}
	verifier = base64.RawURLEncoding.EncodeToString(buf)
	h := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(h[:])
	return verifier, challenge, nil
}

// GenerateState creates a random state parameter for CSRF protection.
func GenerateState() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generating state: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// Login sends credentials to the login endpoint and returns an auth code.
func Login(ctx context.Context, loginURL string, req *LoginRequest) (*LoginResponse, error) {
	resp, err := restyClient.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(req).
		Post(loginURL)
	if err != nil {
		return nil, fmt.Errorf("sending login request: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, parseErrorResponse(resp.StatusCode(), resp.Body())
	}

	var envelope struct {
		Data LoginResponse `json:"data"`
	}
	if err := json.Unmarshal(resp.Body(), &envelope); err != nil {
		return nil, fmt.Errorf("decoding login response: %w", err)
	}
	return &envelope.Data, nil
}

// ExchangeCode exchanges an authorization code for tokens.
func ExchangeCode(ctx context.Context, tokenURL, code, verifier, redirectURI string) (*TokenResponse, error) {
	resp, err := restyClient.R().
		SetContext(ctx).
		SetFormData(map[string]string{
			"grant_type":    "authorization_code",
			"code":          code,
			"code_verifier": verifier,
			"redirect_uri":  redirectURI,
			"client_id":     "flowmi-cli",
		}).
		Post(tokenURL)
	if err != nil {
		return nil, fmt.Errorf("exchanging code: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, parseErrorResponse(resp.StatusCode(), resp.Body())
	}

	var token TokenResponse
	if err := json.Unmarshal(resp.Body(), &token); err != nil {
		return nil, fmt.Errorf("decoding token response: %w", err)
	}
	return &token, nil
}

// RefreshTokens exchanges a refresh token for a new token pair.
func RefreshTokens(ctx context.Context, refreshURL, refreshToken string) (*TokenResponse, error) {
	resp, err := restyClient.R().
		SetContext(ctx).
		SetFormData(map[string]string{
			"grant_type":    "refresh_token",
			"refresh_token": refreshToken,
			"client_id":     "flowmi-cli",
		}).
		Post(refreshURL)
	if err != nil {
		return nil, fmt.Errorf("refreshing tokens: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, parseErrorResponse(resp.StatusCode(), resp.Body())
	}

	var token TokenResponse
	if err := json.Unmarshal(resp.Body(), &token); err != nil {
		return nil, fmt.Errorf("decoding refresh response: %w", err)
	}
	return &token, nil
}

// parseErrorResponse extracts an error message from a non-200 response.
// It tries the server envelope format {"error":{"message":"..."}} first,
// then falls back to OAuth2 standard {"error":"...", "error_description":"..."}.
func parseErrorResponse(statusCode int, body []byte) error {
	if len(body) == 0 {
		return fmt.Errorf("server returned status %d", statusCode)
	}

	// Try envelope format: {"success":false,"error":{"code":"...","message":"..."}}
	var envelope struct {
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &envelope) == nil && envelope.Error != nil && envelope.Error.Message != "" {
		return fmt.Errorf("%s (status %d)", envelope.Error.Message, statusCode)
	}

	// Try OAuth2 standard format: {"error":"...","error_description":"..."}
	var oauth2Err struct {
		Error       string `json:"error"`
		Description string `json:"error_description"`
	}
	if json.Unmarshal(body, &oauth2Err) == nil && oauth2Err.Error != "" {
		if oauth2Err.Description != "" {
			return fmt.Errorf("%s: %s (status %d)", oauth2Err.Error, oauth2Err.Description, statusCode)
		}
		return fmt.Errorf("%s (status %d)", oauth2Err.Error, statusCode)
	}

	return fmt.Errorf("server returned status %d", statusCode)
}

// CallbackResult holds the code and state received from the OAuth callback.
type CallbackResult struct {
	Code  string
	State string
	Err   error
}

// StartCallbackServer starts a localhost HTTP server that listens for the OAuth
// callback. It returns the port the server is listening on and a channel that
// will receive the authorization code. The server shuts down after receiving
// the callback or when the context is cancelled.
func StartCallbackServer(ctx context.Context) (port int, resultCh <-chan CallbackResult, err error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, nil, fmt.Errorf("starting callback listener: %w", err)
	}
	port = listener.Addr().(*net.TCPAddr).Port

	ch := make(chan CallbackResult, 1)

	mux := http.NewServeMux()
	srv := &http.Server{Handler: mux}

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")

		if code == "" {
			errMsg := r.URL.Query().Get("error")
			if errMsg == "" {
				errMsg = "missing authorization code"
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, errorHTML, errMsg)
			ch <- CallbackResult{Err: fmt.Errorf("callback error: %s", errMsg)}
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, successHTML)
		ch <- CallbackResult{Code: code, State: state}
	})

	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			ch <- CallbackResult{Err: fmt.Errorf("callback server: %w", err)}
		}
	}()

	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background())
	}()

	return port, ch, nil
}

// BuildAuthorizeURL constructs the authorization URL with PKCE and state parameters.
func BuildAuthorizeURL(serverURL, redirectURI, state, challenge string) string {
	params := url.Values{
		"client_id":             {"flowmi-cli"},
		"redirect_uri":          {redirectURI},
		"response_type":         {"code"},
		"state":                 {state},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}
	return serverURL + "/authorize?" + params.Encode()
}

// OpenBrowser opens the given URL in the default browser.
func OpenBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return cmd.Start()
}

const successHTML = `<!DOCTYPE html>
<html><head><title>flowmi — Login Successful</title>
<style>body{font-family:system-ui,sans-serif;display:flex;justify-content:center;align-items:center;height:100vh;margin:0;background:#f8f9fa}
.card{text-align:center;padding:2rem;border-radius:12px;background:#fff;box-shadow:0 2px 8px rgba(0,0,0,.1)}
h1{color:#22c55e;margin-bottom:.5rem}p{color:#6b7280}</style></head>
<body><div class="card"><h1>&#10003; Login Successful</h1><p>You can close this window and return to the terminal.</p></div></body></html>`

const errorHTML = `<!DOCTYPE html>
<html><head><title>flowmi — Login Failed</title>
<style>body{font-family:system-ui,sans-serif;display:flex;justify-content:center;align-items:center;height:100vh;margin:0;background:#f8f9fa}
.card{text-align:center;padding:2rem;border-radius:12px;background:#fff;box-shadow:0 2px 8px rgba(0,0,0,.1)}
h1{color:#ef4444;margin-bottom:.5rem}p{color:#6b7280}</style></head>
<body><div class="card"><h1>&#10007; Login Failed</h1><p>%s</p></div></body></html>`
