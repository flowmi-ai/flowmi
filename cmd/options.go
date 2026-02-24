package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var optionsCmd = &cobra.Command{
	Use:   "options",
	Short: "Show global flags",
	Long:  "Show global flags used across all commands.",
	Example: `  flowmi options
  flowmi options --format json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		format, _ := cmd.Flags().GetString("format")
		if strings.EqualFold(format, "json") {
			payload := struct {
				Command string     `json:"command"`
				Flags   []helpFlag `json:"flags"`
			}{
				Command: cmd.CommandPath(),
				Flags:   collectRootGlobalFlags(rootCmd),
			}
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(payload)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Global Flags:")
		fmt.Fprintln(cmd.OutOrStdout(), rootCmd.PersistentFlags().FlagUsagesWrapped(80))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(optionsCmd)
}
