package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

// TokenRefresher is a callback that refreshes the access token.
// It receives the request context so cancellation propagates correctly.
// It returns the new access token or an error.
type TokenRefresher func(ctx context.Context) (newAccessToken string, err error)

type Client struct {
	BaseURL        string
	AccessToken    string
	HTTPClient     *resty.Client
	TokenRefresher TokenRefresher
}

type Response struct {
	Success   bool            `json:"success"`
	Data      json.RawMessage `json:"data"`
	Error     *ErrorBody      `json:"error"`
	RequestID string          `json:"requestId"`
}

type ErrorBody struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Hint    string         `json:"hint,omitempty"`
	Details map[string]any `json:"details,omitempty"`
}

type UserProfile struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
}

func NewClient(baseURL, accessToken string) *Client {
	return &Client{
		BaseURL:     baseURL,
		AccessToken: accessToken,
		HTTPClient:  resty.New().SetTimeout(30 * time.Second).SetResponseBodyLimit(1 << 20),
	}
}

func (c *Client) do(ctx context.Context, method, path string, body io.Reader) (*Response, error) {
	// Buffer the request body so we can replay it on token refresh retry.
	var reqBody []byte
	if body != nil {
		var err error
		reqBody, err = io.ReadAll(body)
		if err != nil {
			return nil, &Error{
				Code:    CodeNetworkError,
				Message: fmt.Sprintf("reading request body: %s", err),
				Cause:   err,
			}
		}
	}

	envelope, statusCode, err := c.doOnce(ctx, method, path, reqBody)
	if err != nil && statusCode == http.StatusUnauthorized && c.TokenRefresher != nil {
		newToken, refreshErr := c.TokenRefresher(ctx)
		if refreshErr == nil && newToken != "" {
			c.AccessToken = newToken
			envelope, _, err = c.doOnce(ctx, method, path, reqBody)
		}
	}
	return envelope, err
}

func (c *Client) doOnce(ctx context.Context, method, path string, reqBody []byte) (*Response, int, error) {
	req := c.HTTPClient.R().
		SetContext(ctx).
		SetHeader("Authorization", "Bearer "+c.AccessToken).
		SetHeader("Accept", "application/json")

	if reqBody != nil {
		req.SetHeader("Content-Type", "application/json").
			SetBody(reqBody)
	}

	resp, err := req.Execute(method, c.BaseURL+path)
	if err != nil {
		return nil, 0, &Error{
			Code:    CodeNetworkError,
			Message: fmt.Sprintf("executing request: %s", err),
			Cause:   err,
		}
	}

	statusCode := resp.StatusCode()
	bodyBytes := resp.Body()

	var envelope Response
	if err := json.Unmarshal(bodyBytes, &envelope); err != nil {
		snippet := strings.TrimSpace(string(bodyBytes))
		if snippet == "" {
			snippet = http.StatusText(statusCode)
		}
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}
		return nil, statusCode, &Error{
			Code:       CodeUnexpectedResp,
			Message:    fmt.Sprintf("unexpected response (status %d): %s", statusCode, snippet),
			StatusCode: statusCode,
			Cause:      err,
		}
	}

	if !envelope.Success {
		code := CodeUnknownError
		msg := "unknown error"
		var hint string
		var details map[string]any
		if envelope.Error != nil {
			if envelope.Error.Code != "" {
				code = envelope.Error.Code
			}
			if envelope.Error.Message != "" {
				msg = envelope.Error.Message
			}
			hint = envelope.Error.Hint
			details = envelope.Error.Details
		}
		return nil, statusCode, &Error{
			Code:       code,
			Message:    msg,
			RequestID:  envelope.RequestID,
			StatusCode: statusCode,
			Hint:       hint,
			Details:    details,
		}
	}

	if statusCode < 200 || statusCode >= 300 {
		return nil, statusCode, &Error{
			Code:       CodeUnexpectedResp,
			Message:    fmt.Sprintf("unexpected status %d for successful response", statusCode),
			StatusCode: statusCode,
		}
	}

	return &envelope, statusCode, nil
}

func (c *Client) GetMe(ctx context.Context) (*UserProfile, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/me", nil)
	if err != nil {
		return nil, err
	}

	var profile UserProfile
	if err := json.Unmarshal(resp.Data, &profile); err != nil {
		return nil, fmt.Errorf("decoding user profile: %w", err)
	}
	return &profile, nil
}
