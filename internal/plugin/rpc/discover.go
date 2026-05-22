package rpc

import (
	"os"
	"path/filepath"
	"strings"
)

const prefix = "loadout-plugin-"

// Discover returns absolute paths to all executables named loadout-plugin-<name>
// found in $PATH directories. First match per name wins (PATH order).
func Discover() []string {
	seen := map[string]bool{}
	var results []string

	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasPrefix(e.Name(), prefix) {
				continue
			}
			name := PluginNameFromBinary(e.Name())
			if seen[name] {
				continue
			}
			full := filepath.Join(dir, e.Name())
			info, err := e.Info()
			if err != nil {
				continue
			}
			if info.Mode()&0o111 == 0 {
				continue // not executable
			}
			seen[name] = true
			results = append(results, full)
		}
	}
	return results
}

// PluginNameFromBinary extracts the plugin name from a binary name or path.
// "loadout-plugin-myfoo" → "myfoo"
func PluginNameFromBinary(path string) string {
	base := filepath.Base(path)
	return strings.TrimPrefix(base, prefix)
}
