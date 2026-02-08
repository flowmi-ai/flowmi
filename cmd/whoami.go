package cmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/flowmi/flowmi/internal/api"
	"github.com/flowmi/flowmi/internal/config"
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
	accessToken := viper.GetString("access_token")
	if accessToken == "" {
		return fmt.Errorf("not logged in. run 'flowmi auth login' to get started")
	}

	apiServerURL := viper.GetString("api_server_url")
	client := api.NewClient(apiServerURL, accessToken)

	profile, err := client.GetMe(cmd.Context())
	if err != nil {
		return fmt.Errorf("fetching user info: %w", err)
	}

	output := viper.GetString("output")
	switch output {
	case "json":
		return printJSON(cmd, profile)
	case "table":
		return printTable(cmd, profile)
	case "text", "":
		return printText(cmd, profile)
	default:
		return fmt.Errorf("unsupported output format: %s", output)
	}
}

func printJSON(cmd *cobra.Command, profile *api.UserProfile) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(profile)
}

func printText(cmd *cobra.Command, profile *api.UserProfile) error {
	w := cmd.OutOrStdout()

	label := lipgloss.NewStyle().Faint(true)
	value := lipgloss.NewStyle().Bold(true)

	fmt.Fprintln(w, "Logged in as")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "  %s  %s\n", label.Render("Email:"), value.Render(profile.Email))
	fmt.Fprintf(w, "  %s     %s\n", label.Render("ID:"), value.Render(profile.ID))
	fmt.Fprintf(w, "  %s  %s\n", label.Render("Since:"), value.Render(profile.CreatedAt.Format("2006-01-02")))

	credsPath, err := config.CredentialsFilePath()
	if err == nil {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "  %s %s\n", label.Render("Credentials:"), credsPath)
	}

	return nil
}

func printTable(cmd *cobra.Command, profile *api.UserProfile) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)

	fmt.Fprintln(w, "FIELD\tVALUE")
	fmt.Fprintf(w, "Email\t%s\n", profile.Email)
	fmt.Fprintf(w, "ID\t%s\n", profile.ID)
	fmt.Fprintf(w, "Since\t%s\n", profile.CreatedAt.Format("2006-01-02"))

	credsPath, err := config.CredentialsFilePath()
	if err == nil {
		fmt.Fprintf(w, "Credentials\t%s\n", credsPath)
	}

	return w.Flush()
}
