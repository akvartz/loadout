package restore

import (
	"fmt"
	"strings"

	"github.com/ak/loadout/internal/state"
)

type Shell struct{}

func NewShell() *Shell { return &Shell{} }

func (s *Shell) Name() string { return "shell" }

func (s *Shell) Generate(st state.State) (string, error) {
	var b strings.Builder
	b.WriteString("#!/usr/bin/env sh\nset -e\n\n")

	if pkgs := st.Sources["apt"].Packages; len(pkgs) > 0 {
		fmt.Fprintf(&b, "# apt\nsudo apt-get install -y %s\n\n", strings.Join(pkgs, " "))
	}

	if pkgs := st.Sources["flatpak"].Packages; len(pkgs) > 0 {
		b.WriteString("# flatpak\n")
		for _, p := range pkgs {
			fmt.Fprintf(&b, "flatpak install -y flathub %s\n", p)
		}
		b.WriteString("\n")
	}

	if pkgs := st.Sources["snap"].Packages; len(pkgs) > 0 {
		b.WriteString("# snap\n")
		for _, p := range pkgs {
			fmt.Fprintf(&b, "sudo snap install %s\n", p)
		}
		b.WriteString("\n")
	}

	if pkgs := st.Sources["pip"].Packages; len(pkgs) > 0 {
		fmt.Fprintf(&b, "# pip\npip install --user %s\n\n", strings.Join(pkgs, " "))
	}

	if pkgs := st.Sources["cargo"].Packages; len(pkgs) > 0 {
		fmt.Fprintf(&b, "# cargo\ncargo install %s\n\n", strings.Join(pkgs, " "))
	}

	if pkgs := st.Sources["npm"].Packages; len(pkgs) > 0 {
		fmt.Fprintf(&b, "# npm (global)\nnpm install -g %s\n\n", strings.Join(pkgs, " "))
	}

	if pkgs := st.Sources["brew"].Packages; len(pkgs) > 0 {
		fmt.Fprintf(&b, "# brew\nbrew install %s\n\n", strings.Join(pkgs, " "))
	}

	if pkgs := st.Sources["nix"].Packages; len(pkgs) > 0 {
		fmt.Fprintf(&b, "# nix\nnix-env -iA %s\n\n", strings.Join(pkgs, " "))
	}

	if pkgs := st.Sources["appimage"].Packages; len(pkgs) > 0 {
		b.WriteString("# AppImages — manual download required:\n")
		for _, p := range pkgs {
			fmt.Fprintf(&b, "#   %s\n", p)
		}
		b.WriteString("\n")
	}

	return b.String(), nil
}
