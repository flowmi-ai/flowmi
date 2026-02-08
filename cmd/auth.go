package cmd

import "github.com/spf13/cobra"

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate flowmi",
	Long:  "Manage flowmi's authentication state.",
}

func init() {
	rootCmd.AddCommand(authCmd)
}
