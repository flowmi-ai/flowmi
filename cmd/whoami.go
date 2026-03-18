package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/flowmi-ai/flowmi/internal/api"
	"github.com/flowmi-ai/flowmi/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	Long:  "Display information about the currently authenticated user.",
	RunE:  runAuthStatus,
}

func init() {
	authCmd.AddCommand(authStatusCmd)
}

func runAuthStatus(cmd *cobra.Command, args []string) error {
	client, err := newAPIClient()
	if err != nil {
		return err
	}

	profile, err := client.GetMe(cmd.Context())
	if err != nil {
		return fmt.Errorf("fetching user info: %w", err)
	}

	balance, err := client.GetBalance(cmd.Context())
	if err != nil {
		return fmt.Errorf("fetching credit balance: %w", err)
	}

	if viper.GetBool("json") {
		return printJSON(cmd, profile, balance.Balance)
	}
	return printText(cmd, profile, balance.Balance)
}

// authMethodLabel returns a human-readable label for the current auth method.
func authMethodLabel() string {
	_, method := resolveToken()
	switch method {
	case "api_key":
		return "API key"
	case "token":
		return "OAuth2 token"
	default:
		return "unknown"
	}
}

func printJSON(cmd *cobra.Command, profile *api.UserProfile, credits int64) error {
	combined := struct {
		ID        string    `json:"id"`
		Email     string    `json:"email"`
		CreatedAt time.Time `json:"createdAt"`
		Credits   int64     `json:"credits"`
		Auth      string    `json:"auth"`
	}{
		ID:        profile.ID,
		Email:     profile.Email,
		CreatedAt: profile.CreatedAt,
		Credits:   credits,
		Auth:      authMethodLabel(),
	}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(combined)
}

func printText(cmd *cobra.Command, profile *api.UserProfile, credits int64) error {
	w := cmd.OutOrStdout()

	label := lipgloss.NewStyle().Faint(true)
	value := lipgloss.NewStyle().Bold(true)

	fmt.Fprintln(w, "Logged in as")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  %s  %s\n", label.Render("Email:"), value.Render(profile.Email))
	fmt.Fprintf(w, "  %s     %s\n", label.Render("ID:"), value.Render(profile.ID))
	fmt.Fprintf(w, "  %s  %s\n", label.Render("Since:"), value.Render(profile.CreatedAt.Format("2006-01-02")))
	fmt.Fprintf(w, "  %s %s\n", label.Render("Credits:"), value.Render(formatInt(credits)))
	fmt.Fprintf(w, "  %s    %s\n", label.Render("Auth:"), value.Render(authMethodLabel()))

	credsPath, err := config.CredentialsFilePath()
	if err == nil {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "  %s %s\n", label.Render("Credentials:"), credsPath)
	}

	return nil
}

// formatInt formats an integer with comma separators (e.g., 1200 → "1,200").
func formatInt(n int64) string {
	negative := n < 0
	if negative {
		n = -n
	}
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		if negative {
			return "-" + s
		}
		return s
	}

	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	if negative {
		return "-" + string(result)
	}
	return string(result)
}
