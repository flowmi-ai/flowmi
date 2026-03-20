package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestConfigDir_Default(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")

	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var want string
	if runtime.GOOS == "windows" {
		base, _ := os.UserConfigDir()
		want = filepath.Join(base, "flowmi")
	} else {
		home, _ := os.UserHomeDir()
		want = filepath.Join(home, ".config", "flowmi")
	}
	if dir != want {
		t.Errorf("ConfigDir() = %q, want %q", dir, want)
	}
}

func TestConfigDir_XDG(t *testing.T) {
	custom := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", custom)

	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := filepath.Join(custom, "flowmi")
	if dir != want {
		t.Errorf("ConfigDir() = %q, want %q", dir, want)
	}
}

func TestConfigFilePath(t *testing.T) {
	custom := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", custom)

	p, err := ConfigFilePath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasSuffix(p, "config.toml") {
		t.Errorf("ConfigFilePath() = %q, want suffix config.toml", p)
	}
}

func TestCredentialsFilePath(t *testing.T) {
	custom := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", custom)

	p, err := CredentialsFilePath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasSuffix(p, "credentials.toml") {
		t.Errorf("CredentialsFilePath() = %q, want suffix credentials.toml", p)
	}
}
