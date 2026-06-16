package detector

import (
	"os/exec"
	"strings"
)

type Pacman struct{}

func (p *Pacman) Name() string { return "pacman" }

func (p *Pacman) Available() bool {
	_, err := exec.LookPath("pacman")
	return err == nil
}

func (p *Pacman) Detect() ([]Package, error) {
	if !p.Available() {
		return nil, ErrNotAvailable
	}
	// -Qqe: names only, explicitly installed (the user's own picks).
	out, err := exec.Command("pacman", "-Qqe").Output()
	if err != nil {
		return nil, err
	}
	var pkgs []Package
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			pkgs = append(pkgs, Package{Name: line})
		}
	}
	return pkgs, nil
}
