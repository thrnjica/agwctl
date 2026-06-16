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

package monitor

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/thrnjica/agwctl/internal/client"
	"github.com/thrnjica/agwctl/internal/models"
	"github.com/thrnjica/agwctl/internal/store"
)

// Poller implements the polling loop for detecting and processing new APIs.
type Poller struct {
	client   *client.Client
	repo     *store.Store
	teamMgr  *TeamManager
	proc     *Processor
	interval time.Duration
	pageSize int
	teamIDs  []string
	log      *slog.Logger
	dryRun   bool
}

// NewPoller creates a new poller instance.
func NewPoller(
	c *client.Client,
	repo *store.Store,
	teamMgr *TeamManager,
	proc *Processor,
	interval time.Duration,
	pageSize int,
	teamIDs []string,
	log *slog.Logger,
	dryRun bool,
) *Poller {
	return &Poller{
		client:   c,
		repo:     repo,
		teamMgr:  teamMgr,
		proc:     proc,
		interval: interval,
		pageSize: pageSize,
		teamIDs:  teamIDs,
		log:      log,
		dryRun:   dryRun,
	}
}

// Start begins the polling loop. It blocks until the context is cancelled.
func (p *Poller) Start(ctx context.Context) error {
	p.log.Info("Starting poller",
		slog.Duration("interval", p.interval),
		slog.Int("page_size", p.pageSize),
		slog.Int("target_teams", len(p.teamIDs)),
		slog.Bool("dry_run", p.dryRun))

	// Initial poll immediately
	if err := p.poll(ctx); err != nil {
		p.log.Error("Initial poll failed", slog.Any("error", err))
	}

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.log.Info("Poller stopped", slog.Any("reason", ctx.Err()))
			return ctx.Err()

		case <-ticker.C:
			if err := p.poll(ctx); err != nil {
				p.log.Error("Poll failed", slog.Any("error", err))
			}
		}
	}
}

// poll performs a single poll cycle.
func (p *Poller) poll(ctx context.Context) error {
	start := time.Now()
	p.log.Info("Starting poll cycle")

	// Fetch all API IDs with pagination
	apiIDs, err := p.list(ctx)
	if err != nil {
		return fmt.Errorf("fetch all service IDs: %w", err)
	}

	p.log.Info("Fetched all services",
		slog.Int("total", len(apiIDs)),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()))

	// Look for new APIs
	ids, err := p.pending(apiIDs)
	if err != nil {
		return fmt.Errorf("detect new services: %w", err)
	}

	if len(ids) == 0 {
		p.log.Info("No new services detected")
		if err := p.repo.UpdateLastPoll(time.Now()); err != nil {
			p.log.Warn("Failed to update last poll time", slog.Any("error", err))
		}
		return nil
	}

	p.log.Info("New services detected", slog.Int("count", len(ids)))

	// Process new APIs
	processed := 0
	failed := 0

	for _, id := range ids {
		if err := p.process(ctx, id); err != nil {
			p.log.Error("Failed to process service",
				slog.String("api_id", id),
				slog.Any("error", err))
			failed++
		} else {
			processed++
		}
	}

	// Update last poll time
	if err := p.repo.UpdateLastPoll(time.Now()); err != nil {
		p.log.Warn("Failed to update last poll time", slog.Any("error", err))
	}

	dur := time.Since(start)
	p.log.Info("Poll cycle complete",
		slog.Int("processed", processed),
		slog.Int("failed", failed),
		slog.Int64("duration_ms", dur.Milliseconds()))

	return nil
}

// list fetches all API IDs using pagination.
func (p *Poller) list(ctx context.Context) ([]string, error) {
	var ids []string
	from := 0
	page := 1

	for {
		p.log.Debug("Fetching service page",
			slog.Int("page", page),
			slog.Int("from", from),
			slog.Int("size", p.pageSize))

		res, err := p.client.ListServices(ctx, from, p.pageSize)
		if err != nil {
			return nil, fmt.Errorf("list services (page %d): %w", page, err)
		}

		// Extract API IDs
		for _, item := range res.APIResponse {
			if id := item.API.ID; id != "" {
				ids = append(ids, id)
			}
		}

		// Check if we got fewer results than requested (last page)
		if len(res.APIResponse) < p.pageSize {
			p.log.Debug("Reached last page",
				slog.Int("page", page),
				slog.Int("results", len(res.APIResponse)))
			break
		}

		from += p.pageSize
		page++
	}

	return ids, nil
}

// pending identifies which APIs have not been processed yet.
func (p *Poller) pending(ids []string) ([]string, error) {
	var diff []string

	for _, id := range ids {
		seen, err := p.repo.Processed(id)
		if err != nil {
			return nil, fmt.Errorf("check if processed %s: %w", id, err)
		}

		if !seen {
			diff = append(diff, id)
		}
	}

	return diff, nil
}

// process processes a single new API by adding teams to it.
func (p *Poller) process(ctx context.Context, id string) error {
	p.log.Info("Processing new service", slog.String("api_id", id))

	// Fetch full API document
	api, err := p.client.GetService(ctx, id)
	if err != nil {
		return fmt.Errorf("get service: %w", err)
	}

	// Extract metadata
	meta, err := p.proc.Metadata(api)
	if err != nil {
		return fmt.Errorf("extract metadata: %w", err)
	}

	p.log.Info("API metadata extracted",
		slog.String("api_id", meta.ID),
		slog.String("name", meta.Name),
		slog.String("version", meta.Version),
		slog.String("type", meta.Type),
		slog.Int("existing_teams", len(meta.ExistingTeams)))

	// Add teams to API JSON
	mod, err := p.proc.AddTeamsToAPI(api, p.teamIDs)
	if err != nil {
		return fmt.Errorf("add teams to service: %w", err)
	}

	// Update API (unless dry-run)
	if !p.dryRun {
		if err := p.client.UpdateService(ctx, id, mod); err != nil {
			return fmt.Errorf("update service: %w", err)
		}
		p.log.Info("Service updated successfully",
			slog.String("api_id", id),
			slog.Int("teams_added", len(p.teamIDs)))
	} else {
		p.log.Info("DRY RUN: Would update service",
			slog.String("api_id", id),
			slog.Int("teams_to_add", len(p.teamIDs)))
	}

	// Mark as processed
	processed := &models.Service{
		ID:          meta.ID,
		Name:        meta.Name,
		Version:     meta.Version,
		Type:        meta.Type,
		ProcessedAt: time.Now(),
		TeamsAdded:  p.teamIDs,
	}

	if err := p.repo.MarkProcessed(id, processed); err != nil {
		return fmt.Errorf("mark as processed: %w", err)
	}

	return nil
}

// Made with Bob
