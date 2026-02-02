package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/flowmi/flowmi/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure API key interactively",
	Run: func(cmd *cobra.Command, args []string) {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, ui.ErrorStyle.Render(err.Error()))
			os.Exit(1)
		}
		configPath := filepath.Join(home, ".flowmi.toml")
		if err := runConfigure(os.Stdin, configPath); err != nil {
			fmt.Fprintln(os.Stderr, ui.ErrorStyle.Render(err.Error()))
			os.Exit(1)
		}
	},
}

// runConfigure reads API key from reader and saves to configPath.
func runConfigure(in io.Reader, configPath string) error {
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

	viper.Set("api_key", key)

	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		err = viper.SafeWriteConfigAs(configPath)
	} else {
		err = viper.WriteConfigAs(configPath)
	}
	if err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Println(ui.SuccessStyle.Render("✓ Configuration saved to " + configPath))
	return nil
}

func init() {
	rootCmd.AddCommand(configureCmd)
}
