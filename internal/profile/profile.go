package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/saurabh/logr/internal/filter"
)

// profileDir returns the directory where profiles are stored, creating it if needed.
func profileDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	dir := filepath.Join(home, ".logr", "profiles")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("cannot create profile directory: %w", err)
	}
	return dir, nil
}

func profilePath(name string) (string, error) {
	dir, err := profileDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name+".toml"), nil
}

// tomlConfig is the on-disk representation of a filter.Config.
// SuppressTTL is stored as a human-readable string (e.g. "30s").
type tomlConfig struct {
	MinLevel      string   `toml:"min_level"`
	IncludeFields []string `toml:"include_fields"`
	ExcludeFields []string `toml:"exclude_fields"`
	HierPatterns  []string `toml:"hier_patterns"`
	SuppressTTL   string   `toml:"suppress_ttl"`
	Services      []string `toml:"services"`
	Keys          []string `toml:"keys"`
}

func configToTOML(cfg filter.Config) tomlConfig {
	ttl := ""
	if cfg.SuppressTTL > 0 {
		ttl = cfg.SuppressTTL.String()
	}
	return tomlConfig{
		MinLevel:      cfg.MinLevel,
		IncludeFields: cfg.IncludeFields,
		ExcludeFields: cfg.ExcludeFields,
		HierPatterns:  cfg.HierPatterns,
		SuppressTTL:   ttl,
		Services:      cfg.Services,
		Keys:          cfg.Keys,
	}
}

func tomlToConfig(tc tomlConfig) (filter.Config, error) {
	cfg := filter.Config{
		MinLevel:      tc.MinLevel,
		IncludeFields: tc.IncludeFields,
		ExcludeFields: tc.ExcludeFields,
		HierPatterns:  tc.HierPatterns,
		Services:      tc.Services,
		Keys:          tc.Keys,
	}
	if tc.SuppressTTL != "" {
		d, err := time.ParseDuration(tc.SuppressTTL)
		if err != nil {
			return cfg, fmt.Errorf("invalid suppress_ttl %q: %w", tc.SuppressTTL, err)
		}
		cfg.SuppressTTL = d
	}
	return cfg, nil
}

// Save persists cfg under the given profile name.
func Save(name string, cfg filter.Config) error {
	p, err := profilePath(name)
	if err != nil {
		return err
	}
	f, err := os.Create(p)
	if err != nil {
		return fmt.Errorf("cannot create profile file: %w", err)
	}
	defer f.Close()

	if err := toml.NewEncoder(f).Encode(configToTOML(cfg)); err != nil {
		return fmt.Errorf("cannot encode profile: %w", err)
	}
	return nil
}

// Load reads a named profile and returns the corresponding filter.Config.
func Load(name string) (filter.Config, error) {
	p, err := profilePath(name)
	if err != nil {
		return filter.Config{}, err
	}

	// Check existence before decoding so we can give a clean error message.
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return filter.Config{}, fmt.Errorf("profile %q not found", name)
	}

	var tc tomlConfig
	if _, err := toml.DecodeFile(p, &tc); err != nil {
		return filter.Config{}, fmt.Errorf("cannot decode profile: %w", err)
	}

	return tomlToConfig(tc)
}

// List returns the names of all saved profiles.
func List() ([]string, error) {
	dir, err := profileDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("cannot read profile directory: %w", err)
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".toml") {
			names = append(names, strings.TrimSuffix(e.Name(), ".toml"))
		}
	}
	return names, nil
}

// Delete removes a named profile.
func Delete(name string) error {
	p, err := profilePath(name)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("profile %q not found", name)
		}
		return fmt.Errorf("cannot delete profile: %w", err)
	}
	return nil
}
