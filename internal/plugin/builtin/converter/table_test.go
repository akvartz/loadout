package converter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseTableAndLookup(t *testing.T) {
	tbl, err := ParseTable([]byte(`
[packages]
fd = { apt = "fd-find", brew = "fd", snap = "" }
`))
	if err != nil {
		t.Fatal(err)
	}

	if got, ok := tbl.Lookup("apt", "fd-find", "brew"); !ok || got != "fd" {
		t.Errorf(`Lookup(apt fd-find -> brew) = %q, %v`, got, ok)
	}
	// Reverse direction works from the same entry.
	if got, ok := tbl.Lookup("brew", "fd", "apt"); !ok || got != "fd-find" {
		t.Errorf(`Lookup(brew fd -> apt) = %q, %v`, got, ok)
	}
	// "" marks the package unavailable: no forward result, no reverse index.
	if _, ok := tbl.Lookup("apt", "fd-find", "snap"); ok {
		t.Error(`Lookup into "" mapping should fail`)
	}
	if _, ok := tbl.Lookup("snap", "", "apt"); ok {
		t.Error(`empty name must not be indexed`)
	}
	// Unknown name and unknown manager.
	if _, ok := tbl.Lookup("apt", "missing", "brew"); ok {
		t.Error("Lookup of unknown name should fail")
	}
	if _, ok := tbl.Lookup("pacman", "fd", "brew"); ok {
		t.Error("Lookup from unknown manager should fail")
	}
}

func TestReverseCollisionIsDeterministic(t *testing.T) {
	// Both canonicals claim nix name "openssh"; alphabetically-first wins.
	data := []byte(`
[packages]
openssh-server = { apt = "openssh-server", nix = "openssh" }
openssh-client = { apt = "openssh-client", nix = "openssh" }
`)
	for i := 0; i < 20; i++ {
		tbl, err := ParseTable(data)
		if err != nil {
			t.Fatal(err)
		}
		if got, _ := tbl.Lookup("nix", "openssh", "apt"); got != "openssh-client" {
			t.Fatalf("run %d: collision resolved to %q, want openssh-client", i, got)
		}
	}
}

func TestMergeFile(t *testing.T) {
	tbl, err := ParseTable([]byte(`
[packages]
fd = { apt = "fd-find", brew = "fd" }
`))
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "user.toml")
	content := `
[packages]
fd = { brew = "fd-user", nix = "fd" }
new = { apt = "new-pkg", brew = "new" }
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := tbl.MergeFile(path); err != nil {
		t.Fatal(err)
	}

	if got, _ := tbl.Lookup("apt", "fd-find", "brew"); got != "fd-user" {
		t.Errorf("override = %q, want fd-user", got)
	}
	if got, _ := tbl.Lookup("apt", "fd-find", "nix"); got != "fd" {
		t.Errorf("added key = %q, want fd", got)
	}
	if got, _ := tbl.Lookup("apt", "new-pkg", "brew"); got != "new" {
		t.Errorf("new entry = %q, want new", got)
	}
	// The index follows the merge: the old reverse name is replaced.
	if _, ok := tbl.Lookup("brew", "fd", "apt"); ok {
		t.Error("stale reverse mapping survived the merge")
	}
	if got, _ := tbl.Lookup("brew", "fd-user", "apt"); got != "fd-find" {
		t.Errorf("reverse of overridden name = %q, want fd-find", got)
	}
}

func TestBuiltinTable(t *testing.T) {
	tbl, err := builtinTable()
	if err != nil {
		t.Fatal(err)
	}
	if tbl.Len() < 100 {
		t.Errorf("built-in table has %d entries, expected at least 100", tbl.Len())
	}

	// Spot-check one mapping per supported manager pair direction.
	cases := []struct{ from, name, to, want string }{
		{"apt", "ripgrep", "nix", "ripgrep"},
		{"apt", "exa", "brew", "eza"},
		{"brew", "the_silver_searcher", "nix", "silver-searcher"},
		{"nix", "du-dust", "cargo", "du-dust"},
		{"snap", "zoom-client", "flatpak", "us.zoom.Zoom"},
		{"flatpak", "org.videolan.VLC", "apt", "vlc"},
		{"pip", "yt-dlp", "brew", "yt-dlp"},
		{"cargo", "watchexec-cli", "brew", "watchexec"},
		{"npm", "typescript", "apt", "node-typescript"},
	}
	for _, tc := range cases {
		got, ok := tbl.Lookup(tc.from, tc.name, tc.to)
		if !ok || got != tc.want {
			t.Errorf("Lookup(%s %q -> %s) = %q, %v; want %q", tc.from, tc.name, tc.to, got, ok, tc.want)
		}
	}
}
