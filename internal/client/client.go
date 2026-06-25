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
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
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
	url  string
	http *http.Client
	log  *slog.Logger
}

// New creates a new API Gateway client with optimized transport.
func New(
	url,
	username,
	password,
	version string,
	rps int,
	log *slog.Logger,
) *Client {
	return &Client{
		url: url,
		http: &http.Client{
			Timeout:   30 * time.Second,
			Transport: newTransport(username, password, version, rps),
		},
		log: log,
	}
}

// call performs an HTTP request.
func (c *Client) call(
	ctx context.Context,
	method, path string,
	body []byte,
) ([]byte, int, error) {
	// Build URL properly to handle trailing slashes
	base, err := url.Parse(c.url)
	if err != nil {
		return nil, 0, fmt.Errorf("parse base URL: %w", err)
	}
	// Join paths properly, handling trailing/leading slashes
	base.Path = strings.TrimSuffix(base.Path, "/") + "/" + strings.TrimPrefix(path, "/")
	fullURL := base.String()

	// Create request
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, r)
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
		slog.String("url", fullURL),
		slog.Int("status", status),
		slog.Int("body_size", len(data)),
		slog.Int64("duration_ms", dur.Milliseconds()))

	// Handle non-2xx status codes
	if status < 200 || status >= 300 {
		return data, status, fmt.Errorf("HTTP %d: %s", status, string(data))
	}

	return data, status, nil
}

// ListAPIs fetches a page of APIs from the gateway.
func (c *Client) ListAPIs(
	ctx context.Context,
	from, size int,
) (*models.APIListResponse, error) {
	path := fmt.Sprintf("/apis?from=%d&size=%d", from, size)

	res, _, err := c.call(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("list APIs: %w", err)
	}

	var model models.APIListResponse
	if err := unmarshal(res, &model); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &model, nil
}

// GetAPI fetches the full API document as raw JSON.
func (c *Client) GetAPI(ctx context.Context, id string) ([]byte, error) {
	path := fmt.Sprintf("/apis/%s", url.PathEscape(id))

	res, _, err := c.call(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("get API: %w", err)
	}

	return res, nil
}

// UpdateAPI updates an API with the provided JSON document.
func (c *Client) UpdateAPI(
	ctx context.Context,
	id string,
	payload []byte,
) error {
	path := fmt.Sprintf("/apis/%s", url.PathEscape(id))

	_, _, err := c.call(ctx, http.MethodPut, path, payload)
	if err != nil {
		return fmt.Errorf("update api: %w", err)
	}

	return nil
}

// ListTeams fetches all teams from the gateway.
func (c *Client) ListTeams(
	ctx context.Context,
) (*models.TeamListResponse, error) {
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

// ListAliases fetches all aliases from the gateway.
// Note: API does NOT support pagination - returns all aliases in one call.
func (c *Client) ListAliases(
	ctx context.Context,
) ([]models.EndpointAlias, error) {
	path := "/alias"

	res, _, err := c.call(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("list aliases: %w", err)
	}

	var response models.AliasResponseModel
	if err := unmarshal(res, &response); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return response.Alias, nil
}

// GetAlias fetches a single alias by ID.
func (c *Client) GetAlias(
	ctx context.Context,
	aliasID string,
) (*models.EndpointAlias, error) {
	path := fmt.Sprintf("/alias/%s", url.PathEscape(aliasID))

	res, _, err := c.call(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("get alias: %w", err)
	}

	var alias models.EndpointAlias
	if err := unmarshal(res, &alias); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &alias, nil
}

// Made with Bob
