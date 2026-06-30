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
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/thrnjica/agwctl/internal/models"
)

func TestFormatTable(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		aliases        []models.AliasInfo
		wantContains   []string // Strings that should be in output
		wantNotContain []string // Strings that should NOT be in output
	}{
		{
			name: "successful aliases with IPs",
			aliases: []models.AliasInfo{
				{
					Name:        "TestAlias1",
					EndpointURL: "https://example.com",
					Hostname:    "example.com",
					IPAddresses: []string{"93.184.216.34"},
					Resolved:    true,
				},
			},
			wantContains: []string{
				"ALIAS NAME",
				"ENDPOINT URL",
				"IP ADDRESSES",
				"TestAlias1",
				"https://example.com",
				"93.184.216.34",
			},
		},
		{
			name: "failed DNS lookup",
			aliases: []models.AliasInfo{
				{
					Name:        "TestAlias2",
					EndpointURL: "https://invalid.local",
					Hostname:    "invalid.local",
					Resolved:    false,
					Error:       "lookup failed",
				},
			},
			wantContains: []string{
				"TestAlias2",
				"https://invalid.local",
				"ERROR",
			},
		},
		{
			name: "skipped DNS resolution",
			aliases: []models.AliasInfo{
				{
					Name:        "TestAlias3",
					EndpointURL: "https://skipped.com",
					Hostname:    "skipped.com",
					Resolved:    false,
					Error:       "skipped",
				},
			},
			wantContains: []string{
				"TestAlias3",
				"https://skipped.com",
				"<skipped>",
			},
			wantNotContain: []string{
				"<error:",
				"<DNS lookup failed>",
			},
		},
		{
			name:    "empty list",
			aliases: []models.AliasInfo{},
			wantContains: []string{
				"ALIAS NAME",
				"ENDPOINT URL",
				"IP ADDRESSES",
			},
		},
		{
			name: "long strings truncation",
			aliases: []models.AliasInfo{
				{
					Name:        "VeryLongAliasNameThatExceedsTheMaximumLength",
					EndpointURL: "https://very-long-url-that-exceeds-maximum-length.example.com/path",
					Hostname:    "very-long-url.example.com",
					IPAddresses: []string{"1.2.3.4", "5.6.7.8", "9.10.11.12"},
					Resolved:    true,
				},
			},
			wantContains: []string{
				"...", // Truncation indicator
				"1.2.3.4",
			},
		},
		{
			name: "multiple aliases",
			aliases: []models.AliasInfo{
				{
					Name:        "Alias1",
					EndpointURL: "https://api1.com",
					Hostname:    "api1.com",
					IPAddresses: []string{"1.1.1.1"},
					Resolved:    true,
				},
				{
					Name:        "Alias2",
					EndpointURL: "https://api2.com",
					Hostname:    "api2.com",
					IPAddresses: []string{"2.2.2.2"},
					Resolved:    true,
				},
			},
			wantContains: []string{
				"Alias1",
				"Alias2",
				"1.1.1.1",
				"2.2.2.2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := FormatTable(&buf, tt.aliases)
			if err != nil {
				t.Errorf("FormatTable() error = %v", err)
				return
			}

			output := buf.String()

			// Check for expected strings
			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("FormatTable() output missing expected string: %q\nOutput:\n%s", want, output)
				}
			}

			// Check for strings that should NOT be present
			for _, notWant := range tt.wantNotContain {
				if strings.Contains(output, notWant) {
					t.Errorf("FormatTable() output contains unexpected string: %q\nOutput:\n%s", notWant, output)
				}
			}
		})
	}
}

func TestFormatJSON(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		aliases []models.AliasInfo
		wantErr bool
	}{
		{
			name: "valid JSON output",
			aliases: []models.AliasInfo{
				{
					Name:        "TestAlias",
					EndpointURL: "https://example.com",
					Hostname:    "example.com",
					IPAddresses: []string{"93.184.216.34"},
					Resolved:    true,
				},
			},
			wantErr: false,
		},
		{
			name:    "empty list",
			aliases: []models.AliasInfo{},
			wantErr: false,
		},
		{
			name: "multiple aliases",
			aliases: []models.AliasInfo{
				{
					Name:        "Alias1",
					EndpointURL: "https://api1.com",
					Hostname:    "api1.com",
					IPAddresses: []string{"1.1.1.1"},
					Resolved:    true,
				},
				{
					Name:        "Alias2",
					EndpointURL: "https://api2.com",
					Hostname:    "api2.com",
					Resolved:    false,
					Error:       "lookup failed",
				},
			},
			wantErr: false,
		},
		{
			name: "alias with skipped DNS",
			aliases: []models.AliasInfo{
				{
					Name:        "SkippedAlias",
					EndpointURL: "https://skipped.com",
					Hostname:    "skipped.com",
					Resolved:    false,
					Error:       "skipped",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			err := FormatJSON(&buf, tt.aliases)
			if (err != nil) != tt.wantErr {
				t.Errorf("FormatJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Validate JSON is valid
			if !tt.wantErr {
				var result map[string]interface{}
				if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
					t.Errorf("Invalid JSON output: %v\nOutput:\n%s", err, buf.String())
					return
				}

				// Verify structure
				aliases, ok := result["aliases"]
				if !ok {
					t.Error("JSON output missing 'aliases' key")
					return
				}

				// Verify aliases is an array
				aliasesArray, ok := aliases.([]interface{})
				if !ok {
					t.Errorf("'aliases' is not an array, got type %T", aliases)
					return
				}

				// Verify count matches
				if len(aliasesArray) != len(tt.aliases) {
					t.Errorf("JSON output has %d aliases, want %d", len(aliasesArray), len(tt.aliases))
				}
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "short string",
			input:  "short",
			maxLen: 10,
			want:   "short",
		},
		{
			name:   "exact length",
			input:  "exactly10c",
			maxLen: 10,
			want:   "exactly10c",
		},
		{
			name:   "long string",
			input:  "this is a very long string",
			maxLen: 10,
			want:   "this is...",
		},
		{
			name:   "empty string",
			input:  "",
			maxLen: 10,
			want:   "",
		},
		{
			name:   "one character over",
			input:  "12345678901",
			maxLen: 10,
			want:   "1234567...",
		},
		{
			name:   "very short maxLen",
			input:  "hello",
			maxLen: 3,
			want:   "...",
		},
		{
			name:   "maxLen equals string length",
			input:  "test",
			maxLen: 4,
			want:   "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := truncate(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

// Made with Bob
