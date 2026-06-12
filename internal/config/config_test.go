package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingFileReturnsZeroConfig(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "missing.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Plugins.Enabled) != 0 {
		t.Errorf("missing config should enable no plugins, got %v", cfg.Plugins.Enabled)
	}
}

func TestLoadParsesPluginsAndSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	content := `
[plugins]
enabled = ["converter", "cloud"]

[plugin.cloud]
destination = "~/backups/loadout.toml"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if !cfg.IsEnabled("converter") || !cfg.IsEnabled("cloud") {
		t.Errorf("enabled = %v, want converter and cloud enabled", cfg.Plugins.Enabled)
	}
	if cfg.IsEnabled("other") {
		t.Error("IsEnabled(\"other\") = true, want false")
	}
	if got := cfg.Settings("cloud")["destination"]; got != "~/backups/loadout.toml" {
		t.Errorf("cloud destination = %q", got)
	}
	if got := cfg.Settings("converter"); got == nil || len(got) != 0 {
		t.Errorf("Settings for plugin without a table should be an empty map, got %v", got)
	}
}

func TestExpandTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	if got := ExpandTilde("~/x"); got != filepath.Join(home, "x") {
		t.Errorf("ExpandTilde(~/x) = %q", got)
	}
	if got := ExpandTilde("/abs/path"); got != "/abs/path" {
		t.Errorf("ExpandTilde should leave absolute paths alone, got %q", got)
	}
}
