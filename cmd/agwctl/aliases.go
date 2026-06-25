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

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/thrnjica/agwctl/internal/alias"
	"github.com/thrnjica/agwctl/internal/client"
	"github.com/thrnjica/agwctl/internal/logger"
)

// aliasesCommand handles the 'aliases list' subcommand.
// Lists all endpoint aliases from the API Gateway and resolves their IP addresses.
func aliasesCommand(args []string) error {
	fs := flag.NewFlagSet("aliases", flag.ExitOnError)

	// Define flags
	gatewayURL := fs.String("gateway-url", "", "API Gateway base URL (required)")
	username := fs.String("username", "", "Basic auth username (required)")
	password := fs.String("password", "", "Basic auth password (required)")
	format := fs.String("format", "table", "Output format: table or json")
	timeout := fs.Int("timeout", 60, "DNS lookup timeout in seconds")
	rateLimit := fs.Int("rate-limit", 10, "Max requests per second")
	logLevel := fs.String("log-level", "info", "Log level: debug, info, warn, error")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Validate required flags
	if *gatewayURL == "" || *username == "" || *password == "" {
		fs.Usage()
		return fmt.Errorf("--gateway-url, --username, and --password are required")
	}

	if *format != "table" && *format != "json" {
		return fmt.Errorf("--format must be 'table' or 'json'")
	}

	// Setup logger
	log := logger.Setup(*logLevel)

	log.Info("Starting alias listing",
		"gateway_url", *gatewayURL,
		"format", *format,
		"timeout", *timeout,
		"rate_limit", *rateLimit)

	// Create client
	c := client.New(*gatewayURL, *username, *password, Version, *rateLimit, log)

	// Create alias manager
	mgr := alias.NewManager(c, time.Duration(*timeout)*time.Second, log)

	// Fetch aliases with IPs
	ctx := context.Background()
	aliases, err := mgr.ListWithIPs(ctx)
	if err != nil {
		return fmt.Errorf("list aliases: %w", err)
	}

	// Format output
	if *format == "json" {
		return alias.FormatJSON(os.Stdout, aliases)
	}
	return alias.FormatTable(os.Stdout, aliases)
}

// Made with Bob
