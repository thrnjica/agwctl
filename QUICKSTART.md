# Quick Start Guide

This guide will help you get started with the API Gateway Automator CLI in minutes.

## Prerequisites

- Go 1.26.4 or later installed
- Access to IBM webMethods API Gateway 10.15
- Valid credentials with "Manage APIs" privilege

## Installation

### Option 1: Build from Source

```bash
# Clone the repository
git clone https://github.com/thrnjica/agwctl.git
cd agwctl

# Download dependencies
go mod download

# Build the binary
go build -o agwctl ./cmd/agwctl

# Verify installation
./agwctl --help
```

### Option 2: Install to GOPATH

```bash
go install github.com/thrnjica/agwctl/cmd/agwctl@latest
```

## Basic Usage

### 1. Test Connection (Dry Run)

First, verify you can connect to the API Gateway:

```bash
./agwctl \
  --gateway-url=https://your-gateway.example.com:5555/rest/apigateway \
  --username=your-username \
  --password=your-password \
  --teams="YourTeamName" \
  --dry-run \
  --log-level=debug
```

**What this does:**
- Connects to the API Gateway
- Fetches all access profiles (teams)
- Polls for APIs
- Simulates adding teams (without actually modifying anything)
- Shows detailed debug logs

### 2. Run in Production

Once you've verified the connection works:

```bash
./agwctl \
  --gateway-url=https://your-gateway.example.com:5555/rest/apigateway \
  --username=your-username \
  --password=your-password \
  --teams="Team1,Team2,Team3" \
  --interval=60 \
  --log-level=info
```

**What this does:**
- Polls every 60 seconds
- Detects new APIs
- Automatically adds Team1, Team2, and Team3 to new APIs
- Logs operations in JSON format

### 3. Run as Background Service

```bash
# Using nohup
nohup ./agwctl \
  --gateway-url=https://your-gateway.example.com:5555/rest/apigateway \
  --username=automation-user \
  --password="${GATEWAY_PASSWORD}" \
  --teams="ProductionTeam" \
  --interval=300 \
  > agwctl.log 2>&1 &

# Save the PID
echo $! > agwctl.pid

# To stop later
kill $(cat agwctl.pid)
```

## Common Scenarios

### Scenario 1: Development Environment

```bash
./agwctl \
  --gateway-url=https://gateway-dev.example.com:5555/rest/apigateway \
  --username=admin \
  --password=admin \
  --teams="DevTeam" \
  --interval=30 \
  --dry-run \
  --log-level=debug
```

### Scenario 2: Production with Multiple Teams

```bash
./agwctl \
  --gateway-url=https://gateway-prod.example.com:5555/rest/apigateway \
  --username=automation-user \
  --password="${GATEWAY_PASSWORD}" \
  --teams="SecurityTeam,ComplianceTeam,OpsTeam" \
  --interval=300 \
  --rate-limit=5 \
  --log-level=info
```

### Scenario 3: High-Volume Environment

```bash
./agwctl \
  --gateway-url=https://gateway.example.com:5555/rest/apigateway \
  --username=automation-user \
  --password="${GATEWAY_PASSWORD}" \
  --teams="Team1,Team2" \
  --interval=120 \
  --page-size=200 \
  --rate-limit=15 \
  --db-path=/var/lib/agwctl/db \
  --log-level=info
```

## Understanding the Output

### Successful Startup

```json
{"time":"2026-06-16T17:00:00Z","level":"INFO","msg":"Starting API Gateway Automator","gateway_url":"https://gateway.example.com:5555/rest/apigateway","username":"admin","teams":["DevTeam"],"interval":"60s","page_size":100,"rate_limit":10,"db_path":"data","dry_run":false}
{"time":"2026-06-16T17:00:00Z","level":"INFO","msg":"Database opened","path":"data"}
{"time":"2026-06-16T17:00:01Z","level":"INFO","msg":"Refreshing access profiles cache"}
{"time":"2026-06-16T17:00:02Z","level":"INFO","msg":"Access profiles cache refreshed","count":5}
{"time":"2026-06-16T17:00:02Z","level":"INFO","msg":"Team names resolved","teams":["DevTeam"],"team_ids":["abc-123"]}
{"time":"2026-06-16T17:00:02Z","level":"INFO","msg":"Database stats","stats":{"processed_apis_count":0}}
{"time":"2026-06-16T17:00:02Z","level":"INFO","msg":"Starting polling loop"}
```

### Poll Cycle

```json
{"time":"2026-06-16T17:00:02Z","level":"INFO","msg":"Starting poll cycle"}
{"time":"2026-06-16T17:00:15Z","level":"INFO","msg":"Fetched all APIs","total":2847,"duration_ms":13245}
{"time":"2026-06-16T17:00:15Z","level":"INFO","msg":"New APIs detected","count":3}
{"time":"2026-06-16T17:00:16Z","level":"INFO","msg":"Processing new API","api_id":"api-123"}
{"time":"2026-06-16T17:00:17Z","level":"INFO","msg":"API metadata extracted","api_id":"api-123","name":"PetStore API","version":"1.0","type":"REST","existing_teams":0}
{"time":"2026-06-16T17:00:18Z","level":"INFO","msg":"API updated successfully","api_id":"api-123","teams_added":1}
{"time":"2026-06-16T17:00:30Z","level":"INFO","msg":"Poll complete","processed":3,"failed":0,"duration_ms":28456}
```

## Troubleshooting

### Issue: "teams not found"

```
Error: resolve team names: teams not found: [TeamName]
```

**Solution:** The team name doesn't exist in the API Gateway. Check available teams:

1. Log in to API Gateway UI
2. Go to User Management → Access Profiles
3. Note the exact team name (case-sensitive)

### Issue: "HTTP 401: Unauthorized"

```
Error: HTTP 401: Unauthorized
```

**Solution:**
- Verify username and password are correct
- Ensure user has "Manage APIs" privilege
- Check if user account is active

### Issue: "HTTP 429: Too Many Requests"

```
Error: HTTP 429: Too Many Requests
```

**Solution:** Reduce the rate limit:

```bash
./agwctl --rate-limit=5 [other flags...]
```

### Issue: Database corruption

```
Error: open database: corrupted
```

**Solution:** Delete and recreate the database:

```bash
rm -rf data
./agwctl [flags...]
```

## Best Practices

### 1. Use Environment Variables for Credentials

```bash
export GATEWAY_URL="https://gateway.example.com:5555/rest/apigateway"
export GATEWAY_USERNAME="automation-user"
export GATEWAY_PASSWORD="secret"

./agwctl \
  --gateway-url="${GATEWAY_URL}" \
  --username="${GATEWAY_USERNAME}" \
  --password="${GATEWAY_PASSWORD}" \
  --teams="Team1,Team2"
```

### 2. Monitor Logs

```bash
# Tail logs in real-time
./agwctl [flags...] 2>&1 | tee -a agwctl.log

# Parse JSON logs with jq
tail -f agwctl.log | jq -r '.msg'

# Filter for errors
tail -f agwctl.log | jq 'select(.level=="ERROR")'
```

### 3. Run as Systemd Service (Linux)

Create `/etc/systemd/system/agwctl.service`:

```ini
[Unit]
Description=API Gateway Automator
After=network.target

[Service]
Type=simple
User=agwctl
WorkingDirectory=/opt/agwctl
Environment="GATEWAY_PASSWORD=secret"
ExecStart=/opt/agwctl/agwctl \
  --gateway-url=https://gateway.example.com:5555/rest/apigateway \
  --username=automation-user \
  --password=${GATEWAY_PASSWORD} \
  --teams=Team1,Team2 \
  --interval=300 \
  --db-path=/var/lib/agwctl/db \
  --log-level=info
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable agwctl
sudo systemctl start agwctl
sudo systemctl status agwctl
```

### 4. Health Monitoring

Monitor the process:

```bash
# Check if running
ps aux | grep agwctl

# Monitor resource usage
top -p $(pgrep agwctl)

# Check database size
du -sh data
```

## Next Steps

- Read the full [README.md](README.md) for detailed documentation
- Review [DESIGN.md](DESIGN.md) for architecture details
- Check [DEPENDENCIES.md](DEPENDENCIES.md) for dependency information
- Explore the source code in `internal/` for customization

## Getting Help

If you encounter issues:

1. Enable debug logging: `--log-level=debug`
2. Check the logs for error messages
3. Verify API Gateway connectivity
4. Review the troubleshooting section above
5. Open an issue on GitHub with logs and configuration

## Example: Complete Production Setup

```bash
#!/bin/bash
# production-setup.sh

# Configuration
GATEWAY_URL="https://gateway.prod.example.com:5555/rest/apigateway"
GATEWAY_USERNAME="automation-user"
GATEWAY_PASSWORD="${GATEWAY_PASSWORD:-}"  # Set via environment
TEAMS="SecurityTeam,ComplianceTeam,OpsTeam"
INTERVAL=300
RATE_LIMIT=10
DB_PATH="/var/lib/agwctl/db"
LOG_FILE="/var/log/agwctl/agwctl.log"

# Validate
if [ -z "$GATEWAY_PASSWORD" ]; then
    echo "Error: GATEWAY_PASSWORD environment variable not set"
    exit 1
fi

# Create directories
mkdir -p "$(dirname "$DB_PATH")"
mkdir -p "$(dirname "$LOG_FILE")"

# Run
./agwctl \
  --gateway-url="$GATEWAY_URL" \
  --username="$GATEWAY_USERNAME" \
  --password="$GATEWAY_PASSWORD" \
  --teams="$TEAMS" \
  --interval="$INTERVAL" \
  --rate-limit="$RATE_LIMIT" \
  --db-path="$DB_PATH" \
  --log-level=info \
  2>&1 | tee -a "$LOG_FILE"
```

Make it executable and run:

```bash
chmod +x production-setup.sh
export GATEWAY_PASSWORD="your-secret-password"
./production-setup.sh
