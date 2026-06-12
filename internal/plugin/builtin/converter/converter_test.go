package converter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/akvartz/loadout/internal/plugin"
)

func newConverter(t *testing.T, settings map[string]string) *Converter {
	t.Helper()
	c, err := New(settings)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func convertOne(t *testing.T, c *Converter, from, to, pkg string) plugin.ConvertResult {
	t.Helper()
	results, err := c.Convert(plugin.ConvertRequest{From: from, To: to, Packages: []string{pkg}})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	return results[0]
}

func TestConvertAptToNix(t *testing.T) {
	c := newConverter(t, nil)
	results, err := c.Convert(plugin.ConvertRequest{
		From:     "apt",
		To:       "nix",
		Packages: []string{"fd-find", "golang-go", "no-such-package"},
	})
	if err != nil {
		t.Fatal(err)
	}

	want := []plugin.ConvertResult{
		{Original: "fd-find", Converted: "fd", Found: true},
		{Original: "golang-go", Converted: "go", Found: true},
		{Original: "no-such-package", Found: false},
	}
	for i, w := range want {
		if results[i] != w {
			t.Errorf("results[%d] = %+v, want %+v", i, results[i], w)
		}
	}
}

func TestConvertAptToBrew(t *testing.T) {
	c := newConverter(t, nil)
	cases := map[string]string{
		"silversearcher-ag": "the_silver_searcher",
		"nodejs":            "node",
		"fd-find":           "fd",
	}
	for in, want := range cases {
		if got := convertOne(t, c, "apt", "brew", in); !got.Found || got.Converted != want {
			t.Errorf("apt %q -> brew = %+v, want %q", in, got, want)
		}
	}
}

func TestConvertIsBidirectional(t *testing.T) {
	c := newConverter(t, nil)
	cases := []struct{ from, to, in, want string }{
		{"brew", "apt", "fd", "fd-find"},
		{"brew", "apt", "the_silver_searcher", "silversearcher-ag"},
		{"nix", "apt", "gnumake", "make"},
		{"nix", "brew", "kubernetes-helm", "helm"},
	}
	for _, tc := range cases {
		if got := convertOne(t, c, tc.from, tc.to, tc.in); !got.Found || got.Converted != tc.want {
			t.Errorf("%s %q -> %s = %+v, want %q", tc.from, tc.in, tc.to, got, tc.want)
		}
	}
}

func TestConvertSnapAndFlatpak(t *testing.T) {
	c := newConverter(t, nil)
	cases := []struct{ from, to, in, want string }{
		{"snap", "nix", "code", "vscode"},
		{"snap", "apt", "nvim", "neovim"},
		{"flatpak", "apt", "org.gimp.GIMP", "gimp"},
		{"flatpak", "nix", "us.zoom.Zoom", "zoom-us"},
		{"apt", "flatpak", "gimp", "org.gimp.GIMP"},
		{"pip", "brew", "ranger-fm", "ranger"},
		{"cargo", "apt", "fd-find", "fd-find"},
		{"npm", "brew", "firebase-tools", "firebase-cli"},
		{"flatpak", "brew-cask", "org.gimp.GIMP", "gimp"},
		{"snap", "brew-cask", "code", "visual-studio-code"},
		{"brew-cask", "apt", "firefox", "firefox"},
	}
	for _, tc := range cases {
		if got := convertOne(t, c, tc.from, tc.to, tc.in); !got.Found || got.Converted != tc.want {
			t.Errorf("%s %q -> %s = %+v, want %q", tc.from, tc.in, tc.to, got, tc.want)
		}
	}
}

func TestConvertUnknownManagers(t *testing.T) {
	c := newConverter(t, nil)
	for _, req := range []plugin.ConvertRequest{
		{From: "pacman", To: "nix", Packages: []string{"htop"}},
		{From: "apt", To: "pacman", Packages: []string{"some-unknown-pkg"}},
	} {
		results, err := c.Convert(req)
		if err != nil {
			t.Fatal(err)
		}
		if len(results) != 1 || results[0].Found {
			t.Errorf("Convert(%s -> %s) = %+v, want single not-found result", req.From, req.To, results)
		}
		if results[0].Original != req.Packages[0] {
			t.Errorf("Convert(%s -> %s) Original = %q, want %q", req.From, req.To, results[0].Original, req.Packages[0])
		}
	}
}

func TestIdentityFallback(t *testing.T) {
	c := newConverter(t, map[string]string{"fallback": "identity"})

	// A table hit still wins over identity.
	if got := convertOne(t, c, "apt", "brew", "fd-find"); got.Converted != "fd" {
		t.Errorf("table hit overridden by identity: %+v", got)
	}
	// Unknown names carry over unchanged.
	if got := convertOne(t, c, "apt", "brew", "obscure-tool"); !got.Found || got.Converted != "obscure-tool" {
		t.Errorf("identity fallback = %+v, want obscure-tool found", got)
	}
	// Dotted names (flatpak IDs, docker.io) never carry over.
	for _, pkg := range []string{"io.github.some.App", "docker.io"} {
		if got := convertOne(t, c, "flatpak", "brew", pkg); got.Found {
			t.Errorf("identity fallback must skip dotted name %q, got %+v", pkg, got)
		}
	}
	// Namespaces with their own naming schemes are never guessed into.
	for _, to := range []string{"brew-cask", "flatpak"} {
		if got := convertOne(t, c, "apt", to, "obscure-tool"); got.Found {
			t.Errorf("identity fallback must not target %s, got %+v", to, got)
		}
	}
}

func TestIdentityFallbackOffByDefault(t *testing.T) {
	c := newConverter(t, nil)
	if got := convertOne(t, c, "apt", "brew", "obscure-tool"); got.Found {
		t.Errorf("identity fallback should be off by default, got %+v", got)
	}
}

func TestInvalidFallbackSetting(t *testing.T) {
	if _, err := New(map[string]string{"fallback": "guess"}); err == nil {
		t.Error("New with invalid fallback should return an error")
	}
}

func TestUserTableOverridesAndExtends(t *testing.T) {
	path := filepath.Join(t.TempDir(), "conversions.toml")
	content := `
[packages]
# Override one mapping of a built-in entry and disable another.
fd = { brew = "fd-custom", nix = "" }
# Brand new entry.
mytool = { apt = "mytool-bin", brew = "mytool" }
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	c := newConverter(t, map[string]string{"table": path})

	if got := convertOne(t, c, "apt", "brew", "fd-find"); got.Converted != "fd-custom" {
		t.Errorf("override not applied: %+v", got)
	}
	if got := convertOne(t, c, "apt", "nix", "fd-find"); got.Found {
		t.Errorf("disabled mapping still found: %+v", got)
	}
	// Untouched keys of the merged entry survive.
	if got := convertOne(t, c, "apt", "cargo", "fd-find"); !got.Found || got.Converted != "fd-find" {
		t.Errorf("merge clobbered untouched key: %+v", got)
	}
	if got := convertOne(t, c, "apt", "brew", "mytool-bin"); !got.Found || got.Converted != "mytool" {
		t.Errorf("new entry not added: %+v", got)
	}
}

func TestUserTableMissingFile(t *testing.T) {
	if _, err := New(map[string]string{"table": filepath.Join(t.TempDir(), "nope.toml")}); err == nil {
		t.Error("New with missing table file should return an error")
	}
}

func TestConverterIsRegisteredAsBuiltin(t *testing.T) {
	for _, name := range plugin.BuiltinNames() {
		if name == "converter" {
			return
		}
	}
	t.Error("converter not found in plugin.BuiltinNames()")
}
