package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"text/tabwriter"
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

	output := viper.GetString("output")
	switch output {
	case "json":
		return printJSON(cmd, profile, balance.Balance)
	case "table":
		return printTable(cmd, profile, balance.Balance)
	case "text", "":
		return printText(cmd, profile, balance.Balance)
	default:
		return fmt.Errorf("unsupported output format: %s", output)
	}
}

func printJSON(cmd *cobra.Command, profile *api.UserProfile, credits int64) error {
	combined := struct {
		ID        string    `json:"id"`
		Email     string    `json:"email"`
		CreatedAt time.Time `json:"createdAt"`
		Credits   int64     `json:"credits"`
	}{
		ID:        profile.ID,
		Email:     profile.Email,
		CreatedAt: profile.CreatedAt,
		Credits:   credits,
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

	credsPath, err := config.CredentialsFilePath()
	if err == nil {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "  %s %s\n", label.Render("Credentials:"), credsPath)
	}

	return nil
}

func printTable(cmd *cobra.Command, profile *api.UserProfile, credits int64) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)

	fmt.Fprintln(w, "FIELD\tVALUE")
	fmt.Fprintf(w, "Email\t%s\n", profile.Email)
	fmt.Fprintf(w, "ID\t%s\n", profile.ID)
	fmt.Fprintf(w, "Since\t%s\n", profile.CreatedAt.Format("2006-01-02"))
	fmt.Fprintf(w, "Credits\t%s\n", formatInt(credits))

	credsPath, err := config.CredentialsFilePath()
	if err == nil {
		fmt.Fprintf(w, "Credentials\t%s\n", credsPath)
	}

	return w.Flush()
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
