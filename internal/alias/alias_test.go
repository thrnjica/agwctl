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
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/thrnjica/agwctl/internal/models"
)

// mockClient implements a mock API client for testing.
type mockClient struct {
	aliases []models.EndpointAlias
	err     error
}

func (m *mockClient) ListAliases(ctx context.Context) ([]models.EndpointAlias, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.aliases, nil
}

// TestNewManager verifies that NewManager creates a manager with correct configuration.
func TestNewManager(t *testing.T) {
	t.Parallel()

	client := &mockClient{}
	timeout := 30 * time.Second
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	mgr := NewManager(client, timeout, log)

	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}
	if mgr.client == nil {
		t.Error("Manager client is nil")
	}
	if mgr.resolver == nil {
		t.Error("Manager resolver is nil")
	}
	if mgr.log == nil {
		t.Error("Manager logger is nil")
	}
}

// TestListWithIPs tests the ListWithIPs method with various scenarios.
func TestListWithIPs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		aliases       []models.EndpointAlias
		clientErr     error
		wantErr       bool
		wantCount     int
		checkResolved bool
	}{
		{
			name: "successful resolution with endpoint aliases",
			aliases: []models.EndpointAlias{
				{
					ID:          "alias1",
					Name:        "test-alias-1",
					Type:        "endpoint",
					EndpointURI: "https://example.com/api",
				},
				{
					ID:          "alias2",
					Name:        "test-alias-2",
					Type:        "endpoint",
					EndpointURI: "https://google.com/api",
				},
			},
			wantErr:       false,
			wantCount:     2,
			checkResolved: true,
		},
		{
			name: "filter non-endpoint aliases",
			aliases: []models.EndpointAlias{
				{
					ID:          "alias1",
					Name:        "endpoint-alias",
					Type:        "endpoint",
					EndpointURI: "https://example.com/api",
				},
				{
					ID:          "alias2",
					Name:        "simple-alias",
					Type:        "simple",
					EndpointURI: "https://example.com/api",
				},
				{
					ID:          "alias3",
					Name:        "routing-alias",
					Type:        "routing",
					EndpointURI: "https://example.com/api",
				},
			},
			wantErr:       false,
			wantCount:     1, // Only endpoint type
			checkResolved: true,
		},
		{
			name: "case insensitive type filtering",
			aliases: []models.EndpointAlias{
				{
					ID:          "alias1",
					Name:        "lowercase",
					Type:        "endpoint",
					EndpointURI: "https://example.com/api",
				},
				{
					ID:          "alias2",
					Name:        "uppercase",
					Type:        "ENDPOINT",
					EndpointURI: "https://example.com/api",
				},
				{
					ID:          "alias3",
					Name:        "mixedcase",
					Type:        "EndPoint",
					EndpointURI: "https://example.com/api",
				},
			},
			wantErr:       false,
			wantCount:     3, // All should be included
			checkResolved: true,
		},
		{
			name: "invalid URL in alias",
			aliases: []models.EndpointAlias{
				{
					ID:          "alias1",
					Name:        "invalid-url",
					Type:        "endpoint",
					EndpointURI: "not-a-valid-url",
				},
			},
			wantErr:       false,
			wantCount:     1,
			checkResolved: false, // Should have error
		},
		{
			name: "empty URL in alias",
			aliases: []models.EndpointAlias{
				{
					ID:          "alias1",
					Name:        "empty-url",
					Type:        "endpoint",
					EndpointURI: "",
				},
			},
			wantErr:       false,
			wantCount:     1,
			checkResolved: false,
		},
		{
			name:      "empty alias list",
			aliases:   []models.EndpointAlias{},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name:      "client error",
			aliases:   nil,
			clientErr: errors.New("API error"),
			wantErr:   true,
			wantCount: 0,
		},
		{
			name: "invalid hostname in URL",
			aliases: []models.EndpointAlias{
				{
					ID:          "alias1",
					Name:        "invalid-hostname",
					Type:        "endpoint",
					EndpointURI: "https://invalid-hostname-that-does-not-exist-12345.com/api",
				},
			},
			wantErr:       false,
			wantCount:     1,
			checkResolved: false, // DNS should fail
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := &mockClient{
				aliases: tt.aliases,
				err:     tt.clientErr,
			}
			log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
			mgr := NewManager(client, 5*time.Second, log)

			ctx := context.Background()
			results, err := mgr.ListWithIPs(ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("ListWithIPs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(results) != tt.wantCount {
				t.Errorf("ListWithIPs() returned %d results, want %d", len(results), tt.wantCount)
			}

			// Check if results have expected resolution status
			if tt.checkResolved && len(results) > 0 {
				hasResolved := false
				for _, r := range results {
					if r.Resolved {
						hasResolved = true
						break
					}
				}
				if !hasResolved && tt.wantCount > 0 {
					t.Error("Expected at least one resolved alias, but none were resolved")
				}
			}

			// Verify that non-resolved aliases have error messages
			for _, r := range results {
				if !r.Resolved && r.Error == "" {
					t.Errorf("Alias %s is not resolved but has no error message", r.Name)
				}
			}
		})
	}
}

// TestListWithoutIPs tests the ListWithoutIPs method.
func TestListWithoutIPs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		aliases   []models.EndpointAlias
		clientErr error
		wantErr   bool
		wantCount int
	}{
		{
			name: "successful listing without DNS",
			aliases: []models.EndpointAlias{
				{
					ID:          "alias1",
					Name:        "test-alias-1",
					Type:        "endpoint",
					EndpointURI: "https://example.com/api",
				},
				{
					ID:          "alias2",
					Name:        "test-alias-2",
					Type:        "endpoint",
					EndpointURI: "https://google.com/api",
				},
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name: "filter non-endpoint aliases",
			aliases: []models.EndpointAlias{
				{
					ID:          "alias1",
					Name:        "endpoint-alias",
					Type:        "endpoint",
					EndpointURI: "https://example.com/api",
				},
				{
					ID:          "alias2",
					Name:        "simple-alias",
					Type:        "simple",
					EndpointURI: "https://example.com/api",
				},
			},
			wantErr:   false,
			wantCount: 1,
		},
		{
			name: "invalid URL fallback to full URI",
			aliases: []models.EndpointAlias{
				{
					ID:          "alias1",
					Name:        "invalid-url",
					Type:        "endpoint",
					EndpointURI: "not-a-valid-url",
				},
			},
			wantErr:   false,
			wantCount: 1,
		},
		{
			name:      "empty alias list",
			aliases:   []models.EndpointAlias{},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name:      "client error",
			aliases:   nil,
			clientErr: errors.New("API error"),
			wantErr:   true,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := &mockClient{
				aliases: tt.aliases,
				err:     tt.clientErr,
			}
			log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
			mgr := NewManager(client, 5*time.Second, log)

			ctx := context.Background()
			results, err := mgr.ListWithoutIPs(ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("ListWithoutIPs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(results) != tt.wantCount {
				t.Errorf("ListWithoutIPs() returned %d results, want %d", len(results), tt.wantCount)
			}

			// Verify all results have "skipped" error marker
			for _, r := range results {
				if r.Error != "skipped" {
					t.Errorf("Alias %s should have 'skipped' error marker, got %q", r.Name, r.Error)
				}
				if r.Resolved {
					t.Errorf("Alias %s should not be marked as resolved", r.Name)
				}
				if len(r.IPAddresses) > 0 {
					t.Errorf("Alias %s should not have IP addresses", r.Name)
				}
			}
		})
	}
}

// TestExtractHostname tests the extractHostname helper function.
func TestExtractHostname(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		url      string
		want     string
		wantErr  bool
		errMatch string
	}{
		{
			name:    "valid HTTPS URL",
			url:     "https://example.com/api/v1",
			want:    "example.com",
			wantErr: false,
		},
		{
			name:    "valid HTTP URL",
			url:     "http://api.example.com:8080/path",
			want:    "api.example.com",
			wantErr: false,
		},
		{
			name:    "URL with port",
			url:     "https://localhost:9443/gateway",
			want:    "localhost",
			wantErr: false,
		},
		{
			name:    "URL with query parameters",
			url:     "https://example.com/api?key=value",
			want:    "example.com",
			wantErr: false,
		},
		{
			name:     "invalid URL",
			url:      "not-a-valid-url",
			want:     "",
			wantErr:  true,
			errMatch: "no hostname",
		},
		{
			name:     "empty URL",
			url:      "",
			want:     "",
			wantErr:  true,
			errMatch: "no hostname",
		},
		{
			name:     "URL without hostname",
			url:      "file:///path/to/file",
			want:     "",
			wantErr:  true,
			errMatch: "no hostname",
		},
		{
			name:    "IP address as hostname",
			url:     "https://192.168.1.1/api",
			want:    "192.168.1.1",
			wantErr: false,
		},
		{
			name:    "IPv6 address as hostname",
			url:     "https://[2001:db8::1]/api",
			want:    "2001:db8::1",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := extractHostname(tt.url)

			if (err != nil) != tt.wantErr {
				t.Errorf("extractHostname() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("extractHostname() = %v, want %v", got, tt.want)
			}

			if tt.wantErr && tt.errMatch != "" && err != nil {
				if !contains(err.Error(), tt.errMatch) {
					t.Errorf("extractHostname() error = %v, should contain %q", err, tt.errMatch)
				}
			}
		})
	}
}

// TestCountResolved tests the countResolved helper function.
func TestCountResolved(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		aliases []models.AliasInfo
		want    int
	}{
		{
			name: "all resolved",
			aliases: []models.AliasInfo{
				{Name: "alias1", Resolved: true},
				{Name: "alias2", Resolved: true},
				{Name: "alias3", Resolved: true},
			},
			want: 3,
		},
		{
			name: "none resolved",
			aliases: []models.AliasInfo{
				{Name: "alias1", Resolved: false},
				{Name: "alias2", Resolved: false},
			},
			want: 0,
		},
		{
			name: "mixed resolution",
			aliases: []models.AliasInfo{
				{Name: "alias1", Resolved: true},
				{Name: "alias2", Resolved: false},
				{Name: "alias3", Resolved: true},
				{Name: "alias4", Resolved: false},
			},
			want: 2,
		},
		{
			name:    "empty list",
			aliases: []models.AliasInfo{},
			want:    0,
		},
		{
			name:    "nil list",
			aliases: nil,
			want:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := countResolved(tt.aliases)
			if got != tt.want {
				t.Errorf("countResolved() = %v, want %v", got, tt.want)
			}
		})
	}
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Made with Bob
