package detector

import (
	"encoding/json"
	"os/exec"
)

type Npm struct{}

func (n *Npm) Name() string { return "npm" }

func (n *Npm) Available() bool {
	_, err := exec.LookPath("npm")
	return err == nil
}

func (n *Npm) Detect() ([]Package, error) {
	if !n.Available() {
		return nil, ErrNotAvailable
	}
	out, err := exec.Command("npm", "list", "-g", "--depth=0", "--json").Output()
	if err != nil {
		// npm exits non-zero when there are peer dep issues; output is still valid JSON
		if len(out) == 0 {
			return nil, err
		}
	}
	var result struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, err
	}
	var pkgs []Package
	for name, info := range result.Dependencies {
		pkgs = append(pkgs, Package{Name: name, Version: info.Version})
	}
	return pkgs, nil
}
