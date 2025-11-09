# go-cache-server-mini

## Overview
`go-cache-server-mini` is a tiny in-memory cache service that exposes a minimal Gin HTTP API for storing and fetching key-value pairs. It is intended for workshops, demos, or lightweight integration tests that need a disposable stateful endpoint with zero external dependencies.

## Features
- `POST /set` accepts arbitrary JSON payloads and stores them verbatim.
- `GET /get` looks up a key and returns the cached JSON blob with precise HTTP status codes.
- Backed by `sync.Map`, so concurrent access is safe, and common errors are centralized in `internal/errors.go`.

## Project Layout
```
cmd/main.go          # Server entrypoint, address configuration
internal/api.go      # Gin router and HTTP handlers
internal/core.go     # Cache struct plus Set/Get operations
internal/errors.go   # Reusable error values surfaced by the API
```

## Quickstart
1. Install Go 1.24.5 or newer.
2. Run the server directly:
   ```bash
   go run ./cmd
   ```
3. Produce a distributable binary when needed:
   ```bash
   go build -o bin/cache-server ./cmd
   ```

## API Examples
```bash
# Store a value
curl -X POST http://localhost:8080/set \
  -H "Content-Type: application/json" \
  -d '{"key":"greeting","value":"\"hello\""}'

# Fetch the value
curl "http://localhost:8080/get?key=greeting"
```
The `value` field is treated as `json.RawMessage`, so any valid JSON (string, object, number) is preserved without transformation.

## Development & Testing
- Run all tests: `go test ./...`
- Toggle the listening port by editing `addr` in `cmd/main.go` when running multiple instances.
- Keep business rules inside `internal/core.go` and leave HTTP-only concerns in `internal/api.go` to maintain a clear separation.
