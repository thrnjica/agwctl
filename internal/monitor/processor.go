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

// ExtractAPIMetadata extracts metadata from an API JSON document using [gjson].
func (p *Processor) ExtractAPIMetadata(apiJSON []byte) (*models.APIMetadata, error) {
	result := gjson.ParseBytes(apiJSON)

	// Extract from apiResponse.api structure
	apiObj := result.Get("apiResponse.api")
	if !apiObj.Exists() {
		return nil, fmt.Errorf("apiResponse.api not found in JSON")
	}

	meta := &models.APIMetadata{
		ID:       apiObj.Get("id").String(),
		Name:     apiObj.Get("apiName").String(),
		Version:  apiObj.Get("apiVersion").String(),
		Type:     apiObj.Get("type").String(),
		IsActive: apiObj.Get("isActive").Bool(),
	}

	// Extract existing teams
	teamsArray := result.Get("apiResponse.teams")
	if teamsArray.Exists() && teamsArray.IsArray() {
		teamsArray.ForEach(func(_, value gjson.Result) bool {
			teamID := value.Get("id").String()
			if teamID != "" {
				meta.ExistingTeams = append(meta.ExistingTeams, teamID)
			}
			return true // continue iteration
		})
	}

	p.log.Debug("Extracted API metadata",
		slog.String("api_id", meta.ID),
		slog.String("name", meta.Name),
		slog.String("version", meta.Version),
		slog.Int("existing_teams", len(meta.ExistingTeams)))

	return meta, nil
}

// AddTeamsToAPI adds teams to an API JSON document, avoiding duplicates.
func (p *Processor) AddTeamsToAPI(apiJSON []byte, teamIDsToAdd []string) ([]byte, error) {
	// Extract existing teams
	meta, err := p.ExtractAPIMetadata(apiJSON)
	if err != nil {
		return nil, fmt.Errorf("extract metadata: %w", err)
	}

	// Build set of existing team IDs
	existing := make(map[string]bool)
	for _, teamID := range meta.ExistingTeams {
		existing[teamID] = true
	}

	// Add new teams (avoiding duplicates)
	var toAdd []string
	for _, teamID := range teamIDsToAdd {
		if !existing[teamID] {
			toAdd = append(toAdd, teamID)
		}
	}

	if len(toAdd) == 0 {
		p.log.Debug("No new teams to add", slog.String("api_id", meta.ID))
		return apiJSON, nil
	}

	p.log.Debug("Adding teams to API",
		slog.String("api_id", meta.ID),
		slog.Int("new_teams", len(toAdd)),
		slog.Int("existing_teams", len(meta.ExistingTeams)))

	// Get existing teams array from apiResponse
	result := gjson.ParseBytes(apiJSON)
	existingTeamsJSON := result.Get("apiResponse.teams")

	// Build new teams array
	var allTeams []any

	// Add existing teams
	if existingTeamsJSON.Exists() && existingTeamsJSON.IsArray() {
		existingTeamsJSON.ForEach(func(_, value gjson.Result) bool {
			allTeams = append(allTeams, map[string]any{
				"id":   value.Get("id").String(),
				"name": value.Get("name").String(),
			})
			return true
		})
	}

	// Add new teams (with just ID, API Gateway will fill in the name)
	for _, teamID := range toAdd {
		allTeams = append(allTeams, map[string]any{
			"id": teamID,
		})
	}

	// Update the JSON using [sjson]
	modJSON, err := sjson.SetBytes(apiJSON, "apiResponse.teams", allTeams)
	if err != nil {
		return nil, fmt.Errorf("set teams in JSON: %w", err)
	}

	p.log.Debug("Teams added to API JSON",
		slog.String("api_id", meta.ID),
		slog.Int("total_teams", len(allTeams)))

	return modJSON, nil
}

// GetTeamIDs extracts team IDs from an API JSON document.
func (p *Processor) GetTeamIDs(apiJSON []byte) ([]string, error) {
	result := gjson.ParseBytes(apiJSON)
	teamsArray := result.Get("apiResponse.teams")

	var teamIDs []string
	if teamsArray.Exists() && teamsArray.IsArray() {
		teamsArray.ForEach(func(_, value gjson.Result) bool {
			teamID := value.Get("id").String()
			if teamID != "" {
				teamIDs = append(teamIDs, teamID)
			}
			return true
		})
	}

	return teamIDs, nil
}

// Made with Bob
