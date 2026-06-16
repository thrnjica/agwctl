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

// AccessProfileManager manages access profiles (teams) and provides name-to-ID resolution.
type AccessProfileManager struct {
	client *client.Client
	cache  map[string]string // name -> ID mapping
	mu     sync.RWMutex
	log    *slog.Logger
}

// NewAccessProfileManager creates a new access profile manager.
func NewAccessProfileManager(c *client.Client, log *slog.Logger) *AccessProfileManager {
	return &AccessProfileManager{
		client: c,
		cache:  make(map[string]string),
		log:    log,
	}
}

// RefreshCache fetches all access profiles and updates the cache.
func (apm *AccessProfileManager) RefreshCache(ctx context.Context) error {
	apm.log.Info("Refreshing access profiles cache")

	resp, err := apm.client.ListAccessProfiles(ctx)
	if err != nil {
		return fmt.Errorf("list access profiles: %w", err)
	}

	apm.mu.Lock()
	defer apm.mu.Unlock()

	// Clear and rebuild cache
	apm.cache = make(map[string]string)
	for _, profile := range resp.AccessProfiles {
		apm.cache[profile.Name] = profile.ID
		apm.log.Debug("Cached access profile",
			slog.String("name", profile.Name),
			slog.String("id", profile.ID))
	}

	apm.log.Info("Access profiles cache refreshed",
		slog.Int("count", len(apm.cache)))

	return nil
}

// ResolveTeamNames resolves team names to their IDs using the cache.
func (apm *AccessProfileManager) ResolveTeamNames(names []string) ([]string, error) {
	apm.mu.RLock()
	defer apm.mu.RUnlock()

	var ids []string
	var missing []string

	for _, name := range names {
		if id, ok := apm.cache[name]; ok {
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

// GetAllTeams returns all cached team names and IDs.
func (apm *AccessProfileManager) GetAllTeams() map[string]string {
	apm.mu.RLock()
	defer apm.mu.RUnlock()

	// Return a copy to avoid concurrent modification
	result := make(map[string]string, len(apm.cache))
	maps.Copy(result, apm.cache)

	return result
}

// Made with Bob
