package cloud

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/akvartz/loadout/internal/state"
)

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

	// We only care about the permission bits
	perms := info.Mode().Perm()
	expectedPerms := os.FileMode(0700)

	// Under umask, the actual permissions might be more restrictive than 0700 (e.g. 0700 & ~umask == 0700),
	// but let's check it's exactly 0700 for this test or bounded by it.
	// Actually os.MkdirAll applies umask, but if we set 0700, umask can only make it stricter.
	// Typically, umask is 0022 or 0002. So 0700 & ~0022 = 0700.
	// If umask is 0077, 0700 & ~0077 = 0700 still (except read/write for owner but we are creating it). Wait, 0700 & ~0077 is 0700 & 0700 = 0700.
	// It's just bounded by 0700.
	if perms != expectedPerms {
		// Just to be safe, check that group and other have no permissions
		if perms&0077 != 0 {
			t.Errorf("destination directory has insecure permissions: %v (expected %v or more restrictive)", perms, expectedPerms)
		}
	}
}
