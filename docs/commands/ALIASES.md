# Aliases Command

Der `aliases` Befehl listet alle Endpoint-Aliase aus dem IBM webMethods API Gateway auf und löst optional deren IP-Adressen auf.

## Übersicht

Endpoint-Aliase sind Gateway-Konfigurationsobjekte, die Backend-URLs für APIs definieren. Dieser Befehl ermöglicht es, alle konfigurierten Aliase zu inspizieren und deren DNS-Auflösung zu überprüfen.

## Verwendung

```bash
agwctl aliases list [flags]
```

## Flags

| Flag | Typ | Standard | Beschreibung |
|------|-----|----------|--------------|
| `--gateway-url` | string | *erforderlich* | API Gateway Basis-URL |
| `--username` | string | *erforderlich* | Basic Auth Benutzername |
| `--password` | string | *erforderlich* | Basic Auth Passwort |
| `--format` | string | `table` | Ausgabeformat: `table` oder `json` |
| `--timeout` | int | `60` | DNS-Lookup-Timeout in Sekunden |
| `--skip-dns-resolution` | bool | `false` | DNS-Auflösung überspringen (schneller) |
| `--rate-limit` | int | `10` | Maximale Anfragen pro Sekunde |
| `--log-level` | string | `info` | Log-Level: debug, info, warn, error |

## Beispiele

### Basis-Verwendung mit Tabellen-Ausgabe

```bash
agwctl aliases list \
  --gateway-url=https://gateway.example.com:5555/rest/apigateway \
  --username=admin \
  --password=secret
```

**Ausgabe:**
```
ALIAS NAME                     ENDPOINT URL                                       IP ADDRESSES                            
----------------------------------------------------------------------------------------------------------------------------
production-backend             https://api.prod.example.com/v1                    203.0.113.10, 203.0.113.11             
staging-backend                https://api.staging.example.com/v1                 198.51.100.5                           
legacy-system                  http://legacy.internal.local:8080/api              192.168.1.100                          
external-partner               https://partner-api.external.com/gateway           <DNS lookup failed>                    
```

### JSON-Ausgabe für Automatisierung

```bash
agwctl aliases list \
  --gateway-url=https://gateway.example.com:5555/rest/apigateway \
  --username=admin \
  --password=secret \
  --format=json
```

**Ausgabe:**
```json
{
  "aliases": [
    {
      "name": "production-backend",
      "endpointUrl": "https://api.prod.example.com/v1",
      "hostname": "api.prod.example.com",
      "ipAddresses": [
        "203.0.113.10",
        "203.0.113.11"
      ],
      "resolved": true
    },
    {
      "name": "staging-backend",
      "endpointUrl": "https://api.staging.example.com/v1",
      "hostname": "api.staging.example.com",
      "ipAddresses": [
        "198.51.100.5"
      ],
      "resolved": true
    },
    {
      "name": "external-partner",
      "endpointUrl": "https://partner-api.external.com/gateway",
      "hostname": "partner-api.external.com",
      "ipAddresses": null,
      "resolved": false,
      "error": "lookup partner-api.external.com: no such host"
    }
  ]
}
```

### Schnelle Auflistung ohne DNS-Auflösung

Wenn Sie nur die konfigurierten Aliase sehen möchten, ohne DNS-Lookups durchzuführen:

```bash
agwctl aliases list \
  --gateway-url=https://gateway.example.com:5555/rest/apigateway \
  --username=admin \
  --password=secret \
  --skip-dns-resolution
```

**Ausgabe:**
```
ALIAS NAME                     ENDPOINT URL                                       IP ADDRESSES                            
----------------------------------------------------------------------------------------------------------------------------
production-backend             https://api.prod.example.com/v1                    <skipped>                              
staging-backend                https://api.staging.example.com/v1                 <skipped>                              
legacy-system                  http://legacy.internal.local:8080/api              <skipped>                              
```

### Mit erhöhtem Timeout für langsame DNS-Server

```bash
agwctl aliases list \
  --gateway-url=https://gateway.example.com:5555/rest/apigateway \
  --username=admin \
  --password=secret \
  --timeout=120
```

### Debug-Modus für Fehlersuche

```bash
agwctl aliases list \
  --gateway-url=https://gateway.example.com:5555/rest/apigateway \
  --username=admin \
  --password=secret \
  --log-level=debug
```

## Funktionsweise

1. **API-Abfrage**: Ruft alle Aliase vom Gateway-Endpoint `/alias` ab
2. **Filterung**: Filtert nur Aliase mit `type: "endpoint"` (case-insensitive)
3. **Hostname-Extraktion**: Extrahiert Hostname aus der Endpoint-URL
4. **DNS-Auflösung**: Löst Hostname zu IP-Adressen auf (optional)
5. **Formatierung**: Gibt Ergebnisse in Tabellen- oder JSON-Format aus

## Alias-Typen

Das Gateway unterstützt verschiedene Alias-Typen. Dieser Befehl zeigt nur **Endpoint-Aliase**:

- ✅ **endpoint**: Backend-URLs für APIs (werden angezeigt)
- ❌ **simple**: Einfache Aliase (werden gefiltert)
- ❌ **routing**: Routing-Regeln (werden gefiltert)

## DNS-Auflösung

### Erfolgreiche Auflösung
```
production-backend    https://api.example.com/v1    203.0.113.10, 203.0.113.11
```

### Fehlgeschlagene Auflösung
```
invalid-backend       https://nonexistent.local/api    <DNS lookup failed>
```

### Übersprungene Auflösung
```
production-backend    https://api.example.com/v1    <skipped>
```

## Performance-Hinweise

### DNS-Auflösung kann langsam sein

- **Mit DNS**: ~1-2 Sekunden pro Alias (abhängig von DNS-Server)
- **Ohne DNS**: ~100ms für alle Aliase

**Empfehlung**: Verwenden Sie `--skip-dns-resolution` für schnelle Übersichten.

### Rate Limiting

Der Befehl respektiert das konfigurierte Rate-Limit:

```bash
# Langsamer, aber sicherer für produktive Systeme
--rate-limit=5

# Schneller für Entwicklungsumgebungen
--rate-limit=20
```

## Fehlerbehandlung

### Authentifizierungsfehler

```
Error: HTTP 401: Unauthorized
```

**Lösung**: Überprüfen Sie Benutzername und Passwort.

### Verbindungsfehler

```
Error: dial tcp: connection refused
```

**Lösung**: Überprüfen Sie die Gateway-URL und Netzwerkverbindung.

### DNS-Timeout

```
Error: lookup api.example.com: i/o timeout
```

**Lösung**: Erhöhen Sie `--timeout` oder verwenden Sie `--skip-dns-resolution`.

### Keine Aliase gefunden

```
ALIAS NAME    ENDPOINT URL    IP ADDRESSES
----------------------------------------
```

**Mögliche Ursachen**:
- Keine Endpoint-Aliase konfiguriert
- Benutzer hat keine Berechtigung zum Lesen von Aliasen
- Alle Aliase haben einen anderen Typ als "endpoint"

## Ausgabeformate

### Tabellen-Format (Standard)

- **Vorteile**: Menschenlesbar, übersichtlich
- **Nachteile**: Nicht für Automatisierung geeignet
- **Verwendung**: Manuelle Inspektion, Debugging

### JSON-Format

- **Vorteile**: Maschinenlesbar, vollständige Informationen
- **Nachteile**: Weniger übersichtlich für Menschen
- **Verwendung**: Automatisierung, Scripting, Integration

**Beispiel-Script**:
```bash
#!/bin/bash
# Extrahiere alle IP-Adressen
agwctl aliases list --format=json | \
  jq -r '.aliases[].ipAddresses[]' | \
  sort -u
```

## Sicherheitshinweise

### Credentials

Verwenden Sie Umgebungsvariablen für Passwörter:

```bash
export GATEWAY_PASSWORD="secret"
agwctl aliases list \
  --gateway-url=https://gateway.example.com:5555/rest/apigateway \
  --username=admin \
  --password="${GATEWAY_PASSWORD}"
```

### HTTPS

Verwenden Sie immer HTTPS in Produktionsumgebungen:

```bash
# ✅ Gut
--gateway-url=https://gateway.example.com:5555/rest/apigateway

# ❌ Unsicher
--gateway-url=http://gateway.example.com:5555/rest/apigateway
```

## Anwendungsfälle

### 1. Netzwerk-Diagnose

Überprüfen Sie, ob alle Backend-Systeme erreichbar sind:

```bash
agwctl aliases list --format=json | \
  jq -r '.aliases[] | select(.resolved == false) | .name'
```

### 2. IP-Inventar

Erstellen Sie eine Liste aller Backend-IP-Adressen:

```bash
agwctl aliases list --format=json | \
  jq -r '.aliases[].ipAddresses[]' | sort -u > backend-ips.txt
```

### 3. Konfigurationsaudit

Exportieren Sie alle Alias-Konfigurationen:

```bash
agwctl aliases list --format=json > alias-backup.json
```

### 4. Monitoring-Integration

Integrieren Sie in Monitoring-Systeme:

```bash
#!/bin/bash
# Prüfe auf fehlgeschlagene DNS-Auflösungen
FAILED=$(agwctl aliases list --format=json | \
  jq '[.aliases[] | select(.resolved == false)] | length')

if [ "$FAILED" -gt 0 ]; then
  echo "WARNING: $FAILED aliases have DNS resolution failures"
  exit 1
fi
```

## Technische Details

### API-Endpoint

```
GET /rest/apigateway/alias
```

### Response-Struktur

```json
{
  "alias": [
    {
      "id": "abc-123",
      "name": "production-backend",
      "type": "endpoint",
      "endPointURI": "https://api.example.com/v1",
      "description": "Production backend system",
      "connectionTimeout": 30000,
      "readTimeout": 60000
    }
  ]
}
```

### Feldnamen-Besonderheiten

⚠️ **Wichtig**: Das API-Feld heißt `endPointURI` (mit großem P), nicht `endpointURI`.

## Siehe auch

- [API Gateway REST API Dokumentation](https://documentation.softwareag.com/webmethods/api_gateway/yai10-15/webhelp/yai-webhelp/index.html#page/yai-webhelp%2Fco-rest_api.html)
- [Hauptdokumentation](../../README.md)
- [Schnellstart-Anleitung](../QUICKSTART.md)
- [Design-Dokumentation](../DESIGN.md)