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

package alias

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/thrnjica/agwctl/internal/client"
	"github.com/thrnjica/agwctl/internal/models"
)

// Manager orchestrates alias listing and DNS resolution.
type Manager struct {
	client   *client.Client
	resolver *Resolver
	log      *slog.Logger
}

// NewManager creates a new alias manager with the specified client, DNS timeout, and logger.
func NewManager(
	client *client.Client,
	timeout time.Duration,
	log *slog.Logger,
) *Manager {
	return &Manager{
		client:   client,
		resolver: NewResolver(timeout),
		log:      log,
	}
}

// ListWithIPs fetches all endpoint aliases and resolves their IP addresses.
// Only aliases with type "endpoint" are processed.
func (m *Manager) ListWithIPs(ctx context.Context) ([]models.AliasInfo, error) {
	// Fetch all aliases (no pagination - API returns all in one call)
	allAliases, err := m.client.ListAliases(ctx)
	if err != nil {
		return nil, fmt.Errorf("list aliases: %w", err)
	}

	m.log.Info("Fetched aliases from gateway",
		slog.Int("total", len(allAliases)))

	// Filter endpoint aliases only (type is "endpoint" - lowercase!)
	var endpointAliases []models.EndpointAlias
	for _, alias := range allAliases {
		if strings.ToLower(alias.Type) == "endpoint" {
			endpointAliases = append(endpointAliases, alias)
		}
	}

	m.log.Info("Filtered endpoint aliases",
		slog.Int("total", len(allAliases)),
		slog.Int("endpoints", len(endpointAliases)))

	// Resolve IPs for each alias
	var results []models.AliasInfo
	for _, alias := range endpointAliases {
		info := models.AliasInfo{
			Name:        alias.Name,
			EndpointURL: alias.EndpointURI, // Note: Field is EndpointURI (capital P)
		}

		// Extract hostname from EndpointURI
		hostname, err := extractHostname(alias.EndpointURI)
		if err != nil {
			info.Error = err.Error()
			results = append(results, info)
			m.log.Warn("Failed to extract hostname",
				slog.String("alias", alias.Name),
				slog.String("url", alias.EndpointURI),
				slog.Any("error", err))
			continue
		}
		info.Hostname = hostname

		// Resolve IPs
		ips, err := m.resolver.ResolveHostname(hostname)
		if err != nil {
			info.Error = err.Error()
			info.Resolved = false
			m.log.Warn("DNS lookup failed",
				slog.String("alias", alias.Name),
				slog.String("hostname", hostname),
				slog.Any("error", err))
		} else {
			info.IPAddresses = ips
			info.Resolved = true
			m.log.Debug("DNS lookup successful",
				slog.String("alias", alias.Name),
				slog.String("hostname", hostname),
				slog.Any("ips", ips))
		}

		results = append(results, info)
	}

	m.log.Info("Completed IP resolution",
		slog.Int("total", len(results)),
		slog.Int("resolved", countResolved(results)))

	return results, nil
}

// ListWithoutIPs fetches all endpoint aliases without DNS resolution.
// This is faster and useful when only alias information is needed.
func (m *Manager) ListWithoutIPs(ctx context.Context) ([]models.AliasInfo, error) {
	// Fetch all aliases
	allAliases, err := m.client.ListAliases(ctx)
	if err != nil {
		return nil, fmt.Errorf("list aliases: %w", err)
	}

	m.log.Info("Fetched aliases from gateway",
		slog.Int("total", len(allAliases)))

	// Filter endpoint aliases only
	var endpointAliases []models.EndpointAlias
	for _, alias := range allAliases {
		if strings.ToLower(alias.Type) == "endpoint" {
			endpointAliases = append(endpointAliases, alias)
		}
	}

	m.log.Info("Filtered endpoint aliases (DNS resolution skipped)",
		slog.Int("total", len(allAliases)),
		slog.Int("endpoints", len(endpointAliases)))

	// Build results without DNS resolution
	var results []models.AliasInfo
	for _, alias := range endpointAliases {
		hostname, err := extractHostname(alias.EndpointURI)
		if err != nil {
			// Fallback to full URI if hostname extraction fails
			hostname = alias.EndpointURI
			m.log.Warn("Failed to extract hostname, using full URI",
				slog.String("alias", alias.Name),
				slog.String("url", alias.EndpointURI),
				slog.Any("error", err))
		}

		info := models.AliasInfo{
			Name:        alias.Name,
			EndpointURL: alias.EndpointURI,
			Hostname:    hostname,
			IPAddresses: nil,
			Resolved:    false,
			Error:       "skipped", // Marker for skipped DNS resolution
		}
		results = append(results, info)
	}

	return results, nil
}

// extractHostname extracts the hostname from a URL.
func extractHostname(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse URL: %w", err)
	}

	hostname := u.Hostname()
	if hostname == "" {
		return "", fmt.Errorf("no hostname in URL")
	}

	return hostname, nil
}

// countResolved counts how many aliases were successfully resolved.
func countResolved(aliases []models.AliasInfo) int {
	count := 0
	for _, alias := range aliases {
		if alias.Resolved {
			count++
		}
	}
	return count
}

// Made with Bob
