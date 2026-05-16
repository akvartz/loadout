package state

import (
	"os"
	"runtime"
	"time"

	"github.com/BurntSushi/toml"
)

type Meta struct {
	SchemaVersion int       `toml:"schema_version"`
	CapturedAt    time.Time `toml:"captured_at"`
	Hostname      string    `toml:"hostname"`
	OS            string    `toml:"os"`
}

type SourceState struct {
	Packages []string `toml:"packages"`
}

type State struct {
	Meta    Meta                   `toml:"meta"`
	Sources map[string]SourceState `toml:"sources"`
}

func New() State {
	hostname, _ := os.Hostname()
	return State{
		Meta: Meta{
			SchemaVersion: 1,
			CapturedAt:    time.Now().UTC(),
			Hostname:      hostname,
			OS:            runtime.GOOS,
		},
		Sources: make(map[string]SourceState),
	}
}

func Read(path string) (State, error) {
	var s State
	if _, err := toml.DecodeFile(path, &s); err != nil {
		return s, err
	}
	if s.Sources == nil {
		s.Sources = make(map[string]SourceState)
	}
	return s, nil
}

func Write(path string, s State) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(s)
}
