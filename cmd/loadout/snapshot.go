package loadout

import (
	"errors"
	"fmt"
	"os"

	"github.com/akvartz/loadout/internal/detector"
	"github.com/akvartz/loadout/internal/plugin"
	"github.com/akvartz/loadout/internal/state"
	"github.com/spf13/cobra"
)

var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Capture currently installed packages and save to file",
	RunE:  runSnapshot,
}

func init() {
	rootCmd.AddCommand(snapshotCmd)
}

func runSnapshot(cmd *cobra.Command, _ []string) error {
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
	s := detectCurrentState(detectors)

	if err := state.Write(stateFile, s); err != nil {
		return fmt.Errorf("writing state: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "saved to %s\n", stateFile)

	// Run post-snapshot hooks. Errors are non-fatal.
	for _, h := range mgr.Hooks() {
		if err := h.PostSnapshot(s, stateFile); err != nil {
			fmt.Fprintf(os.Stderr, "hook %s: PostSnapshot: %v\n", h.Name(), err)
		}
	}

	return nil
}

func detectCurrentState(detectors []detector.Detector) state.State {
	s := state.New()

	for _, d := range detectors {
		if verbose {
			fmt.Fprintf(os.Stderr, "detecting %s...\n", d.Name())
		}
		pkgs, err := d.Detect()
		if errors.Is(err, detector.ErrNotAvailable) {
			if verbose {
				fmt.Fprintf(os.Stderr, "  %s: not available, skipping\n", d.Name())
			}
			continue
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s: error: %v\n", d.Name(), err)
			continue
		}
		names := make([]string, len(pkgs))
		for i, p := range pkgs {
			names[i] = p.Name
		}
		s.Sources[d.Name()] = state.SourceState{Packages: names}
		if verbose {
			fmt.Fprintf(os.Stderr, "  %s: found %d packages\n", d.Name(), len(pkgs))
		}
	}

	return s
}
