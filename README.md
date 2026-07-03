# Prism

Prism is an A/B testing platform. This repository contains the React client and Go API server.

## Repository Layout

- `client` — React, Vite, TypeScript, Tailwind v4, `shadcn/ui`
- `server` — Go API with Postgres, Air hot reload, and SQL migrations

## Quick Start

### Prerequisites

- Node.js and npm
- Go 1.25+
- PostgreSQL
- `air` for Go hot reload
- `golang-migrate` CLI for running migrations

```bash
# Install Go tooling once
go install github.com/air-verse/air@latest
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

### 1. Set up environment

```bash
cp server/.env.example server/.env
# Edit server/.env if you are not using the default Docker Postgres setup
```

### 2. Start Postgres

```bash
make db-up
```

This starts a local Postgres container on `localhost:5432` with:

- database: `prism`
- user: `postgres`
- password: `postgres`

If you use those defaults, `server/.env.example` already matches the container. Update `server/.env` when you change DB credentials, ports, or point the app at a different Postgres instance.

### 3. Run migrations

```bash
make migrate-up
```

The migration command reads `DATABASE_URL` from `server/.env` automatically.

### 4. Start the apps

```bash
# Both client and server together
make dev

# Or individually
make client-dev
make server-dev
```

## Makefile Reference

| Command              | What it does                                      |
|----------------------|---------------------------------------------------|
| `make dev`           | Start client (Vite) and server (Air) together     |
| `make client-dev`    | Start the Vite dev server                         |
| `make server-dev`    | Start the Go server with Air hot reload           |
| `make server-run`    | Start the Go server without hot reload            |
| `make db-up`         | Start the Postgres container                      |
| `make db-down`       | Stop Docker services and remove containers        |
| `make db-logs`       | Tail the Postgres container logs                  |
| `make migrate-up`    | Apply all pending migrations                      |
| `make migrate-down`  | Roll back the most recent migration               |

## Current State

- Frontend is a Prism-branded UI starter with `shadcn/ui` ready to use.
- Backend is a Go HTTP API with a Postgres connection pool and `GET /health`.
- Database schema covers the four core Prism entities: applications, experiments, branches, and assignments.

## Project Docs

- `client/README.md` — frontend setup and structure
- `server/README.md` — backend setup, hot reload, migrations, and data model
