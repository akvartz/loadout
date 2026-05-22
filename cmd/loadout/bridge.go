package loadout

import (
	"github.com/akvartz/loadout/internal/detector"
	"github.com/akvartz/loadout/internal/plugin"
)

// pluginDetectorBridge adapts plugin.Detector to detector.Detector.
type pluginDetectorBridge struct{ d plugin.Detector }

func (b *pluginDetectorBridge) Name() string    { return b.d.Name() }
func (b *pluginDetectorBridge) Available() bool { return b.d.Available() }

func (b *pluginDetectorBridge) Detect() ([]detector.Package, error) {
	pkgs, err := b.d.Detect()
	if err != nil {
		return nil, err
	}
	out := make([]detector.Package, len(pkgs))
	for i, p := range pkgs {
		out[i] = detector.Package{Name: p.Name, Version: p.Version}
	}
	return out, nil
}

func bridgeDetectors(ps []plugin.Detector) []detector.Detector {
	out := make([]detector.Detector, len(ps))
	for i, d := range ps {
		out[i] = &pluginDetectorBridge{d: d}
	}
	return out
}
