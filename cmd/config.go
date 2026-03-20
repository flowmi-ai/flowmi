package cmd

import (
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/flowmi-ai/flowmi/internal/api"
	"github.com/flowmi-ai/flowmi/internal/config"
	"github.com/flowmi-ai/flowmi/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// knownConfigKeys lists all recognized configuration keys and where they are stored.
var knownConfigKeys = map[string]string{
	"api_key":         "credential",
	"auth_server_url": "config",
	"api_server_url":  "config",
}

// sensitiveKeys are credential keys whose values should be masked in output.
var sensitiveKeys = map[string]struct{}{
	"api_key": {},
}

// maskValue masks a sensitive value, showing only the first 4 characters.
func maskValue(value string) string {
	if len(value) <= 4 {
		return strings.Repeat("*", len(value))
	}
	return value[:4] + "****"
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

var configUseCmd = &cobra.Command{
	Use:   "use <profile>",
	Short: "Switch the active profile",
	Example: `  flowmi config use local
  flowmi config use prod`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigUse,
}

func init() {
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configUseCmd)
	rootCmd.AddCommand(configCmd)
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key, value := args[0], args[1]
	profile := viper.GetString("profile")

	store, ok := knownConfigKeys[key]
	if !ok {
		return api.NewError(api.CodeConfigNotFound, fmt.Sprintf("unknown config key: %s", key))
	}

	switch store {
	case "credential":
		creds, err := config.LoadCredentials(profile)
		if err != nil {
			return fmt.Errorf("loading credentials: %w", err)
		}
		creds[key] = value
		if err := config.SaveCredentials(profile, creds); err != nil {
			return fmt.Errorf("saving credentials: %w", err)
		}
	case "config":
		cfg, err := config.LoadConfigProfile(profile)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		cfg[key] = value
		if err := config.SaveConfigProfile(profile, cfg); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}
	}

	msg := fmt.Sprintf(`Set "%s" to "%s"`, key, value)
	if profiles, _, _ := config.ListProfiles(); len(profiles) > 1 {
		msg = fmt.Sprintf(`Set "%s" to "%s" (profile: %s)`, key, value, profile)
	}
	fmt.Fprintln(cmd.OutOrStdout(), ui.SuccessStyle.Render(msg))
	return nil
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	key := args[0]
	if _, ok := knownConfigKeys[key]; !ok {
		return api.NewError(api.CodeConfigNotFound, fmt.Sprintf("unknown config key: %s", key))
	}

	value := viper.GetString(key)
	fmt.Fprintln(cmd.OutOrStdout(), value)
	return nil
}

func runConfigList(cmd *cobra.Command, args []string) error {
	profile := viper.GetString("profile")

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)

	// Only show profile header when multiple profiles exist.
	profiles, _, _ := config.ListProfiles()
	if len(profiles) > 1 {
		fmt.Fprintf(w, "Profile: %s\n\n", ui.TitleStyle.Render(profile))
	}

	fmt.Fprintln(w, "KEY\tVALUE\tSOURCE")

	keys := make([]string, 0, len(knownConfigKeys))
	for k := range knownConfigKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		store := knownConfigKeys[key]
		value := viper.GetString(key)
		source := store
		if value == "" {
			source = "default"
		}
		if _, sensitive := sensitiveKeys[key]; sensitive && value != "" {
			value = maskValue(value)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", key, value, source)
	}

	// Show available profiles only when multiple exist.
	if len(profiles) > 1 {
		sort.Strings(profiles)
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Profiles:")
		for _, p := range profiles {
			marker := "  "
			if p == profile {
				marker = "* "
			}
			fmt.Fprintf(w, "  %s%s\n", marker, p)
		}
	}

	return w.Flush()
}

func runConfigUse(cmd *cobra.Command, args []string) error {
	profile := args[0]

	// Warn if the profile has no config or credentials.
	cfg, _ := config.LoadConfigProfile(profile)
	creds, _ := config.LoadCredentials(profile)
	if len(cfg) == 0 && len(creds) == 0 {
		fmt.Fprintln(cmd.ErrOrStderr(), ui.WarningStyle.Render(
			fmt.Sprintf("Warning: profile %q has no config or credentials yet", profile)))
	}

	if err := config.SetCurrentProfile(profile); err != nil {
		return fmt.Errorf("setting profile: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), ui.SuccessStyle.Render(fmt.Sprintf("Switched to profile %q", profile)))
	return nil
}
