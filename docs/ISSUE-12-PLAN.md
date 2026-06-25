# Implementierungsplan: Issue #12 - Endpoint-Alias Auflistung mit IP-Adress-Ermittlung

## Übersicht

Implementierung eines eigenständigen CLI-Commands `agwctl aliases list`, das alle Endpoint-Aliase vom IBM webMethods API Gateway abruft und deren IP-Adressen via DNS-Lookup ermittelt.

## Architektur-Übersicht

```
┌─────────────────────────────────────────────────────────────┐
│                     agwctl aliases list                      │
│                      (CLI Command)                           │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                    Alias Manager                             │
│  - Filterung (nur Endpoint-Aliase)                          │
│  - Hostname-Extraktion aus URLs                             │
│  - Orchestrierung der Komponenten                           │
└────────┬────────────────────────────────┬───────────────────┘
         │                                │
         ▼                                ▼
┌────────────────────┐          ┌────────────────────────────┐
│   API Gateway      │          │     DNS Resolver           │
│   Client           │          │  - net.LookupIP()          │
│  - ListAliases()   │          │  - Timeout-Handling        │
│  - GetAlias()      │          │  - IPv4/IPv6-Support       │
└────────────────────┘          └────────────────────────────┘
         │                                │
         └────────────────┬───────────────┘
                          ▼
                 ┌────────────────────┐
                 │    Formatter       │
                 │  - Table-Ausgabe   │
                 │  - JSON-Ausgabe    │
                 └────────────────────┘
```

## Detaillierte Implementierungsschritte

### Phase 1: Grundlagen & Datenmodelle

#### 1.1 OpenAPI-Spezifikation verwenden
**Datei:** `spec/alias.openapi.json` (umbenannt von APIGatewayAlias.json)

**Status:** ✅ **ABGESCHLOSSEN** - Swagger 2.0 Spezifikation bereits vom API Gateway vorhanden

**Wichtige Erkenntnisse:**
- Swagger 2.0 Format (nicht OpenAPI 3.0)
- Endpoint: `GET /alias` - Liste aller Aliase (KEINE Pagination-Parameter!)
- Endpoint: `GET /alias/{aliasId}` - Einzelner Alias
- Response-Format: Array von Alias-Objekten direkt (kein Wrapper-Objekt)
- EndpointAlias-Schema definiert mit Feldern:
  - `id`, `name`, `description`, `type`, `owner`, `stage` (von Basis-Alias)
  - `endPointURI` (String) - **Hauptfeld für URL**
  - `connectionTimeout` (int32)
  - `readTimeout` (int32)
  - `passSecurityHeaders` (boolean)
  - `keystoreAlias`, `keyAlias`, `truststoreAlias` (Strings)
  - `optimizationTechnique` (enum: None, MTOM, SwA)

**Wichtig:** API unterstützt **KEINE Pagination** - alle Aliase werden in einem Request zurückgegeben!

---

#### 1.2 Datenmodelle definieren
**Datei:** `internal/models/alias.go`

**Aufgabe:**
- Go-Structs für Alias-Daten basierend auf tatsächlicher API-Struktur erstellen:
  ```go
  // EndpointAlias represents an endpoint alias from the API Gateway.
  // Based on spec/APIGatewayAlias.json - EndpointAlias definition
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
  ```

**Wichtige Anpassungen:**
- ✅ Feldname ist `endPointURI` (nicht `endpointURI`) - Großes P!
- ✅ Type-Wert ist `"endpoint"` (lowercase) für Endpoint-Aliase
- ✅ Keine separate `AliasListResponse` - API gibt Array direkt zurück
- ✅ Alle optionalen Felder mit `omitempty`

**Akzeptanzkriterien:**
- ✅ Structs entsprechen Swagger-Schema exakt
- ✅ JSON-Tags matchen API-Response
- ✅ Dokumentation mit Kommentaren

---

### Phase 2: API-Client-Erweiterung

#### 2.1 Client-Methoden implementieren
**Datei:** `internal/client/client.go`

**Aufgabe:**
- Neue Methoden zum bestehenden Client hinzufügen:
  ```go
  // ListAliases fetches all aliases from the gateway.
  // Note: API does NOT support pagination - returns all aliases in one call.
  func (c *Client) ListAliases(
      ctx context.Context,
  ) ([]models.EndpointAlias, error)

  // GetAlias fetches a single alias by ID.
  func (c *Client) GetAlias(
      ctx context.Context,
      aliasID string,
  ) (*models.EndpointAlias, error)
  ```

**Implementation:**
- Analog zu bestehenden `ListAPIs()` und `GetAPI()` Methoden
- Verwendung der `call()` Hilfsfunktion
- **KEINE Pagination** - API gibt alle Aliase in einem Request zurück
- Response ist direkt ein Array von Aliases (kein Wrapper-Objekt)
- Fehlerbehandlung und Logging

**Wichtige Änderungen:**
- ❌ Keine `from`/`size` Parameter - API unterstützt keine Pagination
- ✅ Return-Type ist `[]models.EndpointAlias` (Array direkt)
- ✅ Einfachere Implementierung ohne Pagination-Loop

**Akzeptanzkriterien:**
- ✅ Methoden folgen bestehendem Client-Pattern
- ✅ Rate-Limiting wird respektiert (via Transport)
- ✅ Strukturiertes Logging implementiert
- ✅ Fehlerbehandlung konsistent
- ✅ Korrekte Deserialisierung des Array-Response

---

### Phase 3: DNS-Resolver

#### 3.1 DNS-Resolver implementieren
**Datei:** `internal/alias/resolver.go`

**Aufgabe:**
- DNS-Lookup mit `net.LookupIP()` und konfigurierbarem Timeout
- IPv4 und IPv6 Unterstützung
- Fehlerbehandlung für nicht auflösbare Hosts

**Implementation:**
```go
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

// NewResolver creates a new DNS resolver.
func NewResolver(timeout time.Duration) *Resolver {
    return &Resolver{timeout: timeout}
}

// ResolveHostname performs DNS lookup for the given hostname.
func (r *Resolver) ResolveHostname(hostname string) ([]string, error) {
    ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
    defer cancel()

    ips, err := net.DefaultResolver.LookupIP(ctx, "ip", hostname)
    if err != nil {
        return nil, err
    }

    var ipStrings []string
    for _, ip := range ips {
        ipStrings = append(ipStrings, ip.String())
    }

    return ipStrings, nil
}
```

**Akzeptanzkriterien:**
- ✅ Timeout-Handling funktioniert (Standard: 60s)
- ✅ IPv4 und IPv6 werden unterstützt
- ✅ Fehler werden sauber zurückgegeben
- ✅ Context-basierte Timeouts

---

#### 3.2 Unit-Tests für Resolver
**Datei:** `internal/alias/resolver_test.go`

**Aufgabe:**
- Test für erfolgreiche DNS-Auflösung
- Test für Timeout-Verhalten
- Test für nicht auflösbare Hosts
- Test für IPv4/IPv6-Unterstützung

**Akzeptanzkriterien:**
- ✅ Mindestens 80% Code-Coverage
- ✅ Edge-Cases abgedeckt
- ✅ Timeout-Tests funktionieren

---

### Phase 4: Alias-Manager

#### 4.1 Alias-Manager implementieren
**Datei:** `internal/alias/alias.go`

**Aufgabe:**
- Orchestrierung der Komponenten
- Filterung: Nur Endpoint-Aliase (Type == "ENDPOINT")
- Hostname-Extraktion aus URLs
- Integration von Client und Resolver

**Implementation:**
```go
package alias

import (
    "context"
    "fmt"
    "log/slog"
    "net/url"
    "time"

    "github.com/thrnjica/agwctl/internal/client"
    "github.com/thrnjica/agwctl/internal/models"
)

// Manager orchestrates alias listing and DNS resolution.
type Manager struct {
    client   *client.Client
    resolver *Resolver
    log      *slog.Logger
}

// NewManager creates a new alias manager.
func NewManager(
    client *client.Client,
    timeout time.Duration,
    log *slog.Logger,
) *Manager {
    return &Manager{
        client:   client,
        resolver: NewResolver(timeout),
        log:      log,
    }
}

// ListWithIPs fetches all endpoint aliases and resolves their IPs.
func (m *Manager) ListWithIPs(ctx context.Context) ([]models.AliasInfo, error) {
    // Fetch all aliases (no pagination - API returns all in one call)
    allAliases, err := m.client.ListAliases(ctx)
    if err != nil {
        return nil, fmt.Errorf("list aliases: %w", err)
    }

    m.log.Info("Fetched aliases from gateway",
        slog.Int("total", len(allAliases)))

    // Filter endpoint aliases only (type is "endpoint" - lowercase!)
    var endpointAliases []models.EndpointAlias
    for _, alias := range allAliases {
        if strings.ToLower(alias.Type) == "endpoint" {
            endpointAliases = append(endpointAliases, alias)
        }
    }

    m.log.Info("Filtered endpoint aliases",
        slog.Int("total", len(allAliases)),
        slog.Int("endpoints", len(endpointAliases)))

    // Resolve IPs for each alias
    var results []models.AliasInfo
    for _, alias := range endpointAliases {
        info := models.AliasInfo{
            Name:        alias.Name,
            EndpointURL: alias.EndpointURI, // Note: Field is EndpointURI (capital P)
        }

        // Extract hostname from EndpointURI
        hostname, err := extractHostname(alias.EndpointURI)
        if err != nil {
            info.Error = err.Error()
            results = append(results, info)
            continue
        }
        info.Hostname = hostname

        // Resolve IPs
        ips, err := m.resolver.ResolveHostname(hostname)
        if err != nil {
            info.Error = err.Error()
            info.Resolved = false
        } else {
            info.IPAddresses = ips
            info.Resolved = true
        }

        results = append(results, info)
    }

    return results, nil
}

// extractHostname extracts the hostname from a URL.
func extractHostname(rawURL string) (string, error) {
    u, err := url.Parse(rawURL)
    if err != nil {
        return "", fmt.Errorf("parse URL: %w", err)
    }
    return u.Hostname(), nil
}
```

**Akzeptanzkriterien:**
- ✅ Pagination funktioniert korrekt
- ✅ Nur Endpoint-Aliase werden verarbeitet
- ✅ Hostname-Extraktion funktioniert
- ✅ Fehlerbehandlung für jeden Alias

---

#### 4.2 Unit-Tests für Manager
**Datei:** `internal/alias/alias_test.go`

**Aufgabe:**
- Test für Filterung (nur Endpoint-Aliase)
- Test für Hostname-Extraktion
- Test für Pagination
- Mock-Client für Tests

**Akzeptanzkriterien:**
- ✅ Mindestens 80% Code-Coverage
- ✅ Mock-Client verwendet
- ✅ Edge-Cases getestet

---

### Phase 5: Ausgabe-Formatierung

#### 5.1 Formatter implementieren
**Datei:** `internal/alias/formatter.go`

**Aufgabe:**
- Tabellarische Ausgabe (ASCII-Table)
- JSON-Ausgabe
- Fehlerbehandlung in Ausgabe

**Implementation:**
```go
package alias

import (
    "encoding/json"
    "fmt"
    "io"
    "strings"

    "github.com/thrnjica/agwctl/internal/models"
)

// FormatTable outputs aliases in table format.
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
func FormatJSON(w io.Writer, aliases []models.AliasInfo) error {
    output := map[string]interface{}{
        "aliases": aliases,
    }

    encoder := json.NewEncoder(w)
    encoder.SetIndent("", "  ")
    return encoder.Encode(output)
}

// truncate truncates a string to maxLen characters.
func truncate(s string, maxLen int) string {
    if len(s) <= maxLen {
        return s
    }
    return s[:maxLen-3] + "..."
}
```

**Akzeptanzkriterien:**
- ✅ Tabellarische Ausgabe lesbar
- ✅ JSON-Ausgabe valide
- ✅ Lange Strings werden gekürzt
- ✅ Fehler werden angezeigt

---

#### 5.2 Unit-Tests für Formatter
**Datei:** `internal/alias/formatter_test.go`

**Aufgabe:**
- Test für Table-Format
- Test für JSON-Format
- Test für String-Truncation

**Akzeptanzkriterien:**
- ✅ Output-Validierung
- ✅ Edge-Cases getestet

---

### Phase 6: CLI-Integration

#### 6.1 CLI-Command implementieren
**Datei:** `cmd/agwctl/aliases.go`

**Aufgabe:**
- Neuer Subcommand `aliases list`
- Flag-Parsing (ohne externe Libraries wie Cobra)
- Integration aller Komponenten

**Implementation:**
```go
package main

import (
    "context"
    "flag"
    "fmt"
    "os"
    "time"

    "github.com/thrnjica/agwctl/internal/alias"
    "github.com/thrnjica/agwctl/internal/client"
    "github.com/thrnjica/agwctl/internal/logger"
)

// aliasesCommand handles the 'aliases' subcommand.
func aliasesCommand(args []string) error {
    fs := flag.NewFlagSet("aliases", flag.ExitOnError)

    // Define flags
    gatewayURL := fs.String("gateway-url", "", "API Gateway base URL (required)")
    username := fs.String("username", "", "Basic auth username (required)")
    password := fs.String("password", "", "Basic auth password (required)")
    format := fs.String("format", "table", "Output format: table or json")
    timeout := fs.Int("timeout", 60, "DNS lookup timeout in seconds")
    rateLimit := fs.Int("rate-limit", 10, "Max requests per second")
    logLevel := fs.String("log-level", "info", "Log level: debug, info, warn, error")

    if err := fs.Parse(args); err != nil {
        return err
    }

    // Validate required flags
    if *gatewayURL == "" || *username == "" || *password == "" {
        return fmt.Errorf("--gateway-url, --username, and --password are required")
    }

    if *format != "table" && *format != "json" {
        return fmt.Errorf("--format must be 'table' or 'json'")
    }

    // Setup logger
    log := logger.Setup(*logLevel)

    // Create client
    c := client.New(*gatewayURL, *username, *password, Version, *rateLimit, log)

    // Create alias manager
    mgr := alias.NewManager(c, time.Duration(*timeout)*time.Second, log)

    // Fetch aliases with IPs
    ctx := context.Background()
    aliases, err := mgr.ListWithIPs(ctx)
    if err != nil {
        return fmt.Errorf("list aliases: %w", err)
    }

    // Format output
    if *format == "json" {
        return alias.FormatJSON(os.Stdout, aliases)
    }
    return alias.FormatTable(os.Stdout, aliases)
}
```

**Akzeptanzkriterien:**
- ✅ Alle Flags funktionieren
- ✅ Validierung der Pflichtfelder
- ✅ Fehlerbehandlung
- ✅ Logging integriert

---

#### 6.2 Main-Funktion anpassen
**Datei:** `cmd/agwctl/main.go`

**Aufgabe:**
- Subcommand-Routing implementieren
- Bestehende Funktionalität beibehalten
- Hilfe-Text erweitern

**Implementation:**
```go
func main() {
    if len(os.Args) < 2 {
        if err := run(); err != nil {
            fmt.Fprintf(os.Stderr, "Error: %v\n", err)
            os.Exit(1)
        }
        return
    }

    // Handle subcommands
    switch os.Args[1] {
    case "aliases":
        if len(os.Args) < 3 || os.Args[2] != "list" {
            fmt.Fprintf(os.Stderr, "Usage: agwctl aliases list [flags]\n")
            os.Exit(1)
        }
        if err := aliasesCommand(os.Args[3:]); err != nil {
            fmt.Fprintf(os.Stderr, "Error: %v\n", err)
            os.Exit(1)
        }
    default:
        // Run original monitoring functionality
        if err := run(); err != nil {
            fmt.Fprintf(os.Stderr, "Error: %v\n", err)
            os.Exit(1)
        }
    }
}
```

**Akzeptanzkriterien:**
- ✅ Subcommand-Routing funktioniert
- ✅ Bestehende Funktionalität unverändert
- ✅ Hilfe-Text aktualisiert

---

### Phase 7: Tests & Dokumentation

#### 7.1 Integration-Tests
**Datei:** `internal/alias/integration_test.go`

**Aufgabe:**
- End-to-End Test mit Mock-Gateway
- Test für kompletten Workflow
- Test für Fehlerszenarien

**Akzeptanzkriterien:**
- ✅ Mock-Gateway implementiert
- ✅ Kompletter Workflow getestet
- ✅ Fehlerszenarien abgedeckt

---

#### 7.2 README aktualisieren
**Datei:** `README.md`

**Aufgabe:**
- Neuen `aliases list` Command dokumentieren
- Beispiele hinzufügen
- Flag-Beschreibungen

**Beispiel-Sektion:**
```markdown
### Alias Management

List all endpoint aliases with their IP addresses:

```bash
agwctl aliases list \
  --gateway-url=https://gateway.example.com:5555/rest/apigateway \
  --username=admin \
  --password=secret \
  --format=table
```

Output in JSON format:

```bash
agwctl aliases list \
  --gateway-url=https://gateway.example.com:5555/rest/apigateway \
  --username=admin \
  --password=secret \
  --format=json
```
```

**Akzeptanzkriterien:**
- ✅ Vollständige Dokumentation
- ✅ Beispiele funktionieren
- ✅ Alle Flags dokumentiert

---

#### 7.3 QUICKSTART erweitern
**Datei:** `docs/QUICKSTART.md`

**Aufgabe:**
- Quick-Start-Beispiel für Alias-Command
- Troubleshooting-Tipps
- Häufige Anwendungsfälle

**Akzeptanzkriterien:**
- ✅ Schnelleinstieg dokumentiert
- ✅ Troubleshooting-Sektion
- ✅ Beispiele getestet

---

## Abhängigkeiten

### Externe Dependencies
- **Keine neuen Dependencies erforderlich**
- Verwendung von Standard-Library:
  - `net` für DNS-Lookup
  - `encoding/json` für JSON-Ausgabe
  - `flag` für CLI-Parsing
  - `context` für Timeouts

### Interne Dependencies
- `internal/client` - Bestehender HTTP-Client
- `internal/logger` - Bestehende Logging-Infrastruktur
- `internal/models` - Erweitert um Alias-Modelle

---

## Zeitplan

| Phase | Aufwand | Abhängigkeiten |
|-------|---------|----------------|
| Phase 1: Grundlagen | 0.5 Tage | - |
| Phase 2: Client | 0.5 Tage | Phase 1 |
| Phase 3: DNS-Resolver | 1 Tag | - |
| Phase 4: Alias-Manager | 1 Tag | Phase 2, 3 |
| Phase 5: Formatter | 0.5 Tage | Phase 4 |
| Phase 6: CLI-Integration | 1 Tag | Phase 1-5 |
| Phase 7: Tests & Docs | 1 Tag | Phase 1-6 |
| **Gesamt** | **5.5 Tage** | |

---

## Risiken & Mitigationen

### Risiko 1: API-Dokumentation unvollständig
**Wahrscheinlichkeit:** Mittel  
**Impact:** Hoch  
**Mitigation:** 
- Frühzeitig API-Endpoints testen
- Bei Unklarheiten IBM-Support kontaktieren
- Reverse-Engineering via Browser DevTools

### Risiko 2: DNS-Timeouts in großen Umgebungen
**Wahrscheinlichkeit:** Mittel  
**Impact:** Mittel  
**Mitigation:**
- Konfigurierbarer Timeout (Standard: 60s)
- Parallele DNS-Lookups (optional für Phase 2)
- Fehlerbehandlung pro Alias

### Risiko 3: Rate-Limiting bei vielen Aliases
**Wahrscheinlichkeit:** Niedrig  
**Impact:** Mittel  
**Mitigation:**
- Bestehender Rate-Limiter im Client
- Konfigurierbar via `--rate-limit` Flag
- Pagination-Support

---

## Offene Fragen

1. **Sollen DNS-Lookups parallel ausgeführt werden?**
   - Pro: Schneller bei vielen Aliases
   - Contra: Komplexere Implementierung
   - **Entscheidung:** Zunächst sequentiell, Parallelisierung in Phase 2

2. **Soll DNS-Caching implementiert werden?**
   - Pro: Schnellere wiederholte Lookups
   - Contra: Zusätzliche Komplexität
   - **Entscheidung:** Nein, da Command einmalig ausgeführt wird

3. **Sollen andere Alias-Typen unterstützt werden?**
   - Laut Issue nur Endpoint-Aliase
   - **Entscheidung:** Nur Endpoint-Aliase in Phase 1

---

## Akzeptanzkriterien (Gesamt)

- ✅ OpenAPI-Spezifikation für `/alias` Endpoint erstellt
- ✅ CLI-Command `agwctl aliases list` funktioniert
- ✅ Nur Endpoint-Aliase werden aufgelistet
- ✅ DNS-Lookup mit `net.LookupIP()` funktioniert mit konfigurierbarem Timeout (Standard: 60s)
- ✅ Unterstützung für IPv4 und IPv6
- ✅ Ausgabe in Table- und JSON-Format möglich
- ✅ Rate-Limiting wird respektiert
- ✅ Fehlerbehandlung für nicht auflösbare Hosts
- ✅ Unit- und Integration-Tests vorhanden (>80% Coverage)
- ✅ Dokumentation aktualisiert (README, QUICKSTART)
- ✅ Keine neuen externen Dependencies

---

## Nächste Schritte

1. Plan-Review mit Stakeholder
2. Bei Freigabe: Wechsel in Code-Modus
3. Implementierung gemäß Todo-Liste
4. Kontinuierliche Tests während Entwicklung
5. Finale Review vor Merge

---

**Erstellt:** 2026-06-25  
**Branch:** `feature/12-endpoint-alias-listing`  
**Issue:** #12