# Prism Server

The Prism server is the Go API for the A/B testing platform.

## Stack

- Go 1.25+
- Standard library `net/http`
- `pgx/v5` — PostgreSQL driver and connection pool
- `godotenv` — `.env` loading for local development
- Air — hot reload during development
- `golang-migrate` — SQL migration runner

## Prerequisites

Install tooling once:

```bash
# Hot reload
go install github.com/air-verse/air@latest

# Migration runner
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

## Environment Setup

Copy the example env file and fill in your values:

```bash
cp .env.example .env
```

**Update `.env` when:** setting up a new machine, changing database credentials, or switching between local and remote Postgres instances.

The required variables are:

| Variable       | Description                        | Default                                                       |
|----------------|------------------------------------|---------------------------------------------------------------|
| `DATABASE_URL` | PostgreSQL connection string       | `postgres://postgres:postgres@localhost:5432/prism?sslmode=disable` |
| `PORT`         | HTTP port the server listens on    | `8080`                                                        |

## Database

If you are using the repo's Docker setup from the project root, start Postgres with:

```bash
make db-up
```

That container already creates the `prism` database with:

- user: `postgres`
- password: `postgres`
- port: `5432`

Update `.env` if you are not using those defaults.

If you are using a separate Postgres instance, create the Prism database first:

```sql
CREATE DATABASE prism;
```

Then run migrations:

```bash
make migrate-up
```

The migration targets read `DATABASE_URL` from `server/.env` automatically when that file exists.

Roll back one step:

```bash
make migrate-down
```

Create a new migration:

```bash
make migrate-create name=add_experiment_metadata
```

## Development

Run with hot reload:

```bash
make dev
```

Run directly without Air:

```bash
go run .
```

Build:

```bash
go build ./...
```

## Project Structure

```
server/
├── main.go                  # entrypoint
├── .air.toml                # Air reload config
├── .env.example             # environment variable template
├── Makefile
├── migrations/              # SQL migration files (golang-migrate)
│   ├── 000001_create_applications.up.sql
│   ├── 000001_create_applications.down.sql
│   ├── 000002_create_experiments.up.sql
│   ├── 000002_create_experiments.down.sql
│   ├── 000003_create_branches.up.sql
│   ├── 000003_create_branches.down.sql
│   ├── 000004_create_assignments.up.sql
│   └── 000004_create_assignments.down.sql
└── internal/
    ├── config/              # environment-based config loading
    ├── db/                  # Postgres connection pool
    ├── router/              # HTTP route wiring
    └── handlers/            # request handlers
```

## Data Model

| Table          | Description                                      |
|----------------|--------------------------------------------------|
| `applications` | Top-level Prism project / API key holder         |
| `experiments`  | A/B test belonging to an application             |
| `branches`     | Variants within an experiment with traffic weights |
| `assignments`  | Tracks which branch a user was assigned to       |

Applications support both `status` (`active` or `inactive`) and soft delete. Inactive applications remain visible, but cannot create new experiments.

## Current API

See [`API.md`](./API.md) for the full route reference including request/response shapes.

Current routes:

| Method | Path | Description |
|--------|------|-------------|
| `GET`  | `/health` | Service health check |
| `GET`  | `/api/v1/applications` | List all applications |
| `POST` | `/api/v1/applications` | Create an application |
| `GET`  | `/api/v1/applications/{id}` | Get an application |
| `PUT`  | `/api/v1/applications/{id}` | Update an application name or status |
| `DELETE` | `/api/v1/applications/{id}` | Soft-delete an application (cascades to experiments and branches) |
| `GET`  | `/api/v1/applications/{appID}/experiments` | List experiments (branches embedded) |
| `POST` | `/api/v1/applications/{appID}/experiments` | Create an experiment (optional initial branches) |
| `GET`  | `/api/v1/applications/{appID}/experiments/{id}` | Get an experiment (branches embedded) |
| `PUT`  | `/api/v1/applications/{appID}/experiments/{id}` | Update an experiment (branches embedded in response) |
| `DELETE` | `/api/v1/applications/{appID}/experiments/{id}` | Soft-delete an experiment (cascades to branches) |
| `POST` | `/api/v1/applications/{appID}/experiments/{experimentID}/branches` | Add a branch to an experiment |
| `PUT`  | `/api/v1/applications/{appID}/experiments/{experimentID}/branches/{id}` | Update a branch |
| `DELETE` | `/api/v1/applications/{appID}/experiments/{experimentID}/branches/{id}` | Soft-delete a branch |
