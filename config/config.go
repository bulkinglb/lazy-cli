package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	defaultPrefix = "%&"
	defaultPort   = 8080
	defaultMode   = "normal"
)

// Config holds all persistent CLI configuration
type Config struct {
	Mode        string            `json:"mode"`         // ultra-safe, normal, yolo
	Port        int               `json:"port"`         // llama-server port
	Prefix      string            `json:"prefix"`       // command prefix (default %&)
	LogEnabled  bool              `json:"log_enabled"`  // whether logging is active
	LogPath     string            `json:"log_path"`     // log directory path
	PathAliases map[string]string `json:"path_aliases"` // known path shortcuts

	filePath string // where this config is stored (not serialized)
}

// Default returns a Config with sensible defaults
func Default() *Config {
	home, _ := os.UserHomeDir()
	return &Config{
		Mode:        defaultMode,
		Port:        defaultPort,
		Prefix:      defaultPrefix,
		LogEnabled:  true,
		LogPath:     filepath.Join(home, ".lazy-cli", "logs"),
		PathAliases: make(map[string]string),
	}
}

// Load reads config from ~/.lazy-cli/config.json, creating defaults if missing
func Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Default(), nil
	}

	cfgDir := filepath.Join(home, ".lazy-cli")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		return Default(), nil
	}

	cfgPath := filepath.Join(cfgDir, "config.json")
	cfg := Default()
	cfg.filePath = cfgPath

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		// No config file yet — save defaults and return
		cfg.Save()
		return cfg, nil
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return cfg, fmt.Errorf("config: parse error: %w", err)
	}

	// Ensure maps are initialized after unmarshal
	if cfg.PathAliases == nil {
		cfg.PathAliases = make(map[string]string)
	}

	return cfg, nil
}

// Save writes the current config to disk
func (c *Config) Save() error {
	if c.filePath == "" {
		home, _ := os.UserHomeDir()
		c.filePath = filepath.Join(home, ".lazy-cli", "config.json")
	}

	dir := filepath.Dir(c.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("config: create dir: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("config: marshal: %w", err)
	}

	return os.WriteFile(c.filePath, data, 0644)
}

// FilePath returns the config file location
func (c *Config) FilePath() string {
	return c.filePath
}

// ValidModes returns the allowed mode strings
func ValidModes() []string {
	return []string{"ultra-safe", "normal", "yolo"}
}

// IsValidMode checks if a mode string is valid
func IsValidMode(mode string) bool {
	for _, m := range ValidModes() {
		if m == mode {
			return true
		}
	}
	return false
}
