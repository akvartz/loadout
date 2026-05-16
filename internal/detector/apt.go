package detector

import (
	"os/exec"
	"strings"
)

type Apt struct{}

func (a *Apt) Name() string { return "apt" }

func (a *Apt) Available() bool {
	_, err := exec.LookPath("apt-mark")
	return err == nil
}

func (a *Apt) Detect() ([]Package, error) {
	if !a.Available() {
		return nil, ErrNotAvailable
	}
	out, err := exec.Command("apt-mark", "showmanual").Output()
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
