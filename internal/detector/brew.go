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
	out, err := exec.Command("brew", "list", "--formula", "--casks", "-1").Output()
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
