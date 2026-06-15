package cloud

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/akvartz/loadout/internal/config"
	"github.com/akvartz/loadout/internal/plugin"
	"github.com/akvartz/loadout/internal/state"
)

func init() {
	plugin.Register("cloud", func(settings map[string]string) (interface{}, error) {
		dest := settings["destination"]
		if dest == "" {
			return nil, fmt.Errorf("plugin.cloud: missing required setting: destination")
		}
		return &Cloud{destination: config.ExpandTilde(dest)}, nil
	})
}

// Cloud copies the state file to a configured destination after each snapshot.
type Cloud struct {
	destination string
}

func (c *Cloud) Name() string { return "cloud" }

func (c *Cloud) PostSnapshot(s state.State, stateFile string) error {
	if err := os.MkdirAll(filepath.Dir(c.destination), 0o755); err != nil {
		return fmt.Errorf("cloud: create destination dir: %w", err)
	}
	return copyFile(stateFile, c.destination)
}

func (c *Cloud) PreRestore(_ state.State, _ string) error  { return nil }
func (c *Cloud) PostRestore(_ state.State, _ string) error { return nil }

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
