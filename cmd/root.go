package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/flowmi-ai/flowmi/internal/api"
	"github.com/flowmi-ai/flowmi/internal/config"
	"github.com/flowmi-ai/flowmi/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var cfgFile string
var defaultHelpFunc func(*cobra.Command, []string)
var globalHelpFlagNames = map[string]struct{}{
	"config": {},
	"format": {},
	"output": {},
}

const (
	defaultAuthServerURL = "https://flowmi.ai"
	defaultAPIServerURL  = "https://api.flowmi.ai"
)

var rootCmd = &cobra.Command{
	Use:           "flowmi",
	Short:         "Flowmi CLI",
	Long:          `flowmi (fm) — notes, auth, and more from your terminal.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() {
	// Support "fm" as alias: adapt Use field based on binary name
	bin := filepath.Base(os.Args[0])
	if bin == "fm" {
		rootCmd.Use = "fm"
	}

	if err := rootCmd.Execute(); err != nil {
		exitCode := formatError(err)
		os.Exit(exitCode)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default ~/.config/flowmi/config.toml)")
	rootCmd.PersistentFlags().String("format", "text", "help/options format: text, json")
	rootCmd.PersistentFlags().StringP("output", "o", "text", "output format: text, json, table")
	viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))

	cobra.AddTemplateFunc("hasParentInheritedFlags", hasParentInheritedFlags)
	cobra.AddTemplateFunc("parentInheritedFlagUsages", parentInheritedFlagUsages)
	cobra.AddTemplateFunc("hasGlobalInheritedFlags", hasGlobalInheritedFlags)
	cobra.AddTemplateFunc("isRootCommand", isRootCommand)
	cobra.AddTemplateFunc("globalHelpHint", globalHelpHint)
	rootCmd.SetHelpTemplate(helpTemplate)

	defaultHelpFunc = rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(renderHelp)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		configDir, err := config.ConfigDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		viper.AddConfigPath(configDir)
		viper.SetConfigName("config")
		viper.SetConfigType("toml")
	}

	viper.SetDefault("auth_server_url", defaultAuthServerURL)
	viper.SetDefault("api_server_url", defaultAPIServerURL)
	viper.SetEnvPrefix("FLOWMI")
	viper.AutomaticEnv()
	viper.ReadInConfig() // silently ignore if config file not found

	// Inject credentials into viper so the rest of the app can use
	// viper.GetString("access_token") etc. as before.
	creds, err := config.LoadCredentials()
	if err != nil {
		fmt.Fprintln(os.Stderr, "warning: could not load credentials:", err)
		return
	}
	for k, v := range creds {
		viper.SetDefault(k, v)
	}
}

func hasParentInheritedFlags(cmd *cobra.Command) bool {
	return flagUsages(cmd, false) != ""
}

func parentInheritedFlagUsages(cmd *cobra.Command) string {
	return flagUsages(cmd, false)
}

func hasGlobalInheritedFlags(cmd *cobra.Command) bool {
	return flagUsages(cmd, true) != ""
}

func isRootCommand(cmd *cobra.Command) bool {
	return cmd.Parent() == nil
}

func globalHelpHint(cmd *cobra.Command) string {
	return fmt.Sprintf(
		"Global Flags hidden by default (common: --config, --output). Run '%s options' or use '--help --format json'.",
		cmd.Root().CommandPath(),
	)
}

func flagUsages(cmd *cobra.Command, global bool) string {
	fs := pflag.NewFlagSet("help", pflag.ContinueOnError)
	fs.SortFlags = false
	cmd.InheritedFlags().VisitAll(func(f *pflag.Flag) {
		_, isGlobal := globalHelpFlagNames[f.Name]
		if isGlobal == global {
			fs.AddFlag(f)
		}
	})
	return fs.FlagUsagesWrapped(80)
}

type helpFlag struct {
	Name        string `json:"name"`
	Shorthand   string `json:"shorthand,omitempty"`
	Type        string `json:"type"`
	Default     string `json:"default,omitempty"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Scope       string `json:"scope"`
	Inherited   bool   `json:"inherited"`
	Hidden      bool   `json:"hidden,omitempty"`
}

type helpSubcommand struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Summary string `json:"summary"`
}

type helpJSON struct {
	Command     string           `json:"command"`
	Path        string           `json:"path"`
	Summary     string           `json:"summary,omitempty"`
	Description string           `json:"description,omitempty"`
	Usage       string           `json:"usage"`
	Examples    []string         `json:"examples,omitempty"`
	Subcommands []helpSubcommand `json:"subcommands,omitempty"`
	Flags       struct {
		Local  []helpFlag `json:"local,omitempty"`
		Parent []helpFlag `json:"parent,omitempty"`
		Global []helpFlag `json:"global,omitempty"`
	} `json:"flags"`
}

func renderHelp(cmd *cobra.Command, args []string) {
	format, _ := cmd.Flags().GetString("format")
	if strings.EqualFold(format, "json") {
		if err := encodeHelpJSON(cmd); err != nil {
			fmt.Fprintln(cmd.ErrOrStderr(), "Error:", err)
		}
		return
	}
	defaultHelpFunc(cmd, args)
}

func encodeHelpJSON(cmd *cobra.Command) error {
	help := helpJSON{
		Command:     cmd.Name(),
		Path:        cmd.CommandPath(),
		Summary:     cmd.Short,
		Description: cmd.Long,
		Usage:       cmd.UseLine(),
		Examples:    splitExamples(cmd.Example),
		Subcommands: collectSubcommands(cmd),
	}
	help.Flags.Local = collectLocalHelpFlags(cmd)
	help.Flags.Parent = collectInheritedHelpFlags(cmd, false)
	help.Flags.Global = collectInheritedHelpFlags(cmd, true)
	if isRootCommand(cmd) {
		help.Flags.Global = collectRootGlobalFlags(cmd)
	}

	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(help)
}

func splitExamples(examples string) []string {
	if strings.TrimSpace(examples) == "" {
		return nil
	}
	lines := strings.Split(examples, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func collectSubcommands(cmd *cobra.Command) []helpSubcommand {
	var subs []helpSubcommand
	for _, sub := range cmd.Commands() {
		if !sub.IsAvailableCommand() {
			continue
		}
		subs = append(subs, helpSubcommand{
			Name:    sub.Name(),
			Path:    sub.CommandPath(),
			Summary: sub.Short,
		})
	}
	return subs
}

func collectLocalHelpFlags(cmd *cobra.Command) []helpFlag {
	flags := make([]helpFlag, 0)
	cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
		flags = append(flags, toHelpFlag(f, "local", false))
	})
	return flags
}

func collectInheritedHelpFlags(cmd *cobra.Command, global bool) []helpFlag {
	flags := make([]helpFlag, 0)
	cmd.InheritedFlags().VisitAll(func(f *pflag.Flag) {
		_, isGlobal := globalHelpFlagNames[f.Name]
		if isGlobal != global {
			return
		}
		scope := "parent"
		if global {
			scope = "global"
		}
		flags = append(flags, toHelpFlag(f, scope, true))
	})
	return flags
}

func collectRootGlobalFlags(cmd *cobra.Command) []helpFlag {
	flags := make([]helpFlag, 0)
	cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
		if _, ok := globalHelpFlagNames[f.Name]; ok {
			flags = append(flags, toHelpFlag(f, "global", false))
		}
	})
	return flags
}

func toHelpFlag(f *pflag.Flag, scope string, inherited bool) helpFlag {
	return helpFlag{
		Name:        f.Name,
		Shorthand:   f.Shorthand,
		Type:        f.Value.Type(),
		Default:     f.DefValue,
		Description: f.Usage,
		Required:    isRequiredFlag(f),
		Scope:       scope,
		Inherited:   inherited,
		Hidden:      f.Hidden,
	}
}

func isRequiredFlag(f *pflag.Flag) bool {
	values, ok := f.Annotations[cobra.BashCompOneRequiredFlag]
	if !ok {
		return false
	}
	for _, v := range values {
		if strings.EqualFold(v, "true") {
			return true
		}
	}
	return false
}

func formatError(err error) int {
	var apiErr *api.Error
	if errors.As(err, &apiErr) {
		if viper.GetString("output") == "json" {
			formatErrorJSON(apiErr)
		} else {
			formatErrorText(apiErr)
		}
		return apiErr.ExitCode()
	}

	// Unstructured error fallback.
	if viper.GetString("output") == "json" {
		je := struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
			ExitCode int `json:"exitCode"`
		}{}
		je.Error.Code = api.CodeCommandError
		je.Error.Message = err.Error()
		je.ExitCode = api.ExitBusiness
		_ = json.NewEncoder(os.Stderr).Encode(je)
	} else {
		fmt.Fprintln(os.Stderr, ui.ErrorStyle.Render("Error:")+" "+err.Error())
	}
	return api.ExitBusiness
}

func formatErrorJSON(e *api.Error) {
	type jsonError struct {
		Code    string         `json:"code"`
		Message string         `json:"message"`
		Hint    string         `json:"hint,omitempty"`
		Details map[string]any `json:"details,omitempty"`
	}
	je := struct {
		Error     jsonError `json:"error"`
		RequestID string    `json:"requestId,omitempty"`
		ExitCode  int       `json:"exitCode"`
	}{
		Error: jsonError{
			Code:    e.Code,
			Message: e.Message,
			Hint:    e.Hint,
			Details: e.Details,
		},
		RequestID: e.RequestID,
		ExitCode:  e.ExitCode(),
	}
	_ = json.NewEncoder(os.Stderr).Encode(je)
}

func formatErrorText(e *api.Error) {
	fmt.Fprintln(os.Stderr, ui.ErrorStyle.Render(fmt.Sprintf("Error [%s]:", e.Code))+" "+e.Message)
	if e.Hint != "" {
		fmt.Fprintln(os.Stderr, ui.SubtleStyle.Render("Hint: "+e.Hint))
	}
	if e.RequestID != "" {
		fmt.Fprintln(os.Stderr, ui.SubtleStyle.Render("Request ID: "+e.RequestID))
	}
}

const helpTemplate = `{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}{{"\n\n"}}{{end}}{{if or .Runnable .HasSubCommands}}Usage:
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

{{if isRootCommand .}}Global Flags:{{else}}Flags:{{end}}
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}
{{end}}{{if hasParentInheritedFlags .}}

Parent Flags:
{{parentInheritedFlagUsages . | trimTrailingWhitespaces}}
{{end}}{{if hasGlobalInheritedFlags .}}
{{globalHelpHint .}}
{{end}}{{if .HasExample}}

Examples:
{{.Example | trimTrailingWhitespaces}}
{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`
