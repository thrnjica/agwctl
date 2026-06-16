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

package config

import (
	"testing"
	"time"
)

func TestValidate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				GatewayURL: "https://gateway.example.com",
				Username:   "admin",
				Password:   "secret",
				Teams:      []string{"Team1", "Team2"},
				Interval:   60 * time.Second,
				PageSize:   100,
				RateLimit:  10,
				LogLevel:   "info",
			},
			wantErr: false,
		},
		{
			name: "missing gateway URL",
			config: &Config{
				Username:  "admin",
				Password:  "secret",
				Teams:     []string{"Team1"},
				Interval:  60 * time.Second,
				PageSize:  100,
				RateLimit: 10,
				LogLevel:  "info",
			},
			wantErr: true,
		},
		{
			name: "missing username",
			config: &Config{
				GatewayURL: "https://gateway.example.com",
				Password:   "secret",
				Teams:      []string{"Team1"},
				Interval:   60 * time.Second,
				PageSize:   100,
				RateLimit:  10,
				LogLevel:   "info",
			},
			wantErr: true,
		},
		{
			name: "missing password",
			config: &Config{
				GatewayURL: "https://gateway.example.com",
				Username:   "admin",
				Teams:      []string{"Team1"},
				Interval:   60 * time.Second,
				PageSize:   100,
				RateLimit:  10,
				LogLevel:   "info",
			},
			wantErr: true,
		},
		{
			name: "missing teams",
			config: &Config{
				GatewayURL: "https://gateway.example.com",
				Username:   "admin",
				Password:   "secret",
				Interval:   60 * time.Second,
				PageSize:   100,
				RateLimit:  10,
				LogLevel:   "info",
			},
			wantErr: true,
		},
		{
			name: "invalid interval",
			config: &Config{
				GatewayURL: "https://gateway.example.com",
				Username:   "admin",
				Password:   "secret",
				Teams:      []string{"Team1"},
				Interval:   0,
				PageSize:   100,
				RateLimit:  10,
				LogLevel:   "info",
			},
			wantErr: true,
		},
		{
			name: "invalid page size - too small",
			config: &Config{
				GatewayURL: "https://gateway.example.com",
				Username:   "admin",
				Password:   "secret",
				Teams:      []string{"Team1"},
				Interval:   60 * time.Second,
				PageSize:   0,
				RateLimit:  10,
				LogLevel:   "info",
			},
			wantErr: true,
		},
		{
			name: "invalid page size - too large",
			config: &Config{
				GatewayURL: "https://gateway.example.com",
				Username:   "admin",
				Password:   "secret",
				Teams:      []string{"Team1"},
				Interval:   60 * time.Second,
				PageSize:   2000,
				RateLimit:  10,
				LogLevel:   "info",
			},
			wantErr: true,
		},
		{
			name: "invalid rate limit",
			config: &Config{
				GatewayURL: "https://gateway.example.com",
				Username:   "admin",
				Password:   "secret",
				Teams:      []string{"Team1"},
				Interval:   60 * time.Second,
				PageSize:   100,
				RateLimit:  0,
				LogLevel:   "info",
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			config: &Config{
				GatewayURL: "https://gateway.example.com",
				Username:   "admin",
				Password:   "secret",
				Teams:      []string{"Team1"},
				Interval:   60 * time.Second,
				PageSize:   100,
				RateLimit:  10,
				LogLevel:   "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigString(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		GatewayURL: "https://gateway.example.com",
		Username:   "admin",
		Password:   "secret",
		Teams:      []string{"Team1", "Team2"},
		Interval:   60 * time.Second,
		PageSize:   100,
		RateLimit:  10,
		DBPath:     "data",
		LogLevel:   "info",
		DryRun:     false,
	}

	str := cfg.String()

	// Should contain non-sensitive info
	if str == "" {
		t.Error("String() returned empty string")
	}

	// Should not contain password
	if contains(str, "secret") {
		t.Error("String() contains password")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Made with Bob
