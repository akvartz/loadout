package loadout

import (
	"reflect"
	"strings"
	"testing"

	"github.com/akvartz/loadout/internal/config"
	"github.com/akvartz/loadout/internal/plugin"
	"github.com/akvartz/loadout/internal/state"
)

func TestConversionManagers(t *testing.T) {
	cases := map[string][]string{
		"shell":    nil,                   // native managers — nothing to convert into
		"brewfile": {"brew", "brew-cask"}, // formulas first, casks for the rest
		"nix":      {"nix"},
		"custom":   {"custom"}, // plugin-provided targets pass through
	}
	for target, want := range cases {
		if got := conversionManagers(target); !reflect.DeepEqual(got, want) {
			t.Errorf("conversionManagers(%q) = %v, want %v", target, got, want)
		}
	}
}

// Regression test: restoring to a Brewfile with --convert must translate apt
// names into the "brew" source the generator reads, not a "brewfile" source.
func TestConvertedRestoreToBrewfile(t *testing.T) {
	var cfg config.Config
	cfg.Plugins.Enabled = []string{"converter"}
	mgr, err := plugin.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Close() //nolint:errcheck

	s := state.New()
	s.Sources["apt"] = state.SourceState{Packages: []string{"fd-find", "libdebian-only-dev"}}
	s.Sources["flatpak"] = state.SourceState{Packages: []string{"org.gimp.GIMP"}}
	s.Sources["snap"] = state.SourceState{Packages: []string{"spotify"}}

	namespaces := conversionManagers("brewfile")
	for _, ns := range namespaces {
		s = mgr.ApplyConversion(s, ns, namespaces...)
	}

	gen, err := resolveGenerator("brewfile", mgr)
	if err != nil {
		t.Fatal(err)
	}
	script, err := gen.Generate(s)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(script, `brew "fd"`) {
		t.Errorf("Brewfile missing converted apt package:\n%s", script)
	}
	// GUI apps from flatpak/snap come out as casks in the second pass.
	for _, want := range []string{`cask "gimp"`, `cask "spotify"`} {
		if !strings.Contains(script, want) {
			t.Errorf("Brewfile missing %s:\n%s", want, script)
		}
	}
	if !strings.Contains(script, "#   libdebian-only-dev") {
		t.Errorf("Brewfile should keep untranslatable apt packages as comments:\n%s", script)
	}
}

// Moving machines across managers: a brew/snap/flatpak snapshot restored on a
// Debian box with --target shell --convert-to apt must come out as apt installs.
func TestConvertedRestoreToShellViaConvertTo(t *testing.T) {
	var cfg config.Config
	cfg.Plugins.Enabled = []string{"converter"}
	mgr, err := plugin.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Close() //nolint:errcheck

	s := state.New()
	s.Sources["brew"] = state.SourceState{Packages: []string{"fd", "the_silver_searcher"}}
	s.Sources["snap"] = state.SourceState{Packages: []string{"nvim"}}
	s.Sources["flatpak"] = state.SourceState{Packages: []string{"org.gimp.GIMP"}}

	s = mgr.ApplyConversion(s, "apt")

	gen, err := resolveGenerator("shell", mgr)
	if err != nil {
		t.Fatal(err)
	}
	script, err := gen.Generate(s)
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{"fd-find", "silversearcher-ag", "neovim", "gimp"} {
		if !strings.Contains(script, want) {
			t.Errorf("shell script missing converted package %q:\n%s", want, script)
		}
	}
	if !strings.Contains(script, "sudo apt-get install -y") {
		t.Errorf("shell script missing apt install line:\n%s", script)
	}
	for _, gone := range []string{"brew install", "snap install", "flatpak install"} {
		if strings.Contains(script, gone) {
			t.Errorf("shell script still installs via source manager (%q):\n%s", gone, script)
		}
	}
}

func TestConvertedRestoreToNix(t *testing.T) {
	var cfg config.Config
	cfg.Plugins.Enabled = []string{"converter"}
	mgr, err := plugin.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Close() //nolint:errcheck

	s := state.New()
	s.Sources["apt"] = state.SourceState{Packages: []string{"golang-go"}}

	for _, ns := range conversionManagers("nix") {
		s = mgr.ApplyConversion(s, ns)
	}

	gen, err := resolveGenerator("nix", mgr)
	if err != nil {
		t.Fatal(err)
	}
	script, err := gen.Generate(s)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(script, "  go\n") {
		t.Errorf("nix snippet missing converted apt package:\n%s", script)
	}
}

func TestResolveGeneratorUnknownTarget(t *testing.T) {
	mgr, err := plugin.New(config.Config{})
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Close() //nolint:errcheck

	if _, err := resolveGenerator("bogus", mgr); err == nil {
		t.Error("resolveGenerator(\"bogus\") should return an error")
	}
}
