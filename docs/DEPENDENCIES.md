# Dependencies

## Required Go Modules

Add these dependencies to `go.mod`:

```go
module github.com/thrnjica/agwctl

go 1.26.4

require (
    github.com/nutsdb/nutsdb v1.0.4
    github.com/tidwall/gjson v1.17.1
    github.com/tidwall/sjson v1.2.5
    golang.org/x/time v0.5.0
)
```

## Dependency Justification

### github.com/nutsdb/nutsdb
**Purpose:** Embedded key-value database for local state persistence

**Why:**
- Fast O(1) lookups for 2000-3000 API IDs
- ACID guarantees for data consistency
- No external database server required
- Efficient batch operations
- Automatic compaction and cleanup
- Pure Go implementation

**Usage:**
- Store processed API IDs and metadata
- Track last poll timestamp
- Query which APIs are new vs. processed

### github.com/tidwall/gjson
**Purpose:** Fast JSON path evaluation

**Why:**
- Extract specific fields without full JSON parsing
- 10x faster than standard encoding/json for reads
- Simple JSONPath-like syntax
- Zero allocations for most operations
- Read-only operations on API documents

**Usage:**
- Extract existing teams from API JSON: `gjson.Get(apiJSON, "teams")`
- Extract API metadata: `gjson.Get(apiJSON, "api.apiName")`
- Check if field exists without parsing entire document

### github.com/tidwall/sjson
**Purpose:** Fast JSON modification

**Why:**
- Modify JSON without full unmarshaling/marshaling
- Companion to gjson for write operations
- Maintains JSON structure and formatting
- Efficient for targeted updates

**Usage:**
- Add teams to API: `sjson.Set(apiJSON, "teams", updatedTeams)`
- Update specific fields in API documents

### golang.org/x/time/rate
**Purpose:** Token bucket rate limiter

**Why:**
- Standard Go rate limiting implementation
- Prevents 429 (Too Many Requests) errors
- Configurable requests per second
- Context-aware (respects cancellation)
- Thread-safe

**Usage:**
- Limit API Gateway requests to 10/sec (configurable)
- Apply to all HTTP operations
- Shared across concurrent workers

## Installation

```bash
go get github.com/nutsdb/nutsdb@v1.0.4
go get github.com/tidwall/gjson@v1.17.1
go get github.com/tidwall/sjson@v1.2.5
go get golang.org/x/time@v0.5.0
```

## Standard Library Usage

The following standard library packages are also used:

- `net/http` - HTTP client
- `log/slog` - Structured logging
- `flag` - CLI argument parsing
- `context` - Cancellation and timeouts
- `time` - Time operations
- `sync` - Concurrency primitives
- `os`, `io` - File operations
- `fmt`, `errors` - Error handling
