package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/akvartz/loadout/internal/plugin/rpc"
)

// TestServeProtocol drives the plugin loop the way loadout's rpc.Client does:
// handshake (with settings), converter.convert, shutdown.
func TestServeProtocol(t *testing.T) {
	fake := &fakeRepology{projects: map[string]string{
		"debian_13|binname|fd-find": fdProject,
	}}
	srv := httptest.NewServer(fake)
	defer srv.Close()

	var in bytes.Buffer
	enc := json.NewEncoder(&in)
	settings := map[string]string{
		"base_url":     srv.URL,
		"min_interval": "0s",
		"cache":        filepath.Join(t.TempDir(), "cache.json"),
	}
	for id, req := range []rpc.Request{
		{JSONRPC: "2.0", ID: 1, Method: "handshake", Params: rpc.HandshakeParams{Settings: settings}},
		{JSONRPC: "2.0", ID: 2, Method: "converter.convert", Params: rpc.ConverterConvertParams{
			From: "apt", To: "brew", Packages: []string{"fd-find", "unknown-pkg"},
		}},
		{JSONRPC: "2.0", ID: 3, Method: "bogus.method"},
		{JSONRPC: "2.0", ID: -1, Method: "shutdown"},
	} {
		if err := enc.Encode(req); err != nil {
			t.Fatalf("request %d: %v", id, err)
		}
	}

	var out bytes.Buffer
	if err := serve(&in, &out); err != nil {
		t.Fatal(err)
	}

	dec := json.NewDecoder(&out)
	var hs rpc.Response
	if err := dec.Decode(&hs); err != nil {
		t.Fatal(err)
	}
	var hsResult rpc.HandshakeResult
	if err := json.Unmarshal(hs.Result, &hsResult); err != nil {
		t.Fatal(err)
	}
	if hsResult.Name != "repology" || len(hsResult.Capabilities) != 1 || hsResult.Capabilities[0] != "converter" {
		t.Errorf("handshake = %+v", hsResult)
	}

	var conv rpc.Response
	if err := dec.Decode(&conv); err != nil {
		t.Fatal(err)
	}
	var convResult rpc.ConverterConvertResult
	if err := json.Unmarshal(conv.Result, &convResult); err != nil {
		t.Fatal(err)
	}
	if len(convResult.Results) != 2 {
		t.Fatalf("got %d results, want 2: %+v", len(convResult.Results), convResult.Results)
	}
	if r := convResult.Results[0]; !r.Found || r.Converted != "fd" {
		t.Errorf("results[0] = %+v, want fd found", r)
	}
	if r := convResult.Results[1]; r.Found {
		t.Errorf("results[1] = %+v, want not found", r)
	}

	var bogus rpc.Response
	if err := dec.Decode(&bogus); err != nil {
		t.Fatal(err)
	}
	if bogus.Error == "" {
		t.Error("unknown method should return an error response")
	}
}

func TestServeConvertBeforeHandshake(t *testing.T) {
	var in bytes.Buffer
	enc := json.NewEncoder(&in)
	_ = enc.Encode(rpc.Request{JSONRPC: "2.0", ID: 1, Method: "converter.convert",
		Params: rpc.ConverterConvertParams{From: "apt", To: "brew", Packages: []string{"x"}}})

	var out bytes.Buffer
	if err := serve(&in, &out); err != nil {
		t.Fatal(err)
	}
	var resp rpc.Response
	if err := json.NewDecoder(&out).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Error == "" {
		t.Error("convert before handshake should return an error response")
	}
}
