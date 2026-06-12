package converter

import (
	"fmt"
	"strings"

	"github.com/akvartz/loadout/internal/config"
	"github.com/akvartz/loadout/internal/plugin"
)

func init() {
	plugin.Register("converter", func(settings map[string]string) (interface{}, error) {
		return New(settings)
	})
}

// Converter translates package names between managers using the embedded
// canonical table, optionally overlaid with a user-provided table.
//
// Settings (from [plugin.converter] in config.toml):
//
//	table    — path to a user table (same TOML format as the built-in one);
//	           its mappings extend and override the built-ins
//	fallback — "none" (default) or "identity": assume an untranslated name is
//	           the same in the target manager
type Converter struct {
	tbl      *Table
	identity bool
}

// New builds a Converter from plugin settings.
func New(settings map[string]string) (*Converter, error) {
	tbl, err := builtinTable()
	if err != nil {
		return nil, err
	}
	if path := settings["table"]; path != "" {
		path = config.ExpandTilde(path)
		if err := tbl.MergeFile(path); err != nil {
			return nil, fmt.Errorf("loading conversion table %s: %w", path, err)
		}
	}

	c := &Converter{tbl: tbl}
	switch settings["fallback"] {
	case "", "none":
	case "identity":
		c.identity = true
	default:
		return nil, fmt.Errorf("unknown fallback %q (valid: none, identity)", settings["fallback"])
	}
	return c, nil
}

// identityUnsafe lists target namespaces the identity fallback must not
// guess into: their names follow conventions of their own, so a same-name
// assumption is almost always wrong.
var identityUnsafe = map[string]bool{
	"brew-cask": true,
	"flatpak":   true,
}

func (c *Converter) Name() string { return "converter" }

func (c *Converter) Convert(req plugin.ConvertRequest) ([]plugin.ConvertResult, error) {
	results := make([]plugin.ConvertResult, len(req.Packages))
	for i, pkg := range req.Packages {
		if translated, ok := c.tbl.Lookup(req.From, pkg, req.To); ok {
			results[i] = plugin.ConvertResult{Original: pkg, Converted: translated, Found: true}
			continue
		}
		// Identity fallback: assume the name carries over unchanged. Names
		// containing a dot never do (flatpak app IDs, apt's docker.io), so
		// they are left untranslated regardless. Namespaces with their own
		// naming schemes (casks, flatpak IDs) are never targeted blindly.
		if c.identity && !strings.Contains(pkg, ".") && !identityUnsafe[req.To] {
			results[i] = plugin.ConvertResult{Original: pkg, Converted: pkg, Found: true}
			continue
		}
		results[i] = plugin.ConvertResult{Original: pkg, Found: false}
	}
	return results, nil
}
