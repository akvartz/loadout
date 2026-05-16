package plugin

import (
	"fmt"
	"os"

	"github.com/akvartz/loadout/internal/config"
	"github.com/akvartz/loadout/internal/plugin/rpc"
	"github.com/akvartz/loadout/internal/state"
)

// Manager holds all active plugins for a single CLI invocation.
type Manager struct {
	hooks   []Hook
	dets    []Detector
	gens    []Generator
	convs   []Converter
	closers []func() error
}

// New loads plugins according to cfg. Built-in plugins must have been registered
// via Register() before New is called (i.e. their packages blank-imported in cmd/).
// External plugins are discovered from PATH and started as subprocesses.
// Plugin-level errors are printed to stderr; the plugin is skipped.
func New(cfg config.Config) (*Manager, error) {
	m := &Manager{}

	// Activate enabled built-ins.
	for _, name := range cfg.Plugins.Enabled {
		factory, ok := builtins[name]
		if !ok {
			continue // might be an external plugin name
		}
		inst, err := factory(cfg.Settings(name))
		if err != nil {
			fmt.Fprintf(os.Stderr, "plugin %s: %v\n", name, err)
			continue
		}
		m.classify(inst)
	}

	// Discover and start enabled external plugins.
	for _, bin := range rpc.Discover() {
		name := rpc.PluginNameFromBinary(bin)
		if !cfg.IsEnabled(name) {
			continue
		}
		client, hs, err := rpc.Start(bin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "plugin %s: %v\n", name, err)
			continue
		}
		m.closers = append(m.closers, client.Close)
		for _, cap := range hs.Capabilities {
			switch Capability(cap) {
			case CapHook:
				m.hooks = append(m.hooks, &ExternalHook{name: name, client: client})
			case CapDetector:
				m.dets = append(m.dets, &ExternalDetector{name: name, client: client})
			case CapGenerator:
				m.gens = append(m.gens, &ExternalGenerator{name: name, client: client})
			case CapConverter:
				m.convs = append(m.convs, &ExternalConverter{name: name, client: client})
			}
		}
	}

	return m, nil
}

// classify inspects inst and registers it under every interface it satisfies.
func (m *Manager) classify(inst interface{}) {
	if h, ok := inst.(Hook); ok {
		m.hooks = append(m.hooks, h)
	}
	if d, ok := inst.(Detector); ok {
		m.dets = append(m.dets, d)
	}
	if g, ok := inst.(Generator); ok {
		m.gens = append(m.gens, g)
	}
	if cv, ok := inst.(Converter); ok {
		m.convs = append(m.convs, cv)
	}
}

func (m *Manager) Hooks() []Hook         { return m.hooks }
func (m *Manager) Detectors() []Detector { return m.dets }
func (m *Manager) Generators() []Generator { return m.gens }
func (m *Manager) Converters() []Converter { return m.convs }

// Close shuts down all external plugin processes.
func (m *Manager) Close() error {
	var last error
	for _, fn := range m.closers {
		if err := fn(); err != nil {
			last = err
		}
	}
	return last
}

// ApplyConversion rewrites s so that packages from each source are translated
// into target's namespace using registered converters. Converted packages are
// merged into s.Sources[target]; untranslated packages stay in their source.
func (m *Manager) ApplyConversion(s state.State, target string) state.State {
	if len(m.convs) == 0 {
		return s
	}

	out := state.State{Meta: s.Meta, Sources: make(map[string]state.SourceState)}
	for src, ss := range s.Sources {
		out.Sources[src] = ss
	}

	for src, ss := range s.Sources {
		if src == target || len(ss.Packages) == 0 {
			continue
		}

		var converted, unconverted []string
		for _, pkg := range ss.Packages {
			found := false
			for _, conv := range m.convs {
				results, err := conv.Convert(ConvertRequest{From: src, To: target, Packages: []string{pkg}})
				if err != nil {
					fmt.Fprintf(os.Stderr, "converter %s: %v\n", conv.Name(), err)
					continue
				}
				if len(results) > 0 && results[0].Found {
					converted = append(converted, results[0].Converted)
					found = true
					break
				}
			}
			if !found {
				unconverted = append(unconverted, pkg)
			}
		}

		// Merge converted into target source.
		if len(converted) > 0 {
			tgt := out.Sources[target]
			tgt.Packages = append(tgt.Packages, converted...)
			out.Sources[target] = tgt
		}

		// Leave only unconverted in the original source (delete if empty).
		if len(unconverted) > 0 {
			orig := out.Sources[src]
			orig.Packages = unconverted
			out.Sources[src] = orig
		} else {
			delete(out.Sources, src)
		}
	}
	return out
}
