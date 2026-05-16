package detector

import (
	"os/exec"
	"strings"
)

type Snap struct{}

func (s *Snap) Name() string { return "snap" }

func (s *Snap) Available() bool {
	_, err := exec.LookPath("snap")
	return err == nil
}

func (s *Snap) Detect() ([]Package, error) {
	if !s.Available() {
		return nil, ErrNotAvailable
	}
	out, err := exec.Command("snap", "list", "--unicode=never").Output()
	if err != nil {
		return nil, err
	}
	var pkgs []Package
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	// skip header line
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pkgs = append(pkgs, Package{Name: fields[0], Version: fields[1]})
	}
	return pkgs, nil
}
