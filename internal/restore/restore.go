package restore

import "github.com/akvartz/loadout/internal/state"

type Generator interface {
	Name() string
	Generate(s state.State) (string, error)
}
