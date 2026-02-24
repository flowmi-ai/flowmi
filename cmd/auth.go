package cmd

import "github.com/spf13/cobra"

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate flowmi",
	Long:  "Manage flowmi's authentication state.",
	Example: `  flowmi auth login
  flowmi auth status`,
}

func init() {
	rootCmd.AddCommand(authCmd)
}
