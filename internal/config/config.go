package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Plugins struct {
		Enabled []string `toml:"enabled"`
	} `toml:"plugins"`
	Plugin map[string]map[string]string `toml:"plugin"`
}

// Load reads the config file at path. If the file does not exist, a zero-value
// Config (no plugins enabled) is returned without error.
func Load(path string) (Config, error) {
	var cfg Config
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}
	_, err := toml.DecodeFile(path, &cfg)
	return cfg, err
}

// DefaultPath returns ~/.config/loadout/config.toml, respecting XDG_CONFIG_HOME.
func DefaultPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "loadout", "config.toml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "loadout", "config.toml")
}

// ExpandTilde replaces a leading ~ with the user's home directory.
func ExpandTilde(s string) string {
	if !strings.HasPrefix(s, "~") {
		return s
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return s
	}
	return filepath.Join(home, s[1:])
}

// Settings returns the per-plugin settings map for the named plugin,
// or an empty map if no settings are configured.
func (c Config) Settings(name string) map[string]string {
	if c.Plugin == nil {
		return map[string]string{}
	}
	if s, ok := c.Plugin[name]; ok {
		return s
	}
	return map[string]string{}
}

// IsEnabled reports whether the named plugin appears in plugins.enabled.
func (c Config) IsEnabled(name string) bool {
	for _, n := range c.Plugins.Enabled {
		if n == name {
			return true
		}
	}
	return false
}
