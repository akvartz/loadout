// Command loadout-plugin-repology is an external loadout converter plugin
// that translates package names between managers by querying the Repology
// API. Install it on PATH and enable it with:
//
//	[plugins]
//	enabled = ["repology"]
//
// It speaks the loadout plugin JSON-RPC protocol on stdin/stdout.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/akvartz/loadout/internal/plugin/rpc"
)

const version = "0.1.0"

// request mirrors rpc.Request with raw params so each method can decode its
// own parameter type.
type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

func main() {
	if err := serve(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "loadout-plugin-repology:", err)
		os.Exit(1)
	}
}

func serve(in io.Reader, out io.Writer) error {
	dec := json.NewDecoder(in)
	enc := json.NewEncoder(out)
	var conv *converter

	for {
		var req request
		if err := dec.Decode(&req); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		switch req.Method {
		case "handshake":
			var params rpc.HandshakeParams
			if len(req.Params) > 0 {
				if err := json.Unmarshal(req.Params, &params); err != nil {
					replyError(enc, req.ID, "bad handshake params: "+err.Error())
					continue
				}
			}
			c, err := newConverter(params.Settings)
			if err != nil {
				replyError(enc, req.ID, err.Error())
				continue
			}
			conv = c
			reply(enc, req.ID, rpc.HandshakeResult{
				Name:         "repology",
				Version:      version,
				Capabilities: []string{"converter"},
			})

		case "shutdown":
			// loadout does not wait for a response to shutdown.
			if conv != nil {
				conv.cache.save()
			}
			return nil

		case "converter.name":
			reply(enc, req.ID, rpc.ConverterNameResult{Name: "repology"})

		case "converter.convert":
			if conv == nil {
				replyError(enc, req.ID, "handshake required before converter.convert")
				continue
			}
			var params rpc.ConverterConvertParams
			if err := json.Unmarshal(req.Params, &params); err != nil {
				replyError(enc, req.ID, "bad convert params: "+err.Error())
				continue
			}
			results := conv.convert(params)
			conv.cache.save()
			reply(enc, req.ID, rpc.ConverterConvertResult{Results: results})

		default:
			replyError(enc, req.ID, "unknown method "+req.Method)
		}
	}
}

func reply(enc *json.Encoder, id int64, result interface{}) {
	raw, err := json.Marshal(result)
	if err != nil {
		replyError(enc, id, "encode result: "+err.Error())
		return
	}
	_ = enc.Encode(rpc.Response{JSONRPC: "2.0", ID: id, Result: raw})
}

func replyError(enc *json.Encoder, id int64, msg string) {
	_ = enc.Encode(rpc.Response{JSONRPC: "2.0", ID: id, Error: msg})
}
