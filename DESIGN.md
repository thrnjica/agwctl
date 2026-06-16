# API Gateway Automator - High-Level Design

## Overview

A Go CLI tool that monitors the IBM webMethods API Gateway 10.15 for newly created APIs and automatically adds specified teams to them. The tool handles large-scale deployments (2000-3000 APIs) with pagination, rate limiting, and efficient local storage.

## Dependencies

**Standard Library:**
- `net/http` - HTTP client
- `log/slog` - Structured logging
- `flag` - CLI argument parsing
- `context`, `time`, `sync` - Concurrency and timing
- `os`, `io` - File operations

**Third-Party Libraries:**
- `github.com/nutsdb/nutsdb` - Embedded key-value database for local state persistence
- `github.com/tidwall/gjson` - Fast JSON path evaluation for efficient document manipulation

## Architecture

### System Components

```
┌─────────────────────────────────────────────────────────────┐
│                         CLI Entry Point                      │
│                      (cmd/agwctl/main.go)                    │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│                     Command Handler                          │
│  - Parse flags (credentials, teams, interval, state file)   │
│  - Initialize components                                     │
│  - Start monitoring loop                                     │
└────────────────────┬────────────────────────────────────────┘
                     │
        ┌────────────┼────────────┐
        ▼            ▼            ▼
┌──────────┐  ┌──────────┐  ┌──────────────┐
│   API    │  │  State   │  │ AccessProfile│
│  Client  │  │ Manager  │  │   Manager    │
└──────────┘  └──────────┘  └──────────────┘
     │                              │
     │                              │
     ▼                              ▼
/apis endpoint              /accessProfiles endpoint
(API operations)            (Team/AccessProfile operations)
```

### Package Structure

```
api-gateway-automator/
├── cmd/
│   └── agwctl/
│       └── main.go              # CLI entry point
├── internal/
│   ├── client/
│   │   ├── client.go            # HTTP client wrapper with rate limiting
│   │   ├── auth.go              # Authentication logic
│   │   ├── api.go               # API-specific methods
│   │   └── ratelimit.go         # Rate limiter implementation
│   ├── models/
│   │   ├── api.go               # API data structures
│   │   ├── team.go              # Team data structures
│   │   └── response.go          # API response structures
│   ├── storage/
│   │   ├── nutsdb.go            # NutsDB wrapper
│   │   ├── repository.go        # Data access layer
│   │   └── models.go            # Storage models
│   ├── monitor/
│   │   ├── poller.go            # Polling logic with pagination
│   │   ├── detector.go          # Change detection
│   │   └── processor.go         # API processing with gjson
│   └── config/
│       └── config.go            # Configuration management
├── spec/
│   └── apis.openapi.json        # API specification
├── go.mod
├── Makefile
└── README.md
```

## Core Components

### 1. CLI Entry Point (`cmd/agwctl/main.go`)

**Responsibilities:**
- Parse command-line flags
- Validate input parameters
- Initialize and coordinate components
- Handle graceful shutdown (SIGINT, SIGTERM)

**Command-line Flags:**
```
--gateway-url     string   API Gateway base URL (required)
--username        string   Basic auth username (required)
--password        string   Basic auth password (required)
--teams           string   Comma-separated team names to add (required)
--interval        int      Polling interval in seconds (default: 60)
--db-path         string   Path to NutsDB database directory (default: data)
--page-size       int      Number of APIs to fetch per page (default: 100)
--rate-limit      int      Max requests per second (default: 10)
--log-level       string   Log level: debug, info, warn, error (default: info)
--dry-run         bool     Simulate without making changes (default: false)
```

**Example Usage:**
```bash
agwctl \
  --gateway-url=https://gateway.example.com:5555/rest/apigateway \
  --username=admin \
  --password=secret \
  --teams="DevTeam,QATeam" \
  --interval=60
```

### 2. API Client with Rate Limiting (`internal/client/`)

**Purpose:** Encapsulate all HTTP communication with rate limiting to avoid 429 errors.

**Key Methods:**

```go
type Client struct {
    baseURL     string
    httpClient  *http.Client
    auth        *BasicAuth
    rateLimiter *RateLimiter
    logger      *slog.Logger
}

type RateLimiter struct {
    limiter *rate.Limiter
    mu      sync.Mutex
}

// Authentication
func (c *Client) Authenticate() error

// API Operations with Pagination
func (c *Client) ListAPIs(from, size int) (*APIListResponse, error)
func (c *Client) GetAPI(apiID string) ([]byte, error) // Returns raw JSON
func (c *Client) UpdateAPI(apiID string, apiJSON []byte) error

// AccessProfile (Team) Operations
func (c *Client) ListAccessProfiles() (*AccessProfileListResponse, error)
func (c *Client) GetAccessProfile(accessProfileID string) (*AccessProfile, error)

// Rate Limiting
func (rl *RateLimiter) Wait(ctx context.Context) error
func (rl *RateLimiter) SetRate(requestsPerSecond int)
```

**Implementation Details:**
- Use `net/http` for HTTP requests
- Implement Basic Authentication using `Authorization` header
- Set appropriate headers: `Content-Type: application/json`, `Accept: application/json`
- Handle HTTP status codes: 200 (success), 401 (auth error), 404 (not found), 429 (rate limit), 500 (server error)
- Implement token bucket rate limiting using `golang.org/x/time/rate`
- Use `slog` for structured request/response logging
- Return raw JSON bytes for APIs to avoid full parsing overhead

**Error Handling:**
- Wrap errors with context using `fmt.Errorf`
- Distinguish between network errors, auth errors, rate limit errors, and API errors
- Implement exponential backoff for retryable errors (5xx, network timeouts)
- Special handling for 429: wait and retry with backoff

### 3. Data Models (`internal/models/`)

**API Structure:**
```go
type API struct {
    ID              string   `json:"id"`
    APIName         string   `json:"apiName"`
    APIVersion      string   `json:"apiVersion"`
    APIDescription  string   `json:"apiDescription"`
    IsActive        bool     `json:"isActive"`
    Type            string   `json:"type"`
    Teams           []Team   `json:"teams"`
    CreationDate    string   `json:"creationDate"`
    SystemVersion   int      `json:"systemVersion"`
    // ... other fields as needed
}

type Team struct {
    ID        string `json:"id"`
    Name      string `json:"name"`
    Source    string `json:"source,omitempty"`
    CanDelete string `json:"canDelete,omitempty"`
}

type AccessProfile struct {
    ID          string   `json:"id"`
    Name        string   `json:"name"`
    Description string   `json:"description"`
    GroupIDs    []string `json:"groupIds"`
    // ... other fields as needed
}

type AccessProfileListResponse struct {
    AccessProfiles []AccessProfile `json:"accessProfiles"`
}

type APIListResponse struct {
    APIResponse []APIResponseItem `json:"apiResponse"`
}

type APIResponseItem struct {
    API            API    `json:"api"`
    ResponseStatus string `json:"responseStatus"`
    Teams          []Team `json:"teams"`
}
```

### 4. Storage Layer with NutsDB (`internal/storage/`)

**Purpose:** Efficiently store and query processed APIs using embedded database.

**NutsDB Schema:**
```
Bucket: "processed_apis"
Key: API ID (string)
Value: JSON document with metadata
{
  "id": "api-id-1",
  "name": "PetStore API",
  "version": "1.0",
  "processedAt": "2026-06-16T15:30:00Z",
  "teamsAdded": ["DevTeam", "QATeam"],
  "rawAPI": "..." // Original API JSON for reference
}

Bucket: "metadata"
Key: "last_poll"
Value: ISO 8601 timestamp
```

**Key Methods:**
```go
type Repository struct {
    db     *nutsdb.DB
    logger *slog.Logger
}

func NewRepository(dbPath string) (*Repository, error)
func (r *Repository) Close() error

// API Operations
func (r *Repository) IsProcessed(apiID string) (bool, error)
func (r *Repository) MarkProcessed(apiID string, metadata *ProcessedAPI) error
func (r *Repository) GetProcessedAPI(apiID string) (*ProcessedAPI, error)
func (r *Repository) GetAllProcessedIDs() ([]string, error)

// Metadata Operations
func (r *Repository) SetLastPoll(timestamp time.Time) error
func (r *Repository) GetLastPoll() (time.Time, error)

// Batch Operations
func (r *Repository) MarkProcessedBatch(apis []*ProcessedAPI) error
```

**Implementation Details:**
- Use NutsDB for fast key-value storage with ACID guarantees
- Store raw API JSON to avoid re-fetching
- Use separate buckets for different data types
- Implement batch operations for efficiency
- Handle database corruption gracefully
- Automatic compaction and cleanup

### 5. Monitor with Pagination (`internal/monitor/`)

**Purpose:** Implement the polling loop with pagination support for large API collections.

**Polling Strategy:**
```go
type Poller struct {
    client      *client.Client
    repository  *storage.Repository
    interval    time.Duration
    pageSize    int
    targetTeams []string
    logger      *slog.Logger
    dryRun      bool
}

func (p *Poller) Start(ctx context.Context) error
func (p *Poller) pollOnce(ctx context.Context) error
func (p *Poller) fetchAllAPIs(ctx context.Context) ([]string, error) // Returns API IDs
func (p *Poller) detectNewAPIs(apiIDs []string) ([]string, error)
func (p *Poller) processNewAPI(ctx context.Context, apiID string) error
```

**Processor with gjson:**
```go
type Processor struct {
    client      *client.Client
    logger      *slog.Logger
}

func (p *Processor) AddTeamsToAPI(apiJSON []byte, teamIDs []string) ([]byte, error) {
    // Use gjson to extract existing teams
    // Merge with new teams (avoid duplicates)
    // Use sjson to update teams array
    // Return modified JSON
}

func (p *Processor) ExtractAPIMetadata(apiJSON []byte) (*APIMetadata, error) {
    // Use gjson to extract: id, name, version, creationDate
}
```

**Polling Flow:**

```
┌─────────────────────────────────────────────────────────────┐
│                      Start Polling                           │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│              Wait for interval or context cancel             │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│           Fetch all APIs from /apis endpoint                 │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│    For each API: Check if ID exists in state                │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│              New API detected? Process it                    │
│  1. Fetch full API details (GET /apis/{apiId})              │
│  2. Resolve target team IDs from team names                 │
│  3. Merge target teams with existing teams                  │
│  4. Update API with new teams (PUT /apis/{apiId})           │
│  5. Mark API as processed in state                          │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│              Save state to disk                              │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────────┐
│              Log summary and repeat                          │
└─────────────────────────────────────────────────────────────┘
```

**Change Detection Logic:**
- Fetch all API IDs via paginated requests
- Query NutsDB for processed API IDs
- New API = ID not in NutsDB
- Efficient set difference operation
- Handle 2000-3000 APIs efficiently

### 6. AccessProfile (Team) Resolution

**Challenge:** The user provides team names, but the API requires team IDs.

**Solution:** Use the `/accessProfiles` endpoint to list all teams and resolve names to IDs.

```go
type AccessProfileManager struct {
    client *client.Client
    cache  map[string]string // name -> ID mapping
    mu     sync.RWMutex
}

func (apm *AccessProfileManager) ResolveTeamNames(names []string) ([]string, error) {
    // 1. Check cache first
    // 2. If not in cache, fetch all access profiles via GET /accessProfiles
    // 3. Build name->ID mapping from response
    // 4. Cache the mapping for future use
    // 5. Return team IDs for requested names
}

func (apm *AccessProfileManager) RefreshCache() error {
    // Fetch all access profiles and update cache
}
```

**Key Insights from API Spec:**
- Teams are called "Access Profiles" in the API Gateway
- Endpoint: `GET /accessProfiles` returns all teams with their IDs and names
- Each AccessProfile has: `id`, `name`, `description`, `groupIds`, etc.
- System-defined teams: "Administrators", "API-Gateway-Providers"
- Custom teams have UUID-style IDs (e.g., "8b6f2e10-1d82-4813-b927-4c1cf4a4d029")

### 7. Configuration (`internal/config/`)

**Purpose:** Centralize configuration management.

```go
type Config struct {
    GatewayURL  string
    Username    string
    Password    string
    Teams       []string
    Interval    time.Duration
    StateFile   string
    LogLevel    string
    DryRun      bool
}

func LoadFromFlags() (*Config, error)
func (c *Config) Validate() error
```

### 8. Structured Logging with slog

**Implementation:**
- Use `log/slog` for structured logging
- Support multiple log levels: DEBUG, INFO, WARN, ERROR
- Add context fields for traceability
- Log to stdout with JSON formatting option

**Log Examples:**
```json
{"time":"2026-06-16T17:00:00Z","level":"INFO","msg":"Starting API Gateway monitor","interval":"60s","teams":["DevTeam","QATeam"],"page_size":100,"rate_limit":10}
{"time":"2026-06-16T17:00:01Z","level":"INFO","msg":"Fetching APIs","page":1,"from":0,"size":100}
{"time":"2026-06-16T17:00:02Z","level":"INFO","msg":"Fetching APIs","page":2,"from":100,"size":100}
{"time":"2026-06-16T17:00:15Z","level":"INFO","msg":"Pagination complete","total_apis":2847,"pages":29,"duration":"14.2s"}
{"time":"2026-06-16T17:00:15Z","level":"INFO","msg":"New APIs detected","count":5}
{"time":"2026-06-16T17:00:16Z","level":"INFO","msg":"Processing API","api_id":"abc-123","name":"PetStore API","version":"1.0"}
{"time":"2026-06-16T17:00:17Z","level":"INFO","msg":"Teams added","api_id":"abc-123","teams":["DevTeam","QATeam"]}
{"time":"2026-06-16T17:00:30Z","level":"INFO","msg":"Poll complete","processed":5,"errors":0,"duration":"29.5s"}
```

## Workflow

### Initial Setup
1. User runs CLI with credentials and target teams
2. CLI validates inputs and initializes components
3. State manager loads existing state (or creates new)
4. API client authenticates with gateway

### Polling Cycle
1. Wait for interval (60 seconds by default)
2. Fetch all APIs from `/apis` endpoint
3. Compare API IDs against state file
4. For each new API:
   - Fetch full API details
   - Resolve team names to IDs
   - Merge with existing teams (avoid duplicates)
   - Update API via `PUT /apis/{apiId}`
   - Mark as processed in state
5. Save state to disk
6. Log summary and repeat

### Graceful Shutdown
1. Catch SIGINT/SIGTERM signals
2. Complete current poll cycle
3. Save state to disk
4. Exit cleanly

## Error Handling Strategy

### Retryable Errors
- Network timeouts: Retry with exponential backoff (max 3 attempts)
- 5xx server errors: Retry with exponential backoff (max 3 attempts)
- Rate limiting (429): Wait and retry

### Non-Retryable Errors
- 401 Unauthorized: Log error and exit (invalid credentials)
- 404 Not Found: Skip API and continue
- 400 Bad Request: Log error and skip API
- State file corruption: Create new state file

### Error Recovery
- Continue processing remaining APIs even if one fails
- Log all errors with context
- Maintain state consistency (only mark as processed on success)

## Security Considerations

1. **Credentials:**
   - Accept via command-line flags (not ideal for production)
   - Consider environment variables as alternative
   - Never log credentials
   - Clear sensitive data from memory when possible

2. **HTTPS:**
   - Enforce HTTPS for API Gateway communication
   - Validate TLS certificates (configurable for dev environments)

3. **State File:**
   - Store in user's home directory or specified location
   - Set appropriate file permissions (0600)
   - Don't store credentials in state file

## Testing Strategy

### Unit Tests
- Test each component in isolation
- Mock HTTP responses for API client tests
- Test state manager with temporary files
- Test team resolution logic
- Test change detection algorithm

### Integration Tests
- Test against mock API Gateway server
- Verify end-to-end workflow
- Test error scenarios

### Manual Testing
- Test against actual API Gateway instance (if available)
- Verify team addition works correctly
- Test graceful shutdown
- Test state persistence across restarts

## Performance Considerations

### 1. Pagination Strategy
**Challenge:** 2000-3000 APIs require efficient pagination
- Fetch in pages of 100 APIs (configurable)
- Total requests: ~30 pages per poll cycle
- Use `from` and `size` query parameters
- Track pagination progress for resumability

### 2. Rate Limiting
**Challenge:** Avoid 429 (Too Many Requests) errors
- Implement token bucket rate limiter
- Default: 10 requests per second (configurable)
- Apply to all API calls (GET, PUT)
- Exponential backoff on 429 responses
- Monitor rate limit headers if available

### 3. Efficient JSON Processing
**Challenge:** Avoid parsing entire API documents
- Use `gjson` for read operations (extract teams, metadata)
- Use `sjson` for write operations (update teams array)
- Avoid full JSON unmarshaling/marshaling
- Process only required fields

### 4. Database Performance
**Challenge:** Fast lookups for 2000-3000 API IDs
- NutsDB provides O(1) key lookups
- Batch operations for bulk inserts
- Periodic compaction to maintain performance
- Index on API IDs for fast existence checks

### 5. Concurrency
**Challenge:** Process multiple APIs efficiently
- Process new APIs concurrently (5-10 goroutines)
- Use worker pool pattern with rate limiting
- Shared rate limiter across workers
- Context-based cancellation

### 6. Memory Management
**Challenge:** Handle large datasets efficiently
- Stream API IDs during pagination (don't load all at once)
- Process APIs in batches
- Release memory after each batch
- Monitor memory usage

**Performance Targets:**
- Pagination: ~30 seconds for 3000 APIs (10 req/sec)
- Processing: ~5-10 seconds per new API
- Total poll cycle: <2 minutes for typical workload
- Memory: <100MB for 3000 APIs

## Future Enhancements

1. **Webhook Support:** Listen for API creation events instead of polling
2. **Team Removal:** Support removing teams from APIs
3. **Conditional Logic:** Add teams based on API properties (name, version, type)
4. **Metrics:** Export metrics for monitoring (Prometheus format)
5. **Configuration File:** Support YAML/JSON config file
6. **Multiple Gateways:** Monitor multiple API Gateway instances
7. **Dry-Run Mode:** Simulate changes without applying them
8. **Rollback:** Remove teams if API is deleted

## Constraints Validation

✅ **Go 1.26.4 Standard Library Only:**
- `net/http` for HTTP client
- `encoding/json/v2` for JSON handling
- `log/slog` for logging
- `flag` for CLI parsing
- `os`, `io`, `time`, `context`, `sync` for core functionality

✅ **No Third-Party Dependencies:**
- All functionality implemented using standard library
- No external packages required

## Open Questions

1. **Team ID vs Name:** Are team IDs the same as team names, or do we need a mapping?
2. **API Update Payload:** Does `PUT /apis/{apiId}` require the full API object or just the teams field?
3. **Team Endpoint:** Is there an undocumented `/teams` endpoint for listing teams?
4. **Duplicate Prevention:** Should we check if teams already exist before adding?
5. **API Types:** Should we filter by API type (REST, SOAP, WebSocket, OData)?

## Recommendations

1. **Start Simple:** Implement basic polling and team addition first
2. **Incremental Development:** Add features iteratively (logging, error handling, concurrency)
3. **Test Early:** Write tests alongside implementation
4. **Documentation:** Document API assumptions and limitations
5. **Configuration:** Make behavior configurable (intervals, batch sizes, etc.)
6. **Observability:** Add comprehensive logging for troubleshooting

## Resolved Questions

1. ✅ **Team Management:** Teams are managed via the `/accessProfiles` endpoint in the User Management Service
   - Endpoint: `GET /accessProfiles` returns all teams (called "Access Profiles")
   - Each AccessProfile has: `id`, `name`, `description`, `groupIds`
   - System-defined teams: "Administrators", "API-Gateway-Providers"
   - Custom teams have UUID-style IDs

2. ✅ **Team Resolution:** Use `/accessProfiles` to build a name-to-ID mapping cache at startup

## Remaining Open Questions

1. **API Update Payload:** Does `PUT /apis/{apiId}` require the full API object or just the teams field?
2. **Duplicate Prevention:** Should we check if teams already exist before adding?
3. **API Types:** Should we filter by API type (REST, SOAP, WebSocket, OData)?
4. **Team Assignment Mechanism:** How exactly are teams added to APIs? Via the `teams` array in the API object during PUT?
