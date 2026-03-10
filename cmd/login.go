package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/flowmi-ai/flowmi/internal/auth"
	"github.com/flowmi-ai/flowmi/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with your flowmi account",
	Long: `Authenticate with your flowmi account.

By default, opens a browser window for OAuth2 login (supports social login).
Use --email and --password flags for direct email/password login (e.g. CI/CD).

Use --no-browser to print the login URL instead of opening the browser automatically.`,
	Example: `  flowmi auth login
  flowmi auth login --no-browser
  flowmi auth login --email test@example.com --password "$FLOWMI_PASSWORD"`,
	RunE: runLogin,
}

func init() {
	loginCmd.Flags().String("email", "", "email address for direct login (skips browser)")
	loginCmd.Flags().String("password", "", "password for direct login (used with --email)")
	loginCmd.Flags().Bool("no-browser", false, "print the login URL instead of opening the browser")
	authCmd.AddCommand(loginCmd)
}

func runLogin(cmd *cobra.Command, args []string) error {
	email, _ := cmd.Flags().GetString("email")
	if email != "" {
		return passwordLogin(cmd, email)
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

// passwordLogin authenticates directly via email/password (CI/CD flow).
func passwordLogin(cmd *cobra.Command, email string) error {
	apiServerURL := viper.GetString("api_server_url")

	// Get password.
	password, _ := cmd.Flags().GetString("password")
	if password == "" {
		fmt.Fprint(cmd.OutOrStdout(), "Password: ")
		raw, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(cmd.OutOrStdout()) // newline after hidden input
		if err != nil {
			return fmt.Errorf("reading password: %w", err)
		}
		password = string(raw)
		if password == "" {
			return fmt.Errorf("password is required")
		}
	}

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

	// Login: send credentials to get auth code.
	loginURL := apiServerURL + "/api/v1/oauth2/login"
	loginResp, err := auth.Login(cmd.Context(), loginURL, &auth.LoginRequest{
		Email:               email,
		Password:            password,
		ClientID:            "flowmi-cli",
		RedirectURI:         auth.PlaceholderRedirectURI,
		ResponseType:        "code",
		CodeChallenge:       challenge,
		CodeChallengeMethod: "S256",
		State:               state,
	})
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	// Validate state.
	if loginResp.State != state {
		return fmt.Errorf("state mismatch: possible CSRF attack")
	}

	// Exchange code for tokens.
	tokenURL := apiServerURL + "/api/v1/oauth2/token"
	token, err := auth.ExchangeCode(cmd.Context(), tokenURL, loginResp.Code, verifier, auth.PlaceholderRedirectURI)
	if err != nil {
		return fmt.Errorf("exchanging code for tokens: %w", err)
	}

	// Save tokens.
	if err := saveTokens(token); err != nil {
		return fmt.Errorf("saving tokens: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Login successful!")
	return nil
}

func saveTokens(token *auth.TokenResponse) error {
	creds, err := config.LoadCredentials()
	if err != nil {
		return err
	}

	creds["access_token"] = token.AccessToken
	creds["refresh_token"] = token.RefreshToken

	return config.SaveCredentials(creds)
}
