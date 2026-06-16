// Package monitor provides API monitoring and processing functionality.
package monitor

import (
	"fmt"
	"log/slog"

	"github.com/thrnjica/agwctl/internal/models"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// Processor handles JSON manipulation for API documents using [gjson] and [sjson].
type Processor struct {
	log *slog.Logger
}

// NewProcessor creates a new JSON processor.
func NewProcessor(log *slog.Logger) *Processor {
	return &Processor{
		log: log,
	}
}

// Metadata extracts metadata from an API JSON document using [gjson].
func (p *Processor) Metadata(doc []byte) (*models.ServiceIMetadata, error) {
	result := gjson.ParseBytes(doc)

	// Extract from apiResponse.api structure
	const apiPath = "apiResponse.api"

	api := result.Get(apiPath)
	if !api.Exists() {
		return nil, fmt.Errorf("%s not found in JSON", apiPath)
	}

	meta := &models.ServiceIMetadata{
		ID:       api.Get("id").String(),
		Name:     api.Get("apiName").String(),
		Version:  api.Get("apiVersion").String(),
		Type:     api.Get("type").String(),
		IsActive: api.Get("isActive").Bool(),
	}

	// Extract existing teams
	teams := result.Get("apiResponse.teams")
	if teams.Exists() && teams.IsArray() {
		teams.ForEach(func(_, value gjson.Result) bool {
			id := value.Get("id").String()
			if id != "" {
				meta.ExistingTeams = append(meta.ExistingTeams, id)
			}
			return true // continue iteration
		})
	}

	p.log.Debug("Extracted service metadata",
		slog.String("api_id", meta.ID),
		slog.String("name", meta.Name),
		slog.String("version", meta.Version),
		slog.Int("existing_teams", len(meta.ExistingTeams)))

	return meta, nil
}

// AddTeamsToAPI adds teams to an API JSON document, avoiding duplicates.
func (p *Processor) AddTeamsToAPI(doc []byte, ids []string) ([]byte, error) {
	// Extract existing teams
	meta, err := p.Metadata(doc)
	if err != nil {
		return nil, fmt.Errorf("extract metadata: %w", err)
	}

	// Build set of existing team IDs
	existing := make(map[string]bool)
	for _, teamID := range meta.ExistingTeams {
		existing[teamID] = true
	}

	// Add new teams (avoiding duplicates)
	var queue []string
	for _, id := range ids {
		if !existing[id] {
			queue = append(queue, id)
		}
	}

	if len(queue) == 0 {
		p.log.Debug("No new teams to add", slog.String("api_id", meta.ID))
		return doc, nil
	}

	p.log.Debug("Adding teams to API",
		slog.String("api_id", meta.ID),
		slog.Int("new_teams", len(queue)),
		slog.Int("existing_teams", len(meta.ExistingTeams)))

	// Get existing teams array from apiResponse
	teams := gjson.ParseBytes(doc).Get("apiResponse.teams")

	// Populate new teams array
	var all []any

	// Add existing teams
	if teams.Exists() && teams.IsArray() {
		teams.ForEach(func(_, value gjson.Result) bool {
			all = append(all, map[string]any{
				"id":   value.Get("id").String(),
				"name": value.Get("name").String(),
			})
			return true
		})
	}

	// Add new teams (with just ID, API Gateway will fill in the name)
	for _, id := range queue {
		all = append(all, map[string]any{
			"id": id,
		})
	}

	// Update the JSON using [sjson]
	mod, err := sjson.SetBytes(doc, "apiResponse.teams", all)
	if err != nil {
		return nil, fmt.Errorf("set teams in JSON: %w", err)
	}

	p.log.Debug("Teams added to API JSON",
		slog.String("api_id", meta.ID),
		slog.Int("total_teams", len(all)))

	return mod, nil
}

// GetTeamIDs extracts team IDs from an API JSON document.
func (p *Processor) GetTeamIDs(doc []byte) ([]string, error) {
	result := gjson.ParseBytes(doc)
	teams := result.Get("apiResponse.teams")

	var ids []string
	if teams.Exists() && teams.IsArray() {
		teams.ForEach(func(_, value gjson.Result) bool {
			id := value.Get("id").String()
			if id != "" {
				ids = append(ids, id)
			}
			return true
		})
	}

	return ids, nil
}

// Made with Bob
