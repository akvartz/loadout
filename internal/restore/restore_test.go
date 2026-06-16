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
		"dnf":       {"htop"},
		"pacman":    {"fzf", "bat"},
		"flatpak":   {"org.signal.Signal"},
		"snap":      {"code"},
		"pip":       {"httpie"},
		"cargo":     {"ripgrep"},
		"npm":       {"prettier"},
		"brew":      {"jq"},
		"brew-cask": {"firefox"},
		"nix":       {"fzf"},
		"appimage":  {"Neovim.appimage"},
	}))
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"#!/usr/bin/env sh",
		"set -e",
		"sudo apt-get install -y curl git",
		"sudo dnf install -y htop",
		"sudo pacman -S --needed --noconfirm fzf bat",
		"flatpak install -y flathub org.signal.Signal",
		"sudo snap install code",
		"pip install --user httpie",
		"cargo install ripgrep",
		"npm install -g prettier",
		"brew install jq",
		"brew install --cask firefox",
		"nix-env -iA fzf",
		"#   Neovim.appimage",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("script missing %q:\n%s", want, script)
		}
	}
}

func TestBrewfileGenerate(t *testing.T) {
	script, err := NewBrewfile().Generate(stateWith(map[string][]string{
		"brew":      {"fd", "ripgrep"},
		"brew-cask": {"firefox"},
		"apt":       {"cowsay"},
	}))
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{`brew "fd"`, `brew "ripgrep"`, `cask "firefox"`} {
		if !strings.Contains(script, want) {
			t.Errorf("Brewfile missing %q:\n%s", want, script)
		}
	}
	// Packages from other managers appear only as comments.
	if !strings.Contains(script, "#   cowsay") {
		t.Errorf("Brewfile should list apt packages as comments:\n%s", script)
	}
	if strings.Contains(script, `brew "cowsay"`) {
		t.Errorf("Brewfile must not emit apt packages as brew entries:\n%s", script)
	}
}

func TestGeneratorsHandleEmptyState(t *testing.T) {
	for _, gen := range []Generator{NewShell(), NewBrewfile(), NewNix()} {
		if _, err := gen.Generate(state.New()); err != nil {
			t.Errorf("%s: Generate(empty) returned error: %v", gen.Name(), err)
		}
	}
}

func TestGeneratorNames(t *testing.T) {
	for _, tc := range []struct {
		gen  Generator
		name string
	}{
		{NewShell(), "shell"},
		{NewBrewfile(), "brewfile"},
		{NewNix(), "nix"},
	} {
		if tc.gen.Name() != tc.name {
			t.Errorf("expected generator name %q, got %q", tc.name, tc.gen.Name())
		}
	}
}
