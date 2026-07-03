// Package vector provides a minimal Qdrant HTTP client used by the memory
// module to store conversation embeddings. It intentionally uses the REST
// API to avoid a gRPC dependency.
package vector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	baseURL string
	httpc   *http.Client
}

// New returns a Qdrant client, or nil when no URL is configured.
func New(baseURL string) *Client {
	if baseURL == "" {
		return nil
	}
	return &Client{baseURL: baseURL, httpc: &http.Client{Timeout: 10 * time.Second}}
}

func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/readyz", nil)
	if err != nil {
		return err
	}
	resp, err := c.httpc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("qdrant readyz: status %d", resp.StatusCode)
	}
	return nil
}

// EnsureCollection creates the collection if it does not exist.
func (c *Client) EnsureCollection(ctx context.Context, name string, vectorSize int) error {
	body := map[string]any{
		"vectors": map[string]any{"size": vectorSize, "distance": "Cosine"},
	}
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, fmt.Sprintf("%s/collections/%s", c.baseURL, name), bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// 409 means it already exists — fine.
	if resp.StatusCode >= 300 && resp.StatusCode != http.StatusConflict {
		return fmt.Errorf("qdrant create collection: status %d", resp.StatusCode)
	}
	return nil
}

// Upsert stores one point with payload.
func (c *Client) Upsert(ctx context.Context, collection, id string, vector []float32, payload map[string]any) error {
	body := map[string]any{
		"points": []map[string]any{{"id": id, "vector": vector, "payload": payload}},
	}
	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, fmt.Sprintf("%s/collections/%s/points", c.baseURL, collection), bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("qdrant upsert: status %d", resp.StatusCode)
	}
	return nil
}
