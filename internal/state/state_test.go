package state

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestWriteReadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "loadout.toml")

	s := New()
	s.Sources["apt"] = SourceState{Packages: []string{"curl", "git"}}
	s.Sources["flatpak"] = SourceState{Packages: []string{"org.signal.Signal"}}

	if err := Write(path, s); err != nil {
		t.Fatal(err)
	}
	got, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got.Sources, s.Sources) {
		t.Errorf("sources after round trip = %v, want %v", got.Sources, s.Sources)
	}
	if got.Meta.SchemaVersion != 1 {
		t.Errorf("schema version = %d, want 1", got.Meta.SchemaVersion)
	}
	if !got.Meta.CapturedAt.Equal(s.Meta.CapturedAt) {
		t.Errorf("captured_at = %v, want %v", got.Meta.CapturedAt, s.Meta.CapturedAt)
	}
}

func TestReadMissingFile(t *testing.T) {
	if _, err := Read(filepath.Join(t.TempDir(), "missing.toml")); err == nil {
		t.Error("Read of missing file should return an error")
	}
}

func TestReadInitializesNilSources(t *testing.T) {
	path := filepath.Join(t.TempDir(), "loadout.toml")
	if err := Write(path, State{Meta: Meta{SchemaVersion: 1}}); err != nil {
		t.Fatal(err)
	}
	got, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Sources == nil {
		t.Error("Read should initialize a nil Sources map")
	}
}
