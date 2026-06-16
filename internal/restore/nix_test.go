package restore

import (
	"strings"
	"testing"

	"github.com/akvartz/loadout/internal/state"
)

func TestNix_Name(t *testing.T) {
	n := NewNix()
	if got := n.Name(); got != "nix" {
		t.Errorf("Name() = %v, want %v", got, "nix")
	}
}

func TestNix_Generate(t *testing.T) {
	tests := []struct {
		name    string
		state   state.State
		wantOut []string
	}{
		{
			name:  "empty state",
			state: state.New(),
			wantOut: []string{
				"home.packages = with pkgs; [",
				"  # no nix-managed packages found in snapshot",
				"];",
			},
		},
		{
			name: "only nix packages",
			state: stateWith(map[string][]string{
				"nix": {"fd", "htop"},
			}),
			wantOut: []string{
				"home.packages = with pkgs; [",
				"  fd",
				"  htop",
				"];",
			},
		},
		{
			name: "unsupported packages only",
			state: stateWith(map[string][]string{
				"apt": {"cowsay"},
			}),
			wantOut: []string{
				"home.packages = with pkgs; [",
				"  # no nix-managed packages found in snapshot",
				"];",
				"# apt packages (add nix equivalents manually):",
				"#   cowsay",
			},
		},
		{
			name: "mixed packages",
			state: stateWith(map[string][]string{
				"nix":       {"fd", "htop"},
				"apt":       {"cowsay"},
				"brew-cask": {"firefox"},
			}),
			wantOut: []string{
				"home.packages = with pkgs; [",
				"  fd",
				"  htop",
				"];",
				"# apt packages (add nix equivalents manually):",
				"#   cowsay",
				"# brew-cask packages (add nix equivalents manually):",
				"#   firefox",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := NewNix()
			got, err := n.Generate(tt.state)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			for _, want := range tt.wantOut {
				if !strings.Contains(got, want) {
					t.Errorf("Generate() output missing %q\nGot:\n%s", want, got)
				}
			}
		})
	}
}
