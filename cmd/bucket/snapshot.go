package bucket

import (
	"errors"
	"fmt"
	"os"

	"github.com/ak/loadout/internal/detector"
	"github.com/ak/loadout/internal/state"
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
	detectors := detector.All()
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

	if err := state.Write(stateFile, s); err != nil {
		return fmt.Errorf("writing state: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "saved to %s\n", stateFile)
	return nil
}
