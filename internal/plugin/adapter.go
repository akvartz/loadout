package plugin

import (
	"time"

	"github.com/akvartz/loadout/internal/plugin/rpc"
	"github.com/akvartz/loadout/internal/state"
)

// stateToWire converts internal state to the JSON wire format.
func stateToWire(s state.State) rpc.WireState {
	sources := make(map[string]rpc.WireSourceState, len(s.Sources))
	for k, v := range s.Sources {
		sources[k] = rpc.WireSourceState{Packages: v.Packages}
	}
	return rpc.WireState{
		Meta: rpc.WireMeta{
			SchemaVersion: s.Meta.SchemaVersion,
			CapturedAt:    s.Meta.CapturedAt.Format(time.RFC3339),
			Hostname:      s.Meta.Hostname,
			OS:            s.Meta.OS,
		},
		Sources: sources,
	}
}

// ExternalHook implements Hook over an rpc.Client.
type ExternalHook struct {
	name   string
	client *rpc.Client
}

func (h *ExternalHook) Name() string { return h.name }

func (h *ExternalHook) PostSnapshot(s state.State, stateFile string) error {
	return h.client.Call("hook.post_snapshot", rpc.HookPostSnapshotParams{
		State: stateToWire(s), StateFile: stateFile,
	}, nil)
}

func (h *ExternalHook) PreRestore(s state.State, target string) error {
	return h.client.Call("hook.pre_restore", rpc.HookPreRestoreParams{
		State: stateToWire(s), Target: target,
	}, nil)
}

func (h *ExternalHook) PostRestore(s state.State, target string) error {
	return h.client.Call("hook.post_restore", rpc.HookPostRestoreParams{
		State: stateToWire(s), Target: target,
	}, nil)
}

// ExternalDetector implements Detector over an rpc.Client.
type ExternalDetector struct {
	name   string
	client *rpc.Client
}

func (d *ExternalDetector) Name() string { return d.name }

func (d *ExternalDetector) Available() bool {
	var res rpc.DetectorAvailableResult
	if err := d.client.Call("detector.available", nil, &res); err != nil {
		return false
	}
	return res.Available
}

func (d *ExternalDetector) Detect() ([]DetectorPackage, error) {
	var res rpc.DetectorDetectResult
	if err := d.client.Call("detector.detect", nil, &res); err != nil {
		return nil, err
	}
	pkgs := make([]DetectorPackage, len(res.Packages))
	for i, p := range res.Packages {
		pkgs[i] = DetectorPackage{Name: p.Name, Version: p.Version}
	}
	return pkgs, nil
}

// ExternalGenerator implements Generator over an rpc.Client.
type ExternalGenerator struct {
	name   string
	client *rpc.Client
}

func (g *ExternalGenerator) Name() string { return g.name }

func (g *ExternalGenerator) Generate(s state.State) (string, error) {
	var res rpc.GeneratorGenerateResult
	if err := g.client.Call("generator.generate", rpc.GeneratorGenerateParams{
		State: stateToWire(s),
	}, &res); err != nil {
		return "", err
	}
	return res.Script, nil
}

// ExternalConverter implements Converter over an rpc.Client.
type ExternalConverter struct {
	name   string
	client *rpc.Client
}

func (cv *ExternalConverter) Name() string { return cv.name }

func (cv *ExternalConverter) Convert(req ConvertRequest) ([]ConvertResult, error) {
	var res rpc.ConverterConvertResult
	if err := cv.client.Call("converter.convert", rpc.ConverterConvertParams{
		From:     req.From,
		To:       req.To,
		Packages: req.Packages,
	}, &res); err != nil {
		return nil, err
	}
	results := make([]ConvertResult, len(res.Results))
	for i, r := range res.Results {
		results[i] = ConvertResult{
			Original:  r.Original,
			Converted: r.Converted,
			Found:     r.Found,
		}
	}
	return results, nil
}
