package plugin

import (
	"reflect"
	"sort"
	"testing"
)

func TestRegisterAndBuiltinNames(t *testing.T) {
	// Save the original builtins map so we don't mess up global state
	originalBuiltins := builtins
	defer func() { builtins = originalBuiltins }()

	// Reset the global map for our tests
	builtins = make(map[string]Factory)

	// Test case 1: Empty registry
	if names := BuiltinNames(); len(names) != 0 {
		t.Errorf("expected empty names list, got %v", names)
	}

	// Helper factory
	dummyFactory := func(settings map[string]string) (interface{}, error) {
		return nil, nil
	}

	// Test case 2: Register single plugin
	Register("test_plugin1", dummyFactory)

	names := BuiltinNames()
	if len(names) != 1 || names[0] != "test_plugin1" {
		t.Errorf("expected [test_plugin1], got %v", names)
	}

	// Verify the factory is actually registered correctly
	if builtins["test_plugin1"] == nil {
		t.Errorf("expected factory to be registered for test_plugin1")
	}

	// Test case 3: Register multiple plugins
	Register("test_plugin2", dummyFactory)
	Register("test_plugin3", dummyFactory)

	names = BuiltinNames()
	if len(names) != 3 {
		t.Errorf("expected 3 plugins, got %d", len(names))
	}

	// Map iteration is non-deterministic in Go, so we need to sort to compare
	sort.Strings(names)
	expected := []string{"test_plugin1", "test_plugin2", "test_plugin3"}
	if !reflect.DeepEqual(names, expected) {
		t.Errorf("expected %v, got %v", expected, names)
	}

	// Test case 4: Overwriting an existing plugin
	newFactory := func(settings map[string]string) (interface{}, error) {
		return "overwritten", nil
	}
	Register("test_plugin1", newFactory)

	// The length should still be 3
	names = BuiltinNames()
	if len(names) != 3 {
		t.Errorf("expected 3 plugins after overwrite, got %d", len(names))
	}

	// Verify it was actually overwritten
	val, _ := builtins["test_plugin1"](nil)
	if val != "overwritten" {
		t.Errorf("expected plugin to be overwritten")
	}
}
