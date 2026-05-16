package bucket

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/ak/loadout/internal/restore"
	"github.com/ak/loadout/internal/state"
	"github.com/spf13/cobra"
)

var (
	restoreTarget string
	applyRestore  bool
)

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Generate install commands from saved state",
	RunE:  runRestore,
}

func init() {
	restoreCmd.Flags().StringVarP(&restoreTarget, "target", "t", "shell", "restore target: shell, brewfile, nix")
	restoreCmd.Flags().BoolVar(&applyRestore, "apply", false, "execute the generated script (default: dry-run)")
	rootCmd.AddCommand(restoreCmd)
}

func runRestore(_ *cobra.Command, _ []string) error {
	s, err := state.Read(stateFile)
	if err != nil {
		return fmt.Errorf("reading state file: %w", err)
	}

	var gen restore.Generator
	switch restoreTarget {
	case "shell":
		gen = restore.NewShell()
	case "brewfile":
		gen = restore.NewBrewfile()
	case "nix":
		gen = restore.NewNix()
	default:
		return fmt.Errorf("unknown target %q — valid targets: shell, brewfile, nix", restoreTarget)
	}

	script, err := gen.Generate(s)
	if err != nil {
		return err
	}

	if !applyRestore {
		fmt.Print(script)
		return nil
	}

	fmt.Fprintln(os.Stderr, "WARNING: about to execute the generated script. Press Ctrl-C within 3 seconds to abort.")
	fmt.Print(script)
	fmt.Fprintln(os.Stderr, "\n--- executing ---")

	c := exec.Command("sh", "-c", script)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	return c.Run()
}
