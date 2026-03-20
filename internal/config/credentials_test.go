package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSaveAndLoadCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	creds := map[string]string{
		"access_token":  "tok_abc",
		"refresh_token": "tok_xyz",
	}

	if err := SaveCredentials("prod", creds); err != nil {
		t.Fatalf("SaveCredentials: %v", err)
	}

	// Verify file permissions (Unix only — Windows ignores permission bits).
	if runtime.GOOS != "windows" {
		path := filepath.Join(tmpDir, "flowmi", "credentials.toml")
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat credentials file: %v", err)
		}
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Errorf("credentials file perm = %o, want 0600", perm)
		}
	}

	// Load and verify contents.
	loaded, err := LoadCredentials("prod")
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

	creds, err := LoadCredentials("prod")
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

	if err := SaveCredentials("prod", map[string]string{"access_token": "old"}); err != nil {
		t.Fatalf("first save: %v", err)
	}

	if err := SaveCredentials("prod", map[string]string{"access_token": "new"}); err != nil {
		t.Fatalf("second save: %v", err)
	}

	loaded, err := LoadCredentials("prod")
	if err != nil {
		t.Fatalf("LoadCredentials: %v", err)
	}

	if got := loaded["access_token"]; got != "new" {
		t.Errorf("access_token = %q, want %q", got, "new")
	}
}

func TestDeleteCredentialKeys(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	if err := SaveCredentials("prod", map[string]string{
		"access_token":  "tok_abc",
		"refresh_token": "ref_xyz",
		"api_key":       "key_123",
	}); err != nil {
		t.Fatalf("SaveCredentials: %v", err)
	}

	if err := DeleteCredentialKeys("prod", "access_token", "refresh_token"); err != nil {
		t.Fatalf("DeleteCredentialKeys: %v", err)
	}

	creds, err := LoadCredentials("prod")
	if err != nil {
		t.Fatalf("LoadCredentials: %v", err)
	}
	if _, ok := creds["access_token"]; ok {
		t.Error("access_token should have been deleted")
	}
	if _, ok := creds["refresh_token"]; ok {
		t.Error("refresh_token should have been deleted")
	}
	if got := creds["api_key"]; got != "key_123" {
		t.Errorf("api_key = %q, want %q (should be preserved)", got, "key_123")
	}
}

func TestDeleteCredentialKeys_RemovesEmptyProfile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	if err := SaveCredentials("staging", map[string]string{"api_key": "k"}); err != nil {
		t.Fatalf("SaveCredentials: %v", err)
	}
	if err := DeleteCredentialKeys("staging", "api_key"); err != nil {
		t.Fatalf("DeleteCredentialKeys: %v", err)
	}

	creds, err := LoadCredentials("staging")
	if err != nil {
		t.Fatalf("LoadCredentials: %v", err)
	}
	if len(creds) != 0 {
		t.Errorf("expected empty map after deleting all keys, got %v", creds)
	}
}

func TestCredentials_MultipleProfiles(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	if err := SaveCredentials("prod", map[string]string{"api_key": "prod_key"}); err != nil {
		t.Fatalf("save production: %v", err)
	}
	if err := SaveCredentials("local", map[string]string{"api_key": "local_key"}); err != nil {
		t.Fatalf("save local: %v", err)
	}

	prod, err := LoadCredentials("prod")
	if err != nil {
		t.Fatalf("load production: %v", err)
	}
	if got := prod["api_key"]; got != "prod_key" {
		t.Errorf("production api_key = %q, want %q", got, "prod_key")
	}

	local, err := LoadCredentials("local")
	if err != nil {
		t.Fatalf("load local: %v", err)
	}
	if got := local["api_key"]; got != "local_key" {
		t.Errorf("local api_key = %q, want %q", got, "local_key")
	}
}

func TestLegacyFlatCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Write a legacy flat credentials file.
	dir := filepath.Join(tmpDir, "flowmi")
	os.MkdirAll(dir, 0o755)
	legacy := []byte("access_token = 'tok_legacy'\nrefresh_token = 'ref_legacy'\n")
	os.WriteFile(filepath.Join(dir, "credentials.toml"), legacy, 0o600)

	// Loading with "prod" profile should find the migrated legacy data.
	creds, err := LoadCredentials("prod")
	if err != nil {
		t.Fatalf("LoadCredentials: %v", err)
	}
	if got := creds["access_token"]; got != "tok_legacy" {
		t.Errorf("access_token = %q, want %q", got, "tok_legacy")
	}
}

func TestConfigProfile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	if err := SaveConfigProfile("local", map[string]string{
		"api_server_url":  "http://localhost:8080",
		"auth_server_url": "http://localhost:5173",
	}); err != nil {
		t.Fatalf("SaveConfigProfile: %v", err)
	}

	cfg, err := LoadConfigProfile("local")
	if err != nil {
		t.Fatalf("LoadConfigProfile: %v", err)
	}
	if got := cfg["api_server_url"]; got != "http://localhost:8080" {
		t.Errorf("api_server_url = %q, want %q", got, "http://localhost:8080")
	}
}

func TestCurrentProfile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Default when no file exists.
	p, err := CurrentProfile()
	if err != nil {
		t.Fatalf("CurrentProfile: %v", err)
	}
	if p != DefaultProfile {
		t.Errorf("default profile = %q, want %q", p, DefaultProfile)
	}

	// Set and read back.
	if err := SetCurrentProfile("local"); err != nil {
		t.Fatalf("SetCurrentProfile: %v", err)
	}
	p, err = CurrentProfile()
	if err != nil {
		t.Fatalf("CurrentProfile: %v", err)
	}
	if p != "local" {
		t.Errorf("profile = %q, want %q", p, "local")
	}
}

func TestMixedFormatCredentials(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Write a mixed-format file: top-level scalars alongside a profile section.
	dir := filepath.Join(tmpDir, "flowmi")
	os.MkdirAll(dir, 0o755)
	mixed := []byte("stray_key = 'stray_value'\n\n[prod]\naccess_token = 'tok_prod'\n")
	os.WriteFile(filepath.Join(dir, "credentials.toml"), mixed, 0o600)

	// Top-level scalar should be preserved in the default profile, not dropped.
	creds, err := LoadCredentials("prod")
	if err != nil {
		t.Fatalf("LoadCredentials: %v", err)
	}
	if got := creds["access_token"]; got != "tok_prod" {
		t.Errorf("access_token = %q, want %q", got, "tok_prod")
	}
	if got := creds["stray_key"]; got != "stray_value" {
		t.Errorf("stray_key = %q, want %q (should be preserved, not dropped)", got, "stray_value")
	}
}

func TestMixedFormatCredentials_ConflictingKeys(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Stray top-level key conflicts with a key inside [prod] — section value must win.
	dir := filepath.Join(tmpDir, "flowmi")
	os.MkdirAll(dir, 0o755)
	mixed := []byte("access_token = 'stale_token'\n\n[prod]\naccess_token = 'real_token'\n")
	os.WriteFile(filepath.Join(dir, "credentials.toml"), mixed, 0o600)

	creds, err := LoadCredentials("prod")
	if err != nil {
		t.Fatalf("LoadCredentials: %v", err)
	}
	if got := creds["access_token"]; got != "real_token" {
		t.Errorf("access_token = %q, want %q (section value must take precedence over stray scalar)", got, "real_token")
	}
}

func TestLegacyFlatConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Write a legacy flat config file.
	dir := filepath.Join(tmpDir, "flowmi")
	os.MkdirAll(dir, 0o755)
	legacy := []byte("api_server_url = 'http://localhost:8080'\nauth_server_url = 'http://localhost:5173'\n")
	os.WriteFile(filepath.Join(dir, "config.toml"), legacy, 0o644)

	// Should read legacy flat values as "prod" profile.
	cfg, err := LoadConfigProfile("prod")
	if err != nil {
		t.Fatalf("LoadConfigProfile: %v", err)
	}
	if got := cfg["api_server_url"]; got != "http://localhost:8080" {
		t.Errorf("api_server_url = %q, want %q", got, "http://localhost:8080")
	}
}
