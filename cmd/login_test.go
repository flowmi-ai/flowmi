package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flowmi/flowmi/internal/auth"
	"github.com/spf13/viper"
)

func TestLoginFlow(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Set up a mock API server with /api/v1/oauth2/login and /api/v1/oauth2/token.
	mockAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/oauth2/login":
			var req auth.LoginRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("decoding login request: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if req.Email != "test@example.com" {
				t.Errorf("email = %q, want test@example.com", req.Email)
			}
			if req.Password != "testpass" {
				t.Errorf("password = %q, want testpass", req.Password)
			}
			if req.ClientID != "flowmi-cli" {
				t.Errorf("client_id = %q, want flowmi-cli", req.ClientID)
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"data": auth.LoginResponse{
					Code:        "mock_code",
					RedirectURI: req.RedirectURI,
					State:       req.State,
				},
			})
		case "/api/v1/oauth2/token":
			if err := r.ParseForm(); err != nil {
				t.Errorf("ParseForm: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if r.FormValue("grant_type") != "authorization_code" {
				t.Errorf("grant_type = %q", r.FormValue("grant_type"))
			}
			if r.FormValue("code") != "mock_code" {
				t.Errorf("code = %q", r.FormValue("code"))
			}
			if r.FormValue("client_id") != "flowmi-cli" {
				t.Errorf("client_id = %q", r.FormValue("client_id"))
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(auth.TokenResponse{
				AccessToken:  "test_access_token",
				RefreshToken: "test_refresh_token",
				TokenType:    "Bearer",
				ExpiresIn:    3600,
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockAPI.Close()

	// Point viper at the mock server.
	viper.Set("api_server_url", mockAPI.URL)
	defer viper.Set("api_server_url", "")

	// Execute the login command with flags (non-interactive).
	rootCmd.SetArgs([]string{"auth", "login", "--email", "test@example.com", "--password", "testpass"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("login command failed: %v", err)
	}

	// Verify credentials.toml was written.
	credsPath := filepath.Join(tmpDir, "flowmi", "credentials.toml")
	data, err := os.ReadFile(credsPath)
	if err != nil {
		t.Fatalf("reading credentials.toml: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "test_access_token") {
		t.Errorf("credentials.toml missing access_token, got:\n%s", content)
	}
	if !strings.Contains(content, "test_refresh_token") {
		t.Errorf("credentials.toml missing refresh_token, got:\n%s", content)
	}
}

func TestLoginCmdHelp(t *testing.T) {
	t.Cleanup(func() { resetHelpFlags(rootCmd) })
	t.Cleanup(resetHelpState)
	rootCmd.SetArgs([]string{"auth", "login", "--help"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("login --help failed: %v", err)
	}
}

func TestLoginCmdHasFlags(t *testing.T) {
	f := loginCmd.Flags().Lookup("email")
	if f == nil {
		t.Fatal("--email flag not found")
	}

	f = loginCmd.Flags().Lookup("password")
	if f == nil {
		t.Fatal("--password flag not found")
	}
}
