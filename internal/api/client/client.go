// Package client is the shared Go client for the daemon's local API. Both the
// CLI and the GUI use it, so they operate on identical state. It dials the IPC
// transport (Unix socket / named pipe) rather than a network port.
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/shruggietech/go-scheduler/internal/api/server"
	"github.com/shruggietech/go-scheduler/internal/ipc"
)

// Client talks to the daemon over the IPC endpoint.
type Client struct {
	http     *http.Client
	endpoint string
}

// New returns a client bound to the given IPC endpoint (socket path / pipe name).
func New(endpoint string) *Client {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return ipc.DialContext(ctx, endpoint)
		},
	}
	return &Client{http: &http.Client{Transport: transport}, endpoint: endpoint}
}

// baseURL uses a fixed dummy host; the transport ignores it and dials the IPC
// endpoint instead.
const baseURL = "http://ipc"

// Health calls GET /v1/health.
func (c *Client) Health(ctx context.Context) (server.HealthResponse, error) {
	var out server.HealthResponse
	if err := c.get(ctx, "/v1/health", &out); err != nil {
		return server.HealthResponse{}, err
	}
	return out, nil
}

// get performs a GET and decodes a JSON body, surfacing the API error envelope.
func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+path, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("api: %s: %w (is the daemon running?)", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		var apiErr server.APIError
		if decErr := json.NewDecoder(resp.Body).Decode(&apiErr); decErr == nil && apiErr.Error.Message != "" {
			return fmt.Errorf("api: %s: %s", apiErr.Error.Code, apiErr.Error.Message)
		}
		return fmt.Errorf("api: %s: unexpected status %d", path, resp.StatusCode)
	}
	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("api: decode %s: %w", path, err)
		}
	}
	return nil
}
