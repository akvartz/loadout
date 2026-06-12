package plugin

import (
	"errors"
	"reflect"
	"testing"

	"github.com/akvartz/loadout/internal/state"
)

// fakeConverter translates via a static from→to→name map, like the built-in.
type fakeConverter struct {
	name  string
	table map[string]map[string]map[string]string
	err   error
	short bool // return fewer results than inputs to exercise the guard
}

func (f *fakeConverter) Name() string { return f.name }

func (f *fakeConverter) Convert(req ConvertRequest) ([]ConvertResult, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.short {
		return nil, nil
	}
	var toMap map[string]string
	if fromMap := f.table[req.From]; fromMap != nil {
		toMap = fromMap[req.To]
	}
	results := make([]ConvertResult, len(req.Packages))
	for i, pkg := range req.Packages {
		if translated, ok := toMap[pkg]; ok {
			results[i] = ConvertResult{Original: pkg, Converted: translated, Found: true}
		} else {
			results[i] = ConvertResult{Original: pkg, Found: false}
		}
	}
	return results, nil
}

func stateWith(sources map[string][]string) state.State {
	s := state.New()
	for src, pkgs := range sources {
		s.Sources[src] = state.SourceState{Packages: pkgs}
	}
	return s
}

func aptToBrewConverter() *fakeConverter {
	return &fakeConverter{
		name: "fake",
		table: map[string]map[string]map[string]string{
			"apt": {"brew": {
				"fd-find":   "fd",
				"golang-go": "go",
				"git":       "git",
			}},
		},
	}
}

func TestApplyConversionMergesIntoTarget(t *testing.T) {
	m := &Manager{convs: []Converter{aptToBrewConverter()}}
	s := stateWith(map[string][]string{
		"apt":  {"fd-find", "golang-go", "cowsay"},
		"brew": {"wget"},
	})

	out := m.ApplyConversion(s, "brew")

	if got, want := out.Sources["brew"].Packages, []string{"wget", "fd", "go"}; !reflect.DeepEqual(got, want) {
		t.Errorf("brew packages = %v, want %v", got, want)
	}
	if got, want := out.Sources["apt"].Packages, []string{"cowsay"}; !reflect.DeepEqual(got, want) {
		t.Errorf("apt packages = %v, want %v", got, want)
	}
}

func TestApplyConversionDeletesFullyConvertedSource(t *testing.T) {
	m := &Manager{convs: []Converter{aptToBrewConverter()}}
	s := stateWith(map[string][]string{"apt": {"fd-find", "git"}})

	out := m.ApplyConversion(s, "brew")

	if _, ok := out.Sources["apt"]; ok {
		t.Errorf("apt source should be deleted when all packages convert, got %v", out.Sources["apt"])
	}
	if got, want := out.Sources["brew"].Packages, []string{"fd", "git"}; !reflect.DeepEqual(got, want) {
		t.Errorf("brew packages = %v, want %v", got, want)
	}
}

func TestApplyConversionSkipsTargetSource(t *testing.T) {
	m := &Manager{convs: []Converter{aptToBrewConverter()}}
	s := stateWith(map[string][]string{"brew": {"wget", "git"}})

	out := m.ApplyConversion(s, "brew")

	if got, want := out.Sources["brew"].Packages, []string{"wget", "git"}; !reflect.DeepEqual(got, want) {
		t.Errorf("brew packages = %v, want %v", got, want)
	}
}

func TestApplyConversionNoConvertersReturnsInput(t *testing.T) {
	m := &Manager{}
	s := stateWith(map[string][]string{"apt": {"fd-find"}})

	out := m.ApplyConversion(s, "brew")

	if !reflect.DeepEqual(out, s) {
		t.Errorf("state changed with no converters: %v", out)
	}
}

func TestApplyConversionDeduplicatesTarget(t *testing.T) {
	m := &Manager{convs: []Converter{aptToBrewConverter()}}
	s := stateWith(map[string][]string{
		"apt":  {"git"},
		"brew": {"git"},
	})

	out := m.ApplyConversion(s, "brew")

	if got, want := out.Sources["brew"].Packages, []string{"git"}; !reflect.DeepEqual(got, want) {
		t.Errorf("brew packages = %v, want %v", got, want)
	}
}

func TestApplyConversionFallsThroughToNextConverter(t *testing.T) {
	second := &fakeConverter{
		name: "second",
		table: map[string]map[string]map[string]string{
			"apt": {"brew": {"cowsay": "cowsay"}},
		},
	}
	m := &Manager{convs: []Converter{aptToBrewConverter(), second}}
	s := stateWith(map[string][]string{"apt": {"fd-find", "cowsay"}})

	out := m.ApplyConversion(s, "brew")

	if got, want := out.Sources["brew"].Packages, []string{"fd", "cowsay"}; !reflect.DeepEqual(got, want) {
		t.Errorf("brew packages = %v, want %v", got, want)
	}
	if _, ok := out.Sources["apt"]; ok {
		t.Errorf("apt source should be deleted, got %v", out.Sources["apt"])
	}
}

func TestApplyConversionSkipsFailingConverter(t *testing.T) {
	failing := &fakeConverter{name: "failing", err: errors.New("boom")}
	m := &Manager{convs: []Converter{failing, aptToBrewConverter()}}
	s := stateWith(map[string][]string{"apt": {"fd-find"}})

	out := m.ApplyConversion(s, "brew")

	if got, want := out.Sources["brew"].Packages, []string{"fd"}; !reflect.DeepEqual(got, want) {
		t.Errorf("brew packages = %v, want %v", got, want)
	}
}

func TestApplyConversionSkipsConverterWithWrongResultCount(t *testing.T) {
	m := &Manager{convs: []Converter{&fakeConverter{name: "short", short: true}}}
	s := stateWith(map[string][]string{"apt": {"fd-find", "git"}})

	out := m.ApplyConversion(s, "brew")

	// Nothing converted; original packages must not be lost.
	if got, want := out.Sources["apt"].Packages, []string{"fd-find", "git"}; !reflect.DeepEqual(got, want) {
		t.Errorf("apt packages = %v, want %v", got, want)
	}
	if _, ok := out.Sources["brew"]; ok {
		t.Errorf("unexpected brew source: %v", out.Sources["brew"])
	}
}

func TestApplyConversionKeepsSiblingNamespaces(t *testing.T) {
	// A converter that knows brew "docker" also exists as a cask. Without
	// keep, a brew->brew-cask pass would move it out of the formula list.
	conv := &fakeConverter{
		name: "fake",
		table: map[string]map[string]map[string]string{
			"brew": {"brew-cask": {"docker": "docker"}},
			"snap": {"brew-cask": {"spotify": "spotify"}},
		},
	}
	m := &Manager{convs: []Converter{conv}}
	s := stateWith(map[string][]string{
		"brew": {"docker"},
		"snap": {"spotify"},
	})

	out := m.ApplyConversion(s, "brew-cask", "brew", "brew-cask")

	if got, want := out.Sources["brew"].Packages, []string{"docker"}; !reflect.DeepEqual(got, want) {
		t.Errorf("kept brew packages = %v, want %v", got, want)
	}
	if got, want := out.Sources["brew-cask"].Packages, []string{"spotify"}; !reflect.DeepEqual(got, want) {
		t.Errorf("brew-cask packages = %v, want %v", got, want)
	}
}

func TestApplyConversionDeterministicAcrossSources(t *testing.T) {
	conv := &fakeConverter{
		name: "multi",
		table: map[string]map[string]map[string]string{
			"apt":  {"nix": {"fd-find": "fd"}},
			"snap": {"nix": {"htop": "htop"}},
		},
	}
	m := &Manager{convs: []Converter{conv}}
	s := stateWith(map[string][]string{
		"apt":  {"fd-find"},
		"snap": {"htop"},
	})

	// Sources are merged in sorted order (apt before snap), every run.
	want := []string{"fd", "htop"}
	for i := 0; i < 20; i++ {
		out := m.ApplyConversion(s, "nix")
		if got := out.Sources["nix"].Packages; !reflect.DeepEqual(got, want) {
			t.Fatalf("run %d: nix packages = %v, want %v", i, got, want)
		}
	}
}
