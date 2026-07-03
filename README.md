# Prism

Prism is an A/B testing platform. This repository currently contains the initial client and server scaffolds for the product.

## Repository Layout

- `client` - React, Vite, TypeScript, Tailwind v4, and `shadcn/ui`
- `server` - Go API starter with Air hot reload support

## Quick Start

### Prerequisites

- Node.js and npm
- Go 1.21+
- `air` for Go hot reload

Install Air once:

```bash
go install github.com/air-verse/air@latest
```

### Run The Client

```bash
cd client
npm install
npm run dev
```

### Run The Server

```bash
cd server
go run .
```

### Run The Server With Hot Reload

```bash
cd server
make dev
```

### Run Both Apps From The Repo Root

```bash
make dev
```

## Current State

- The frontend is a Prism-branded starter app with the UI foundation in place.
- The backend is a minimal HTTP server with a `GET /health` endpoint.
- The root `Makefile` can start the frontend and backend together.

## Project Docs

- `client/README.md` - frontend setup and structure
- `server/README.md` - backend setup, hot reload, and API starter details
