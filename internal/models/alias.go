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

// EndpointAlias represents an endpoint alias from the API Gateway.
// Based on spec/alias.openapi.json - EndpointAlias definition.
type EndpointAlias struct {
	ID                    string `json:"id"`
	Name                  string `json:"name"`
	Description           string `json:"description,omitempty"`
	Type                  string `json:"type"` // Must be "endpoint" for filtering
	Owner                 string `json:"owner,omitempty"`
	Stage                 string `json:"stage,omitempty"`
	EndpointURI           string `json:"endPointURI"` // Note: Capital P in URI!
	ConnectionTimeout     int32  `json:"connectionTimeout,omitempty"`
	ReadTimeout           int32  `json:"readTimeout,omitempty"`
	PassSecurityHeaders   bool   `json:"passSecurityHeaders,omitempty"`
	KeystoreAlias         string `json:"keystoreAlias,omitempty"`
	KeyAlias              string `json:"keyAlias,omitempty"`
	TruststoreAlias       string `json:"truststoreAlias,omitempty"`
	OptimizationTechnique string `json:"optimizationTechnique,omitempty"` // None, MTOM, SwA
}

// AliasResponseModel represents the response from GET /alias endpoint.
// Based on spec/alias.openapi.json - AliasResponseModel definition.
type AliasResponseModel struct {
	Alias []EndpointAlias `json:"alias"`
}

// AliasInfo contains alias information with resolved IP addresses.
// This is our output format for the CLI command.
type AliasInfo struct {
	Name        string   `json:"name"`
	EndpointURL string   `json:"endpointUrl"`
	Hostname    string   `json:"hostname"`
	IPAddresses []string `json:"ipAddresses"`
	Resolved    bool     `json:"resolved"`
	Error       string   `json:"error,omitempty"`
}

// Made with Bob
