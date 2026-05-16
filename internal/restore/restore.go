package restore

import "github.com/ak/loadout/internal/state"

type Generator interface {
	Name() string
	Generate(s state.State) (string, error)
}
