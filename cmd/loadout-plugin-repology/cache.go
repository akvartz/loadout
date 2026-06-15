package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type cacheEntry struct {
	Name    string    `json:"name"`
	Found   bool      `json:"found"`
	Fetched time.Time `json:"fetched"`
}

// cache stores lookup results (positive and negative) keyed by
// "from|package|to", persisted as JSON. A zero path disables persistence
// but keeps the in-memory layer.
type cache struct {
	path    string
	ttl     time.Duration
	entries map[string]cacheEntry
	dirty   bool
}

// loadCache reads the cache file at path; a missing or unreadable file
// yields an empty cache.
func loadCache(path string, ttl time.Duration) *cache {
	c := &cache{path: path, ttl: ttl, entries: map[string]cacheEntry{}}
	if path == "" {
		return c
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return c
	}
	_ = json.Unmarshal(data, &c.entries)
	return c
}

func (c *cache) get(key string) (name string, found, ok bool) {
	e, ok := c.entries[key]
	if !ok || time.Since(e.Fetched) > c.ttl {
		return "", false, false
	}
	return e.Name, e.Found, true
}

func (c *cache) put(key, name string, found bool) {
	c.entries[key] = cacheEntry{Name: name, Found: found, Fetched: time.Now()}
	c.dirty = true
}

// save persists the cache if anything changed. Errors are ignored — the
// cache is an optimization, not state.
func (c *cache) save() {
	if c.path == "" || !c.dirty {
		return
	}
	data, err := json.Marshal(c.entries)
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(c.path), 0o700); err != nil {
		return
	}
	_ = os.WriteFile(c.path, data, 0o600)
	c.dirty = false
}
