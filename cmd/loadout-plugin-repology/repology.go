package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/akvartz/loadout/internal/plugin/rpc"
)

const userAgent = "loadout-plugin-repology/" + version + " (+https://github.com/akvartz/loadout)"

// fromRepos maps a loadout manager name to the Repology repositories used to
// resolve a package name to a Repology project. Tried in order.
var fromRepos = map[string][]string{
	"apt":       {"debian_13", "ubuntu_24_04"},
	"brew":      {"homebrew"},
	"brew-cask": {"homebrew_casks"},
	"nix":       {"nix_unstable"},
}

// repoMatches reports whether a Repology repository belongs to a manager.
func repoMatches(manager, repo string) bool {
	switch manager {
	case "apt":
		return strings.HasPrefix(repo, "debian_") || strings.HasPrefix(repo, "ubuntu_")
	case "brew":
		return repo == "homebrew"
	case "brew-cask":
		return repo == "homebrew_casks"
	case "nix":
		return repo == "nix_unstable"
	}
	return false
}

// projectEntry is one package record in a Repology project.
type projectEntry struct {
	Repo        string `json:"repo"`
	SrcName     string `json:"srcname"`
	BinName     string `json:"binname"`
	VisibleName string `json:"visiblename"`
	Status      string `json:"status"`
}

func (e projectEntry) name() string {
	switch {
	case e.BinName != "":
		return e.BinName
	case e.SrcName != "":
		return e.SrcName
	}
	return e.VisibleName
}

// converter resolves package names through the Repology API
// (https://repology.org/api). Calls are throttled and results — including
// negative ones — are cached on disk to respect Repology's rate limits.
type converter struct {
	baseURL     string
	client      *http.Client
	minInterval time.Duration
	lastReq     time.Time
	cache       *cache
}

func newConverter(settings map[string]string) (*converter, error) {
	c := &converter{
		baseURL:     "https://repology.org",
		client:      &http.Client{Timeout: 15 * time.Second},
		minInterval: time.Second,
	}
	if v := settings["base_url"]; v != "" {
		c.baseURL = strings.TrimRight(v, "/")
	}
	if v := settings["min_interval"]; v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return nil, fmt.Errorf("invalid min_interval %q: %w", v, err)
		}
		c.minInterval = d
	}

	ttl := 7 * 24 * time.Hour
	if v := settings["cache_ttl"]; v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return nil, fmt.Errorf("invalid cache_ttl %q: %w", v, err)
		}
		ttl = d
	}
	path := settings["cache"]
	switch path {
	case "off":
		path = ""
	case "":
		if dir, err := os.UserCacheDir(); err == nil {
			path = filepath.Join(dir, "loadout", "repology.json")
		}
	default:
		path = expandTilde(path)
	}
	c.cache = loadCache(path, ttl)
	return c, nil
}

// convert never fails the batch: per-package errors degrade to "not found"
// with a note on stderr, and the result count always matches the input.
func (c *converter) convert(req rpc.ConverterConvertParams) []rpc.WireConvertResult {
	results := make([]rpc.WireConvertResult, len(req.Packages))
	for i, pkg := range req.Packages {
		name, found, err := c.lookup(req.From, req.To, pkg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "repology: %s %q: %v\n", req.From, pkg, err)
		}
		results[i] = rpc.WireConvertResult{Original: pkg, Converted: name, Found: found}
	}
	return results
}

func (c *converter) lookup(from, to, pkg string) (string, bool, error) {
	if _, ok := fromRepos[from]; !ok {
		return "", false, nil
	}
	if _, ok := fromRepos[to]; !ok {
		return "", false, nil
	}

	key := from + "|" + pkg + "|" + to
	if name, found, ok := c.cache.get(key); ok {
		return name, found, nil
	}

	for _, repo := range fromRepos[from] {
		for _, nameType := range []string{"binname", "srcname"} {
			entries, ok, err := c.projectBy(repo, nameType, pkg)
			if err != nil {
				return "", false, err // transient: not cached
			}
			if !ok {
				continue
			}
			name, found := pickName(entries, to)
			c.cache.put(key, name, found)
			return name, found, nil
		}
	}
	c.cache.put(key, "", false)
	return "", false, nil
}

// pickName chooses the target manager's name from a project's entries,
// preferring current ("newest") packages.
func pickName(entries []projectEntry, to string) (string, bool) {
	var fallback string
	for _, e := range entries {
		if !repoMatches(to, e.Repo) || e.name() == "" {
			continue
		}
		if e.Status == "newest" {
			return e.name(), true
		}
		if fallback == "" {
			fallback = e.name()
		}
	}
	return fallback, fallback != ""
}

// projectBy resolves a package name in one repository to its Repology project
// entries. ok=false means the name is unknown there (a definitive miss);
// err covers transient failures (network, rate limiting, server errors).
func (c *converter) projectBy(repo, nameType, name string) ([]projectEntry, bool, error) {
	c.throttle()

	u := fmt.Sprintf("%s/tools/project-by?repo=%s&name_type=%s&target_page=api_v1_project&name=%s",
		c.baseURL, url.QueryEscape(repo), url.QueryEscape(nameType), url.QueryEscape(name))
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusNotFound:
		return nil, false, nil
	case resp.StatusCode != http.StatusOK:
		return nil, false, fmt.Errorf("repology returned %s", resp.Status)
	}

	var entries []projectEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, false, err
	}
	return entries, true, nil
}

// throttle enforces the minimum interval between requests. Calls arrive
// serialized over the plugin protocol, so no locking is needed.
func (c *converter) throttle() {
	if c.minInterval <= 0 {
		return
	}
	if wait := c.minInterval - time.Since(c.lastReq); wait > 0 {
		time.Sleep(wait)
	}
	c.lastReq = time.Now()
}

func expandTilde(s string) string {
	if !strings.HasPrefix(s, "~") {
		return s
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return s
	}
	return filepath.Join(home, s[1:])
}
