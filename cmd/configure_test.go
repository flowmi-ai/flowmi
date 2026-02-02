package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigure_ValidKey(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	in := strings.NewReader("sk-test123\n")
	if err := runConfigure(in); err != nil {
		t.Fatalf("unexpected error: %v", err)
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

func TestConfigure_EmptyKey(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	in := strings.NewReader("\n")
	err := runConfigure(in)
	if err == nil {
		t.Fatal("expected error for empty key, got nil")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("expected empty key error, got: %v", err)
	}
}

func TestConfigure_OverwriteExisting(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Write initial config
	in := strings.NewReader("sk-old-key\n")
	if err := runConfigure(in); err != nil {
		t.Fatalf("first configure failed: %v", err)
	}

	// Overwrite with new key
	in = strings.NewReader("sk-new-key\n")
	if err := runConfigure(in); err != nil {
		t.Fatalf("second configure failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "flowmi", "credentials.toml"))
	if err != nil {
		t.Fatalf("failed to read credentials file: %v", err)
	}

	content := string(data)
	if strings.Contains(content, "sk-old-key") {
		t.Errorf("credentials file still contains old key:\n%s", content)
	}
	if !strings.Contains(content, "sk-new-key") {
		t.Errorf("credentials file missing new key, got:\n%s", content)
	}
}
