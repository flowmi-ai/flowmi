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

func newAPIClient() (*api.Client, error) {
	accessToken := viper.GetString("access_token")
	if accessToken == "" {
		return nil, api.NewError(api.CodeAuthRequired, "not logged in").
			WithHint("Run 'flowmi auth login' to authenticate.")
	}
	apiServerURL := viper.GetString("api_server_url")
	client := api.NewClient(apiServerURL, accessToken)

	if viper.GetBool("debug") {
		client.HTTPClient.SetDebug(true)
		auth.SetDebug(true)
	}

	// Set up automatic token refresh on 401.
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
		creds, _ := config.LoadCredentials()
		if creds == nil {
			creds = map[string]string{}
		}
		creds["access_token"] = tokens.AccessToken
		if tokens.RefreshToken != "" {
			creds["refresh_token"] = tokens.RefreshToken
		}
		if err := config.SaveCredentials(creds); err != nil {
			return "", fmt.Errorf("saving refreshed credentials: %w", err)
		}

		// Update viper so subsequent calls in the same session use the new token.
		viper.Set("access_token", tokens.AccessToken)
		if tokens.RefreshToken != "" {
			viper.Set("refresh_token", tokens.RefreshToken)
		}

		return tokens.AccessToken, nil
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
