package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/flowmi-ai/flowmi/internal/api"
	"github.com/flowmi-ai/flowmi/internal/auth"
	"github.com/flowmi-ai/flowmi/internal/config"
	"github.com/spf13/viper"
)

// resolveToken returns the best available auth token and its type.
// Priority: FLOWMI_API_KEY env → api_key in credentials → access_token.
func resolveToken() (token, method string) {
	if key := viper.GetString("api_key"); key != "" {
		return key, "api_key"
	}
	if tok := viper.GetString("access_token"); tok != "" {
		return tok, "token"
	}
	return "", ""
}

func newAPIClient() (*api.Client, error) {
	token, method := resolveToken()
	if token == "" {
		return nil, api.NewError(api.CodeAuthRequired, "not logged in").
			WithHint("Run 'flowmi auth login' to authenticate.")
	}

	apiServerURL := viper.GetString("api_server_url")
	client := api.NewClient(apiServerURL, token)

	if viper.GetBool("debug") {
		client.HTTPClient.SetDebug(true)
		auth.SetDebug(true)
	}

	// Only set up token refresh for OAuth2 access tokens.
	// API keys don't expire via the refresh flow.
	if method == "token" {
		client.TokenRefresher = func(ctx context.Context) (string, error) {
			refreshToken := viper.GetString("refresh_token")
			if refreshToken == "" {
				return "", fmt.Errorf("no refresh token available")
			}
			tokenURL := apiServerURL + "/api/v1/oauth2/refresh"

			tokens, err := auth.RefreshTokens(ctx, tokenURL, refreshToken)
			if err != nil {
				return "", err
			}

			// Persist the new tokens.
			profile := viper.GetString("profile")
			creds, _ := config.LoadCredentials(profile)
			if creds == nil {
				creds = map[string]string{}
			}
			creds["access_token"] = tokens.AccessToken
			if tokens.RefreshToken != "" {
				creds["refresh_token"] = tokens.RefreshToken
			}
			if err := config.SaveCredentials(profile, creds); err != nil {
				return "", fmt.Errorf("saving refreshed credentials: %w", err)
			}

			// Update viper so subsequent calls in the same session use the new token.
			viper.Set("access_token", tokens.AccessToken)
			if tokens.RefreshToken != "" {
				viper.Set("refresh_token", tokens.RefreshToken)
			}

			return tokens.AccessToken, nil
		}
	}

	return client, nil
}

// truncate truncates s to maxRunes runes, appending "..." if truncated.
func truncate(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes-3]) + "..."
}

// formatLabels formats a label slice for display.
func formatLabels(labels []string) string {
	if len(labels) == 0 {
		return "(none)"
	}
	return strings.Join(labels, ", ")
}
