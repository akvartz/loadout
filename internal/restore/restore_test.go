package restore

import (
	"strings"
	"testing"

	"github.com/akvartz/loadout/internal/state"
)

func stateWith(sources map[string][]string) state.State {
	s := state.New()
	for src, pkgs := range sources {
		s.Sources[src] = state.SourceState{Packages: pkgs}
	}
	return s
}

func TestShellGenerate(t *testing.T) {
	script, err := NewShell().Generate(stateWith(map[string][]string{
		"apt":       {"curl", "git"},
		"flatpak":   {"org.signal.Signal"},
		"cargo":     {"ripgrep"},
		"brew-cask": {"firefox"},
	}))
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"#!/usr/bin/env sh",
		"set -e",
		"sudo apt-get install -y curl git",
		"flatpak install -y flathub org.signal.Signal",
		"cargo install ripgrep",
		"brew install --cask firefox",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("script missing %q:\n%s", want, script)
		}
	}
}

func TestNixGenerate(t *testing.T) {
	script, err := NewNix().Generate(stateWith(map[string][]string{
		"nix": {"fd", "htop"},
		"apt": {"cowsay"},
	}))
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"home.packages = with pkgs; [",
		"  fd\n",
		"  htop\n",
		"#   cowsay",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("nix snippet missing %q:\n%s", want, script)
		}
	}
}

func TestGeneratorsHandleEmptyState(t *testing.T) {
	for _, gen := range []Generator{NewShell(), NewBrewfile(), NewNix()} {
		if _, err := gen.Generate(state.New()); err != nil {
			t.Errorf("%s: Generate(empty) returned error: %v", gen.Name(), err)
		}
	}
}
