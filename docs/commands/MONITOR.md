# Monitor Command

Der `monitor` Befehl überwacht das IBM webMethods API Gateway kontinuierlich auf neu erstellte APIs und fügt automatisch konfigurierte Teams hinzu.

## Übersicht

Der Monitor-Befehl ist die Hauptfunktion von `agwctl`. Er läuft als Daemon und prüft in regelmäßigen Abständen, ob neue APIs im Gateway erstellt wurden. Für jede neue API werden automatisch die konfigurierten Teams (Access Profiles) hinzugefügt.

## Verwendung

```bash
agwctl [flags]
```

oder explizit:

```bash
agwctl monitor [flags]
```

## Flags

| Flag | Typ | Standard | Beschreibung |
|------|-----|----------|--------------|
| `--gateway-url` | string | *erforderlich* | API Gateway Basis-URL |
| `--username` | string | *erforderlich* | Basic Auth Benutzername |
| `--password` | string | *erforderlich* | Basic Auth Passwort |
| `--teams` | string | *erforderlich* | Komma-getrennte Team-Namen |
| `--interval` | int | `60` | Polling-Intervall in Sekunden |
| `--page-size` | int | `100` | Anzahl APIs pro Seite |
| `--rate-limit` | int | `10` | Maximale Anfragen pro Sekunde |
| `--db-path` | string | `data` | Pfad zum NutsDB-Datenbankverzeichnis |
| `--log-level` | string | `info` | Log-Level: debug, info, warn, error |
| `--dry-run` | bool | `false` | Simulation ohne Änderungen |

## Beispiele

### Minimal-Konfiguration

```bash
agwctl \
  --gateway-url=https://gateway.example.com:5555/rest/apigateway \
  --username=admin \
  --password=secret \
  --teams="DevTeam,QATeam"
```

### Produktions-Deployment

```bash
agwctl \
  --gateway-url=https://gateway.prod.example.com:5555/rest/apigateway \
  --username=automation-user \
  --password="${GATEWAY_PASSWORD}" \
  --teams="ProductionTeam,SecurityTeam" \
  --interval=300 \
  --rate-limit=5 \
  --log-level=info
```

### Entwicklung mit Dry-Run

```bash
agwctl \
  --gateway-url=https://gateway.dev.example.com:5555/rest/apigateway \
  --username=admin \
  --password=admin \
  --teams="IBM_Support" \
  --dry-run \
  --log-level=debug
```

### High-Volume-Umgebung

```bash
agwctl \
  --gateway-url=https://gateway.example.com:5555/rest/apigateway \
  --username=admin \
  --password="${GATEWAY_PASSWORD}" \
  --teams="Team1,Team2,Team3" \
  --interval=120 \
  --page-size=200 \
  --rate-limit=15 \
  --db-path=/var/lib/agwctl/db
```

## Funktionsweise

### 1. Initialisierung

- Verbindung zum API Gateway herstellen
- Alle Access Profiles (Teams) abrufen und Name-zu-ID-Mapping erstellen
- Lokale NutsDB-Datenbank für State-Tracking öffnen
- Konfigurierte Teams validieren

### 2. Polling-Schleife

```
┌─────────────────────────────────────────────────────────┐
│ 1. Alle API-IDs mit Pagination abrufen                  │
│    (respektiert Rate-Limits)                            │
└─────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────┐
│ 2. Datenbank abfragen: Welche APIs sind neu?           │
│    (noch nicht verarbeitet)                             │
└─────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────┐
│ 3. Für jede neue API:                                   │
│    - Vollständiges API-Dokument abrufen                 │
│    - Bestehende Teams mit gjson extrahieren             │
│    - Ziel-Teams hinzufügen (Duplikate vermeiden)        │
│    - API mit sjson aktualisieren                        │
│    - Als verarbeitet in Datenbank markieren             │
└─────────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────┐
│ 4. Warten bis zum nächsten Intervall                    │
└─────────────────────────────────────────────────────────┘
```

### 3. Graceful Shutdown

- SIGINT/SIGTERM-Signale abfangen
- Aktuellen Poll-Zyklus abschließen
- State speichern und Datenbank schließen
- Sauber beenden

## State-Management

### NutsDB-Datenbank

Die Anwendung verwendet NutsDB (embedded Key-Value-Datenbank) zum Tracking verarbeiteter APIs:

**Speicherort**: `data/` (konfigurierbar mit `--db-path`)

**Buckets**:
- `processed_apis`: Speichert API-ID → Metadaten-Mappings
- `metadata`: Speichert letzten Poll-Zeitstempel

**Vorteile**:
- Schnelle O(1)-Lookups
- ACID-Garantien
- Keine externe Datenbank erforderlich
- Automatische Kompaktierung

### Datenbank-Struktur

```
data/
├── 000001.dat          # Datendatei
├── 000001.idx          # Index-Datei
└── LOCK                # Lock-Datei
```

## Performance

### Typische Performance-Metriken

| Metrik | Wert |
|--------|------|
| **Pagination** | ~30 Sekunden für 3000 APIs bei 10 req/sec |
| **Verarbeitung** | 5-10 Sekunden pro neuer API |
| **Speicher** | <100MB für 3000 APIs |
| **Gesamt-Poll-Zyklus** | <2 Minuten für typische Workload |

### Optimierungs-Tipps

1. **Rate-Limit anpassen**
   ```bash
   # Erhöhen, wenn Gateway es verarbeiten kann
   --rate-limit=20
   ```

2. **Page-Size optimieren**
   ```bash
   # Größere Seiten = weniger Requests, aber mehr Speicher
   --page-size=200
   ```

3. **Intervall erhöhen**
   ```bash
   # Weniger häufiges Polling, wenn APIs selten erstellt werden
   --interval=300
   ```

4. **Logs überwachen**
   ```bash
   # Debug-Modus zur Identifikation von Bottlenecks
   --log-level=debug
   ```

## Logging

### Strukturierte JSON-Logs

Die Anwendung verwendet strukturiertes JSON-Logging für einfaches Parsing und Monitoring:

```json
{"time":"2026-06-16T17:00:00Z","level":"INFO","msg":"Starting API Gateway monitor","interval":"60s","teams":["DevTeam","QATeam"]}
{"time":"2026-06-16T17:00:15Z","level":"INFO","msg":"Pagination complete","total_apis":2847,"pages":29,"duration":"14.2s"}
{"time":"2026-06-16T17:00:15Z","level":"INFO","msg":"New APIs detected","count":5}
{"time":"2026-06-16T17:00:16Z","level":"INFO","msg":"Processing API","api_id":"abc-123","name":"PetStore API"}
{"time":"2026-06-16T17:00:30Z","level":"INFO","msg":"Poll complete","processed":5,"errors":0,"duration":"29.5s"}
```

### Log-Levels

- **debug**: Detaillierte Informationen für Troubleshooting
- **info**: Allgemeine Betriebsmeldungen
- **warn**: Warnmeldungen (nicht-kritische Probleme)
- **error**: Fehlermeldungen (Fehler)

### Log-Parsing-Beispiele

**Anzahl verarbeiteter APIs pro Stunde**:
```bash
grep "Poll complete" agwctl.log | \
  jq -r '.processed' | \
  awk '{sum+=$1} END {print sum}'
```

**Durchschnittliche Poll-Dauer**:
```bash
grep "Poll complete" agwctl.log | \
  jq -r '.duration' | \
  sed 's/s$//' | \
  awk '{sum+=$1; count++} END {print sum/count "s"}'
```

## Fehlerbehandlung

### Authentifizierungsfehler

```
Error: HTTP 401: Unauthorized
```

**Lösung**: Benutzername und Passwort überprüfen. Benutzer benötigt "Manage APIs"-Berechtigung.

### Rate-Limiting

```
Error: HTTP 429: Too Many Requests
```

**Lösung**: `--rate-limit` reduzieren oder `--interval` erhöhen.

### Team nicht gefunden

```
Error: teams not found: [TeamName]
```

**Lösung**: Team-Namen im Gateway überprüfen. `/accessProfiles`-Endpoint prüfen.

### Datenbank-Korruption

```
Error: open database: corrupted
```

**Lösung**: Datenbankverzeichnis löschen und neu starten:
```bash
rm -rf data
agwctl [flags...]
```

### Netzwerk-Timeouts

```
Error: context deadline exceeded
```

**Lösung**: 
- Netzwerkverbindung prüfen
- Gateway-Verfügbarkeit prüfen
- `--rate-limit` reduzieren

## Dry-Run-Modus

Der Dry-Run-Modus simuliert alle Operationen ohne tatsächliche Änderungen:

```bash
agwctl --dry-run [other flags...]
```

**Was passiert im Dry-Run**:
- ✅ APIs werden abgerufen
- ✅ Neue APIs werden identifiziert
- ✅ Team-Zuweisungen werden berechnet
- ❌ Keine API-Updates werden durchgeführt
- ✅ Logs zeigen, was passieren würde

**Verwendung**:
- Konfiguration testen
- Auswirkungen vor Produktions-Deployment prüfen
- Debugging

## Sicherheitshinweise

### Credentials

**Niemals Credentials in Version Control committen**:

```bash
# ✅ Gut: Umgebungsvariablen verwenden
export GATEWAY_PASSWORD="secret"
agwctl --password="${GATEWAY_PASSWORD}" [other flags...]

# ❌ Schlecht: Passwort im Klartext
agwctl --password="secret" [other flags...]
```

**Secrets Manager in Produktion verwenden**:
- AWS Secrets Manager
- HashiCorp Vault
- Azure Key Vault

### HTTPS

**Immer HTTPS in Produktion verwenden**:

```bash
# ✅ Sicher
--gateway-url=https://gateway.example.com:5555/rest/apigateway

# ❌ Unsicher
--gateway-url=http://gateway.example.com:5555/rest/apigateway
```

### Datenbank-Sicherheit

- Datenbank an sicherem Ort mit entsprechenden Berechtigungen speichern
- Datenbank enthält API-Metadaten, aber keine Credentials
- Regelmäßige Backups erstellen

### Berechtigungen

Der Benutzer benötigt folgende Berechtigungen:
- **Manage APIs**: Zum Lesen und Aktualisieren von APIs
- **Manage Access Profiles**: Zum Lesen von Teams

## Monitoring und Alerting

### Prometheus-Metriken

Logs können in Prometheus-Metriken konvertiert werden:

```bash
# Beispiel: mtail-Konfiguration
counter api_processed_total by status
/Poll complete/ {
  api_processed_total["success"] += $processed
}
```

### Health-Checks

```bash
#!/bin/bash
# Prüfe, ob agwctl läuft und erfolgreich pollt
LAST_POLL=$(grep "Poll complete" agwctl.log | tail -1 | jq -r '.time')
CURRENT_TIME=$(date -u +%s)
LAST_POLL_TIME=$(date -d "$LAST_POLL" +%s)
DIFF=$((CURRENT_TIME - LAST_POLL_TIME))

if [ $DIFF -gt 300 ]; then
  echo "ERROR: Last poll was $DIFF seconds ago"
  exit 1
fi
```

### Alerting-Regeln

**Beispiel: Prometheus Alert**:
```yaml
- alert: AgwctlNotPolling
  expr: time() - agwctl_last_poll_timestamp > 300
  for: 5m
  annotations:
    summary: "agwctl has not polled in 5 minutes"
```

## Deployment

### Systemd Service

```ini
[Unit]
Description=API Gateway Automator
After=network.target

[Service]
Type=simple
User=agwctl
WorkingDirectory=/opt/agwctl
Environment="GATEWAY_PASSWORD=secret"
ExecStart=/usr/local/bin/agwctl \
  --gateway-url=https://gateway.example.com:5555/rest/apigateway \
  --username=automation-user \
  --password=${GATEWAY_PASSWORD} \
  --teams="ProductionTeam,SecurityTeam" \
  --db-path=/var/lib/agwctl/db \
  --log-level=info
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

### Docker

```dockerfile
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY . .
RUN make build

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/agwctl /usr/local/bin/
ENTRYPOINT ["agwctl"]
```

```bash
docker run -d \
  --name agwctl \
  -e GATEWAY_PASSWORD=secret \
  -v /var/lib/agwctl:/data \
  agwctl:latest \
  --gateway-url=https://gateway.example.com:5555/rest/apigateway \
  --username=admin \
  --password="${GATEWAY_PASSWORD}" \
  --teams="DevTeam,QATeam" \
  --db-path=/data
```

## Siehe auch

- [Hauptdokumentation](../../README.md)
- [Schnellstart-Anleitung](../QUICKSTART.md)
- [Design-Dokumentation](../DESIGN.md)
- [Aliases Command](ALIASES.md)