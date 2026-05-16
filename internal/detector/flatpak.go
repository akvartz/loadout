package detector

import (
	"os/exec"
	"strings"
)

type Flatpak struct{}

func (f *Flatpak) Name() string { return "flatpak" }

func (f *Flatpak) Available() bool {
	_, err := exec.LookPath("flatpak")
	return err == nil
}

func (f *Flatpak) Detect() ([]Package, error) {
	if !f.Available() {
		return nil, ErrNotAvailable
	}
	out, err := exec.Command("flatpak", "list", "--app", "--columns=application,version").Output()
	if err != nil {
		return nil, err
	}
	var pkgs []Package
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		p := Package{Name: parts[0]}
		if len(parts) > 1 {
			p.Version = parts[1]
		}
		pkgs = append(pkgs, p)
	}
	return pkgs, nil
}
