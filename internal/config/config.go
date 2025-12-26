package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds the hub-tui configuration.
type Config struct {
	ServerURL string `json:"server_url"`
	Token     string `json:"token,omitempty"`
	TokenExp  string `json:"token_expires,omitempty"`
}

// DefaultPath returns the default config file path.
func DefaultPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "hub-tui", "config.json"), nil
}

// Load reads the config from the default path.
// If the file doesn't exist, returns a zero Config (not an error).
func Load() (*Config, error) {
	path, err := DefaultPath()
	if err != nil {
		return nil, err
	}
	return LoadFrom(path)
}

// LoadFrom reads the config from the specified path.
// If the file doesn't exist, returns a zero Config (not an error).
func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Save writes the config to the default path.
func (c *Config) Save() error {
	path, err := DefaultPath()
	if err != nil {
		return err
	}
	return c.SaveTo(path)
}

// SaveTo writes the config to the specified path.
func (c *Config) SaveTo(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}
