package detector

import (
	"encoding/json"
	"os/exec"
)

type Pip struct {
	binPath string
	checked bool
}

func (p *Pip) Name() string { return "pip" }

func (p *Pip) Available() bool {
	return p.binary() != ""
}

func (p *Pip) binary() string {
	if p.checked {
		return p.binPath
	}
	p.checked = true
	for _, bin := range []string{"pip3", "pip"} {
		if path, err := exec.LookPath(bin); err == nil {
			p.binPath = path
			return path
		}
	}
	return ""
}

func (p *Pip) Detect() ([]Package, error) {
	bin := p.binary()
	if bin == "" {
		return nil, ErrNotAvailable
	}
	out, err := exec.Command(bin, "list", "--user", "--format=json").Output()
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
