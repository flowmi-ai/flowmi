package cmd

import "github.com/spf13/cobra"

var completionCmd = &cobra.Command{
	Use:   "completion <shell>",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for flowmi.

To load completions:

  Bash:
    $ source <(flowmi completion bash)

    # To load completions for each session, execute once:
    # Linux:
    $ flowmi completion bash > /etc/bash_completion.d/flowmi
    # macOS:
    $ flowmi completion bash > $(brew --prefix)/etc/bash_completion.d/flowmi

  Zsh:
    # If shell completion is not already enabled in your environment, you
    # will need to enable it. You can execute the following once:
    $ echo "autoload -U compinit; compinit" >> ~/.zshrc

    # To load completions for each session, execute once:
    $ flowmi completion zsh > "${fpath[1]}/_flowmi"

    # You will need to start a new shell for this setup to take effect.

  Fish:
    $ flowmi completion fish | source

    # To load completions for each session, execute once:
    $ flowmi completion fish > ~/.config/fish/completions/flowmi.fish

  PowerShell:
    PS> flowmi completion powershell | Out-String | Invoke-Expression

    # To load completions for every new session, add the output of the above
    # command to your PowerShell profile.
`,
	Example: `  flowmi completion bash
  flowmi completion zsh
  fm completion fish`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		out := cmd.OutOrStdout()
		switch args[0] {
		case "bash":
			return cmd.Root().GenBashCompletionV2(out, true)
		case "zsh":
			return cmd.Root().GenZshCompletion(out)
		case "fish":
			return cmd.Root().GenFishCompletion(out, true)
		case "powershell":
			return cmd.Root().GenPowerShellCompletionWithDesc(out)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
