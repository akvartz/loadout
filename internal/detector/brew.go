package detector

import (
	"os/exec"
	"strings"
)

type Brew struct{}

func (b *Brew) Name() string { return "brew" }

func (b *Brew) Available() bool {
	_, err := exec.LookPath("brew")
	return err == nil
}

func (b *Brew) Detect() ([]Package, error) {
	if !b.Available() {
		return nil, ErrNotAvailable
	}
	out, err := exec.Command("brew", "list", "--formula", "-1").Output()
	if err != nil {
		return nil, err
	}
	return brewLines(out), nil
}

// BrewCask captures Homebrew casks separately from formulas so that restore
// targets can emit the right install form (brew install --cask / cask "x").
type BrewCask struct{}

func (b *BrewCask) Name() string { return "brew-cask" }

func (b *BrewCask) Available() bool {
	_, err := exec.LookPath("brew")
	return err == nil
}

func (b *BrewCask) Detect() ([]Package, error) {
	if !b.Available() {
		return nil, ErrNotAvailable
	}
	out, err := exec.Command("brew", "list", "--cask", "-1").Output()
	if err != nil {
		return nil, err
	}
	return brewLines(out), nil
}

func brewLines(out []byte) []Package {
	var pkgs []Package
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			pkgs = append(pkgs, Package{Name: line})
		}
	}
	return pkgs
}
