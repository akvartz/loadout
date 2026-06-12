package loadout

import (
	"fmt"
	"os"
	"sort"
	"strings"

	// Blank imports activate built-in plugin init() registrations.
	_ "github.com/akvartz/loadout/internal/plugin/builtin/cloud"
	_ "github.com/akvartz/loadout/internal/plugin/builtin/converter"

	"github.com/akvartz/loadout/internal/plugin"
	"github.com/akvartz/loadout/internal/plugin/rpc"
	"github.com/spf13/cobra"
)

var pluginsCmd = &cobra.Command{
	Use:   "plugins",
	Short: "Manage loadout plugins",
}

var pluginsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available plugins (built-in and external)",
	RunE:  runPluginsList,
}

func init() {
	pluginsCmd.AddCommand(pluginsListCmd)
	rootCmd.AddCommand(pluginsCmd)
}

func runPluginsList(cmd *cobra.Command, _ []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	w := cmd.OutOrStdout()

	// Built-ins.
	fmt.Fprintln(w, "Built-in plugins:")
	names := plugin.BuiltinNames()
	sort.Strings(names)
	for _, name := range names {
		status := "disabled"
		if cfg.IsEnabled(name) {
			status = "enabled"
		}
		fmt.Fprintf(w, "  %-20s %s\n", name, status)
	}

	// External.
	fmt.Fprintln(w, "\nExternal plugins (from PATH):")
	binaries := rpc.Discover()
	if len(binaries) == 0 {
		fmt.Fprintln(w, "  (none found — name binaries loadout-plugin-<name> and put them on PATH)")
		return nil
	}
	for _, bin := range binaries {
		name := rpc.PluginNameFromBinary(bin)
		client, hs, err := rpc.Start(bin, cfg.Settings(name))
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %s: failed to start: %v\n", name, err)
			continue
		}
		client.Close() //nolint:errcheck
		status := "disabled"
		if cfg.IsEnabled(name) {
			status = "enabled"
		}
		caps := strings.Join(hs.Capabilities, ", ")
		fmt.Fprintf(w, "  %-20s %-10s v%s [%s]\n", name, status, hs.Version, caps)
	}
	return nil
}
