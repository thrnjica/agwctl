// Package monitor provides API monitoring and processing functionality.
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
		return fmt.Errorf("fetch all API IDs: %w", err)
	}

	p.log.Info("Fetched all APIs",
		slog.Int("total", len(apiIDs)),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()))

	// Detect new APIs
	newAPIIDs, err := p.pending(apiIDs)
	if err != nil {
		return fmt.Errorf("detect new APIs: %w", err)
	}

	if len(newAPIIDs) == 0 {
		p.log.Info("No new APIs detected")
		if err := p.repo.UpdateLastPoll(time.Now()); err != nil {
			p.log.Warn("Failed to update last poll time", slog.Any("error", err))
		}
		return nil
	}

	p.log.Info("New APIs detected", slog.Int("count", len(newAPIIDs)))

	// Process new APIs
	processed := 0
	failed := 0

	for _, apiID := range newAPIIDs {
		if err := p.process(ctx, apiID); err != nil {
			p.log.Error("Failed to process API",
				slog.String("api_id", apiID),
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
	var allIDs []string
	from := 0
	page := 1

	for {
		p.log.Debug("Fetching API page",
			slog.Int("page", page),
			slog.Int("from", from),
			slog.Int("size", p.pageSize))

		resp, err := p.client.ListServices(ctx, from, p.pageSize)
		if err != nil {
			return nil, fmt.Errorf("list APIs (page %d): %w", page, err)
		}

		// Extract API IDs
		for _, item := range resp.APIResponse {
			if item.API.ID != "" {
				allIDs = append(allIDs, item.API.ID)
			}
		}

		// Check if we got fewer results than requested (last page)
		if len(resp.APIResponse) < p.pageSize {
			p.log.Debug("Reached last page",
				slog.Int("page", page),
				slog.Int("results", len(resp.APIResponse)))
			break
		}

		from += p.pageSize
		page++
	}

	return allIDs, nil
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
func (p *Poller) process(ctx context.Context, apiID string) error {
	p.log.Info("Processing new API", slog.String("api_id", apiID))

	// Fetch full API document
	api, err := p.client.GetService(ctx, apiID)
	if err != nil {
		return fmt.Errorf("get API: %w", err)
	}

	// Extract metadata
	meta, err := p.proc.ExtractAPIMetadata(api)
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
		return fmt.Errorf("add teams to API: %w", err)
	}

	// Update API (unless dry-run)
	if !p.dryRun {
		if err := p.client.UpdateService(ctx, apiID, mod); err != nil {
			return fmt.Errorf("update API: %w", err)
		}
		p.log.Info("API updated successfully",
			slog.String("api_id", apiID),
			slog.Int("teams_added", len(p.teamIDs)))
	} else {
		p.log.Info("DRY RUN: Would update API",
			slog.String("api_id", apiID),
			slog.Int("teams_to_add", len(p.teamIDs)))
	}

	// Mark as processed
	procAPI := &models.Service{
		ID:          meta.ID,
		Name:        meta.Name,
		Version:     meta.Version,
		Type:        meta.Type,
		ProcessedAt: time.Now(),
		TeamsAdded:  p.teamIDs,
	}

	if err := p.repo.MarkProcessed(apiID, procAPI); err != nil {
		return fmt.Errorf("mark as processed: %w", err)
	}

	return nil
}

// Made with Bob
