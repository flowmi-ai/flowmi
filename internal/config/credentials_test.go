package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	creds := map[string]string{
		"access_token":  "tok_abc",
		"refresh_token": "tok_xyz",
	}

	if err := SaveCredentials(creds); err != nil {
		t.Fatalf("SaveCredentials: %v", err)
	}

	// Verify file permissions.
	path := filepath.Join(tmpDir, "flowmi", "credentials.toml")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat credentials file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("credentials file perm = %o, want 0600", perm)
	}

	// Load and verify contents.
	loaded, err := LoadCredentials()
	if err != nil {
		t.Fatalf("LoadCredentials: %v", err)
	}

	for k, want := range creds {
		if got := loaded[k]; got != want {
			t.Errorf("loaded[%q] = %q, want %q", k, got, want)
		}
	}
}

func TestLoadCredentials_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	creds, err := LoadCredentials()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(creds) != 0 {
		t.Errorf("expected empty map, got %v", creds)
	}
}

func TestSaveCredentials_OverwriteExisting(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	if err := SaveCredentials(map[string]string{"access_token": "old"}); err != nil {
		t.Fatalf("first save: %v", err)
	}

	if err := SaveCredentials(map[string]string{"access_token": "new"}); err != nil {
		t.Fatalf("second save: %v", err)
	}

	loaded, err := LoadCredentials()
	if err != nil {
		t.Fatalf("LoadCredentials: %v", err)
	}

	if got := loaded["access_token"]; got != "new" {
		t.Errorf("access_token = %q, want %q", got, "new")
	}
}
