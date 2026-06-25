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
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/thrnjica/agwctl/internal/models"
)

// FormatTable outputs aliases in table format.
// Displays alias name, endpoint URL, and resolved IP addresses in a formatted table.
func FormatTable(w io.Writer, aliases []models.AliasInfo) error {
	// Header
	fmt.Fprintf(w, "%-30s %-50s %-40s\n", "ALIAS NAME", "ENDPOINT URL", "IP ADDRESSES")
	fmt.Fprintf(w, "%s\n", strings.Repeat("-", 120))

	// Rows
	for _, alias := range aliases {
		ips := "<DNS lookup failed>"
		if alias.Resolved {
			ips = strings.Join(alias.IPAddresses, ", ")
		} else if alias.Error != "" {
			ips = fmt.Sprintf("<error: %s>", alias.Error)
		}

		fmt.Fprintf(w, "%-30s %-50s %-40s\n",
			truncate(alias.Name, 30),
			truncate(alias.EndpointURL, 50),
			truncate(ips, 40))
	}

	return nil
}

// FormatJSON outputs aliases in JSON format.
// Returns a JSON object with an "aliases" array containing all alias information.
func FormatJSON(w io.Writer, aliases []models.AliasInfo) error {
	output := map[string]interface{}{
		"aliases": aliases,
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// truncate truncates a string to maxLen characters.
// If the string is longer than maxLen, it is truncated and "..." is appended.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// Made with Bob
