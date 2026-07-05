# Prism Server v1

The Prism server is the Go API for the A/B testing platform.

This document describes the version 1 backend surface: application management, experiments, branches, and the public assignment endpoint.

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
| `AUTH_JWT_SECRET` | HMAC secret for access tokens   | `dev-secret-change-me`                                        |
| `AUTH_ACCESS_TOKEN_TTL` | Access token lifetime     | `15m`                                                         |
| `AUTH_REFRESH_TOKEN_TTL` | Refresh token lifetime   | `168h`                                                        |
| `AUTH_INVITE_TOKEN_TTL` | Invite link lifetime     | `72h`                                                         |
| `AUTH_REFRESH_COOKIE_NAME` | Refresh cookie name    | `prism_refresh`                                               |
| `AUTH_COOKIE_SECURE` | Secure refresh cookie flag   | `false`                                                       |
| `AUTH_COOKIE_SAME_SITE` | Refresh cookie same-site mode | `lax`                                                    |
| `AUTH_REFRESH_COOKIE_PATH` | Refresh cookie path    | `/api/v1/auth`                                                |
| `AUTH_COOKIE_DOMAIN` | Optional refresh cookie domain | _empty_                                                   |
| `APP_BASE_URL` | Frontend base URL used in invite links | `http://localhost:5713`                             |
| `SMTP_HOST` | SMTP server host for invite emails | `smtp.gmail.com`                                          |
| `SMTP_PORT` | SMTP server port | `587`                                                                     |
| `SMTP_USERNAME` | SMTP username | _empty_                                                                     |
| `SMTP_PASSWORD` | SMTP password or Gmail app password | _empty_                                             |
| `SMTP_FROM_EMAIL` | Sender email address | _empty_                                                              |
| `SMTP_FROM_NAME` | Sender display name | `Prism Admin`                                                           |
| `BOOTSTRAP_ADMIN_EMAIL` | Bootstrap admin email     | `admin@example.com`                                           |
| `BOOTSTRAP_ADMIN_PASSWORD` | Bootstrap admin password | `change-me-admin-password`                                 |

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
| `users`        | Admin accounts used to access the management API |
| `refresh_tokens` | Rotating refresh tokens for admin sessions     |
| `invitation_tokens` | One-time activation tokens for invited admins |

Applications support both `status` (`active` or `inactive`) and soft delete. Inactive applications remain visible, but cannot create new experiments.

Users are global admin identities for the Prism dashboard. There is no public signup flow: the first admin is bootstrapped from environment variables, then additional admins are invited by email through the protected `/api/v1/users` API. Invited admins activate their account from a one-time link, set a password, and are signed in automatically.

## Gmail SMTP

Prism uses SMTP directly for admin invite emails. For Gmail:

1. Turn on 2-Step Verification for the Gmail account.
2. Create an App Password in the Google account settings.
3. Set `SMTP_USERNAME` to the Gmail address and `SMTP_PASSWORD` to the App Password.
4. Set `SMTP_HOST=smtp.gmail.com` and `SMTP_PORT=587`.
5. Set `APP_BASE_URL` to the frontend origin that serves the activation page.

## Current API

See [`API.md`](./API.md) for the full route reference including request/response shapes.

Current routes:

| Method | Path | Description |
|--------|------|-------------|
| `GET`  | `/health` | Service health check |
| `POST` | `/api/v1/assign` | Assign a user to a branch using the application API key |
| `POST` | `/api/v1/auth/login` | Sign in an admin and set the refresh cookie |
| `POST` | `/api/v1/auth/refresh` | Rotate the refresh cookie and issue a new access token |
| `POST` | `/api/v1/auth/logout` | Revoke the current refresh token and clear the cookie |
| `GET`  | `/api/v1/auth/invitations/{token}` | Validate an invite token and return invite details |
| `POST` | `/api/v1/auth/invitations/activate` | Activate an invited admin, set password, and issue a session |
| `GET`  | `/api/v1/auth/me` | Return the current admin user |
| `GET`  | `/api/v1/users` | List admin users |
| `POST` | `/api/v1/users` | Send an admin invite email |
| `PATCH` | `/api/v1/users/{id}` | Activate or deactivate an admin user |
| `GET`  | `/api/v1/applications` | List all applications (admin only) |
| `POST` | `/api/v1/applications` | Create an application (admin only) |
| `GET`  | `/api/v1/applications/{id}` | Get an application (admin only) |
| `PUT`  | `/api/v1/applications/{id}` | Update an application name or status (admin only) |
| `DELETE` | `/api/v1/applications/{id}` | Soft-delete an application (admin only) |
| `GET`  | `/api/v1/applications/{appID}/experiments` | List experiments (admin only) |
| `POST` | `/api/v1/applications/{appID}/experiments` | Create an experiment (admin only) |
| `GET`  | `/api/v1/applications/{appID}/experiments/{id}` | Get an experiment (admin only) |
| `PUT`  | `/api/v1/applications/{appID}/experiments/{id}` | Update an experiment (admin only) |
| `DELETE` | `/api/v1/applications/{appID}/experiments/{id}` | Soft-delete an experiment (admin only) |
| `POST` | `/api/v1/applications/{appID}/experiments/{experimentID}/branches` | Add a branch (admin only) |
| `PUT`  | `/api/v1/applications/{appID}/experiments/{experimentID}/branches/{id}` | Update a branch (admin only) |
| `DELETE` | `/api/v1/applications/{appID}/experiments/{experimentID}/branches/{id}` | Soft-delete a branch (admin only) |
