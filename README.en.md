# go-cache-server-mini

## Overview
`go-cache-server-mini` is a tiny in-memory cache service that exposes a minimal Gin HTTP API for storing and fetching key-value pairs. It is intended for workshops, demos, or lightweight integration tests that need a disposable stateful endpoint with zero external dependencies.

## Features
- Data mutations: `POST /set`, `GET /get`, `DELETE /del`
- Introspection: `GET /exists`, `GET /keys`, `GET /ttl`
- Maintenance: `POST /expire`, `POST /flush`, `GET /ping`
- Values are stored as `json.RawMessage`, so any valid JSON document survives round trips.
- Powered by `sync.Map` + reusable error values in `internal/errors.go`

## Project Layout
```
cmd/main.go                 # Entry point, signal handling, graceful shutdown
internal/api/api.go         # Gin bootstrap + router
internal/api/handler/*.go   # HTTP handlers (set/get/del/…)
internal/api/dto/*.go       # Request / response DTOs
internal/core/core.go       # Cache, TTL logic, expire worker
internal/core/cache_interface.go
internal/config/config.go   # YAML loader
internal/errors.go          # Shared errors
config.yml                  # Default TTL + HTTP binding
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

## Configuration
Edit `config.yml` or provide environment variables embedded in the file (e.g., `${PORT}`).

```yaml
ttl:
  default: 86400   # used when clients omit TTL or send <= 0
  max: 604800      # protects the cache by clamping overly large TTLs
http:
  enabled: true
  address: ":8080"
```

## Graceful shutdown & failure propagation
- The main process listens for `SIGINT`/`SIGTERM`, cancels the shared context, and waits for the API goroutine plus the TTL worker to finish before exiting.
- `api.StartAPIServer` now returns an error when the HTTP server fails to bind (e.g., port already in use). The error is forwarded to `cmd/main.go`, which logs it, shuts the cache down, and exits with status code `1` instead of hanging forever without an HTTP listener.

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
- If macOS sandboxing blocks `$HOME/Library/Caches/go-build`, pin the build cache inside the repo:
  ```bash
  mkdir -p .gocache
  GOCACHE=$(pwd)/.gocache go test ./...
  ```
- Toggle the listening port by editing `addr` in `cmd/main.go` when running multiple instances.
- Keep business rules inside `internal/core.go` and leave HTTP-only concerns in `internal/api.go` to maintain a clear separation.
