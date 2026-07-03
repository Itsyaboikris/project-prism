# Prism Server

The Prism server is the Go API for the A/B testing platform.

## Stack

- Go 1.21+
- Standard library `net/http`
- Air for hot reload during development

## Development

Run the server directly:

```bash
go run .
```

Build the server:

```bash
go build ./...
```

Run the server with Air:

```bash
make dev
```

Install Air if needed:

```bash
go install github.com/air-verse/air@latest
```

## Current Structure

- `main.go` - server entrypoint
- `internal/config` - runtime configuration loading
- `internal/router` - HTTP route wiring
- `internal/handlers` - request handlers
- `.air.toml` - Air reload configuration

## Current API

### Health Check

`GET /health`

Returns a small JSON payload confirming the API is running.

## Notes

This backend is currently a thin starter. It is ready to grow into Prism's experiment management, variant assignment, exposure tracking, and reporting APIs.
