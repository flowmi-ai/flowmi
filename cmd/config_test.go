package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigSet_ApiKey(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	rootCmd.SetArgs([]string{"config", "set", "api_key", "sk-test123"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("config set failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "flowmi", "credentials.toml"))
	if err != nil {
		t.Fatalf("failed to read credentials file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "api_key") {
		t.Errorf("credentials file missing api_key, got:\n%s", content)
	}
	if !strings.Contains(content, "sk-test123") {
		t.Errorf("credentials file missing key value, got:\n%s", content)
	}
}

func TestConfigSet_ServerURL(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	rootCmd.SetArgs([]string{"config", "set", "api_server_url", "https://custom.api.example.com"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("config set failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "flowmi", "config.toml"))
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "api_server_url") {
		t.Errorf("config file missing api_server_url, got:\n%s", content)
	}
	if !strings.Contains(content, "https://custom.api.example.com") {
		t.Errorf("config file missing url value, got:\n%s", content)
	}
}

func TestConfigSet_UnknownKey(t *testing.T) {
	rootCmd.SetArgs([]string{"config", "set", "unknown_key", "value"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for unknown key, got nil")
	}
}

func TestConfigSet_NoArgs(t *testing.T) {
	rootCmd.SetArgs([]string{"config", "set"})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing args, got nil")
	}
}

func TestConfigGet_ApiKey(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Set first
	rootCmd.SetArgs([]string{"config", "set", "api_key", "sk-gettest"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("config set failed: %v", err)
	}

	// Get
	rootCmd.SetArgs([]string{"config", "get", "api_key"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("config get failed: %v", err)
	}
}
