// Package monitor provides API monitoring and processing functionality.
package monitor

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"sync"

	"github.com/thrnjica/agwctl/internal/client"
)

// TeamManager manages teams and provides name-to-ID resolution.
type TeamManager struct {
	client *client.Client
	cache  map[string]string
	mu     sync.RWMutex
	log    *slog.Logger
}

// NewTeamManager creates a new team manager with the given client and logger.
func NewTeamManager(c *client.Client, log *slog.Logger) *TeamManager {
	return &TeamManager{
		client: c,
		cache:  make(map[string]string),
		log:    log,
	}
}

// Refresh fetches all teams and updates the cache.
func (t *TeamManager) Refresh(ctx context.Context) error {
	t.log.Info("Refreshing team cache")

	res, err := t.client.ListTeams(ctx)
	if err != nil {
		return fmt.Errorf("list teams: %w", err)
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	// Clear and rebuild cache
	t.cache = make(map[string]string)
	for _, team := range res.Teams {
		t.cache[team.Name] = team.ID
		t.log.Debug("Cached team",
			slog.String("name", team.Name),
			slog.String("id", team.ID))
	}

	t.log.Info("Team cache refreshed",
		slog.Int("count", len(t.cache)))

	return nil
}

// Resolve resolves team names to their IDs using the cache.
func (t *TeamManager) Resolve(names []string) ([]string, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var ids []string
	var missing []string

	for _, name := range names {
		if id, ok := t.cache[name]; ok {
			ids = append(ids, id)
		} else {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("teams not found: %v", missing)
	}

	return ids, nil
}

// All returns all cached team names and IDs.
func (t *TeamManager) All() map[string]string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Return a copy to avoid concurrent modification
	result := make(map[string]string, len(t.cache))
	maps.Copy(result, t.cache)

	return result
}

// Made with Bob
