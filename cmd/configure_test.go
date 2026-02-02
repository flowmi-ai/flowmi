package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigure_ValidKey(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".flowmi.toml")

	in := strings.NewReader("sk-test123\n")
	if err := runConfigure(in, configPath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "api_key") {
		t.Errorf("config file missing api_key, got:\n%s", content)
	}
	if !strings.Contains(content, "sk-test123") {
		t.Errorf("config file missing key value, got:\n%s", content)
	}
}

func TestConfigure_EmptyKey(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".flowmi.toml")

	in := strings.NewReader("\n")
	err := runConfigure(in, configPath)
	if err == nil {
		t.Fatal("expected error for empty key, got nil")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("expected empty key error, got: %v", err)
	}
}

func TestConfigure_OverwriteExisting(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".flowmi.toml")

	// Write initial config
	in := strings.NewReader("sk-old-key\n")
	if err := runConfigure(in, configPath); err != nil {
		t.Fatalf("first configure failed: %v", err)
	}

	// Overwrite with new key
	in = strings.NewReader("sk-new-key\n")
	if err := runConfigure(in, configPath); err != nil {
		t.Fatalf("second configure failed: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}

	content := string(data)
	if strings.Contains(content, "sk-old-key") {
		t.Errorf("config file still contains old key:\n%s", content)
	}
	if !strings.Contains(content, "sk-new-key") {
		t.Errorf("config file missing new key, got:\n%s", content)
	}
}
