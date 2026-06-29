---
name: Feature Request - Hardcoded Values Detection
about: Scan-Tool zur Erkennung hardcodierter Werte in API-Konfigurationen
title: 'Feature: Hardcoded Values Detection Scanner'
labels: enhancement, security, api-scanning
assignees: ''
---

## Beschreibung

Implementierung eines Scan-Tools zur automatischen Erkennung von hardcodierten Werten (IP-Adressen, URLs, Hostnames, Ports) in API-Konfigurationen des IBM webMethods API Gateway.

### Beispiele für hardcodierte Werte

**1. Policy Expression mit hardcodierten IP-Adressen und Hostname:**

```
(HTTP.REQ.HOSTNAME.SET_TEXT_MODE(IGNORECASE).EQ("ods-prd-rest.oebb.at") && HTTP.REQ.URL.STARTSWITH("/ords")) && (CLIENT.IP.SRC.EQ(10.66.16.96).NOT && CLIENT.IP.SRC.EQ(10.66.16.97).NOT && CLIENT.IP.SRC.EQ(10.66.16.98).NOT && CLIENT.IP.SRC.EQ(10.66.16.99).NOT && CLIENT.IP.SRC.EQ(10.66.25.151).NOT)
```

Zu erkennende Werte:
- **Hostname**: `ods-prd-rest.oebb.at`
- **IPv4-Adressen**: `10.66.16.96`, `10.66.16.97`, `10.66.16.98`, `10.66.16.99`, `10.66.25.151`
- **URL-Path**: `/ords`

**2. Endpoint mit hardcodierter URL (statt Endpoint Alias):**

```json
{
  "apiDefinition": {
    "routing": {
      "endpointURI": "https://backend-prod.internal.company.com:8443/api/v1"
    }
  }
}
```

Zu erkennende Werte:
- **URL**: `https://backend-prod.internal.company.com:8443/api/v1`
- **Hostname**: `backend-prod.internal.company.com`
- **Port**: `8443`

> **Hinweis**: Best Practice wäre die Verwendung eines Endpoint Alias statt einer hardcodierten URL. Das Tool soll solche Fälle identifizieren.

## Anforderungen

### Funktionale Anforderungen

**1. Scan-Funktionalität**
- Alle APIs im Gateway scannen
- Hardcodierte Werte in folgenden Bereichen erkennen:
  - **HTTP Headers** (in Policy Expressions)
  - **API Endpoints** (URLs, Routing-Konfigurationen)
- Einmalige Scan-Operation per CLI-Aufruf

**2. Zu erkennende Werte**

| Typ | Beispiel | Beschreibung |
|-----|----------|--------------|
| **IPv4** | `CLIENT.IP.SRC.EQ(10.66.16.96)` | IPv4-Adressen in Policy Expressions |
| **IPv6** | `CLIENT.IP.SRC.EQ(2001:db8::1)` | IPv6-Adressen in Policy Expressions |
| **Hostnames** | `HTTP.REQ.HOSTNAME.EQ("ods-prd-rest.oebb.at")` | FQDNs in Hostname-Checks |
| **URLs** | `HTTP.REQ.URL.CONTAINS("https://api.example.com")` | Vollständige URLs |
| **Ports** | `HTTP.REQ.URL.CONTAINS(":8080")` | Port-Nummern in URLs |

**3. Ausgabe-Format**

Tabellarische Darstellung mit drei Spalten:

```
API NAME              LOCATION                       HARDCODED VALUE
ODS API v1            policies.0.expression          10.66.16.96
ODS API v1            policies.1.expression          ods-prd-rest.oebb.at
Payment API v2        endpoint.routing.url           https://backend.internal:8443
User Service v1       policies.2.expression          :8080
```

**Alternative JSON-Ausgabe** (via `--format json`):
```json
{
  "scannedAt": "2026-06-24T14:00:00Z",
  "totalAPIs": 15,
  "totalFindings": 23,
  "findings": [
    {
      "apiName": "ODS API v1",
      "apiVersion": "1.0",
      "location": "policies.0.expression",
      "type": "ipv4",
      "value": "10.66.16.96",
      "context": "CLIENT.IP.SRC.EQ(10.66.16.96)"
    },
    {
      "apiName": "ODS API v1",
      "apiVersion": "1.0",
      "location": "policies.1.expression",
      "type": "hostname",
      "value": "ods-prd-rest.oebb.at",
      "context": "HTTP.REQ.HOSTNAME.EQ(\"ods-prd-rest.oebb.at\")"
    }
  ]
}
```

### Technische Anforderungen

**1. CLI-Integration als separater Command**
```bash
agwctl scan hardcoded-values \
  --gateway-url=https://gateway.example.com:5555/rest/apigateway \
  --username=admin \
  --password=secret \
  --format=table
```

**Command-Line Flags:**
| Flag | Type | Default | Beschreibung |
|------|------|---------|--------------|
| `--gateway-url` | string | *required* | API Gateway base URL |
| `--username` | string | *required* | Basic auth username |
| `--password` | string | *required* | Basic auth password |
| `--format` | string | `table` | Ausgabeformat: `table` oder `json` |
| `--output` | string | - | Ausgabedatei (optional, sonst stdout) |
| `--log-level` | string | `info` | Log level: debug, info, warn, error |

**2. Code-Struktur**
```
internal/
├── detector/
│   ├── detector.go          # Hauptlogik für Detection
│   ├── detector_test.go     # Unit-Tests
│   ├── patterns.go          # Regex-Patterns für Value-Typen
│   └── parser.go            # JSON-Parser für API-Dokumente
├── models/
│   └── hardcoded.go         # NEU: Datenstrukturen für Findings
├── client/
│   └── client.go            # Bestehend: API-Zugriff
cmd/
└── agwctl/
    └── scan.go              # NEU: Scan-Subcommand
```

**3. Datenmodelle**
```go
// HardcodedValueType definiert den Typ des hardcodierten Werts
type HardcodedValueType string

const (
    HardcodedIPv4     HardcodedValueType = "ipv4"
    HardcodedIPv6     HardcodedValueType = "ipv6"
    HardcodedHostname HardcodedValueType = "hostname"
    HardcodedURL      HardcodedValueType = "url"
    HardcodedPort     HardcodedValueType = "port"
)

// Finding repräsentiert einen erkannten hardcodierten Wert
type Finding struct {
    APIName    string             `json:"apiName"`
    APIVersion string             `json:"apiVersion"`
    Location   string             `json:"location"`    // z.B. "policies.0.expression"
    Type       HardcodedValueType `json:"type"`
    Value      string             `json:"value"`
    Context    string             `json:"context"`     // Umgebender Code
}

// ScanResult ist das Gesamt-Ergebnis des Scans
type ScanResult struct {
    ScannedAt     time.Time `json:"scannedAt"`
    TotalAPIs     int       `json:"totalAPIs"`
    TotalFindings int       `json:"totalFindings"`
    Findings      []Finding `json:"findings"`
}
```

**4. Regex-Patterns**
```go
// internal/detector/patterns.go
var patterns = map[HardcodedValueType]*regexp.Regexp{
    // IPv4: Erkennt IP-Adressen in Policy Expressions
    HardcodedIPv4: regexp.MustCompile(
        `(?:CLIENT\.IP\.SRC\.EQ|IP\.SRC|IP\.DST|\.EQ)\s*\(\s*(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})\s*\)`,
    ),
    
    // IPv6: Erkennt IPv6-Adressen
    HardcodedIPv6: regexp.MustCompile(
        `(?:CLIENT\.IP\.SRC\.EQ|IP\.SRC|IP\.DST|\.EQ)\s*\(\s*([0-9a-fA-F:]+)\s*\)`,
    ),
    
    // Hostname: Erkennt Hostnames in Expressions
    HardcodedHostname: regexp.MustCompile(
        `(?:HOSTNAME|HOST|REQ\.HOSTNAME)\.(?:EQ|CONTAINS|STARTSWITH)\s*\(\s*["']([a-zA-Z0-9.-]+\.[a-zA-Z]{2,})["']\s*\)`,
    ),
    
    // URL: Erkennt vollständige URLs
    HardcodedURL: regexp.MustCompile(
        `(?:URL|URI)\.(?:EQ|CONTAINS|STARTSWITH)\s*\(\s*["'](https?://[^"']+)["']\s*\)`,
    ),
    
    // Port: Erkennt Port-Nummern
    HardcodedPort: regexp.MustCompile(
        `(?:PORT|:)(\d{2,5})`,
    ),
}
```

**5. Detector Implementation**
```go
// internal/detector/detector.go
type Detector struct {
    patterns map[HardcodedValueType]*regexp.Regexp
    logger   *slog.Logger
}

func NewDetector(logger *slog.Logger) *Detector {
    return &Detector{
        patterns: initPatterns(),
        logger:   logger,
    }
}

// DetectInAPI analysiert eine API-Definition und findet hardcodierte Werte
func (d *Detector) DetectInAPI(apiJSON []byte) ([]Finding, error) {
    var findings []Finding
    
    // 1. Policy-Expressions extrahieren und scannen
    policies := d.extractPolicies(apiJSON)
    for _, policy := range policies {
        values := d.detectInText(policy.Expression, policy.Location)
        findings = append(findings, values...)
    }
    
    // 2. Endpoints extrahieren und scannen
    endpoints := d.extractEndpoints(apiJSON)
    for _, endpoint := range endpoints {
        values := d.detectInText(endpoint.URL, endpoint.Location)
        findings = append(findings, values...)
    }
    
    return findings, nil
}
```

## Implementierungsschritte

- [ ] **Detector-Modul** (`internal/detector/`)
  - [ ] `detector.go`: Hauptlogik für Hardcoded-Value-Detection
  - [ ] `patterns.go`: Regex-Patterns für verschiedene Value-Typen
  - [ ] `parser.go`: JSON-Parser für API-Dokumente (gjson)
  - [ ] `detector_test.go`: Unit-Tests für alle Patterns

- [ ] **Datenmodelle** (`internal/models/hardcoded.go`)
  - [ ] `HardcodedValueType` Enum
  - [ ] `Finding` Struct
  - [ ] `ScanResult` Struct

- [ ] **Formatter** (`internal/detector/formatter.go`)
  - [ ] Tabellarische Ausgabe (ASCII-Table)
  - [ ] JSON-Ausgabe
  - [ ] Zusammenfassungs-Statistiken

- [ ] **CLI-Command** (`cmd/agwctl/scan.go`)
  - [ ] Subcommand `agwctl scan hardcoded-values`
  - [ ] Flag-Parsing
  - [ ] API-Abruf über bestehenden Client
  - [ ] Orchestrierung: Scan → Format → Output

- [ ] **Tests**
  - [ ] Unit-Tests für Regex-Patterns
  - [ ] Unit-Tests für JSON-Parser
  - [ ] Integration-Tests mit Mock-APIs
  - [ ] False-Positive-Tests

- [ ] **Dokumentation**
  - [ ] README.md aktualisieren
  - [ ] Beispiele für `agwctl scan hardcoded-values`
  - [ ] QUICKSTART.md erweitern

## Akzeptanzkriterien

- [ ] CLI-Command `agwctl scan hardcoded-values` funktioniert
- [ ] Alle definierten Value-Typen werden erkannt (IPv4, IPv6, Hostname, URL, Port)
- [ ] Scan erfolgt in Policy Expressions und Endpoints
- [ ] Ausgabe in Table- und JSON-Format möglich
- [ ] Ausgabe enthält: API-Name, Location, gefundener Wert
- [ ] Keine False-Positives bei gültigen Variablen/Platzhaltern
- [ ] Unit-Tests für alle Regex-Patterns vorhanden
- [ ] Integration-Tests mit Beispiel-APIs vorhanden
- [ ] Dokumentation aktualisiert

## Use Cases

1. **Security-Audit**: Identifizierung von hardcodierten Produktions-IPs vor Go-Live
2. **Migration-Vorbereitung**: Finden aller hardcodierten Werte vor Umgebungswechsel
3. **Best-Practice-Check**: Regelmäßige Überprüfung auf hardcodierte Werte
4. **Compliance**: Nachweis, dass keine sensiblen Werte hardcodiert sind

## Beispiel-Output

### Table-Format
```
╔═══════════════════╦════════════════════════════╦═══════════════════════════╗
║ API NAME          ║ LOCATION                   ║ HARDCODED VALUE           ║
╠═══════════════════╬════════════════════════════╬═══════════════════════════╣
║ ODS API v1        ║ policies.0.expression      ║ 10.66.16.96               ║
║ ODS API v1        ║ policies.1.expression      ║ ods-prd-rest.oebb.at      ║
║ Payment API v2    ║ endpoint.routing.url       ║ https://backend:8443      ║
╚═══════════════════╩════════════════════════════╩═══════════════════════════╝

Summary: 3 findings in 2 APIs
  - IPv4: 1
  - Hostname: 1
  - URL: 1
```

## Referenzen

- [gjson - JSON Parser](https://github.com/tidwall/gjson)
- [Go regexp Package](https://pkg.go.dev/regexp)
- Bestehende Implementierung: `internal/monitor/processor.go` (JSON-Parsing mit gjson)

## Priorität
High

## Geschätzter Aufwand
3-4 Tage