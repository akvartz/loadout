package detector

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type Cargo struct{}

func (c *Cargo) Name() string { return "cargo" }

func (c *Cargo) Available() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(home, ".cargo"))
	return err == nil
}

func (c *Cargo) Detect() ([]Package, error) {
	if !c.Available() {
		return nil, ErrNotAvailable
	}
	home, _ := os.UserHomeDir()
	cratesFile := filepath.Join(home, ".cargo", ".crates2.json")

	data, err := os.ReadFile(cratesFile)
	if err != nil {
		// fall back to scanning bin directory
		return c.detectFromBin(home)
	}

	var crates struct {
		Installs map[string]struct {
			Bins []string `json:"bins"`
		} `json:"installs"`
	}
	if err := json.Unmarshal(data, &crates); err != nil {
		return c.detectFromBin(home)
	}

	var pkgs []Package
	for key := range crates.Installs {
		// key format: "name version (registry+...)"
		parts := strings.Fields(key)
		if len(parts) >= 2 {
			pkgs = append(pkgs, Package{Name: parts[0], Version: parts[1]})
		}
	}
	return pkgs, nil
}

func (c *Cargo) detectFromBin(home string) ([]Package, error) {
	entries, err := os.ReadDir(filepath.Join(home, ".cargo", "bin"))
	if err != nil {
		return nil, ErrNotAvailable
	}
	var pkgs []Package
	for _, e := range entries {
		if !e.IsDir() {
			pkgs = append(pkgs, Package{Name: e.Name()})
		}
	}
	return pkgs, nil
}
