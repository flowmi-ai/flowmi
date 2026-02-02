package config

import (
	"os"
	"path/filepath"
)

// ConfigDir returns the flowmi configuration directory.
// It checks $XDG_CONFIG_HOME/flowmi first, falling back to ~/.config/flowmi.
func ConfigDir() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "flowmi"), nil
}

// ConfigFilePath returns the full path to config.toml.
func ConfigFilePath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}

// CredentialsFilePath returns the full path to credentials.toml.
func CredentialsFilePath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "credentials.toml"), nil
}
