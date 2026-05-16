package plugin

// Factory is called by the manager to instantiate a built-in plugin.
// settings comes from the [plugin.<name>] table in config.toml.
// The returned value must implement one or more of Hook, Detector, Generator, Converter.
type Factory func(settings map[string]string) (interface{}, error)

var builtins = map[string]Factory{}

// Register is called from each built-in package's init() to make itself
// discoverable. Blank-import the builtin package in cmd/loadout to activate it.
func Register(name string, f Factory) {
	builtins[name] = f
}

// BuiltinNames returns all registered built-in plugin names.
func BuiltinNames() []string {
	names := make([]string, 0, len(builtins))
	for n := range builtins {
		names = append(names, n)
	}
	return names
}
