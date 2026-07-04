# Changelog

All notable changes to Prism should be documented in this file.

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
