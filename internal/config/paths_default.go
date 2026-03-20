//go:build !windows

package config

import (
	"os"
	"path/filepath"
)

// defaultConfigBase returns ~/.config on Unix systems.
func defaultConfigBase() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config"), nil
}
