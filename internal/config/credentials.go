package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// SaveCredentials writes key-value pairs to credentials.toml with 0600 permissions.
func SaveCredentials(creds map[string]string) error {
	path, err := CredentialsFilePath()
	if err != nil {
		return fmt.Errorf("resolving credentials path: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(creds); err != nil {
		return fmt.Errorf("encoding credentials: %w", err)
	}

	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		return fmt.Errorf("writing credentials file: %w", err)
	}

	return nil
}

// LoadCredentials reads key-value pairs from credentials.toml.
// Returns an empty map (not an error) if the file does not exist.
func LoadCredentials() (map[string]string, error) {
	path, err := CredentialsFilePath()
	if err != nil {
		return nil, fmt.Errorf("resolving credentials path: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("reading credentials file: %w", err)
	}

	var creds map[string]string
	if err := toml.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("decoding credentials: %w", err)
	}

	return creds, nil
}

// SaveConfig writes key-value pairs to config.toml.
func SaveConfig(cfg map[string]string) error {
	path, err := ConfigFilePath()
	if err != nil {
		return fmt.Errorf("resolving config path: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(cfg); err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}

// LoadConfig reads key-value pairs from config.toml.
// Returns an empty map (not an error) if the file does not exist.
func LoadConfig() (map[string]string, error) {
	path, err := ConfigFilePath()
	if err != nil {
		return nil, fmt.Errorf("resolving config path: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg map[string]string
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("decoding config: %w", err)
	}

	return cfg, nil
}
