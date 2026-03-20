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

	lock, err := acquireFileLock(path)
	if err != nil {
		return fmt.Errorf("acquiring lock: %w", err)
	}
	defer lock.Release()

	all, err := loadProfiledFile(path)
	if err != nil {
		return err
	}

	// Merge into existing profile data (loaded under lock) so concurrent
	// callers don't silently discard each other's writes.
	existing := all[profile]
	if existing == nil {
		existing = make(map[string]string, len(creds))
	}
	for k, v := range creds {
		existing[k] = v
	}
	all[profile] = existing

	return writeProfiledFile(path, all, 0o600)
}

// DeleteCredentialKeys removes the given keys from a profile in credentials.toml.
func DeleteCredentialKeys(profile string, keys ...string) error {
	path, err := CredentialsFilePath()
	if err != nil {
		return fmt.Errorf("resolving credentials path: %w", err)
	}

	lock, err := acquireFileLock(path)
	if err != nil {
		return fmt.Errorf("acquiring lock: %w", err)
	}
	defer lock.Release()

	all, err := loadProfiledFile(path)
	if err != nil {
		return err
	}

	section, ok := all[profile]
	if !ok {
		return nil
	}
	for _, k := range keys {
		delete(section, k)
	}
	if len(section) == 0 {
		delete(all, profile)
	} else {
		all[profile] = section
	}

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

	lock, err := acquireFileLock(path)
	if err != nil {
		return fmt.Errorf("acquiring lock: %w", err)
	}
	defer lock.Release()

	pc, err := loadConfigFile(path)
	if err != nil {
		return err
	}

	// Merge into existing profile data (loaded under lock).
	existing := pc.Profiles[profile]
	if existing == nil {
		existing = make(map[string]string, len(cfg))
	}
	for k, v := range cfg {
		existing[k] = v
	}
	pc.Profiles[profile] = existing

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

	lock, err := acquireFileLock(path)
	if err != nil {
		return fmt.Errorf("acquiring lock: %w", err)
	}
	defer lock.Release()

	pc, err := loadConfigFile(path)
	if err != nil {
		return err
	}
	pc.CurrentProfile = profile

	return writeConfigFile(path, pc)
}

// ListProfiles returns profile names and the current profile.
// Unions profile names from both config.toml and credentials.toml.
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

	seen := make(map[string]struct{})
	for name := range pc.Profiles {
		seen[name] = struct{}{}
	}

	// Include profiles that exist only in credentials.toml.
	credPath, err := CredentialsFilePath()
	if err == nil {
		credProfiles, loadErr := loadProfiledFile(credPath)
		if loadErr == nil {
			for name := range credProfiles {
				seen[name] = struct{}{}
			}
		}
	}

	for name := range seen {
		profiles = append(profiles, name)
	}
	return profiles, current, nil
}

// --- internal helpers ---

// parseProfiledTOML parses TOML data into profile sections, handling both
// legacy flat format and new profiled format. The "current_profile" key,
// if present, is extracted separately and excluded from profile data.
func parseProfiledTOML(data []byte) (profiles map[string]map[string]string, currentProfile string, err error) {
	var raw map[string]any
	if err := toml.Unmarshal(data, &raw); err != nil {
		return nil, "", err
	}

	if cp, ok := raw["current_profile"].(string); ok {
		currentProfile = cp
	}

	hasSection := false
	for _, v := range raw {
		if _, isMap := v.(map[string]any); isMap {
			hasSection = true
			break
		}
	}

	profiles = map[string]map[string]string{}

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
				profiles[k] = flat
			}
		}
		// Preserve top-level scalar keys (partial migration) into DefaultProfile,
		// but never overwrite keys that already exist in the [prod] section.
		for k, v := range raw {
			if k == "current_profile" {
				continue
			}
			if _, isMap := v.(map[string]any); !isMap {
				if profiles[DefaultProfile] == nil {
					profiles[DefaultProfile] = make(map[string]string)
				}
				if _, exists := profiles[DefaultProfile][k]; !exists {
					profiles[DefaultProfile][k] = fmt.Sprintf("%v", v)
				}
			}
		}
	} else {
		// Legacy flat format: migrate all keys into the default profile.
		flat := make(map[string]string, len(raw))
		for k, v := range raw {
			if k == "current_profile" {
				continue
			}
			flat[k] = fmt.Sprintf("%v", v)
		}
		if len(flat) > 0 {
			profiles[DefaultProfile] = flat
		}
	}

	return profiles, currentProfile, nil
}

// loadConfigFile reads config.toml into a ProfileConfig, handling both
// the new profiled format and the legacy flat format.
func loadConfigFile(path string) (*ProfileConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &ProfileConfig{Profiles: map[string]map[string]string{}}, nil
		}
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}

	profiles, currentProfile, err := parseProfiledTOML(data)
	if err != nil {
		return nil, fmt.Errorf("decoding config: %w", err)
	}

	return &ProfileConfig{
		CurrentProfile: currentProfile,
		Profiles:       profiles,
	}, nil
}

// writeConfigFile writes a ProfileConfig to config.toml.
func writeConfigFile(path string, pc *ProfileConfig) error {
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

	return atomicWriteFile(path, buf.Bytes(), 0o644)
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

	profiles, _, err := parseProfiledTOML(data)
	if err != nil {
		return nil, fmt.Errorf("decoding file %s: %w", path, err)
	}
	return profiles, nil
}

// writeProfiledFile writes profile sections to a TOML file.
func writeProfiledFile(path string, profiles map[string]map[string]string, perm os.FileMode) error {
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(profiles); err != nil {
		return fmt.Errorf("encoding file: %w", err)
	}

	return atomicWriteFile(path, buf.Bytes(), perm)
}

// atomicWriteFile writes data to a unique temporary file and renames it into place,
// preventing both partial writes and collisions between concurrent processes.
//
// On Windows, os.Rename fails if the destination is held open by another process
// (e.g. an external editor). The caller's file lock guards against flowmi's own
// concurrent writes but cannot prevent external programs from holding the file.
func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	f, err := os.CreateTemp(dir, ".flowmi-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temporary file: %w", err)
	}
	tmp := f.Name()

	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("writing temporary file: %w", err)
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("syncing temporary file: %w", err)
	}
	if err := f.Chmod(perm); err != nil {
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("setting file permissions: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("closing temporary file: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("renaming temporary file: %w", err)
	}
	return nil
}
