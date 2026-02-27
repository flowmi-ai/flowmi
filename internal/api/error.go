package api

import (
	"fmt"
	"strings"
)

// CLI-originated error codes.
const (
	CodeAuthRequired   = "AUTH_REQUIRED"
	CodeCommandError   = "COMMAND_ERROR"
	CodeConfigNotFound = "CONFIG_KEY_NOT_FOUND"
	CodeNetworkError   = "NETWORK_ERROR"
	CodeUnexpectedResp = "UNEXPECTED_RESPONSE"
	CodeUnknownError   = "UNKNOWN_ERROR"
)

// Server-originated error codes — must match the server's envelope values exactly.
const (
	// AUTH_* — authentication / authorization failures.
	CodeAuthUnauthorized = "AUTH_UNAUTHORIZED"
	CodeAuthForbidden    = "AUTH_FORBIDDEN"
	CodeAuthTokenExpired = "AUTH_TOKEN_EXPIRED"
	CodeAuthInvalidToken = "AUTH_INVALID_TOKEN"

	// VALIDATION_* — request validation failures.
	CodeValidationError        = "VALIDATION_ERROR"
	CodeValidationBadRequest   = "VALIDATION_BAD_REQUEST"
	CodeValidationInvalidInput = "VALIDATION_INVALID_INPUT"

	// NETWORK_* — network-level failures.
	CodeNetworkTimeout = "NETWORK_TIMEOUT"

	// SERVER_* — server-side failures.
	CodeServerInternal    = "SERVER_INTERNAL"
	CodeServerUnavailable = "SERVER_UNAVAILABLE"

	// RESOURCE_* — resource state errors.
	CodeResourceNotFound = "RESOURCE_NOT_FOUND"
	CodeResourceConflict = "RESOURCE_CONFLICT"

	// RATE_* — rate limiting.
	CodeRateLimitExceeded = "RATE_LIMIT_EXCEEDED"
)

// Exit codes for structured errors.
const (
	ExitSuccess  = 0
	ExitBusiness = 1
	ExitUsage    = 2
	ExitAuth     = 3
	ExitNetwork  = 4
	ExitServer   = 5
)

// Error is a structured API error that preserves the server's error envelope
// fields (code, message, requestId, hint, details) through Go's error chain.
type Error struct {
	Code       string         `json:"code"`
	Message    string         `json:"message"`
	RequestID  string         `json:"requestId,omitempty"`
	StatusCode int            `json:"-"`
	Hint       string         `json:"hint,omitempty"`
	Details    map[string]any `json:"details,omitempty"`
	Cause      error          `json:"-"`
}

func (e *Error) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Cause
}

// ExitCode maps the error code prefix to a CLI exit code.
// The CATEGORY_DETAIL convention means prefix matching is sufficient —
// adding new server codes under an existing prefix works automatically.
func (e *Error) ExitCode() int {
	switch {
	case strings.HasPrefix(e.Code, "AUTH_"):
		return ExitAuth
	case strings.HasPrefix(e.Code, "NETWORK_"):
		return ExitNetwork
	case strings.HasPrefix(e.Code, "VALIDATION_"):
		return ExitUsage
	case strings.HasPrefix(e.Code, "SERVER_"):
		return ExitServer
	}

	// Fall back to HTTP status ranges for unknown code prefixes.
	switch {
	case e.StatusCode == 401 || e.StatusCode == 403:
		return ExitAuth
	case e.StatusCode >= 500:
		return ExitServer
	default:
		return ExitBusiness
	}
}

// NewError creates a new structured Error with the given code and message.
func NewError(code, message string) *Error {
	return &Error{Code: code, Message: message}
}

// NewErrorFrom creates a new structured Error wrapping a cause.
func NewErrorFrom(code, message string, cause error) *Error {
	return &Error{Code: code, Message: message, Cause: cause}
}

// WithHint returns the error with an added hint.
func (e *Error) WithHint(hint string) *Error {
	e.Hint = hint
	return e
}

// WithDetails returns the error with added details.
func (e *Error) WithDetails(details map[string]any) *Error {
	e.Details = details
	return e
}
