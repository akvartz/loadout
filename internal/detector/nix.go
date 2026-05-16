package detector

import (
	"os/exec"
	"strings"
)

type Nix struct{}

func (n *Nix) Name() string { return "nix" }

func (n *Nix) Available() bool {
	_, err := exec.LookPath("nix-env")
	return err == nil
}

func (n *Nix) Detect() ([]Package, error) {
	if !n.Available() {
		return nil, ErrNotAvailable
	}
	out, err := exec.Command("nix-env", "--query", "--installed").Output()
	if err != nil {
		return nil, err
	}
	var pkgs []Package
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// format: name-version (last hyphen-separated segment is version)
		idx := strings.LastIndex(line, "-")
		if idx > 0 {
			pkgs = append(pkgs, Package{Name: line[:idx], Version: line[idx+1:]})
		} else {
			pkgs = append(pkgs, Package{Name: line})
		}
	}
	return pkgs, nil
}
