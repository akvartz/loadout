package converter

import (
	"testing"

	"github.com/akvartz/loadout/internal/plugin"
)

func TestConvertAptToNix(t *testing.T) {
	c := &Converter{}
	results, err := c.Convert(plugin.ConvertRequest{
		From:     "apt",
		To:       "nix",
		Packages: []string{"fd-find", "golang-go", "no-such-package"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Fatalf("got %d results, want 3", len(results))
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
	c := &Converter{}
	results, err := c.Convert(plugin.ConvertRequest{
		From:     "apt",
		To:       "brew",
		Packages: []string{"silversearcher-ag", "nodejs"},
	})
	if err != nil {
		t.Fatal(err)
	}

	want := []plugin.ConvertResult{
		{Original: "silversearcher-ag", Converted: "the_silver_searcher", Found: true},
		{Original: "nodejs", Converted: "node", Found: true},
	}
	for i, w := range want {
		if results[i] != w {
			t.Errorf("results[%d] = %+v, want %+v", i, results[i], w)
		}
	}
}

func TestConvertUnknownManagers(t *testing.T) {
	c := &Converter{}
	for _, req := range []plugin.ConvertRequest{
		{From: "snap", To: "nix", Packages: []string{"htop"}},
		{From: "apt", To: "pacman", Packages: []string{"htop"}},
	} {
		results, err := c.Convert(req)
		if err != nil {
			t.Fatal(err)
		}
		if len(results) != 1 || results[0].Found {
			t.Errorf("Convert(%s→%s) = %+v, want single not-found result", req.From, req.To, results)
		}
		if results[0].Original != "htop" {
			t.Errorf("Convert(%s→%s) Original = %q, want %q", req.From, req.To, results[0].Original, "htop")
		}
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
