package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestGeneratePKCE(t *testing.T) {
	verifier, challenge, err := GeneratePKCE()
	if err != nil {
		t.Fatalf("GeneratePKCE() error: %v", err)
	}

	if verifier == "" || challenge == "" {
		t.Fatal("GeneratePKCE() returned empty strings")
	}

	// Verify the challenge is the S256 hash of the verifier.
	h := sha256.Sum256([]byte(verifier))
	expected := base64.RawURLEncoding.EncodeToString(h[:])
	if challenge != expected {
		t.Errorf("challenge mismatch:\n  got:  %s\n  want: %s", challenge, expected)
	}

	// Verify uniqueness.
	v2, c2, _ := GeneratePKCE()
	if verifier == v2 || challenge == c2 {
		t.Error("GeneratePKCE() returned duplicate values")
	}
}

func TestGenerateState(t *testing.T) {
	s1, err := GenerateState()
	if err != nil {
		t.Fatalf("GenerateState() error: %v", err)
	}
	if s1 == "" {
		t.Fatal("GenerateState() returned empty string")
	}

	s2, _ := GenerateState()
	if s1 == s2 {
		t.Error("GenerateState() returned duplicate values")
	}
}

func TestBuildAuthorizeURL(t *testing.T) {
	u := BuildAuthorizeURL("https://auth.example.com", "http://127.0.0.1:9999/callback", "mystate", "mychallenge")
	parsed, err := url.Parse(u)
	if err != nil {
		t.Fatalf("invalid URL: %v", err)
	}

	tests := map[string]string{
		"client_id":             "flowmi-cli",
		"redirect_uri":          "http://127.0.0.1:9999/callback",
		"response_type":         "code",
		"state":                 "mystate",
		"code_challenge":        "mychallenge",
		"code_challenge_method": "S256",
	}
	for key, want := range tests {
		if got := parsed.Query().Get(key); got != want {
			t.Errorf("param %s = %q, want %q", key, got, want)
		}
	}
}

func TestStartCallbackServer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	port, resultCh, err := StartCallbackServer(ctx)
	if err != nil {
		t.Fatalf("StartCallbackServer() error: %v", err)
	}
	if port == 0 {
		t.Fatal("StartCallbackServer() returned port 0")
	}

	// Simulate callback.
	callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback?code=testcode&state=teststate", port)
	resp, err := http.Get(callbackURL)
	if err != nil {
		t.Fatalf("GET callback: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("callback status = %d, want 200", resp.StatusCode)
	}

	result := <-resultCh
	if result.Err != nil {
		t.Fatalf("callback result error: %v", result.Err)
	}
	if result.Code != "testcode" {
		t.Errorf("code = %q, want %q", result.Code, "testcode")
	}
	if result.State != "teststate" {
		t.Errorf("state = %q, want %q", result.State, "teststate")
	}
}

func TestStartCallbackServerMissingCode(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	port, resultCh, err := StartCallbackServer(ctx)
	if err != nil {
		t.Fatalf("StartCallbackServer() error: %v", err)
	}

	callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback?error=access_denied", port)
	resp, err := http.Get(callbackURL)
	if err != nil {
		t.Fatalf("GET callback: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("callback status = %d, want 400", resp.StatusCode)
	}

	result := <-resultCh
	if result.Err == nil {
		t.Fatal("expected error for missing code")
	}
}

func TestLogin(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("content-type = %s, want application/json", ct)
		}

		var req LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		if req.Email != "user@example.com" {
			t.Errorf("email = %q, want user@example.com", req.Email)
		}
		if req.Password != "secret123" {
			t.Errorf("password = %q, want secret123", req.Password)
		}
		if req.ClientID != "flowmi-cli" {
			t.Errorf("client_id = %q, want flowmi-cli", req.ClientID)
		}
		if req.ResponseType != "code" {
			t.Errorf("response_type = %q, want code", req.ResponseType)
		}
		if req.CodeChallengeMethod != "S256" {
			t.Errorf("code_challenge_method = %q, want S256", req.CodeChallengeMethod)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": LoginResponse{
				Code:        "auth_code_123",
				RedirectURI: req.RedirectURI,
				State:       req.State,
			},
		})
	}))
	defer mockServer.Close()

	resp, err := Login(context.Background(), mockServer.URL, &LoginRequest{
		Email:               "user@example.com",
		Password:            "secret123",
		ClientID:            "flowmi-cli",
		RedirectURI:         PlaceholderRedirectURI,
		ResponseType:        "code",
		CodeChallenge:       "challenge_abc",
		CodeChallengeMethod: "S256",
		State:               "state_xyz",
	})
	if err != nil {
		t.Fatalf("Login() error: %v", err)
	}
	if resp.Code != "auth_code_123" {
		t.Errorf("code = %q, want auth_code_123", resp.Code)
	}
	if resp.State != "state_xyz" {
		t.Errorf("state = %q, want state_xyz", resp.State)
	}
}

func TestLoginInvalidCredentials(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"error": map[string]string{
				"code":    "INVALID_CREDENTIALS",
				"message": "invalid email or password",
			},
		})
	}))
	defer mockServer.Close()

	_, err := Login(context.Background(), mockServer.URL, &LoginRequest{
		Email:    "wrong@example.com",
		Password: "bad",
		ClientID: "flowmi-cli",
	})
	if err == nil {
		t.Fatal("expected error for invalid credentials")
	}
	if got := err.Error(); got == "" {
		t.Error("error message should not be empty")
	}
}

func TestExchangeCode(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Errorf("content-type = %s, want application/x-www-form-urlencoded", ct)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if got := r.FormValue("grant_type"); got != "authorization_code" {
			t.Errorf("grant_type = %q, want authorization_code", got)
		}
		if got := r.FormValue("code"); got != "authcode123" {
			t.Errorf("code = %q, want authcode123", got)
		}
		if got := r.FormValue("code_verifier"); got != "verifier123" {
			t.Errorf("code_verifier = %q, want verifier123", got)
		}
		if got := r.FormValue("client_id"); got != "flowmi-cli" {
			t.Errorf("client_id = %q, want flowmi-cli", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(TokenResponse{
			AccessToken:  "access_abc",
			RefreshToken: "refresh_xyz",
			TokenType:    "Bearer",
			ExpiresIn:    3600,
		})
	}))
	defer mockServer.Close()

	token, err := ExchangeCode(context.Background(), mockServer.URL, "authcode123", "verifier123", "http://127.0.0.1:9999/callback")
	if err != nil {
		t.Fatalf("ExchangeCode() error: %v", err)
	}
	if token.AccessToken != "access_abc" {
		t.Errorf("access_token = %q, want access_abc", token.AccessToken)
	}
	if token.RefreshToken != "refresh_xyz" {
		t.Errorf("refresh_token = %q, want refresh_xyz", token.RefreshToken)
	}
}

func TestExchangeCodeServerError(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer mockServer.Close()

	_, err := ExchangeCode(context.Background(), mockServer.URL, "code", "verifier", "http://127.0.0.1:9999/callback")
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}
}

func TestRefreshTokens(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if got := r.FormValue("grant_type"); got != "refresh_token" {
			t.Errorf("grant_type = %q, want refresh_token", got)
		}
		if got := r.FormValue("refresh_token"); got != "old_refresh" {
			t.Errorf("refresh_token = %q, want old_refresh", got)
		}
		if got := r.FormValue("client_id"); got != "flowmi-cli" {
			t.Errorf("client_id = %q, want flowmi-cli", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(TokenResponse{
			AccessToken:  "new_access",
			RefreshToken: "new_refresh",
			TokenType:    "Bearer",
			ExpiresIn:    3600,
		})
	}))
	defer mockServer.Close()

	token, err := RefreshTokens(context.Background(), mockServer.URL, "old_refresh")
	if err != nil {
		t.Fatalf("RefreshTokens() error: %v", err)
	}
	if token.AccessToken != "new_access" {
		t.Errorf("access_token = %q, want new_access", token.AccessToken)
	}
	if token.RefreshToken != "new_refresh" {
		t.Errorf("refresh_token = %q, want new_refresh", token.RefreshToken)
	}
}

func TestRefreshTokensInvalidToken(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error":             "invalid_grant",
			"error_description": "refresh token is expired or revoked",
		})
	}))
	defer mockServer.Close()

	_, err := RefreshTokens(context.Background(), mockServer.URL, "expired_token")
	if err == nil {
		t.Fatal("expected error for invalid refresh token")
	}
}
