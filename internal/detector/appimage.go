package detector

import (
	"os"
	"path/filepath"
	"strings"
)

type AppImage struct{}

func (a *AppImage) Name() string { return "appimage" }

func (a *AppImage) Available() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	for _, dir := range searchDirs(home) {
		if _, err := os.Stat(dir); err == nil {
			return true
		}
	}
	return false
}

func (a *AppImage) Detect() ([]Package, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, ErrNotAvailable
	}

	var pkgs []Package
	for _, dir := range searchDirs(home) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".appimage") {
				pkgs = append(pkgs, Package{Name: e.Name()})
			}
		}
	}
	if len(pkgs) == 0 {
		return nil, ErrNotAvailable
	}
	return pkgs, nil
}

func searchDirs(home string) []string {
	return []string{
		filepath.Join(home, "Applications"),
		filepath.Join(home, ".local", "share", "applications"),
	}
}
