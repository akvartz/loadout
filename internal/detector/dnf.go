package detector

import (
	"os/exec"
	"strings"
)

type Dnf struct{}

func (d *Dnf) Name() string { return "dnf" }

func (d *Dnf) Available() bool {
	_, err := exec.LookPath("dnf")
	return err == nil
}

func (d *Dnf) Detect() ([]Package, error) {
	if !d.Available() {
		return nil, ErrNotAvailable
	}
	out, err := exec.Command("dnf", "repoquery", "--userinstalled", "--qf", "%{name}\n").Output()
	if err != nil {
		return nil, err
	}
	// One line per installed NEVRA; multiple architectures of the same
	// package collapse to one name.
	seen := map[string]bool{}
	var pkgs []Package
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || seen[line] {
			continue
		}
		seen[line] = true
		pkgs = append(pkgs, Package{Name: line})
	}
	return pkgs, nil
}
