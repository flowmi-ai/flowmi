package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/flowmi-ai/flowmi/internal/api"
	"github.com/flowmi-ai/flowmi/internal/auth"
	"github.com/flowmi-ai/flowmi/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with your flowmi account",
	Long: `Authenticate with your flowmi account.

By default, opens a browser window for OAuth2 login (supports social login).
Use --with-token to provide a setup token (fst_) or API key (flk_) directly or via stdin.

Use --no-browser to print the login URL instead of opening the browser automatically.`,
	Example: `  flowmi auth login
  flowmi auth login --no-browser
  flowmi auth login --with-token fst_...
  flowmi auth login --with-token flk_...
  echo "fst_..." | flowmi auth login --with-token`,
	RunE: runLogin,
}

func init() {
	loginCmd.Flags().Bool("no-browser", false, "print the login URL instead of opening the browser")
	loginCmd.Flags().String("with-token", "", "setup token (fst_) or API key (flk_); reads from stdin if no value given")
	authCmd.AddCommand(loginCmd)
}

func runLogin(cmd *cobra.Command, args []string) error {
	if cmd.Flags().Changed("with-token") {
		return tokenLogin(cmd)
	}
	return browserLogin(cmd)
}

// browserLogin opens a browser for OAuth2 PKCE login (default flow).
func browserLogin(cmd *cobra.Command) error {
	noBrowser, _ := cmd.Flags().GetBool("no-browser")
	authServerURL := viper.GetString("auth_server_url")
	apiServerURL := viper.GetString("api_server_url")

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
	authorizeURL := auth.BuildAuthorizeURL(authServerURL, redirectURI, state, challenge)

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
		tokenURL := apiServerURL + "/api/v1/oauth2/token"
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

// tokenLogin authenticates with a setup token (fst_) or API key (flk_).
// Reads from the --with-token flag value, or falls back to stdin.
func tokenLogin(cmd *cobra.Command) error {
	apiServerURL := viper.GetString("api_server_url")

	// Try flag value first, fall back to stdin.
	token, _ := cmd.Flags().GetString("with-token")
	token = strings.TrimSpace(token)
	if token == "" {
		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("reading token from stdin: %w", err)
			}
			return fmt.Errorf("no token provided")
		}
		token = strings.TrimSpace(scanner.Text())
		if token == "" {
			return fmt.Errorf("no token provided")
		}
	}

	switch {
	case strings.HasPrefix(token, "fst_"):
		// Setup token → exchange for API key.
		exchangeURL := apiServerURL + "/api/v1/setup-tokens/exchange"
		apiKey, err := auth.ExchangeSetupToken(cmd.Context(), exchangeURL, token)
		if err != nil {
			return err
		}
		if err := saveAPIKey(apiKey); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Login successful! API key saved.")

	case strings.HasPrefix(token, "flk_"):
		// API key → verify and save.
		client := api.NewClient(apiServerURL, token)
		if viper.GetBool("debug") {
			client.HTTPClient.SetDebug(true)
		}
		if _, err := client.GetMe(cmd.Context()); err != nil {
			return fmt.Errorf("API key verification failed: %w", err)
		}
		if err := saveAPIKey(token); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Login successful! API key saved.")

	default:
		return fmt.Errorf("unrecognized token format: expected fst_ (setup token) or flk_ (API key)")
	}

	return nil
}

func saveAPIKey(apiKey string) error {
	profile := viper.GetString("profile")

	// Clear OAuth2 tokens — API key auth replaces token-based auth.
	if err := config.DeleteCredentialKeys(profile, "access_token", "refresh_token"); err != nil {
		return fmt.Errorf("clearing old tokens: %w", err)
	}

	creds, err := config.LoadCredentials(profile)
	if err != nil {
		return fmt.Errorf("loading credentials: %w", err)
	}
	creds["api_key"] = apiKey
	if err := config.SaveCredentials(profile, creds); err != nil {
		return fmt.Errorf("saving API key: %w", err)
	}
	viper.Set("api_key", apiKey)
	return nil
}

func saveTokens(token *auth.TokenResponse) error {
	profile := viper.GetString("profile")

	// Clear API key — OAuth2 auth replaces key-based auth.
	if err := config.DeleteCredentialKeys(profile, "api_key"); err != nil {
		return fmt.Errorf("clearing old API key: %w", err)
	}

	creds, err := config.LoadCredentials(profile)
	if err != nil {
		return err
	}

	creds["access_token"] = token.AccessToken
	creds["refresh_token"] = token.RefreshToken

	return config.SaveCredentials(profile, creds)
}
