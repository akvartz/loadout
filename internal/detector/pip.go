package detector

import (
	"encoding/json"
	"os/exec"
)

type Pip struct{}

func (p *Pip) Name() string { return "pip" }

func (p *Pip) Available() bool {
	for _, bin := range []string{"pip3", "pip"} {
		if _, err := exec.LookPath(bin); err == nil {
			return true
		}
	}
	return false
}

func (p *Pip) binary() string {
	for _, bin := range []string{"pip3", "pip"} {
		if _, err := exec.LookPath(bin); err == nil {
			return bin
		}
	}
	return ""
}

func (p *Pip) Detect() ([]Package, error) {
	if !p.Available() {
		return nil, ErrNotAvailable
	}
	out, err := exec.Command(p.binary(), "list", "--user", "--format=json").Output()
	if err != nil {
		return nil, err
	}
	var items []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(out, &items); err != nil {
		return nil, err
	}
	pkgs := make([]Package, len(items))
	for i, item := range items {
		pkgs[i] = Package{Name: item.Name, Version: item.Version}
	}
	return pkgs, nil
}
