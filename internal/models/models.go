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
	Groups      []string `json:"groupIds"`
}

// TeamListResponse represents the response from GET /accessProfiles.
type TeamListResponse struct {
	Teams []Team `json:"accessProfiles"`
}

// APIListResponse represents the response from GET /apis.
type APIListResponse struct {
	Items []APIResponseItem `json:"apiResponse"`
}

// APIResponseItem represents a single API in the list response.
type APIResponseItem struct {
	API    APIInfo `json:"api"`
	Status string  `json:"responseStatus"`
}

// APIInfo contains basic API information from the list endpoint.
type APIInfo struct {
	ID      string `json:"id"`
	Name    string `json:"apiName"`
	Version string `json:"apiVersion"`
	Type    string `json:"type"`
	Active  bool   `json:"isActive"`
}

// API represents metadata about a processed API stored in the database.
type API struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Type        string    `json:"type"`
	ProcessedAt time.Time `json:"processedAt"`
	Teams       []string  `json:"teams"`
}

// APIMetadata contains extracted metadata from an API document.
type APIMetadata struct {
	ID      string
	Name    string
	Version string
	Type    string
	Active  bool
	Teams   []string
}

// Made with Bob
