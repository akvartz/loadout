package loadout

import (
	"fmt"
	"os"

	"github.com/akvartz/loadout/internal/config"
	"github.com/spf13/cobra"
)

var (
	stateFile  string
	verbose    bool
	configFile string
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
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "plugin config file (default: ~/.config/loadout/config.toml)")
}

// loadConfig reads the plugin config, resolving the path from --config or the default.
func loadConfig() (config.Config, error) {
	path := configFile
	if path == "" {
		path = config.DefaultPath()
	}
	return config.Load(path)
}
