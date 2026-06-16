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
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/thrnjica/agwctl/internal/client"
	"github.com/thrnjica/agwctl/internal/config"
	"github.com/thrnjica/agwctl/internal/logger"
	"github.com/thrnjica/agwctl/internal/monitor"
	"github.com/thrnjica/agwctl/internal/store"
)

// Version is set via ldflags during build.
var Version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Load configuration from flags
	cfg, err := config.LoadFromFlags()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Setup logger
	log := logger.Setup(cfg.LogLevel)
	log.Info("Starting API Gateway Automator",
		slog.String("gateway_url", cfg.GatewayURL),
		slog.String("username", cfg.Username),
		slog.Any("teams", cfg.Teams),
		slog.Duration("interval", cfg.Interval),
		slog.Int("page_size", cfg.PageSize),
		slog.Int("rate_limit", cfg.RateLimit),
		slog.String("db_path", cfg.DBPath),
		slog.Bool("dry_run", cfg.DryRun))

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		log.Info("Received signal, shutting down gracefully", slog.Any("signal", sig))
		cancel()
	}()

	// Initialize HTTP client
	c := client.New(
		cfg.GatewayURL,
		cfg.Username,
		cfg.Password,
		Version,
		cfg.RateLimit,
		log,
	)

	// Initialize store repository
	repo, err := store.New(cfg.DBPath, log)
	if err != nil {
		return fmt.Errorf("initialize repository: %w", err)
	}
	defer func() {
		if err := repo.Close(); err != nil {
			log.Error("Failed to close repository", slog.Any("error", err))
		}
	}()

	// Initialize team manager
	teamMgr := monitor.NewTeamManager(c, log)

	// Refresh team cache
	log.Info("Fetching teams")
	if err := teamMgr.Refresh(ctx); err != nil {
		return fmt.Errorf("refresh teams: %w", err)
	}

	// Resolve team names to IDs
	teamIDs, err := teamMgr.Resolve(cfg.Teams)
	if err != nil {
		return fmt.Errorf("resolve team names: %w", err)
	}

	log.Info("Team names resolved",
		slog.Any("teams", cfg.Teams),
		slog.Any("team_ids", teamIDs))

	// Initialize JSON processor
	proc := monitor.NewProcessor(log)

	// Initialize poller
	poller := monitor.NewPoller(
		c,
		repo,
		teamMgr,
		proc,
		cfg.Interval,
		cfg.PageSize,
		teamIDs,
		log,
		cfg.DryRun,
	)

	// Print database stats
	stats, err := repo.Stats()
	if err != nil {
		log.Warn("Failed to get database stats", slog.Any("error", err))
	} else {
		log.Info("Database stats report", slog.Any("stats", stats))
	}

	// Start polling
	log.Info("Starting polling loop")
	if err := poller.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("poller error: %w", err)
	}

	log.Info("Shutdown complete")
	return nil
}

// Made with Bob
