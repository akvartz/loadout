package loadout

import (
	"strings"
	"testing"

	"github.com/akvartz/loadout/internal/config"
	"github.com/akvartz/loadout/internal/plugin"
	"github.com/akvartz/loadout/internal/state"
)

func TestConversionManager(t *testing.T) {
	cases := map[string]string{
		"shell":    "",     // shell restores with native managers — nothing to convert into
		"brewfile": "brew", // the Brewfile generator reads the "brew" source
		"nix":      "nix",
		"custom":   "custom", // plugin-provided targets pass through
	}
	for target, want := range cases {
		if got := conversionManager(target); got != want {
			t.Errorf("conversionManager(%q) = %q, want %q", target, got, want)
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
	s.Sources["apt"] = state.SourceState{Packages: []string{"fd-find", "cowsay"}}

	s = mgr.ApplyConversion(s, conversionManager("brewfile"))

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
	if !strings.Contains(script, "#   cowsay") {
		t.Errorf("Brewfile should keep untranslatable apt packages as comments:\n%s", script)
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

	s = mgr.ApplyConversion(s, conversionManager("nix"))

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
