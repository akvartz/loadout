package rpc

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
)

// Client manages a single external plugin subprocess.
// All calls are serialized with a mutex — plugins handle one request at a time.
type Client struct {
	cmd    *exec.Cmd
	enc    *json.Encoder
	dec    *json.Decoder
	mu     sync.Mutex
	nextID int64
}

// Start spawns the plugin binary, performs the handshake, and returns the
// client and the plugin's self-reported capabilities.
func Start(binary string) (*Client, HandshakeResult, error) {
	cmd := exec.Command(binary)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, HandshakeResult{}, fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, HandshakeResult{}, fmt.Errorf("stdout pipe: %w", err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, HandshakeResult{}, fmt.Errorf("start %s: %w", binary, err)
	}

	c := &Client{
		cmd: cmd,
		enc: json.NewEncoder(stdin),
		dec: json.NewDecoder(stdout),
	}

	var hs HandshakeResult
	if err := c.Call("handshake", nil, &hs); err != nil {
		cmd.Process.Kill() //nolint:errcheck
		return nil, HandshakeResult{}, fmt.Errorf("handshake with %s: %w", binary, err)
	}
	return c, hs, nil
}

// Call sends a JSON-RPC request and decodes the result into dest.
// dest may be nil if no result is expected.
func (c *Client) Call(method string, params interface{}, dest interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	id := atomic.AddInt64(&c.nextID, 1)
	req := Request{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}
	if err := c.enc.Encode(req); err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	var resp Response
	if err := c.dec.Decode(&resp); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	if resp.ID != id {
		return fmt.Errorf("response id mismatch: got %d, want %d", resp.ID, id)
	}
	if resp.Error != "" {
		return fmt.Errorf("plugin error: %s", resp.Error)
	}
	if dest != nil && len(resp.Result) > 0 {
		if err := json.Unmarshal(resp.Result, dest); err != nil {
			return fmt.Errorf("decode result: %w", err)
		}
	}
	return nil
}

// Close sends the shutdown method and waits for the process to exit.
func (c *Client) Close() error {
	c.mu.Lock()
	_ = c.enc.Encode(Request{JSONRPC: "2.0", ID: -1, Method: "shutdown"})
	c.mu.Unlock()
	return c.cmd.Wait()
}
