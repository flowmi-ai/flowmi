package cmd

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/flowmi/flowmi/internal/config"
	"github.com/flowmi/flowmi/internal/ui"
	"github.com/spf13/cobra"
)

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure API key interactively",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runConfigure(cmd.InOrStdin())
	},
}

// runConfigure reads API key from reader and saves to credentials.toml.
func runConfigure(in io.Reader) error {
	fmt.Print("Enter your API key: ")

	reader := bufio.NewReader(in)
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return fmt.Errorf("failed to read input: %w", err)
	}

	key := strings.TrimSpace(line)
	if key == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	creds, err := config.LoadCredentials()
	if err != nil {
		return fmt.Errorf("loading credentials: %w", err)
	}

	creds["api_key"] = key

	if err := config.SaveCredentials(creds); err != nil {
		return fmt.Errorf("saving credentials: %w", err)
	}

	credsPath, _ := config.CredentialsFilePath()
	fmt.Println(ui.SuccessStyle.Render("✓ Configuration saved to " + credsPath))
	return nil
}

func init() {
	rootCmd.AddCommand(configureCmd)
}
