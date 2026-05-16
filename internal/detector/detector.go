package detector

import "errors"

// ErrNotAvailable is returned by Detect when the package manager is not present on the system.
var ErrNotAvailable = errors.New("package manager not available on this system")

type Package struct {
	Name    string
	Version string
}

type Detector interface {
	Name() string
	Available() bool
	Detect() ([]Package, error)
}

// All returns the ordered list of all detectors to run during a snapshot.
func All() []Detector {
	return []Detector{
		&Apt{},
		&Flatpak{},
		&Snap{},
		&Pip{},
		&Cargo{},
		&Npm{},
		&AppImage{},
		&Brew{},
		&Nix{},
	}
}
