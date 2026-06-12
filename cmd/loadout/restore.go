package loadout

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/akvartz/loadout/internal/plugin"
	"github.com/akvartz/loadout/internal/restore"
	"github.com/akvartz/loadout/internal/state"
	"github.com/spf13/cobra"
)

var (
	restoreTarget string
	applyRestore  bool
	convertPkgs   bool
)

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Generate install commands from saved state",
	RunE:  runRestore,
}

func init() {
	restoreCmd.Flags().StringVarP(&restoreTarget, "target", "t", "shell", "restore target: shell, brewfile, nix, or a plugin-provided target")
	restoreCmd.Flags().BoolVar(&applyRestore, "apply", false, "execute the generated script (default: dry-run)")
	restoreCmd.Flags().BoolVar(&convertPkgs, "convert", false, "translate package names to the target manager using converters")
	rootCmd.AddCommand(restoreCmd)
}

func runRestore(_ *cobra.Command, _ []string) error {
	s, err := state.Read(stateFile)
	if err != nil {
		return fmt.Errorf("reading state file: %w", err)
	}

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	mgr, err := plugin.New(cfg)
	if err != nil {
		return fmt.Errorf("loading plugins: %w", err)
	}
	defer mgr.Close() //nolint:errcheck

	// Pre-restore hooks.
	for _, h := range mgr.Hooks() {
		if err := h.PreRestore(s, restoreTarget); err != nil {
			fmt.Fprintf(os.Stderr, "hook %s: PreRestore: %v\n", h.Name(), err)
		}
	}

	// Optionally translate package names into the target manager's namespace.
	if convertPkgs {
		switch convTarget := conversionManager(restoreTarget); {
		case convTarget == "":
			fmt.Fprintf(os.Stderr, "warning: --convert has no effect for target %q\n", restoreTarget)
		case len(mgr.Converters()) == 0:
			fmt.Fprintln(os.Stderr, `warning: --convert requested but no converters are enabled — add one to plugins.enabled in the config (e.g. "converter")`)
		default:
			s = mgr.ApplyConversion(s, convTarget)
		}
	}

	gen, err := resolveGenerator(restoreTarget, mgr)
	if err != nil {
		return err
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
	runErr := c.Run()

	// Post-restore hooks only after successful apply.
	if runErr == nil {
		for _, h := range mgr.Hooks() {
			if err := h.PostRestore(s, restoreTarget); err != nil {
				fmt.Fprintf(os.Stderr, "hook %s: PostRestore: %v\n", h.Name(), err)
			}
		}
	}

	return runErr
}

// conversionManager maps a restore target to the package-manager namespace that
// converters translate into and that the target's generator reads from. The
// "shell" target restores with each native manager, so there is nothing to
// convert into ("").
func conversionManager(target string) string {
	switch target {
	case "shell":
		return ""
	case "brewfile":
		return "brew"
	}
	return target
}

// resolveGenerator finds a Generator by name from built-ins then plugin-provided ones.
func resolveGenerator(target string, mgr *plugin.Manager) (restore.Generator, error) {
	switch target {
	case "shell":
		return restore.NewShell(), nil
	case "brewfile":
		return restore.NewBrewfile(), nil
	case "nix":
		return restore.NewNix(), nil
	}
	for _, g := range mgr.Generators() {
		if g.Name() == target {
			return g, nil
		}
	}
	known := []string{"shell", "brewfile", "nix"}
	for _, g := range mgr.Generators() {
		known = append(known, g.Name())
	}
	return nil, fmt.Errorf("unknown target %q — valid targets: %s", target, strings.Join(known, ", "))
}
