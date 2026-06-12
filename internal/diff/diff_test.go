package diff

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/akvartz/loadout/internal/state"
)

func stateWith(sources map[string][]string) state.State {
	s := state.New()
	for src, pkgs := range sources {
		s.Sources[src] = state.SourceState{Packages: pkgs}
	}
	return s
}

func TestComputeAddedAndRemoved(t *testing.T) {
	saved := stateWith(map[string][]string{
		"apt":     {"curl", "cowsay"},
		"flatpak": {"org.signal.Signal"},
	})
	current := stateWith(map[string][]string{
		"apt":     {"curl", "htop"},
		"flatpak": {"org.signal.Signal"},
		"snap":    {"spotify"},
	})

	diffs := Compute(saved, current)

	// Sources are sorted alphabetically: apt, flatpak, snap.
	if len(diffs) != 3 {
		t.Fatalf("got %d source diffs, want 3: %+v", len(diffs), diffs)
	}
	apt := diffs[0]
	if apt.Source != "apt" || !reflect.DeepEqual(apt.Added, []string{"htop"}) || !reflect.DeepEqual(apt.Removed, []string{"cowsay"}) {
		t.Errorf("apt diff = %+v, want +htop -cowsay", apt)
	}
	flatpak := diffs[1]
	if len(flatpak.Added)+len(flatpak.Removed) != 0 {
		t.Errorf("flatpak diff should be empty, got %+v", flatpak)
	}
	snap := diffs[2]
	if snap.Source != "snap" || !reflect.DeepEqual(snap.Added, []string{"spotify"}) {
		t.Errorf("snap diff = %+v, want +spotify", snap)
	}
}

func TestRenderWithDifferences(t *testing.T) {
	var buf bytes.Buffer
	Render([]SourceDiff{
		{Source: "apt", Added: []string{"htop"}, Removed: []string{"cowsay"}},
		{Source: "flatpak"},
	}, &buf)

	out := buf.String()
	for _, want := range []string{"[apt]", "  + htop", "  - cowsay"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "[flatpak]") {
		t.Errorf("sources without changes should not be rendered:\n%s", out)
	}
}

func TestRenderNoDifferences(t *testing.T) {
	var buf bytes.Buffer
	Render([]SourceDiff{{Source: "apt"}}, &buf)
	if got := strings.TrimSpace(buf.String()); got != "no differences" {
		t.Errorf("output = %q, want %q", got, "no differences")
	}
}
