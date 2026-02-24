package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func runHelpOutput(t *testing.T, args ...string) string {
	t.Helper()
	buf := new(bytes.Buffer)
	resetHelpState()
	t.Cleanup(func() { resetHelpFlags(rootCmd) })
	t.Cleanup(resetHelpState)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("command failed: %v", err)
	}
	return buf.String()
}

func resetHelpState() {
	if f := rootCmd.PersistentFlags().Lookup("format"); f != nil {
		f.Changed = false
		_ = rootCmd.PersistentFlags().Set("format", "text")
	}
}

func resetHelpFlags(cmd *cobra.Command) {
	if f := cmd.Flags().Lookup("help"); f != nil {
		f.Changed = false
		_ = cmd.Flags().Set("help", "false")
	}
	for _, sub := range cmd.Commands() {
		resetHelpFlags(sub)
	}
}

func TestLoginHelp_HidesGlobalFlagsByDefault(t *testing.T) {
	output := runHelpOutput(t, "auth", "login", "--help")
	if strings.Contains(output, "Global Flags:\n") {
		t.Fatalf("help unexpectedly shows full global flags section:\n%s", output)
	}
	if !strings.Contains(output, "Global Flags hidden by default") {
		t.Fatalf("help should explain how to view global flags:\n%s", output)
	}
}

func TestSearchWebHelp_ShowsParentFlagsOnly(t *testing.T) {
	output := runHelpOutput(t, "search", "web", "--help")
	if !strings.Contains(output, "Parent Flags:") {
		t.Fatalf("help should include parent flags section:\n%s", output)
	}
	if !strings.Contains(output, "--limit") {
		t.Fatalf("help should include inherited search --limit flag:\n%s", output)
	}
	if strings.Contains(output, "Global Flags:\n") {
		t.Fatalf("help should hide full root global flags section by default:\n%s", output)
	}
}

func TestOptionsCommand_ShowsGlobalFlags(t *testing.T) {
	output := runHelpOutput(t, "options")
	if !strings.Contains(output, "Global Flags:") {
		t.Fatalf("options should include Global Flags heading:\n%s", output)
	}
	if !strings.Contains(output, "--config") {
		t.Fatalf("options should include --config:\n%s", output)
	}
	if !strings.Contains(output, "--output") {
		t.Fatalf("options should include --output:\n%s", output)
	}
}

func TestRootHelp_UsesGlobalFlagsHeading(t *testing.T) {
	output := runHelpOutput(t, "--help")
	if !strings.Contains(output, "Global Flags:") {
		t.Fatalf("root help should use Global Flags heading:\n%s", output)
	}
	if strings.Contains(output, "\nFlags:\n") {
		t.Fatalf("root help should not use generic Flags heading:\n%s", output)
	}
}

func TestLoginHelp_ShowsExamplesAndHint(t *testing.T) {
	output := runHelpOutput(t, "auth", "login", "--help")
	if !strings.Contains(output, "Examples:") {
		t.Fatalf("help should include examples:\n%s", output)
	}
	if !strings.Contains(output, "common: --config, --output") {
		t.Fatalf("help should include common global flag hint:\n%s", output)
	}
}

func TestHelpJSONFormat(t *testing.T) {
	output := runHelpOutput(t, "auth", "login", "--help", "--format", "json")
	var payload struct {
		Path  string `json:"path"`
		Flags struct {
			Local  []helpFlag `json:"local"`
			Global []helpFlag `json:"global"`
		} `json:"flags"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("expected valid json help output, got error: %v\n%s", err, output)
	}
	if payload.Path != "flowmi auth login" {
		t.Fatalf("path = %q, want %q", payload.Path, "flowmi auth login")
	}
	if len(payload.Flags.Local) == 0 {
		t.Fatalf("expected local flags in json help, got none")
	}
	if len(payload.Flags.Global) == 0 {
		t.Fatalf("expected global flags in json help, got none")
	}
}

func TestOptionsCommand_JSONFormat(t *testing.T) {
	output := runHelpOutput(t, "options", "--format", "json")
	var payload struct {
		Command string     `json:"command"`
		Flags   []helpFlag `json:"flags"`
	}
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("expected valid json options output, got error: %v\n%s", err, output)
	}
	if payload.Command != "flowmi options" {
		t.Fatalf("command = %q, want %q", payload.Command, "flowmi options")
	}
	if len(payload.Flags) == 0 {
		t.Fatalf("expected global flags in options json output, got none")
	}
}
