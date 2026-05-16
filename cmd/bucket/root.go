package bucket

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	stateFile string
	verbose   bool
)

var rootCmd = &cobra.Command{
	Use:   "loadout",
	Short: "Save and restore your installed software as code",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&stateFile, "file", "f", "loadout.toml", "state file path")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
}
