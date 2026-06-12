package converter

import (
	_ "embed"
	"fmt"
	"sort"

	"github.com/BurntSushi/toml"
)

// table.toml maps canonical package identities to their per-manager names.
//
//go:embed table.toml
var builtinTOML []byte

// tableFile is the on-disk schema, shared by the embedded table and user
// override files (the converter's "table" setting):
//
//	[packages]
//	fd = { apt = "fd-find", brew = "fd", nix = "fd", cargo = "fd-find" }
//
// A manager key set to "" means the package is explicitly unavailable there;
// in an override file it removes a built-in mapping.
type tableFile struct {
	Packages map[string]map[string]string `toml:"packages"`
}

// Table translates package names between managers. Every entry maps a
// canonical identity to its name in each manager that ships it, so any
// manager pair is convertible in both directions from one dataset.
type Table struct {
	entries map[string]map[string]string // canonical → manager → name
	index   map[string]map[string]string // manager → name → canonical
}

// ParseTable decodes TOML table data.
func ParseTable(data []byte) (*Table, error) {
	var f tableFile
	if err := toml.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	t := &Table{entries: f.Packages}
	if t.entries == nil {
		t.entries = map[string]map[string]string{}
	}
	t.rebuildIndex()
	return t, nil
}

func builtinTable() (*Table, error) {
	t, err := ParseTable(builtinTOML)
	if err != nil {
		return nil, fmt.Errorf("parsing built-in conversion table: %w", err)
	}
	return t, nil
}

// MergeFile overlays mappings from a user table file. Entries sharing a
// canonical name with a built-in are merged key by key, so an override only
// needs to list the managers it changes.
func (t *Table) MergeFile(path string) error {
	var f tableFile
	if _, err := toml.DecodeFile(path, &f); err != nil {
		return err
	}
	for canon, names := range f.Packages {
		entry := t.entries[canon]
		if entry == nil {
			entry = map[string]string{}
			t.entries[canon] = entry
		}
		for mgr, name := range names {
			entry[mgr] = name
		}
	}
	t.rebuildIndex()
	return nil
}

// rebuildIndex recomputes the reverse lookup. Canonical names are visited in
// sorted order so collisions (two entries claiming the same name in the same
// manager) resolve deterministically: alphabetically-first canonical wins.
func (t *Table) rebuildIndex() {
	t.index = map[string]map[string]string{}

	canons := make([]string, 0, len(t.entries))
	for canon := range t.entries {
		canons = append(canons, canon)
	}
	sort.Strings(canons)

	for _, canon := range canons {
		for mgr, name := range t.entries[canon] {
			if name == "" {
				continue
			}
			byName := t.index[mgr]
			if byName == nil {
				byName = map[string]string{}
				t.index[mgr] = byName
			}
			if _, exists := byName[name]; !exists {
				byName[name] = canon
			}
		}
	}
}

// Lookup translates name from one manager's namespace to another's.
func (t *Table) Lookup(from, name, to string) (string, bool) {
	canon, ok := t.index[from][name]
	if !ok {
		return "", false
	}
	out := t.entries[canon][to]
	if out == "" {
		return "", false
	}
	return out, true
}

// Len returns the number of canonical entries.
func (t *Table) Len() int { return len(t.entries) }
