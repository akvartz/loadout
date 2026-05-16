// Package rpc implements the loadout external plugin JSON-RPC protocol.
// Wire types in this file intentionally import nothing from loadout's internal
// packages so that external plugin authors can copy/reimplement them freely.
package rpc

import "encoding/json"

// Request is a JSON-RPC 2.0 request envelope.
type Request struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int64       `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// Response is a JSON-RPC 2.0 response envelope.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// HandshakeResult is returned by the "handshake" method.
type HandshakeResult struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Capabilities []string `json:"capabilities"`
}

// WirePackage is the JSON form of a detected package.
type WirePackage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// WireSourceState is the JSON form of state.SourceState.
type WireSourceState struct {
	Packages []string `json:"packages"`
}

// WireMeta is the JSON form of state.Meta.
type WireMeta struct {
	SchemaVersion int    `json:"schema_version"`
	CapturedAt    string `json:"captured_at"`
	Hostname      string `json:"hostname"`
	OS            string `json:"os"`
}

// WireState is the JSON form of state.State.
type WireState struct {
	Meta    WireMeta                   `json:"meta"`
	Sources map[string]WireSourceState `json:"sources"`
}

// WireConvertResult is one entry in the converter response.
type WireConvertResult struct {
	Original  string `json:"original"`
	Converted string `json:"converted"`
	Found     bool   `json:"found"`
}

// Per-method param and result types.

type HookPostSnapshotParams struct {
	State     WireState `json:"state"`
	StateFile string    `json:"state_file"`
}

type HookPreRestoreParams struct {
	State  WireState `json:"state"`
	Target string    `json:"target"`
}

type HookPostRestoreParams struct {
	State  WireState `json:"state"`
	Target string    `json:"target"`
}

type DetectorNameResult struct {
	Name string `json:"name"`
}

type DetectorAvailableResult struct {
	Available bool `json:"available"`
}

type DetectorDetectResult struct {
	Packages []WirePackage `json:"packages"`
}

type GeneratorNameResult struct {
	Name string `json:"name"`
}

type GeneratorGenerateParams struct {
	State WireState `json:"state"`
}

type GeneratorGenerateResult struct {
	Script string `json:"script"`
}

type ConverterNameResult struct {
	Name string `json:"name"`
}

type ConverterConvertParams struct {
	From     string   `json:"from"`
	To       string   `json:"to"`
	Packages []string `json:"packages"`
}

type ConverterConvertResult struct {
	Results []WireConvertResult `json:"results"`
}
