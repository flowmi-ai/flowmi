package cmd

import (
	"testing"
)

func TestLoginCmdHelp(t *testing.T) {
	t.Cleanup(func() { resetHelpFlags(rootCmd) })
	t.Cleanup(resetHelpState)
	rootCmd.SetArgs([]string{"auth", "login", "--help"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("login --help failed: %v", err)
	}
}

func TestLoginCmdHasFlags(t *testing.T) {
	f := loginCmd.Flags().Lookup("no-browser")
	if f == nil {
		t.Fatal("--no-browser flag not found")
	}

	f = loginCmd.Flags().Lookup("with-token")
	if f == nil {
		t.Fatal("--with-token flag not found")
	}
}
