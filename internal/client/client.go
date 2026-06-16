// Package client provides HTTP client functionality for the API Gateway REST API.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/thrnjica/agwctl/internal/models"
)

// unmarshalJSON is a helper to unmarshal JSON with better error messages.
func unmarshalJSON(data []byte, v any) error {
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("unmarshal JSON: %w", err)
	}
	return nil
}

// Client provides HTTP communication with the API Gateway REST API.
type Client struct {
	baseURL string
	http    *http.Client
	log     *slog.Logger
}

// New creates a new API Gateway client with optimized transport.
func New(baseURL, username, password, version string, rps int, log *slog.Logger) *Client {
	return &Client{
		baseURL: baseURL,
		http: &http.Client{
			Timeout:   30 * time.Second,
			Transport: buildTransport(username, password, version, rps),
		},
		log: log,
	}
}

// doRequest performs an HTTP request.
func (c *Client) doRequest(ctx context.Context, method, path string, body []byte) ([]byte, int, error) {
	// Build URL
	fullURL := c.baseURL + path

	// Create request
	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Log request
	c.log.Debug("HTTP request",
		slog.String("method", method),
		slog.String("url", fullURL),
		slog.Int("body_size", len(body)))

	// Execute request
	start := time.Now()
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response: %w", err)
	}

	dur := time.Since(start)

	// Log response
	c.log.Debug("HTTP response",
		slog.String("method", method),
		slog.String("url", fullURL),
		slog.Int("status", resp.StatusCode),
		slog.Int("body_size", len(respBody)),
		slog.Int64("duration_ms", dur.Milliseconds()))

	// Handle non-2xx status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return respBody, resp.StatusCode, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, resp.StatusCode, nil
}

// ListAPIs fetches a page of APIs from the gateway.
func (c *Client) ListAPIs(ctx context.Context, from, size int) (*models.APIListResponse, error) {
	path := fmt.Sprintf("/apis?from=%d&size=%d", from, size)

	respBody, _, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("list APIs: %w", err)
	}

	var resp models.APIListResponse
	if err := unmarshalJSON(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &resp, nil
}

// GetAPI fetches the full API document as raw JSON.
func (c *Client) GetAPI(ctx context.Context, apiID string) ([]byte, error) {
	path := fmt.Sprintf("/apis/%s", url.PathEscape(apiID))

	respBody, _, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("get API: %w", err)
	}

	return respBody, nil
}

// UpdateAPI updates an API with the provided JSON document.
func (c *Client) UpdateAPI(ctx context.Context, apiID string, apiJSON []byte) error {
	path := fmt.Sprintf("/apis/%s", url.PathEscape(apiID))

	_, _, err := c.doRequest(ctx, http.MethodPut, path, apiJSON)
	if err != nil {
		return fmt.Errorf("update API: %w", err)
	}

	return nil
}

// ListAccessProfiles fetches all access profiles (teams) from the gateway.
func (c *Client) ListAccessProfiles(ctx context.Context) (*models.AccessProfileListResponse, error) {
	path := "/accessProfiles"

	respBody, _, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("list access profiles: %w", err)
	}

	var resp models.AccessProfileListResponse
	if err := unmarshalJSON(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &resp, nil
}

// Made with Bob
