// Copyright (c) 2026 IBM (https://ibm.com)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/thrnjica/agwctl/internal/models"
)

// unmarshal is a helper to unmarshal JSON with better error messages.
func unmarshal(data []byte, v any) error {
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
			Transport: newTransport(username, password, version, rps),
		},
		log: log,
	}
}

// call performs an HTTP request.
func (c *Client) call(ctx context.Context, method, path string, body []byte) ([]byte, int, error) {
	// Build URL
	url := c.baseURL + path

	// Create request
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, r)
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Log request
	c.log.Debug("HTTP request",
		slog.String("method", method),
		slog.String("url", url),
		slog.Int("body_size", len(body)))

	// Execute request
	start := time.Now()
	res, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("execute request: %w", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	status := res.StatusCode

	// Read response
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, status, fmt.Errorf("read response: %w", err)
	}

	dur := time.Since(start)

	// Log response
	c.log.Debug("HTTP response",
		slog.String("method", method),
		slog.String("url", url),
		slog.Int("status", status),
		slog.Int("body_size", len(data)),
		slog.Int64("duration_ms", dur.Milliseconds()))

	// Handle non-2xx status codes
	if status < 200 || status >= 300 {
		return data, status, fmt.Errorf("HTTP %d: %s", status, string(data))
	}

	return data, status, nil
}

// ListServices fetches a page of APIs from the gateway.
func (c *Client) ListServices(ctx context.Context, from, size int) (*models.ServiceListResponse, error) {
	path := fmt.Sprintf("/apis?from=%d&size=%d", from, size)

	res, _, err := c.call(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("list APIs: %w", err)
	}

	var model models.ServiceListResponse
	if err := unmarshal(res, &model); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &model, nil
}

// GetService fetches the full API document as raw JSON.
func (c *Client) GetService(ctx context.Context, id string) ([]byte, error) {
	path := fmt.Sprintf("/apis/%s", url.PathEscape(id))

	res, _, err := c.call(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("get service: %w", err)
	}

	return res, nil
}

// UpdateService updates an API with the provided JSON document.
func (c *Client) UpdateService(ctx context.Context, id string, payload []byte) error {
	path := fmt.Sprintf("/apis/%s", url.PathEscape(id))

	_, _, err := c.call(ctx, http.MethodPut, path, payload)
	if err != nil {
		return fmt.Errorf("update service: %w", err)
	}

	return nil
}

// ListTeams fetches all teams from the gateway.
func (c *Client) ListTeams(ctx context.Context) (*models.TeamListResponse, error) {
	path := "/accessProfiles"

	res, _, err := c.call(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("list teams: %w", err)
	}

	var model models.TeamListResponse
	if err := unmarshal(res, &model); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &model, nil
}

// Made with Bob
