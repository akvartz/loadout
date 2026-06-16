package cloud

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/akvartz/loadout/internal/state"
)

func TestCopyFile(t *testing.T) {
	tempDir := t.TempDir()
	src := filepath.Join(tempDir, "src.txt")
	dst := filepath.Join(tempDir, "dst.txt")

	err := os.WriteFile(src, []byte("test data"), 0644)
	if err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	err = copyFile(src, dst)
	if err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	info, err := os.Stat(dst)
	if err != nil {
		t.Fatalf("failed to stat destination file: %v", err)
	}

	if info.Mode().Perm() != 0600 {
		t.Errorf("expected permissions 0600, got %v", info.Mode().Perm())
	}
}

func TestCloud_PostSnapshot_CreatesDirWithSecurePermissions(t *testing.T) {
	tempDir := t.TempDir()
	destDir := filepath.Join(tempDir, "backup_dest")
	destFile := filepath.Join(destDir, "loadout.toml")

	c := &Cloud{destination: destFile}

	// Create a dummy source state file
	srcFile := filepath.Join(tempDir, "source.toml")
	if err := os.WriteFile(srcFile, []byte("dummy state"), 0600); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	// Call PostSnapshot
	err := c.PostSnapshot(state.State{}, srcFile)
	if err != nil {
		t.Fatalf("PostSnapshot failed: %v", err)
	}

	// Verify directory permissions
	info, err := os.Stat(destDir)
	if err != nil {
		t.Fatalf("failed to stat destination directory: %v", err)
	}

	// MkdirAll applies the umask, so the result can only be 0700 or stricter.
	perms := info.Mode().Perm()
	expectedPerms := os.FileMode(0700)
	if perms != expectedPerms {
		if perms&0077 != 0 {
			t.Errorf("destination directory has insecure permissions: %v (expected %v or more restrictive)", perms, expectedPerms)
		}
	}
}
