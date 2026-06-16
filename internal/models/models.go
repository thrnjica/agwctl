// Package models defines data structures for API Gateway entities.
package models

import "time"

// AccessProfile represents a team in the API Gateway.
type AccessProfile struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	GroupIDs    []string `json:"groupIds"`
}

// AccessProfileListResponse represents the response from GET /accessProfiles.
type AccessProfileListResponse struct {
	AccessProfiles []AccessProfile `json:"accessProfiles"`
}

// APIListResponse represents the response from GET /apis.
type APIListResponse struct {
	APIResponse []APIResponseItem `json:"apiResponse"`
}

// APIResponseItem represents a single API in the list response.
type APIResponseItem struct {
	API            APIBasic `json:"api"`
	ResponseStatus string   `json:"responseStatus"`
}

// APIBasic contains basic API information from the list endpoint.
type APIBasic struct {
	ID         string `json:"id"`
	APIName    string `json:"apiName"`
	APIVersion string `json:"apiVersion"`
	Type       string `json:"type"`
	IsActive   bool   `json:"isActive"`
}

// ProcessedAPI represents metadata about a processed API stored in the database.
type ProcessedAPI struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Type        string    `json:"type"`
	ProcessedAt time.Time `json:"processedAt"`
	TeamsAdded  []string  `json:"teamsAdded"`
}

// APIMetadata contains extracted metadata from an API document.
type APIMetadata struct {
	ID            string
	Name          string
	Version       string
	Type          string
	IsActive      bool
	ExistingTeams []string
}

// Made with Bob
