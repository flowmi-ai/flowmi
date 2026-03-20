//go:build windows

package config

import "os"

// defaultConfigBase returns %AppData% on Windows via os.UserConfigDir.
func defaultConfigBase() (string, error) {
	return os.UserConfigDir()
}
