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

package alias

import (
	"context"
	"net"
	"time"
)

// Resolver performs DNS lookups with timeout support.
type Resolver struct {
	timeout time.Duration
}

// NewResolver creates a new DNS resolver with the specified timeout.
func NewResolver(timeout time.Duration) *Resolver {
	return &Resolver{timeout: timeout}
}

// ResolveHostname performs DNS lookup for the given hostname.
// Returns a list of IP addresses (IPv4 and IPv6) or an error if the lookup fails.
func (r *Resolver) ResolveHostname(hostname string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	// Use net.LookupIP for DNS resolution (supports both IPv4 and IPv6)
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", hostname)
	if err != nil {
		return nil, err
	}

	// Convert IP addresses to strings
	var ipStrings []string
	for _, ip := range ips {
		ipStrings = append(ipStrings, ip.String())
	}

	return ipStrings, nil
}

// Made with Bob
