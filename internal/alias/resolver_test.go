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
	"testing"
	"time"
)

func TestNewResolver(t *testing.T) {
	t.Parallel()
	timeout := 30 * time.Second
	resolver := NewResolver(timeout)

	if resolver == nil {
		t.Error("NewResolver() returned nil")
	}
	if resolver.timeout != timeout {
		t.Errorf("timeout = %v, want %v", resolver.timeout, timeout)
	}
}

func TestResolveHostname(t *testing.T) {
	t.Parallel()
	resolver := NewResolver(60 * time.Second)

	tests := []struct {
		name     string
		hostname string
		wantErr  bool
		wantIPs  bool // Should return at least one IP
	}{
		{
			name:     "valid hostname - localhost",
			hostname: "localhost",
			wantErr:  false,
			wantIPs:  true,
		},
		{
			name:     "valid hostname - google.com",
			hostname: "google.com",
			wantErr:  false,
			wantIPs:  true,
		},
		{
			name:     "invalid hostname",
			hostname: "this-domain-definitely-does-not-exist-12345.invalid",
			wantErr:  true,
			wantIPs:  false,
		},
		{
			name:     "empty hostname",
			hostname: "",
			wantErr:  true,
			wantIPs:  false,
		},
		{
			name:     "valid hostname - example.com",
			hostname: "example.com",
			wantErr:  false,
			wantIPs:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: Not parallel due to DNS lookups
			ips, err := resolver.ResolveHostname(tt.hostname)

			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveHostname() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantIPs && len(ips) == 0 {
				t.Error("ResolveHostname() returned no IPs")
			}

			if !tt.wantIPs && len(ips) > 0 {
				t.Errorf("ResolveHostname() returned IPs when none expected: %v", ips)
			}

			// Verify IP format if IPs were returned
			if tt.wantIPs && len(ips) > 0 {
				for _, ip := range ips {
					if ip == "" {
						t.Error("ResolveHostname() returned empty IP string")
					}
				}
			}
		})
	}
}

func TestResolveHostnameTimeout(t *testing.T) {
	t.Parallel()
	// Very short timeout to test timeout behavior
	resolver := NewResolver(1 * time.Nanosecond)

	// This should timeout
	_, err := resolver.ResolveHostname("google.com")
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
}

func TestResolveHostnameWithDifferentTimeouts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		timeout  time.Duration
		hostname string
		wantErr  bool
	}{
		{
			name:     "reasonable timeout - localhost",
			timeout:  5 * time.Second,
			hostname: "localhost",
			wantErr:  false,
		},
		{
			name:     "very short timeout",
			timeout:  1 * time.Nanosecond,
			hostname: "google.com",
			wantErr:  true, // Should timeout
		},
		{
			name:     "long timeout - localhost",
			timeout:  60 * time.Second,
			hostname: "localhost",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Not parallel due to DNS lookups
			resolver := NewResolver(tt.timeout)
			_, err := resolver.ResolveHostname(tt.hostname)

			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveHostname() with timeout %v: error = %v, wantErr %v",
					tt.timeout, err, tt.wantErr)
			}
		})
	}
}

// Made with Bob
