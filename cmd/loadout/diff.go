package loadout

import (
	"errors"
	"fmt"
	"os"

	"github.com/akvartz/loadout/internal/detector"
	"github.com/akvartz/loadout/internal/diff"
	"github.com/akvartz/loadout/internal/plugin"
	"github.com/akvartz/loadout/internal/state"
	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Compare saved state against currently installed packages",
	RunE:  runDiff,
}

func init() {
	rootCmd.AddCommand(diffCmd)
}

func runDiff(cmd *cobra.Command, _ []string) error {
	saved, err := state.Read(stateFile)
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

	detectors := append(detector.All(), bridgeDetectors(mgr.Detectors())...)
	current := state.New()

	for _, d := range detectors {
		pkgs, err := d.Detect()
		if errors.Is(err, detector.ErrNotAvailable) {
			continue
		}
		if err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "%s: error: %v\n", d.Name(), err)
			}
			continue
		}
		names := make([]string, len(pkgs))
		for i, p := range pkgs {
			names[i] = p.Name
		}
		current.Sources[d.Name()] = state.SourceState{Packages: names}
	}

	diffs := diff.Compute(saved, current)
	diff.Render(diffs, cmd.OutOrStdout())

	for _, d := range diffs {
		if len(d.Added)+len(d.Removed) > 0 {
			os.Exit(1)
		}
	}
	return nil
}
