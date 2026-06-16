// Package config provides configuration management for the CLI application.
package config

import (
	"errors"
	"flag"
	"fmt"
	"strings"
	"time"
)

// Config holds all configuration for the CLI application.
type Config struct {
	// API Gateway connection
	GatewayURL string
	Username   string
	Password   string

	// Team configuration
	Teams []string

	// Polling configuration
	Interval  time.Duration
	PageSize  int
	RateLimit int

	// Storage
	DBPath string

	// Logging
	LogLevel string
	DryRun   bool
}

// LoadFromFlags parses command-line flags and returns a [Config].
func LoadFromFlags() (*Config, error) {
	cfg := &Config{}

	// Define flags
	flag.StringVar(&cfg.GatewayURL, "gateway-url", "", "API Gateway base URL (required)")
	flag.StringVar(&cfg.Username, "username", "", "Basic auth username (required)")
	flag.StringVar(&cfg.Password, "password", "", "Basic auth password (required)")
	teamsStr := flag.String("teams", "", "Comma-separated team names to add (required)")
	intervalSec := flag.Int("interval", 60, "Polling interval in seconds")
	flag.IntVar(&cfg.PageSize, "page-size", 100, "Number of APIs to fetch per page")
	flag.IntVar(&cfg.RateLimit, "rate-limit", 10, "Max requests per second")
	flag.StringVar(&cfg.DBPath, "db-path", "data", "Path to NutsDB database directory")
	flag.StringVar(&cfg.LogLevel, "log-level", "info", "Log level: debug, info, warn, error")
	flag.BoolVar(&cfg.DryRun, "dry-run", false, "Simulate without making changes")

	flag.Parse()

	// Parse teams
	if *teamsStr != "" {
		cfg.Teams = strings.Split(*teamsStr, ",")
		for i := range cfg.Teams {
			cfg.Teams[i] = strings.TrimSpace(cfg.Teams[i])
		}
	}

	// Convert interval
	cfg.Interval = time.Duration(*intervalSec) * time.Second

	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validate checks if the configuration is valid.
func validate(cfg *Config) error {
	var errs []string

	if cfg.GatewayURL == "" {
		errs = append(errs, "gateway-url is required")
	}
	if cfg.Username == "" {
		errs = append(errs, "username is required")
	}
	if cfg.Password == "" {
		errs = append(errs, "password is required")
	}
	if len(cfg.Teams) == 0 {
		errs = append(errs, "teams is required")
	}
	if cfg.Interval < time.Second {
		errs = append(errs, "interval must be at least 1 second")
	}
	if cfg.PageSize < 1 || cfg.PageSize > 1000 {
		errs = append(errs, "page-size must be between 1 and 1000")
	}
	if cfg.RateLimit < 1 || cfg.RateLimit > 100 {
		errs = append(errs, "rate-limit must be between 1 and 100")
	}
	if cfg.LogLevel != "debug" && cfg.LogLevel != "info" && cfg.LogLevel != "warn" && cfg.LogLevel != "error" {
		errs = append(errs, "log-level must be one of: debug, info, warn, error")
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}

	return nil
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	return validate(c)
}

// String returns a string representation of the config (without sensitive data).
func (c *Config) String() string {
	return fmt.Sprintf("Config{GatewayURL=%s, Username=%s, Teams=%v, Interval=%s, PageSize=%d, RateLimit=%d, DBPath=%s, LogLevel=%s, DryRun=%t}",
		c.GatewayURL, c.Username, c.Teams, c.Interval, c.PageSize, c.RateLimit, c.DBPath, c.LogLevel, c.DryRun)
}

// Made with Bob
