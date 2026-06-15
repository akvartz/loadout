package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
)

// fakeRepology serves canned project-by responses and counts requests.
// projects maps "repo|name_type|name" to a project JSON body.
type fakeRepology struct {
	projects map[string]string
	requests atomic.Int64
}

func (f *fakeRepology) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.requests.Add(1)
	if r.URL.Path != "/tools/project-by" {
		http.NotFound(w, r)
		return
	}
	q := r.URL.Query()
	key := q.Get("repo") + "|" + q.Get("name_type") + "|" + q.Get("name")
	body, ok := f.projects[key]
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, body)
}

const fdProject = `[
	{"repo": "debian_13", "binname": "fd-find", "srcname": "rust-fd-find", "status": "newest"},
	{"repo": "ubuntu_24_04", "binname": "fd-find", "status": "outdated"},
	{"repo": "fedora_rawhide", "binname": "fd-find", "status": "newest"},
	{"repo": "arch", "binname": "fd", "status": "newest"},
	{"repo": "homebrew", "srcname": "fd", "status": "newest"},
	{"repo": "nix_unstable", "srcname": "fd", "status": "newest"},
	{"repo": "freebsd", "srcname": "sysutils/fd", "status": "newest"}
]`

const firefoxProject = `[
	{"repo": "debian_13", "binname": "firefox-esr", "status": "outdated"},
	{"repo": "homebrew_casks", "srcname": "firefox", "status": "newest"}
]`

func testConverter(t *testing.T, srv *httptest.Server, extra map[string]string) *converter {
	t.Helper()
	settings := map[string]string{
		"base_url":     srv.URL,
		"min_interval": "0s",
		"cache":        filepath.Join(t.TempDir(), "cache.json"),
	}
	for k, v := range extra {
		settings[k] = v
	}
	c, err := newConverter(settings)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestLookupTranslates(t *testing.T) {
	fake := &fakeRepology{projects: map[string]string{
		"debian_13|binname|fd-find":      fdProject,
		"homebrew|srcname|fd":            fdProject,
		"fedora_rawhide|binname|fd-find": fdProject,
		"arch|binname|fd":                fdProject,
	}}
	srv := httptest.NewServer(fake)
	defer srv.Close()
	c := testConverter(t, srv, nil)

	cases := []struct{ from, to, pkg, want string }{
		{"apt", "brew", "fd-find", "fd"},
		{"apt", "nix", "fd-find", "fd"},
		{"brew", "apt", "fd", "fd-find"},
		{"dnf", "pacman", "fd-find", "fd"},
		{"pacman", "dnf", "fd", "fd-find"},
		{"apt", "pacman", "fd-find", "fd"},
	}
	for _, tc := range cases {
		got, found, err := c.lookup(tc.from, tc.to, tc.pkg)
		if err != nil {
			t.Fatal(err)
		}
		if !found || got != tc.want {
			t.Errorf("lookup(%s %q -> %s) = %q, %v; want %q", tc.from, tc.pkg, tc.to, got, found, tc.want)
		}
	}
}

func TestLookupCaskTarget(t *testing.T) {
	fake := &fakeRepology{projects: map[string]string{
		"debian_13|binname|firefox-esr": firefoxProject,
	}}
	srv := httptest.NewServer(fake)
	defer srv.Close()
	c := testConverter(t, srv, nil)

	got, found, err := c.lookup("apt", "brew-cask", "firefox-esr")
	if err != nil {
		t.Fatal(err)
	}
	if !found || got != "firefox" {
		t.Errorf("lookup(apt firefox-esr -> brew-cask) = %q, %v; want firefox", got, found)
	}
}

func TestLookupNotFoundAndUnsupported(t *testing.T) {
	fake := &fakeRepology{projects: map[string]string{
		// Project exists but has no homebrew entry.
		"debian_13|binname|debtool": `[{"repo": "debian_13", "binname": "debtool", "status": "newest"}]`,
	}}
	srv := httptest.NewServer(fake)
	defer srv.Close()
	c := testConverter(t, srv, nil)

	if _, found, err := c.lookup("apt", "brew", "debtool"); err != nil || found {
		t.Errorf("lookup of package missing in target = found=%v err=%v, want miss", found, err)
	}
	if _, found, err := c.lookup("apt", "brew", "no-such-pkg"); err != nil || found {
		t.Errorf("lookup of unknown package = found=%v err=%v, want miss", found, err)
	}

	before := fake.requests.Load()
	if _, found, _ := c.lookup("snap", "brew", "anything"); found {
		t.Error("unsupported source manager should miss")
	}
	if _, found, _ := c.lookup("apt", "snap", "anything"); found {
		t.Error("unsupported target manager should miss")
	}
	if got := fake.requests.Load(); got != before {
		t.Errorf("unsupported managers must not hit the API (%d extra requests)", got-before)
	}
}

func TestLookupUsesCache(t *testing.T) {
	fake := &fakeRepology{projects: map[string]string{
		"debian_13|binname|fd-find": fdProject,
	}}
	srv := httptest.NewServer(fake)
	defer srv.Close()
	cachePath := filepath.Join(t.TempDir(), "cache.json")

	c := testConverter(t, srv, map[string]string{"cache": cachePath})
	if _, _, err := c.lookup("apt", "brew", "fd-find"); err != nil {
		t.Fatal(err)
	}
	after := fake.requests.Load()

	// Same lookup again: in-memory cache, no new requests.
	if _, found, _ := c.lookup("apt", "brew", "fd-find"); !found {
		t.Error("cached lookup should still be found")
	}
	// Negative results are cached too.
	if _, _, err := c.lookup("apt", "brew", "missing-pkg"); err != nil {
		t.Fatal(err)
	}
	misses := fake.requests.Load()
	if _, found, _ := c.lookup("apt", "brew", "missing-pkg"); found {
		t.Error("negative cache returned found")
	}
	if got := fake.requests.Load(); got != misses {
		t.Errorf("negative result not cached (%d extra requests)", got-misses)
	}
	if after == 0 {
		t.Fatal("expected at least one API request")
	}

	// A fresh converter reads the persisted cache: no requests at all.
	c.cache.save()
	c2 := testConverter(t, srv, map[string]string{"cache": cachePath})
	before := fake.requests.Load()
	got, found, err := c2.lookup("apt", "brew", "fd-find")
	if err != nil || !found || got != "fd" {
		t.Errorf("lookup from persisted cache = %q, %v, %v", got, found, err)
	}
	if reqs := fake.requests.Load(); reqs != before {
		t.Errorf("persisted cache miss: %d extra requests", reqs-before)
	}
}

func TestLookupServerErrorIsNotCached(t *testing.T) {
	var fail atomic.Bool
	fake := &fakeRepology{projects: map[string]string{
		"debian_13|binname|fd-find": fdProject,
	}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail.Load() {
			http.Error(w, "slow down", http.StatusTooManyRequests)
			return
		}
		fake.ServeHTTP(w, r)
	}))
	defer srv.Close()
	c := testConverter(t, srv, nil)

	fail.Store(true)
	if _, found, err := c.lookup("apt", "brew", "fd-find"); err == nil || found {
		t.Errorf("lookup during 429 = found=%v err=%v, want error", found, err)
	}

	// Once the server recovers the same lookup succeeds — the failure was
	// not cached as a negative result.
	fail.Store(false)
	got, found, err := c.lookup("apt", "brew", "fd-find")
	if err != nil || !found || got != "fd" {
		t.Errorf("lookup after recovery = %q, %v, %v; want fd", got, found, err)
	}
}

func TestNewConverterValidatesSettings(t *testing.T) {
	for _, settings := range []map[string]string{
		{"min_interval": "fast"},
		{"cache_ttl": "soon"},
	} {
		if _, err := newConverter(settings); err == nil {
			t.Errorf("newConverter(%v) should fail", settings)
		}
	}
}
