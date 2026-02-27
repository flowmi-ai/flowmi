package api

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrorMessage(t *testing.T) {
	e := NewError(CodeAuthUnauthorized, "invalid token")
	want := CodeAuthUnauthorized + ": invalid token"
	if got := e.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestErrorMessageNoCode(t *testing.T) {
	e := &Error{Message: "something failed"}
	if got := e.Error(); got != "something failed" {
		t.Errorf("Error() = %q, want something failed", got)
	}
}

func TestErrorUnwrap(t *testing.T) {
	cause := fmt.Errorf("connection refused")
	e := NewErrorFrom(CodeNetworkError, "request failed", cause)
	if e.Unwrap() != cause {
		t.Error("Unwrap() did not return the original cause")
	}
}

func TestErrorUnwrapNil(t *testing.T) {
	e := NewError("TEST", "no cause")
	if e.Unwrap() != nil {
		t.Error("Unwrap() should return nil when no cause")
	}
}

func TestErrorsAsThroughFmtErrorf(t *testing.T) {
	original := NewError(CodeResourceNotFound, "note not found")
	original.RequestID = "req_123"

	wrapped := fmt.Errorf("getting note: %w", original)
	doubleWrapped := fmt.Errorf("command failed: %w", wrapped)

	var apiErr *Error
	if !errors.As(doubleWrapped, &apiErr) {
		t.Fatal("errors.As should find *api.Error through wrapping chain")
	}
	if apiErr.Code != CodeResourceNotFound {
		t.Errorf("Code = %q, want %q", apiErr.Code, CodeResourceNotFound)
	}
	if apiErr.RequestID != "req_123" {
		t.Errorf("RequestID = %q, want req_123", apiErr.RequestID)
	}
}

func TestExitCodeByPrefix(t *testing.T) {
	cases := []struct {
		code string
		want int
	}{
		// AUTH_ → ExitAuth
		{CodeAuthRequired, ExitAuth},
		{CodeAuthUnauthorized, ExitAuth},
		{CodeAuthForbidden, ExitAuth},
		{CodeAuthTokenExpired, ExitAuth},
		{CodeAuthInvalidToken, ExitAuth},

		// NETWORK_ → ExitNetwork
		{CodeNetworkError, ExitNetwork},
		{CodeNetworkTimeout, ExitNetwork},

		// VALIDATION_ → ExitUsage
		{CodeValidationError, ExitUsage},
		{CodeValidationBadRequest, ExitUsage},
		{CodeValidationInvalidInput, ExitUsage},

		// SERVER_ → ExitServer
		{CodeServerInternal, ExitServer},
		{CodeServerUnavailable, ExitServer},

		// No matching prefix → ExitBusiness
		{CodeResourceNotFound, ExitBusiness},
		{CodeResourceConflict, ExitBusiness},
		{CodeRateLimitExceeded, ExitBusiness},
		{CodeCommandError, ExitBusiness},
		{CodeConfigNotFound, ExitBusiness},
		{CodeUnknownError, ExitBusiness},
	}
	for _, tc := range cases {
		e := NewError(tc.code, "test")
		if got := e.ExitCode(); got != tc.want {
			t.Errorf("ExitCode(%q) = %d, want %d", tc.code, got, tc.want)
		}
	}
}

func TestExitCodeHTTPStatusFallback(t *testing.T) {
	cases := []struct {
		name       string
		statusCode int
		want       int
	}{
		{"401", 401, ExitAuth},
		{"403", 403, ExitAuth},
		{"500", 500, ExitServer},
		{"502", 502, ExitServer},
		{"404", 404, ExitBusiness},
		{"unknown", 0, ExitBusiness},
	}
	for _, tc := range cases {
		e := &Error{Code: "SOME_UNKNOWN_CODE", Message: "test", StatusCode: tc.statusCode}
		if got := e.ExitCode(); got != tc.want {
			t.Errorf("ExitCode(status %d) = %d, want %d", tc.statusCode, got, tc.want)
		}
	}
}

func TestWithHint(t *testing.T) {
	e := NewError(CodeAuthRequired, "not logged in").
		WithHint("Run 'flowmi auth login'.")
	if e.Hint != "Run 'flowmi auth login'." {
		t.Errorf("Hint = %q, want Run 'flowmi auth login'.", e.Hint)
	}
}

func TestWithDetails(t *testing.T) {
	details := map[string]any{"field": "email", "reason": "required"}
	e := NewError(CodeValidationError, "invalid input").WithDetails(details)
	if e.Details["field"] != "email" {
		t.Errorf("Details[field] = %v, want email", e.Details["field"])
	}
}

func TestNewErrorFrom(t *testing.T) {
	cause := fmt.Errorf("dial tcp: connection refused")
	e := NewErrorFrom(CodeNetworkError, "request failed", cause)

	if e.Code != CodeNetworkError {
		t.Errorf("Code = %q, want %q", e.Code, CodeNetworkError)
	}
	if e.Cause != cause {
		t.Error("Cause should be the original error")
	}
	if !errors.Is(e, cause) {
		t.Error("errors.Is should match through Unwrap")
	}
}
