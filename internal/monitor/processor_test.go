package monitor

import (
	"log/slog"
	"os"
	"testing"
)

func TestExtractAPIMetadata(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	processor := NewProcessor(logger)

	apiJSON := []byte(`{
		"apiResponse": {
			"api": {
				"id": "test-api-123",
				"apiName": "Test API",
				"apiVersion": "1.0",
				"type": "REST",
				"isActive": true
			},
			"teams": [
				{"id": "team1", "name": "Team 1"},
				{"id": "team2", "name": "Team 2"}
			]
		}
	}`)

	metadata, err := processor.ExtractAPIMetadata(apiJSON)
	if err != nil {
		t.Fatalf("ExtractAPIMetadata() error = %v", err)
	}

	if metadata.ID != "test-api-123" {
		t.Errorf("ID = %v, want test-api-123", metadata.ID)
	}
	if metadata.Name != "Test API" {
		t.Errorf("Name = %v, want Test API", metadata.Name)
	}
	if metadata.Version != "1.0" {
		t.Errorf("Version = %v, want 1.0", metadata.Version)
	}
	if metadata.Type != "REST" {
		t.Errorf("Type = %v, want REST", metadata.Type)
	}
	if !metadata.IsActive {
		t.Errorf("IsActive = %v, want true", metadata.IsActive)
	}
	if len(metadata.ExistingTeams) != 2 {
		t.Errorf("len(ExistingTeams) = %v, want 2", len(metadata.ExistingTeams))
	}
}

func TestAddTeamsToAPI(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	processor := NewProcessor(logger)

	tests := []struct {
		name          string
		apiJSON       []byte
		teamsToAdd    []string
		wantTeamCount int
	}{
		{
			name: "add new teams",
			apiJSON: []byte(`{
				"apiResponse": {
					"api": {
						"id": "test-api-123",
						"apiName": "Test API",
						"apiVersion": "1.0",
						"type": "REST",
						"isActive": true
					},
					"teams": [
						{"id": "team1", "name": "Team 1"}
					]
				}
			}`),
			teamsToAdd:    []string{"team2", "team3"},
			wantTeamCount: 3,
		},
		{
			name: "avoid duplicates",
			apiJSON: []byte(`{
				"apiResponse": {
					"api": {
						"id": "test-api-123",
						"apiName": "Test API",
						"apiVersion": "1.0",
						"type": "REST",
						"isActive": true
					},
					"teams": [
						{"id": "team1", "name": "Team 1"}
					]
				}
			}`),
			teamsToAdd:    []string{"team1", "team2"},
			wantTeamCount: 2,
		},
		{
			name: "no existing teams",
			apiJSON: []byte(`{
				"apiResponse": {
					"api": {
						"id": "test-api-123",
						"apiName": "Test API",
						"apiVersion": "1.0",
						"type": "REST",
						"isActive": true
					}
				}
			}`),
			teamsToAdd:    []string{"team1", "team2"},
			wantTeamCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			modifiedJSON, err := processor.AddTeamsToAPI(tt.apiJSON, tt.teamsToAdd)
			if err != nil {
				t.Fatalf("AddTeamsToAPI() error = %v", err)
			}

			// Extract teams from modified JSON
			teamIDs, err := processor.GetTeamIDs(modifiedJSON)
			if err != nil {
				t.Fatalf("GetTeamIDs() error = %v", err)
			}

			if len(teamIDs) != tt.wantTeamCount {
				t.Errorf("len(teamIDs) = %v, want %v", len(teamIDs), tt.wantTeamCount)
			}
		})
	}
}

func TestGetTeamIDs(t *testing.T) {
	t.Parallel()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	processor := NewProcessor(logger)

	apiJSON := []byte(`{
		"apiResponse": {
			"teams": [
				{"id": "team1", "name": "Team 1"},
				{"id": "team2", "name": "Team 2"},
				{"id": "team3", "name": "Team 3"}
			]
		}
	}`)

	teamIDs, err := processor.GetTeamIDs(apiJSON)
	if err != nil {
		t.Fatalf("GetTeamIDs() error = %v", err)
	}

	if len(teamIDs) != 3 {
		t.Errorf("len(teamIDs) = %v, want 3", len(teamIDs))
	}

	expectedIDs := map[string]bool{"team1": true, "team2": true, "team3": true}
	for _, id := range teamIDs {
		if !expectedIDs[id] {
			t.Errorf("unexpected team ID: %v", id)
		}
	}
}

// Made with Bob
