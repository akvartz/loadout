package diff

import (
	"fmt"
	"io"
	"sort"

	"github.com/ak/loadout/internal/state"
)

type SourceDiff struct {
	Source  string
	Added   []string
	Removed []string
}

func Compute(saved, current state.State) []SourceDiff {
	sources := unionKeys(saved.Sources, current.Sources)
	sort.Strings(sources)

	var diffs []SourceDiff
	for _, src := range sources {
		savedSet := toSet(saved.Sources[src].Packages)
		currentSet := toSet(current.Sources[src].Packages)

		d := SourceDiff{Source: src}
		for pkg := range currentSet {
			if !savedSet[pkg] {
				d.Added = append(d.Added, pkg)
			}
		}
		for pkg := range savedSet {
			if !currentSet[pkg] {
				d.Removed = append(d.Removed, pkg)
			}
		}
		sort.Strings(d.Added)
		sort.Strings(d.Removed)
		diffs = append(diffs, d)
	}
	return diffs
}

func Render(diffs []SourceDiff, w io.Writer) {
	hasDiff := false
	for _, d := range diffs {
		if len(d.Added)+len(d.Removed) == 0 {
			continue
		}
		hasDiff = true
		fmt.Fprintf(w, "[%s]\n", d.Source)
		for _, p := range d.Added {
			fmt.Fprintf(w, "  + %s\n", p)
		}
		for _, p := range d.Removed {
			fmt.Fprintf(w, "  - %s\n", p)
		}
	}
	if !hasDiff {
		fmt.Fprintln(w, "no differences")
	}
}

func toSet(pkgs []string) map[string]bool {
	s := make(map[string]bool, len(pkgs))
	for _, p := range pkgs {
		s[p] = true
	}
	return s
}

func unionKeys(a, b map[string]state.SourceState) []string {
	seen := make(map[string]bool)
	for k := range a {
		seen[k] = true
	}
	for k := range b {
		seen[k] = true
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	return keys
}
