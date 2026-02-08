package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/flowmi/flowmi/internal/config"
	"github.com/flowmi/flowmi/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// knownConfigKeys lists all recognized configuration keys and where they are stored.
var knownConfigKeys = map[string]string{
	"api_key":         "credential",
	"auth_server_url": "config",
	"api_server_url":  "config",
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE:  runConfigSet,
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfigGet,
}

var configListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all configuration values",
	Aliases: []string{"ls"},
	Args:    cobra.NoArgs,
	RunE:    runConfigList,
}

func init() {
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configListCmd)
	rootCmd.AddCommand(configCmd)
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key, value := args[0], args[1]

	store, ok := knownConfigKeys[key]
	if !ok {
		return fmt.Errorf("unknown config key: %s", key)
	}

	switch store {
	case "credential":
		creds, err := config.LoadCredentials()
		if err != nil {
			return fmt.Errorf("loading credentials: %w", err)
		}
		creds[key] = value
		if err := config.SaveCredentials(creds); err != nil {
			return fmt.Errorf("saving credentials: %w", err)
		}
	case "config":
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		cfg[key] = value
		if err := config.SaveConfig(cfg); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}
	}

	fmt.Fprintln(cmd.OutOrStdout(), ui.SuccessStyle.Render(fmt.Sprintf(`Set "%s" to "%s"`, key, value)))
	return nil
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	key := args[0]
	if _, ok := knownConfigKeys[key]; !ok {
		return fmt.Errorf("unknown config key: %s", key)
	}

	value := viper.GetString(key)
	fmt.Fprintln(cmd.OutOrStdout(), value)
	return nil
}

func runConfigList(cmd *cobra.Command, args []string) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "KEY\tVALUE\tSOURCE")
	for key, store := range knownConfigKeys {
		value := viper.GetString(key)
		source := store
		if value == "" {
			source = "default"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", key, value, source)
	}
	return w.Flush()
}
