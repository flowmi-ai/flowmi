package cmd

import (
	"fmt"

	"github.com/flowmi-ai/flowmi/internal/api"
	"github.com/spf13/cobra"
)

var authTokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Print the auth token in use",
	Long:  "Print the current auth token (API key or access token) to stdout for use in scripts.",
	Example: `  flowmi auth token
  curl -H "Authorization: Bearer $(flowmi auth token)" https://api.flowmi.ai/api/v1/me`,
	RunE: runAuthToken,
}

func init() {
	authCmd.AddCommand(authTokenCmd)
}

func runAuthToken(cmd *cobra.Command, args []string) error {
	token, _ := resolveToken()
	if token == "" {
		return api.NewError(api.CodeAuthRequired, "not logged in").
			WithHint("Run 'flowmi auth login' to authenticate.")
	}
	fmt.Fprintln(cmd.OutOrStdout(), token)
	return nil
}
