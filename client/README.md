# Prism Client

The Prism frontend is a React app for the A/B testing platform.

## Stack

- React 19
- Vite
- TypeScript
- Tailwind CSS v4
- `shadcn/ui`

## Development

Install dependencies:

```bash
npm install
```

Start the dev server:

```bash
npm run dev
```

Build for production:

```bash
npm run build
```

Run linting:

```bash
npm run lint
```

## UI Setup

- `shadcn/ui` has been initialized for the project.
- The `@/` alias points to `src`.
- Shared UI components live under `src/components`.
- Shared utilities live under `src/lib`.

## Current App State

The admin console supports:

- Application and experiment management
- Branch editing and experiment status controls
- Assignment listing per experiment
- Event listing per experiment with name filtering and pagination
- Experiment dashboard with assignment distribution and optional conversion metrics by event name

API modules live under `src/api/` and mirror the Go server routes documented in `server/API.md`.

## Routes

| Path | Page |
|------|------|
| `/applications` | Application list |
| `/applications/:id` | Application detail |
| `/applications/:appId/experiments` | Experiment list |
| `/applications/:appId/experiments/:id` | Experiment detail |
| `/applications/:appId/experiments/:id/assignments` | Assignment list |
| `/applications/:appId/experiments/:id/events` | Event list |
| `/applications/:appId/experiments/:id/dashboard` | Assignment and conversion dashboard |
| `/admin/users` | Admin user management |
