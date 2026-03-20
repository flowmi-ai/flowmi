package config

import (
	"os"
	"path/filepath"
	"sync"
)

// configMu guards configFileOverride against concurrent access (e.g. parallel tests).
var configMu sync.RWMutex

// configFileOverride, when non-empty, replaces the default config.toml path.
// Set via SetConfigFile before any config reads.
var configFileOverride string

// SetConfigFile overrides the default config.toml path (e.g. from --config flag).
func SetConfigFile(path string) {
	configMu.Lock()
	defer configMu.Unlock()
	configFileOverride = path
}

// ResetConfigFile clears the config file override. Used in tests.
func ResetConfigFile() {
	configMu.Lock()
	defer configMu.Unlock()
	configFileOverride = ""
}

// ConfigDir returns the flowmi configuration directory.
// It checks $XDG_CONFIG_HOME/flowmi first, falling back to a platform-specific
// default: ~/.config/flowmi on Unix, %AppData%\flowmi on Windows.
func ConfigDir() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		var err error
		base, err = defaultConfigBase()
		if err != nil {
			return "", err
		}
	}
	return filepath.Join(base, "flowmi"), nil
}

// ConfigFilePath returns the full path to config.toml.
// If SetConfigFile was called, returns that path instead.
func ConfigFilePath() (string, error) {
	configMu.RLock()
	override := configFileOverride
	configMu.RUnlock()
	if override != "" {
		return override, nil
	}
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
