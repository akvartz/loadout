package plugin

import "github.com/akvartz/loadout/internal/state"

// Capability identifies what a plugin can do.
type Capability string

const (
	CapHook      Capability = "hook"
	CapDetector  Capability = "detector"
	CapGenerator Capability = "generator"
	CapConverter Capability = "converter"
)

// DetectorPackage mirrors detector.Package to avoid import cycles.
type DetectorPackage struct {
	Name    string
	Version string
}

// ErrNotAvailable is returned by a plugin Detector when the manager is absent.
var ErrNotAvailable = errNotAvailable("package manager not available")

type errNotAvailable string

func (e errNotAvailable) Error() string { return string(e) }

// ConvertRequest describes a batch name-translation operation.
type ConvertRequest struct {
	From     string   // source manager, e.g. "apt"
	To       string   // target manager, e.g. "nix"
	Packages []string // original package names
}

// ConvertResult is one output entry per input package.
type ConvertResult struct {
	Original  string
	Converted string
	Found     bool
}

// Hook receives lifecycle callbacks from loadout commands.
// All errors are printed to stderr; loadout continues regardless.
type Hook interface {
	Name() string
	PostSnapshot(s state.State, stateFile string) error
	PreRestore(s state.State, target string) error
	PostRestore(s state.State, target string) error
}

// Detector adds a new package manager source to snapshot and diff.
type Detector interface {
	Name() string
	Available() bool
	Detect() ([]DetectorPackage, error)
}

// Generator adds a new restore target.
type Generator interface {
	Name() string
	Generate(s state.State) (string, error)
}

// Converter translates package names between managers.
type Converter interface {
	Name() string
	Convert(req ConvertRequest) ([]ConvertResult, error)
}
