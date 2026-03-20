package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

const DefaultProfile = "prod"

// ProfileConfig represents the full config.toml with profile sections.
//
//	current_profile = "prod"
//	[prod]
//	api_server_url = "https://api.flowmi.ai"
//	auth_server_url = "https://flowmi.ai"
//	[local]
//	api_server_url = "http://localhost:8080"
//	auth_server_url = "http://localhost:5173"
type ProfileConfig struct {
	CurrentProfile string                       `toml:"current_profile,omitempty"`
	Profiles       map[string]map[string]string `toml:"-"`
}

// SaveCredentials writes key-value pairs to the given profile section in credentials.toml.
func SaveCredentials(profile string, creds map[string]string) error {
	path, err := CredentialsFilePath()
	if err != nil {
		return fmt.Errorf("resolving credentials path: %w", err)
	}

	all, err := loadProfiledFile(path)
	if err != nil {
		return err
	}
	all[profile] = creds

	return writeProfiledFile(path, all, 0o600)
}

// LoadCredentials reads key-value pairs for the given profile from credentials.toml.
// Returns an empty map (not an error) if the file or profile does not exist.
func LoadCredentials(profile string) (map[string]string, error) {
	path, err := CredentialsFilePath()
	if err != nil {
		return nil, fmt.Errorf("resolving credentials path: %w", err)
	}

	all, err := loadProfiledFile(path)
	if err != nil {
		return nil, err
	}

	if section, ok := all[profile]; ok {
		return section, nil
	}
	return map[string]string{}, nil
}

// SaveConfigProfile writes key-value pairs to the given profile section in config.toml,
// preserving the current_profile top-level key and other profile sections.
func SaveConfigProfile(profile string, cfg map[string]string) error {
	path, err := ConfigFilePath()
	if err != nil {
		return fmt.Errorf("resolving config path: %w", err)
	}

	pc, err := loadConfigFile(path)
	if err != nil {
		return err
	}
	pc.Profiles[profile] = cfg

	return writeConfigFile(path, pc)
}

// LoadConfigProfile reads key-value pairs for the given profile from config.toml.
func LoadConfigProfile(profile string) (map[string]string, error) {
	path, err := ConfigFilePath()
	if err != nil {
		return nil, fmt.Errorf("resolving config path: %w", err)
	}

	pc, err := loadConfigFile(path)
	if err != nil {
		return nil, err
	}

	if section, ok := pc.Profiles[profile]; ok {
		return section, nil
	}
	return map[string]string{}, nil
}

// CurrentProfile reads the current_profile from config.toml.
// Returns DefaultProfile if not set.
func CurrentProfile() (string, error) {
	path, err := ConfigFilePath()
	if err != nil {
		return DefaultProfile, err
	}

	pc, err := loadConfigFile(path)
	if err != nil {
		return DefaultProfile, nil
	}

	if pc.CurrentProfile != "" {
		return pc.CurrentProfile, nil
	}
	return DefaultProfile, nil
}

// SetCurrentProfile writes the current_profile to config.toml.
func SetCurrentProfile(profile string) error {
	path, err := ConfigFilePath()
	if err != nil {
		return fmt.Errorf("resolving config path: %w", err)
	}

	pc, err := loadConfigFile(path)
	if err != nil {
		return err
	}
	pc.CurrentProfile = profile

	return writeConfigFile(path, pc)
}

// ListProfiles returns profile names and the current profile from config.toml.
func ListProfiles() (profiles []string, current string, err error) {
	path, err := ConfigFilePath()
	if err != nil {
		return nil, "", err
	}

	pc, err := loadConfigFile(path)
	if err != nil {
		return nil, DefaultProfile, nil
	}

	current = pc.CurrentProfile
	if current == "" {
		current = DefaultProfile
	}

	for name := range pc.Profiles {
		profiles = append(profiles, name)
	}
	return profiles, current, nil
}

// --- internal helpers ---

// loadConfigFile reads config.toml into a ProfileConfig, handling both
// the new profiled format and the legacy flat format.
func loadConfigFile(path string) (*ProfileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &ProfileConfig{Profiles: map[string]map[string]string{}}, nil
		}
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	// Try the new profiled format first: top-level current_profile + sections.
	var raw map[string]any
	if err := toml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("decoding config: %w", err)
	}

	pc := &ProfileConfig{Profiles: map[string]map[string]string{}}

	if cp, ok := raw["current_profile"].(string); ok {
		pc.CurrentProfile = cp
	}

	// Check if this is the new format (has at least one table/section).
	hasSection := false
	for _, v := range raw {
		if _, isMap := v.(map[string]any); isMap {
			hasSection = true
			break
		}
	}

	if hasSection {
		for k, v := range raw {
			if k == "current_profile" {
				continue
			}
			if section, ok := v.(map[string]any); ok {
				flat := make(map[string]string, len(section))
				for sk, sv := range section {
					flat[sk] = fmt.Sprintf("%v", sv)
				}
				pc.Profiles[k] = flat
			}
		}
	} else {
		// Legacy flat format: migrate all keys into the default profile.
		flat := make(map[string]string)
		for k, v := range raw {
			if k == "current_profile" {
				continue
			}
			flat[k] = fmt.Sprintf("%v", v)
		}
		if len(flat) > 0 {
			pc.Profiles[DefaultProfile] = flat
		}
	}

	return pc, nil
}

// writeConfigFile writes a ProfileConfig to config.toml.
func writeConfigFile(path string, pc *ProfileConfig) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	// Build output: current_profile + profile sections.
	ordered := make(map[string]any, len(pc.Profiles)+1)
	if pc.CurrentProfile != "" {
		ordered["current_profile"] = pc.CurrentProfile
	}
	for name, section := range pc.Profiles {
		ordered[name] = section
	}

	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(ordered); err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// loadProfiledFile reads a TOML file with profile sections (e.g., credentials.toml).
// Handles both legacy flat format (migrates to DefaultProfile) and profiled format.
func loadProfiledFile(path string) (map[string]map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]map[string]string{}, nil
		}
		return nil, fmt.Errorf("reading file %s: %w", path, err)
	}

	var raw map[string]any
	if err := toml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("decoding file %s: %w", path, err)
	}

	// Detect format: if any value is a map, it's the new profiled format.
	hasSection := false
	for _, v := range raw {
		if _, isMap := v.(map[string]any); isMap {
			hasSection = true
			break
		}
	}

	result := map[string]map[string]string{}

	if hasSection {
		for k, v := range raw {
			if section, ok := v.(map[string]any); ok {
				flat := make(map[string]string, len(section))
				for sk, sv := range section {
					flat[sk] = fmt.Sprintf("%v", sv)
				}
				result[k] = flat
			}
		}
	} else {
		// Legacy flat format → migrate into DefaultProfile.
		flat := make(map[string]string, len(raw))
		for k, v := range raw {
			flat[k] = fmt.Sprintf("%v", v)
		}
		if len(flat) > 0 {
			result[DefaultProfile] = flat
		}
	}

	return result, nil
}

// writeProfiledFile writes profile sections to a TOML file.
func writeProfiledFile(path string, profiles map[string]map[string]string, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(profiles); err != nil {
		return fmt.Errorf("encoding file: %w", err)
	}

	return os.WriteFile(path, buf.Bytes(), perm)
}
