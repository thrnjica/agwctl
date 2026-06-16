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
	Teams []Team `json:"accessProfiles"`
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
