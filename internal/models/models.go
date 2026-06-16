// Package models defines data structures for API Gateway entities.
package models

import "time"

// Team represents a team (access profile) in the API Gateway.
type Team struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	GroupIDs    []string `json:"groupIds"`
}

// TeamListResponse represents the response from GET /accessProfiles.
type TeamListResponse struct {
	AccessProfiles []Team `json:"accessProfiles"`
}

// ServiceListResponse represents the response from GET /apis.
type ServiceListResponse struct {
	APIResponse []ServiceResponseItem `json:"apiResponse"`
}

// ServiceResponseItem represents a single API in the list response.
type ServiceResponseItem struct {
	API            ServiceInfo `json:"api"`
	ResponseStatus string      `json:"responseStatus"`
}

// ServiceInfo contains basic API information from the list endpoint.
type ServiceInfo struct {
	ID         string `json:"id"`
	APIName    string `json:"apiName"`
	APIVersion string `json:"apiVersion"`
	Type       string `json:"type"`
	IsActive   bool   `json:"isActive"`
}

// Service represents metadata about a processed API stored in the database.
type Service struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Type        string    `json:"type"`
	ProcessedAt time.Time `json:"processedAt"`
	TeamsAdded  []string  `json:"teamsAdded"`
}

// ServiceIMetadata contains extracted metadata from an API document.
type ServiceIMetadata struct {
	ID            string
	Name          string
	Version       string
	Type          string
	IsActive      bool
	ExistingTeams []string
}

// Made with Bob
