# API Gateway Automator (agwctl)

A Go CLI tool that monitors the IBM webMethods API Gateway 10.15 for newly created APIs and automatically adds specified teams to them.

## Features

- **Automatic Team Assignment**: Monitors for new APIs and automatically adds configured teams
- **Pagination Support**: Efficiently handles large deployments (2000-3000+ APIs)
- **Rate Limiting**: Prevents 429 errors with configurable request rate limiting
- **Persistent State**: Uses embedded NutsDB for fast, reliable state tracking
- **Efficient JSON Processing**: Uses gjson/sjson for minimal overhead
- **Structured Logging**: JSON-formatted logs with configurable levels
- **Graceful Shutdown**: Handles SIGINT/SIGTERM signals cleanly
- **Dry-Run Mode**: Test without making actual changes

## Installation

### Prerequisites

- Go 1.26.4 or later
- Access to IBM webMethods API Gateway 10.15

### Build from Source

```bash
git clone https://github.com/thrnjica/agwctl.git
cd agwctl
go mod download
go build -o agwctl ./cmd/agwctl
```

### Install

```bash
# Install to $GOPATH/bin
go install ./cmd/agwctl

# Or copy the binary to your PATH
sudo cp agwctl /usr/local/bin/
```

## Usage

### Basic Usage

```bash
agwctl \
  --gateway-url=https://gateway.example.com:5555/rest/apigateway \
  --username=admin \
  --password=secret \
  --teams="DevTeam,QATeam"
```

### Command-Line Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--gateway-url` | string | *required* | API Gateway base URL |
| `--username` | string | *required* | Basic auth username |
| `--password` | string | *required* | Basic auth password |
| `--teams` | string | *required* | Comma-separated team names to add |
| `--interval` | int | 60 | Polling interval in seconds |
| `--page-size` | int | 100 | Number of APIs to fetch per page |
| `--rate-limit` | int | 10 | Max requests per second |
| `--db-path` | string | `.agwctl-db` | Path to NutsDB database directory |
| `--log-level` | string | `info` | Log level: debug, info, warn, error |
| `--dry-run` | bool | false | Simulate without making changes |

### Examples

#### Production Deployment

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

#### Development with Dry-Run

```bash
agwctl \
  --gateway-url=https://gateway.dev.example.com:5555/rest/apigateway \
  --username=admin \
  --password=admin \
  --teams="IBM_Support" \
  --dry-run \
  --log-level=debug
```

#### High-Volume Environment

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

## How It Works

1. **Initialization**
   - Connects to API Gateway
   - Fetches all access profiles (teams) and builds name-to-ID mapping
   - Opens local NutsDB database for state tracking

2. **Polling Loop**
   - Fetches all API IDs using pagination (respecting rate limits)
   - Queries database to identify new APIs (not yet processed)
   - For each new API:
     - Fetches full API document
     - Extracts existing teams using gjson
     - Merges target teams (avoiding duplicates)
     - Updates API using sjson
     - Marks as processed in database

3. **Graceful Shutdown**
   - Catches SIGINT/SIGTERM signals
   - Completes current poll cycle
   - Saves state and closes database
   - Exits cleanly

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         CLI Entry Point                      │
│                      (cmd/agwctl/main.go)                    │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│                     Command Handler                          │
│  - Parse flags                                               │
│  - Initialize components                                     │
│  - Start monitoring loop                                     │
└────────────────────┬────────────────────────────────────────┘
                     │
        ┌────────────┼────────────┐
        ▼            ▼            ▼
┌──────────┐  ┌──────────┐  ┌──────────────┐
│   API    │  │  NutsDB  │  │ AccessProfile│
│  Client  │  │ Storage  │  │   Manager    │
└──────────┘  └──────────┘  └──────────────┘
```

## State Management

The tool uses NutsDB (embedded key-value database) to track processed APIs:

- **Location**: `.agwctl-db/` (configurable)
- **Buckets**:
  - `processed_apis`: Stores API ID → metadata mappings
  - `metadata`: Stores last poll timestamp
- **Benefits**:
  - Fast O(1) lookups
  - ACID guarantees
  - No external database required
  - Automatic compaction

## Performance

### Typical Performance Metrics

- **Pagination**: ~30 seconds for 3000 APIs at 10 req/sec
- **Processing**: 5-10 seconds per new API
- **Memory**: <100MB for 3000 APIs
- **Total Poll Cycle**: <2 minutes for typical workload

### Optimization Tips

1. **Adjust Rate Limit**: Increase `--rate-limit` if your gateway can handle it
2. **Tune Page Size**: Larger pages = fewer requests but more memory
3. **Increase Interval**: Reduce polling frequency if APIs are created infrequently
4. **Monitor Logs**: Use `--log-level=debug` to identify bottlenecks

## Logging

The tool uses structured JSON logging for easy parsing and monitoring:

```json
{"time":"2026-06-16T17:00:00Z","level":"INFO","msg":"Starting API Gateway monitor","interval":"60s","teams":["DevTeam","QATeam"]}
{"time":"2026-06-16T17:00:15Z","level":"INFO","msg":"Pagination complete","total_apis":2847,"pages":29,"duration":"14.2s"}
{"time":"2026-06-16T17:00:15Z","level":"INFO","msg":"New APIs detected","count":5}
{"time":"2026-06-16T17:00:16Z","level":"INFO","msg":"Processing API","api_id":"abc-123","name":"PetStore API"}
{"time":"2026-06-16T17:00:30Z","level":"INFO","msg":"Poll complete","processed":5,"errors":0,"duration":"29.5s"}
```

### Log Levels

- **debug**: Detailed information for troubleshooting
- **info**: General operational messages
- **warn**: Warning messages (non-critical issues)
- **error**: Error messages (failures)

## Troubleshooting

### Common Issues

#### Authentication Failures

```
Error: HTTP 401: Unauthorized
```

**Solution**: Verify username and password are correct. Check user has "Manage APIs" privilege.

#### Rate Limiting

```
Error: HTTP 429: Too Many Requests
```

**Solution**: Reduce `--rate-limit` value or increase `--interval`.

#### Team Not Found

```
Error: teams not found: [TeamName]
```

**Solution**: Verify team name exists in API Gateway. Check `/accessProfiles` endpoint.

#### Database Corruption

```
Error: open database: corrupted
```

**Solution**: Delete database directory and restart:
```bash
rm -rf .agwctl-db
agwctl [flags...]
```

### Debug Mode

Enable debug logging to see detailed information:

```bash
agwctl --log-level=debug [other flags...]
```

## Security Considerations

### Credentials

- **Never commit credentials** to version control
- Use environment variables:
  ```bash
  export GATEWAY_PASSWORD="secret"
  agwctl --password="${GATEWAY_PASSWORD}" [other flags...]
  ```
- Consider using a secrets manager in production

### HTTPS

- Always use HTTPS for production deployments
- Verify TLS certificates are valid

### Database

- Store database in secure location with appropriate permissions
- Database contains API metadata but not credentials

## Development

### Project Structure

```
api-gateway-automator/
├── cmd/
│   └── agwctl/
│       └── main.go              # CLI entry point
├── internal/
│   ├── client/
│   │   ├── client.go            # HTTP client
│   │   └── ratelimit.go         # Rate limiter
│   ├── config/
│   │   └── config.go            # Configuration
│   ├── models/
│   │   └── models.go            # Data models
│   ├── monitor/
│   │   ├── accessprofile.go     # Team management
│   │   ├── poller.go            # Polling logic
│   │   └── processor.go         # JSON processing
│   └── storage/
│       └── repository.go        # NutsDB wrapper
├── spec/
│   ├── apis.openapi.json        # API spec
│   └── users.openapi.json       # User management spec
├── go.mod
├── DESIGN.md                    # Architecture documentation
├── DEPENDENCIES.md              # Dependency justification
└── README.md                    # This file
```

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build -o agwctl ./cmd/agwctl
```

### Code Quality

```bash
# Run linter
golangci-lint run

# Format code
go fmt ./...

# Vet code
go vet ./...
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Run linters and tests
6. Submit a pull request

## License

[Add your license here]

## Support

For issues and questions:
- GitHub Issues: [repository URL]
- Documentation: See DESIGN.md for architecture details
- API Gateway Docs: [IBM webMethods documentation]

## Changelog

### v1.0.0 (2026-06-16)

- Initial release
- Pagination support for large API collections
- Rate limiting to prevent 429 errors
- NutsDB for efficient state management
- gjson/sjson for fast JSON processing
- Structured logging with slog
- Dry-run mode
- Graceful shutdown
