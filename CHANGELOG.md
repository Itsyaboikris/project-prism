# Changelog

All notable changes to Prism should be documented in this file.

## [Unreleased]

### Added
- SDK event tracking endpoint at `POST /api/v1/events` for recording user actions with optional experiment and branch attribution
- Admin event list endpoint at `GET /api/v1/applications/{appID}/experiments/{id}/events`
- Experiment dashboard conversion metrics via optional `event_name` query parameter on `GET /api/v1/applications/{appID}/experiments/{id}/dashboard`
- Database migration for the `events` table
- Sample project seed migration with a demo application, experiment, assignments, and events for local testing
- Experiment events page in the admin UI with event name filtering and pagination
- Dashboard conversion metrics UI with optional `event_name` filter and per-branch conversion bars

### Changed
- Experiment detail, assignments, and dashboard pages now link to the events view

## [1.0.4] - 2026-07-05

### Added
- Admin RBAC for the management API using JWT access tokens and rotating HttpOnly refresh cookies
- Admin user management endpoints and UI for listing admins, sending invites, and activating or deactivating users
- Email-based admin invite and activation flow with one-time invitation tokens and Gmail-compatible SMTP delivery
- Public activation page at `/activate` for invited admins to set a password and sign in
- Login page, admin shell layout, and route guards for protected admin console pages
- Database migrations for `users`, `refresh_tokens`, and `invitation_tokens`

### Changed
- All management routes now require admin authentication; SDK-facing `POST /api/v1/assign` remains protected by application API keys
- Admin creation now sends an email invite instead of accepting a password at invite time
- CORS now allows credentials so refresh cookies can be used by the frontend
- Updated `server/.env.example`, `server/README.md`, and `server/API.md` with auth, invite, and SMTP configuration

### Fixed
- Refresh token rotation now handles duplicate bootstrap refresh requests gracefully, avoiding logout on page reload in React Strict Mode
- Corrected user creation SQL to persist invited status during admin invite creation

### Security
- Passwords are hashed with bcrypt; refresh and invitation tokens are stored as SHA-256 hashes
- Invited users cannot log in until they activate their account through a valid invitation link

## [1.0.3] - 2026-07-03

### Changed
- New-user assignment selection now uses count-aware balancing to keep live branch splits closer to configured weights
- Existing user assignments remain sticky while deterministic hashing is now used only as a tie-breaker when branches are equally under target

### Fixed
- Reduced short-run drift for weighted experiments, especially on lower-volume traffic, by selecting the most under-target branch inside a transaction
- Added store coverage for balanced `50/50` and `80/20` splits, sticky assignments, deterministic tie-breaking, and the transactional assignment path

## [1.0.2] - 2026-07-03

### Added
- Experiment-scoped assignment read endpoint at `GET /api/v1/applications/{appID}/experiments/{id}/assignments`
- Experiment dashboard endpoint at `GET /api/v1/applications/{appID}/experiments/{id}/dashboard`
- Dedicated frontend assignments page for viewing experiment user-to-branch assignments
- Dedicated frontend dashboard page for comparing branch configured weights against actual assignment shares

### Changed
- Experiment detail pages now link directly to `Assignments` and `Dashboard` views
- Assignment read responses now expose branch metadata needed for admin-facing analytics and review

### Fixed
- Dashboard summaries now include zero-assignment branches so misbalanced or newly launched experiments remain visible
- Added handler and store coverage for assignment listing and experiment dashboard aggregation

## [1.0.1] - 2026-07-03

### Added
- Shared frontend and backend validation helpers for application names, experiment fields, branch fields, branch weights, and experiment date ranges
- Transactional bulk branch save support with a dedicated `PUT /api/v1/applications/{appID}/experiments/{experimentID}/branches` endpoint
- Reusable `ExperimentStatusToggle` component for per-experiment activation control
- Branch weight migration to convert stored fractional weights to percentages

### Changed
- Branch weights now use percentage-based validation and display while still accepting legacy fractional totals during validation
- Experiment creation and detail flows now use a staged bulk branch editor instead of one-at-a-time branch edits
- Experiment status toggles now live on each experiment card instead of at the application level
- Application names now wrap consistently on list and detail views to avoid layout overflow

### Fixed
- Prevent creating or updating experiments with an `end_date` earlier than `start_date`
- Enforce length limits for application names, experiment names and keys, experiment descriptions, branch names and keys
- Validate branch metadata as a JSON object with a 4 KB serialized size limit
- Improved handler, store, and UI test coverage for validation and bulk branch editing flows

## [1.0.0] - 2026-07-03

### Added
- React/Vite frontend in `client`
- Go API server in `server`
- PostgreSQL migrations for applications, experiments, branches, and assignments
- Docker Compose support for local Postgres development
- Hot reload for the Go server with Air
- CRUD endpoints for applications, experiments, and branches
- Public `POST /api/v1/assign` endpoint for SDK-facing branch assignment
- Deterministic weighted branch assignment based on application, experiment key, and user ID
- Sticky assignment persistence in the `assignments` table
- Backend API documentation in `server/API.md`
- Root `CHANGELOG.md` for release tracking

### Changed
- Experiment read responses now embed branches
- Applications, experiments, and branches now use soft delete behavior
- Applications now support `active` and `inactive` status
- Inactive applications cannot create new experiments

### Fixed
- Backend test coverage increased to the v1 target range with handler, middleware, helper, and store tests
- Client TypeScript compatibility issue in `client/src/api/client.ts` was resolved for successful builds

### Security
- `POST /api/v1/assign` authenticates application requests with `X-API-Key` or `Authorization: Bearer <api_key>`
