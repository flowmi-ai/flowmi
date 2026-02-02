package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/flowmi/flowmi/internal/auth"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with your flowmi account",
	Long: `Opens a browser window to authenticate with your flowmi account using OAuth2.
After successful authentication, your credentials are saved to ~/.flowmi.toml.

Use --no-browser to print the login URL instead of opening the browser automatically.`,
	RunE: runLogin,
}

func init() {
	loginCmd.Flags().Bool("no-browser", false, "print the login URL instead of opening the browser")
	rootCmd.AddCommand(loginCmd)
}

func runLogin(cmd *cobra.Command, args []string) error {
	noBrowser, _ := cmd.Flags().GetBool("no-browser")
	serverURL := viper.GetString("auth_server_url")

	// Generate PKCE pair.
	verifier, challenge, err := auth.GeneratePKCE()
	if err != nil {
		return fmt.Errorf("generating PKCE: %w", err)
	}

	// Generate state.
	state, err := auth.GenerateState()
	if err != nil {
		return fmt.Errorf("generating state: %w", err)
	}

	// Start callback server with 2-minute timeout.
	ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
	defer cancel()

	port, resultCh, err := auth.StartCallbackServer(ctx)
	if err != nil {
		return fmt.Errorf("starting callback server: %w", err)
	}

	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", port)
	authorizeURL := auth.BuildAuthorizeURL(serverURL, redirectURI, state, challenge)

	if noBrowser {
		fmt.Fprintln(cmd.OutOrStdout(), "Open this URL in your browser to log in:")
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintln(cmd.OutOrStdout(), "  "+authorizeURL)
		fmt.Fprintln(cmd.OutOrStdout())
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "Opening browser to log in...")
		if err := auth.OpenBrowser(authorizeURL); err != nil {
			fmt.Fprintln(cmd.ErrOrStderr(), "Could not open browser. Open this URL manually:")
			fmt.Fprintln(cmd.ErrOrStderr(), "  "+authorizeURL)
		}
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Waiting for authentication...")

	// Wait for callback.
	select {
	case result := <-resultCh:
		if result.Err != nil {
			return fmt.Errorf("authentication failed: %w", result.Err)
		}

		// Validate state.
		if result.State != state {
			return fmt.Errorf("state mismatch: possible CSRF attack")
		}

		// Exchange code for tokens.
		tokenURL := serverURL + "/token"
		token, err := auth.ExchangeCode(ctx, tokenURL, result.Code, verifier, redirectURI)
		if err != nil {
			return fmt.Errorf("exchanging code for tokens: %w", err)
		}

		// Save tokens.
		if err := saveTokens(token); err != nil {
			return fmt.Errorf("saving tokens: %w", err)
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Login successful!")
		return nil

	case <-ctx.Done():
		return fmt.Errorf("login timed out — please try again")
	}
}

func saveTokens(token *auth.TokenResponse) error {
	viper.Set("access_token", token.AccessToken)
	viper.Set("refresh_token", token.RefreshToken)

	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("finding home directory: %w", err)
		}
		configFile = filepath.Join(home, ".flowmi.toml")
	}

	return viper.WriteConfigAs(configFile)
}
