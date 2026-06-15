package state

import (
	"os"
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

func TestWriteError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent_dir", "loadout.toml")
	if err := Write(path, State{}); err == nil {
		t.Error("Write to non-existent directory should return an error")
	}
}

func TestReadInvalidData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid.toml")
	if err := os.WriteFile(path, []byte("invalid \n toml \n content"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(path); err == nil {
		t.Error("Read of invalid TOML file should return an error")
	}
}

func TestWriteSuccess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "success.toml")
	err := Write(path, New())
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
}

func TestReadSuccess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "success.toml")
	err := Write(path, New())
	if err != nil {
		t.Fatal(err)
	}

	_, err = Read(path)
	if err != nil {
		t.Errorf("Read failed: %v", err)
	}
}
