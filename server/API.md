# Prism API Reference v1

Base URL: `http://localhost:8080`

All request and response bodies use `application/json`.

SDK-facing assignment and event requests authenticate with an application API key sent in either the `X-API-Key` header or `Authorization: Bearer <api_key>`.

Admin-facing management requests authenticate with a short-lived JWT access token sent as `Authorization: Bearer <access_token>`. Session refresh uses an HttpOnly cookie on the `/api/v1/auth/*` routes.

---

## Health

### `GET /health`

Confirms the API is running. No authentication required.

**Response `200`**
```json
{
  "status": "ok",
  "service": "project-prism-api"
}
```

---

## Assignment

### `POST /api/v1/assign`

Deterministically assigns a user to a branch for an experiment within the application identified by the API key header.

If the user was previously assigned for the same experiment, the existing branch is returned. Otherwise the server hashes `application_id + experiment_key + user_id` into a stable bucket and applies the experiment's branch weights.

**Headers**

| Header | Required | Description |
|--------|----------|-------------|
| `X-API-Key` | yes* | Application API key |
| `Authorization` | yes* | `Bearer <api_key>` |

\* Provide either `X-API-Key` or `Authorization`.

**Request body**
```json
{
  "user_id": "user_123",
  "experiment_key": "checkout-button-color"
}
```

| Field            | Required | Description |
|------------------|----------|-------------|
| `user_id`        | yes      | Stable unique user identifier |
| `experiment_key` | yes      | Experiment key within the authenticated application |

**Response `200`**

Returns a branch object:

```json
{
  "id": "018f1e2a-0002-7d8e-9f0a-1b2c3d4e5f6a",
  "experiment_id": "018f1e2a-0001-7d8e-9f0a-1b2c3d4e5f6a",
  "key": "variant-a",
  "name": "Green Button",
  "weight": 0.5,
  "metadata_json": { "color": "#22c55e" }
}
```

**Error responses**

| Status | Condition |
|--------|-----------|
| `400`  | Request body is not valid JSON |
| `401`  | API key is missing or invalid |
| `403`  | Application is inactive |
| `404`  | Experiment not found for this application |
| `409`  | Experiment is not active/eligible, or its branches are misconfigured |
| `422`  | `user_id` or `experiment_key` is missing |
| `500`  | Database or server error |

---

## Events

SDK-facing event tracking records user actions for analytics and A/B test conversion measurement. Events authenticate with the same application API key as assignment requests.

When `experiment_key` is provided, the server resolves the experiment and looks up the user's existing assignment to attribute the event to a branch. Events are still recorded if no assignment exists, but `branch_id` will be `null`.

### `POST /api/v1/events`

Records a tracking event for the authenticated application.

**Headers**

| Header | Required | Description |
|--------|----------|-------------|
| `X-API-Key` | yes* | Application API key |
| `Authorization` | yes* | `Bearer <api_key>` |

\* Provide either `X-API-Key` or `Authorization`.

**Request body**
```json
{
  "user_id": "user_123",
  "event_name": "purchase",
  "experiment_key": "checkout-button-color",
  "properties": { "amount": 49.99 }
}
```

| Field            | Required | Description |
|------------------|----------|-------------|
| `user_id`        | yes      | Stable unique user identifier |
| `event_name`     | yes      | Action name, max 64 characters |
| `experiment_key` | no       | Links the event to an experiment and enables branch attribution |
| `properties`     | no       | JSON object with extra context, max 4 KB serialized |

**Response `201`**
```json
{
  "id": "018f1e2a-0003-7d8e-9f0a-1b2c3d4e5f6a",
  "user_id": "user_123",
  "event_name": "purchase",
  "experiment_id": "018f1e2a-0001-7d8e-9f0a-1b2c3d4e5f6a",
  "branch_id": "018f1e2a-0002-7d8e-9f0a-1b2c3d4e5f6a",
  "properties": { "amount": 49.99 },
  "occurred_at": "2026-07-05T18:00:00Z"
}
```

**Error responses**

| Status | Condition |
|--------|-----------|
| `400`  | Request body is not valid JSON |
| `401`  | API key is missing or invalid |
| `403`  | Application is inactive |
| `404`  | `experiment_key` was provided but no matching experiment exists |
| `422`  | `user_id` or `event_name` is missing or invalid, or `properties` is invalid |
| `500`  | Database or server error |

---

## Admin Auth

There is no public signup flow. The first admin is bootstrapped from server environment variables, and additional admin users are invited through the protected users API. Invited admins activate their account from a one-time email link.

### `POST /api/v1/auth/login`

Signs in an admin with email/password credentials.

**Request body**
```json
{
  "email": "admin@example.com",
  "password": "correct horse battery staple"
}
```

**Response `200`**
```json
{
  "user": {
    "id": "018f1e2a-3b4c-7d8e-9f0a-1b2c3d4e5f6a",
    "email": "admin@example.com",
    "role": "admin",
    "status": "active",
    "created_at": "2026-07-04T18:00:00Z",
    "updated_at": "2026-07-04T18:00:00Z",
    "last_login_at": "2026-07-04T18:00:00Z"
  },
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "access_token_expires_at": "2026-07-04T18:15:00Z"
}
```

Sets the refresh token as an HttpOnly cookie.

**Error responses**

| Status | Condition |
|--------|-----------|
| `400`  | Request body is not valid JSON |
| `401`  | Email/password is invalid |
| `403`  | User is inactive, or invitation activation is still pending |
| `422`  | `email` or `password` is missing |
| `500`  | Server error |

---

### `POST /api/v1/auth/refresh`

Rotates the refresh cookie and issues a new access token.

**Response `200`** — same shape as the login response.

**Error responses**

| Status | Condition |
|--------|-----------|
| `401`  | Refresh token is missing, expired, or invalid |
| `403`  | User is inactive or no longer authorized |
| `500`  | Server error |

---

### `POST /api/v1/auth/logout`

Revokes the current refresh token and clears the refresh cookie.

**Response `204`** — no body.

---

### `GET /api/v1/auth/invitations/{token}`

Validates a one-time admin invitation token and returns invite details for the activation page.

**Path parameters**

| Parameter | Description |
|-----------|-------------|
| `token`   | One-time invitation token from the email link |

**Response `200`**
```json
{
  "email": "teammate@example.com",
  "expires_at": "2026-07-07T18:00:00Z"
}
```

**Error responses**

| Status | Condition |
|--------|-----------|
| `404`  | Invitation is invalid, expired, or already used |
| `422`  | Invitation token is missing |
| `500`  | Server error |

---

### `POST /api/v1/auth/invitations/activate`

Consumes an invitation token, sets the invited admin's password, activates the account, and signs the user in.

**Request body**
```json
{
  "token": "invite_token_here",
  "password": "correct horse battery staple"
}
```

**Response `200`** — same shape as the login response.

Sets the refresh token as an HttpOnly cookie.

**Error responses**

| Status | Condition |
|--------|-----------|
| `400`  | Request body is not valid JSON |
| `404`  | Invitation is invalid, expired, or already used |
| `422`  | `token` or `password` is missing, or password is too short |
| `500`  | Server error |

---

### `GET /api/v1/auth/me`

Returns the currently authenticated admin user.

**Headers**

| Header | Required | Description |
|--------|----------|-------------|
| `Authorization` | yes | `Bearer <access_token>` |

**Response `200`**
```json
{
  "id": "018f1e2a-3b4c-7d8e-9f0a-1b2c3d4e5f6a",
  "email": "admin@example.com",
  "role": "admin",
  "status": "active",
  "created_at": "2026-07-04T18:00:00Z",
  "updated_at": "2026-07-04T18:00:00Z",
  "last_login_at": "2026-07-04T18:00:00Z"
}
```

**Error responses**

| Status | Condition |
|--------|-----------|
| `401`  | Access token is missing or invalid |
| `403`  | User is inactive or forbidden |
| `500`  | Server error |

---

## Users

All `/api/v1/users` endpoints require an admin access token.

### User object

| Field           | Type     | Description |
|----------------|----------|-------------|
| `id`           | `string` | UUID |
| `email`        | `string` | Unique admin email |
| `role`         | `string` | `admin` |
| `status`       | `string` | `invited` \| `active` \| `inactive` |
| `created_at`   | `string` | ISO 8601 timestamp |
| `updated_at`   | `string` | ISO 8601 timestamp |
| `last_login_at`| `string` | ISO 8601 timestamp or `null` |

---

### `GET /api/v1/users`

Returns all admin users ordered by creation date, newest first.

**Response `200`**
```json
[
  {
    "id": "018f1e2a-3b4c-7d8e-9f0a-1b2c3d4e5f6a",
    "email": "admin@example.com",
    "role": "admin",
    "status": "active",
    "created_at": "2026-07-04T18:00:00Z",
    "updated_at": "2026-07-04T18:00:00Z",
    "last_login_at": "2026-07-04T18:00:00Z"
  }
]
```

---

### `POST /api/v1/users`

Creates an invited admin account and sends the activation email.

**Request body**
```json
{
  "email": "teammate@example.com"
}
```

**Response `201`** — returns the invited user object.

**Error responses**

| Status | Condition |
|--------|-----------|
| `400`  | Request body is not valid JSON |
| `409`  | User already exists, or an active invite already exists for that email |
| `422`  | Email is invalid or missing |
| `503`  | Invite email delivery is not configured |
| `500`  | Server error while creating or sending the invite |

---

### `PATCH /api/v1/users/{id}`

Activates or deactivates an admin user.

**Request body**
```json
{
  "status": "inactive"
}
```

**Response `200`** — returns the updated user object.

**Error responses**

| Status | Condition |
|--------|-----------|
| `400`  | Request body is not valid JSON |
| `404`  | User not found |
| `409`  | The request would deactivate the last active admin |
| `422`  | `status` is not a valid value |
| `500`  | Server error |

---

## Applications

All `/api/v1/applications/*` management routes require an admin access token.

An application is the top-level entity in Prism. Each application has a unique API key used to authenticate SDK and ingestion requests.

Applications also have a lifecycle `status`: `active` applications behave normally, while `inactive` applications remain visible but cannot create new experiments.

Records are soft-deleted: `DELETE` sets `deleted_at` rather than removing the row. Soft-deleted applications are excluded from all reads and cascade soft-delete to their experiments and branches. Assignment history is preserved.

### Application object

| Field        | Type     | Description                                      |
|--------------|----------|--------------------------------------------------|
| `id`         | `string` | UUID                                             |
| `name`       | `string` | Human-readable name                              |
| `api_key`    | `string` | Prefixed API key (`prism_...`). Returned on create. |
| `status`     | `string` | `active` \| `inactive`                           |
| `created_at` | `string` | ISO 8601 timestamp                               |
| `updated_at` | `string` | ISO 8601 timestamp                               |

---

### `GET /api/v1/applications`

Returns all applications ordered by creation date, newest first.

**Response `200`**
```json
[
  {
    "id": "018f1e2a-3b4c-7d8e-9f0a-1b2c3d4e5f6a",
    "name": "My App",
    "api_key": "prism_abc123...",
    "status": "active",
    "created_at": "2026-07-03T20:00:00Z",
    "updated_at": "2026-07-03T20:00:00Z"
  }
]
```

Returns `[]` when no applications exist.

---

### `POST /api/v1/applications`

Creates a new application and generates its API key.

> The `api_key` is only returned on this response. It cannot be retrieved again.

**Request body**
```json
{
  "name": "My App"
}
```

| Field  | Required | Description            |
|--------|----------|------------------------|
| `name` | yes      | Non-empty string       |

**Response `201`**
```json
{
  "id": "018f1e2a-3b4c-7d8e-9f0a-1b2c3d4e5f6a",
  "name": "My App",
  "api_key": "prism_abc123...",
  "status": "active",
  "created_at": "2026-07-03T20:00:00Z",
  "updated_at": "2026-07-03T20:00:00Z"
}
```

**Error responses**

| Status | Condition |
|--------|-----------|
| `400`  | Request body is not valid JSON |
| `422`  | `name` is missing or blank |
| `500`  | API key generation or database error |

---

### `GET /api/v1/applications/{id}`

Returns a single application by its UUID.

**Path parameters**

| Parameter | Description         |
|-----------|---------------------|
| `id`      | Application UUID    |

**Response `200`**
```json
{
  "id": "018f1e2a-3b4c-7d8e-9f0a-1b2c3d4e5f6a",
  "name": "My App",
  "api_key": "prism_abc123...",
  "status": "active",
  "created_at": "2026-07-03T20:00:00Z",
  "updated_at": "2026-07-03T20:00:00Z"
}
```

**Error responses**

| Status | Condition |
|--------|-----------|
| `404`  | No application with that ID |
| `500`  | Database error |

---

### `PUT /api/v1/applications/{id}`

Updates an existing application. The `api_key` cannot be changed.

**Path parameters**

| Parameter | Description         |
|-----------|---------------------|
| `id`      | Application UUID    |

**Request body**
```json
{
  "name": "Renamed App",
  "status": "inactive"
}
```

| Field    | Required | Description |
|----------|----------|-------------|
| `name`   | yes      | Non-empty string |
| `status` | no       | `active` \| `inactive`. Omit to keep the current value |

**Response `200`**
```json
{
  "id": "018f1e2a-3b4c-7d8e-9f0a-1b2c3d4e5f6a",
  "name": "Renamed App",
  "api_key": "prism_abc123...",
  "status": "inactive",
  "created_at": "2026-07-03T20:00:00Z",
  "updated_at": "2026-07-03T20:01:00Z"
}
```

**Error responses**

| Status | Condition |
|--------|-----------|
| `400`  | Request body is not valid JSON |
| `404`  | No application with that ID |
| `422`  | `name` is missing or blank, or `status` is not a valid value |
| `500`  | Database error |

---

### `DELETE /api/v1/applications/{id}`

Soft-deletes an application and cascades to its experiments and branches.

**Path parameters**

| Parameter | Description         |
|-----------|---------------------|
| `id`      | Application UUID    |

**Response `204`** — no body.

**Error responses**

| Status | Condition |
|--------|-----------|
| `404`  | No active application with that ID |
| `500`  | Database error |

---

## Error response shape

All error responses use the same structure:

```json
{
  "error": "description of the problem"
}
```

---

## Experiments

An experiment belongs to an application and represents a single A/B test. Each experiment has a unique `key` within its application (among active records), a `status` representing its lifecycle, and optional date bounds.

Soft-deleted experiments are excluded from reads. Deleting an experiment also soft-deletes its branches. Experiment keys can be reused after deletion. New experiments can only be created while the parent application is `active`.

### Experiment object

| Field            | Type     | Nullable | Description |
|------------------|----------|----------|-------------|
| `id`             | `string` | no  | UUID |
| `application_id` | `string` | no  | Parent application UUID |
| `key`            | `string` | no  | URL-safe identifier, unique per application |
| `name`           | `string` | no  | Human-readable name |
| `description`    | `string` | yes | Optional description |
| `status`         | `string` | no  | `draft` \| `active` \| `paused` \| `completed` |
| `start_date`     | `string` | yes | ISO 8601 timestamp |
| `end_date`       | `string` | yes | ISO 8601 timestamp |
| `created_at`     | `string` | no  | ISO 8601 timestamp |
| `updated_at`     | `string` | no  | ISO 8601 timestamp |

---

### `GET /api/v1/applications/{appID}/experiments`

Returns all experiments for the application, newest first. Each experiment includes its branches.

**Path parameters**

| Parameter | Description          |
|-----------|----------------------|
| `appID`   | Application UUID     |

**Response `200`**
```json
[
  {
    "id": "018f1e2a-0001-7d8e-9f0a-1b2c3d4e5f6a",
    "application_id": "018f1e2a-3b4c-7d8e-9f0a-1b2c3d4e5f6a",
    "key": "checkout-button-color",
    "name": "Checkout Button Color",
    "description": "Testing green vs blue CTA",
    "status": "active",
    "start_date": "2026-07-01T00:00:00Z",
    "end_date": null,
    "created_at": "2026-07-03T20:00:00Z",
    "updated_at": "2026-07-03T20:00:00Z",
    "branches": [
      { "id": "...", "experiment_id": "...", "key": "control",   "name": "Control",     "weight": 0.5, "metadata_json": null },
      { "id": "...", "experiment_id": "...", "key": "variant-a", "name": "Green Button","weight": 0.5, "metadata_json": null }
    ]
  }
]
```

Returns `[]` when no experiments exist. Branches are fetched in a single batched query alongside experiments.

---

### `POST /api/v1/applications/{appID}/experiments`

Creates a new experiment. Status defaults to `draft`. Optionally accepts an initial `branches` array — the experiment and all branches are inserted atomically in a single transaction.

**Path parameters**

| Parameter | Description          |
|-----------|----------------------|
| `appID`   | Application UUID     |

**Request body**
```json
{
  "key": "checkout-button-color",
  "name": "Checkout Button Color",
  "description": "Testing green vs blue CTA",
  "start_date": "2026-07-01T00:00:00Z",
  "end_date": null,
  "branches": [
    { "key": "control",   "name": "Control",      "weight": 0.5 },
    { "key": "variant-a", "name": "Green Button",  "weight": 0.5 }
  ]
}
```

| Field         | Required | Description |
|---------------|----------|-------------|
| `key`         | yes      | URL-safe identifier, unique within the application |
| `name`        | yes      | Non-empty string |
| `description` | no       | Optional text |
| `start_date`  | no       | ISO 8601 timestamp |
| `end_date`    | no       | ISO 8601 timestamp |
| `branches`    | no       | Optional array of branches to create atomically with the experiment |

**Branch object within `branches`**

| Field          | Required | Description |
|----------------|----------|-------------|
| `key`          | yes      | URL-safe identifier, unique within the experiment |
| `name`         | yes      | Non-empty string |
| `weight`       | yes      | Decimal 0–1 |
| `metadata_json`| no       | Arbitrary JSON object |

**Response `201`**
```json
{
  "id": "018f1e2a-0001-7d8e-9f0a-1b2c3d4e5f6a",
  "application_id": "018f1e2a-3b4c-7d8e-9f0a-1b2c3d4e5f6a",
  "key": "checkout-button-color",
  "name": "Checkout Button Color",
  "description": "Testing green vs blue CTA",
  "status": "draft",
  "start_date": "2026-07-01T00:00:00Z",
  "end_date": null,
  "created_at": "2026-07-03T20:00:00Z",
  "updated_at": "2026-07-03T20:00:00Z",
  "branches": [
    { "id": "...", "experiment_id": "...", "key": "control",   "name": "Control",     "weight": 0.5, "metadata_json": null },
    { "id": "...", "experiment_id": "...", "key": "variant-a", "name": "Green Button","weight": 0.5, "metadata_json": null }
  ]
}
```

**Error responses**

| Status | Condition |
|--------|-----------|
| `400`  | Request body is not valid JSON |
| `404`  | Application not found |
| `409`  | Application is inactive, or an experiment with that `key` already exists for this application |
| `422`  | `key` or `name` is missing, or branch validation fails |
| `500`  | Database error |

---

### `GET /api/v1/applications/{appID}/experiments/{id}`

Returns a single experiment. Scoped to the application — returns `404` if the experiment belongs to a different application.

**Path parameters**

| Parameter | Description          |
|-----------|----------------------|
| `appID`   | Application UUID     |
| `id`      | Experiment UUID      |

**Response `200`** — same shape as experiment object above.

**Error responses**

| Status | Condition |
|--------|-----------|
| `404`  | Experiment not found or does not belong to this application |
| `500`  | Database error |

---

### `PUT /api/v1/applications/{appID}/experiments/{id}`

Updates a experiment. `key` cannot be changed after creation.

**Path parameters**

| Parameter | Description          |
|-----------|----------------------|
| `appID`   | Application UUID     |
| `id`      | Experiment UUID      |

**Request body**
```json
{
  "name": "Updated Name",
  "description": "Updated description",
  "status": "active",
  "start_date": "2026-07-01T00:00:00Z",
  "end_date": "2026-08-01T00:00:00Z"
}
```

| Field         | Required | Description |
|---------------|----------|-------------|
| `name`        | yes      | Non-empty string |
| `description` | no       | Pass `null` to clear |
| `status`      | no       | `draft` \| `active` \| `paused` \| `completed`. Defaults to `draft` if omitted |
| `start_date`  | no       | Pass `null` to clear |
| `end_date`    | no       | Pass `null` to clear |

**Response `200`** — same shape as experiment object above with updated `updated_at`.

**Error responses**

| Status | Condition |
|--------|-----------|
| `400`  | Request body is not valid JSON |
| `404`  | Experiment not found or does not belong to this application |
| `422`  | `name` is blank or `status` is not a valid value |
| `500`  | Database error |

---

### `DELETE /api/v1/applications/{appID}/experiments/{id}`

Soft-deletes an experiment and cascades to its branches.

**Path parameters**

| Parameter | Description          |
|-----------|----------------------|
| `appID`   | Application UUID     |
| `id`      | Experiment UUID      |

**Response `204`** — no body.

**Error responses**

| Status | Condition |
|--------|-----------|
| `404`  | Experiment not found or does not belong to this application |
| `500`  | Database error |

---

### `GET /api/v1/applications/{appID}/experiments/{id}/assignments`

Returns user-to-branch assignments for an experiment.

**Path parameters**

| Parameter | Description      |
|-----------|------------------|
| `appID`   | Application UUID |
| `id`      | Experiment UUID  |

**Response `200`**
```json
{
  "experiment_id": "018f1e2a-0001-7d8e-9f0a-1b2c3d4e5f6a",
  "experiment_key": "checkout-button-color",
  "experiment_name": "Checkout Button Color",
  "experiment_status": "active",
  "assignments": [
    {
      "id": "018f1e2a-0004-7d8e-9f0a-1b2c3d4e5f6a",
      "application_id": "018f1e2a-0000-7d8e-9f0a-1b2c3d4e5f6a",
      "experiment_id": "018f1e2a-0001-7d8e-9f0a-1b2c3d4e5f6a",
      "branch_id": "018f1e2a-0002-7d8e-9f0a-1b2c3d4e5f6a",
      "user_id": "user_123",
      "assigned_at": "2026-07-05T18:00:00Z",
      "context_json": null,
      "created_at": "2026-07-05T18:00:00Z",
      "updated_at": "2026-07-05T18:00:00Z",
      "branch_key": "control",
      "branch_name": "Control",
      "branch_weight": 50
    }
  ]
}
```

**Error responses**

| Status | Condition |
|--------|-----------|
| `404`  | Experiment not found or does not belong to this application |
| `500`  | Database error |

---

### `GET /api/v1/applications/{appID}/experiments/{id}/events`

Returns tracking events recorded for an experiment.

**Path parameters**

| Parameter | Description      |
|-----------|------------------|
| `appID`   | Application UUID |
| `id`      | Experiment UUID  |

**Query parameters**

| Parameter    | Required | Description |
|--------------|----------|-------------|
| `event_name` | no       | Filter by event name |
| `limit`      | no       | Page size, default `100`, max `500` |
| `offset`     | no       | Page offset, default `0` |

**Response `200`**
```json
{
  "experiment_id": "018f1e2a-0001-7d8e-9f0a-1b2c3d4e5f6a",
  "experiment_key": "checkout-button-color",
  "experiment_name": "Checkout Button Color",
  "experiment_status": "active",
  "events": [
    {
      "id": "018f1e2a-0005-7d8e-9f0a-1b2c3d4e5f6a",
      "application_id": "018f1e2a-0000-7d8e-9f0a-1b2c3d4e5f6a",
      "experiment_id": "018f1e2a-0001-7d8e-9f0a-1b2c3d4e5f6a",
      "branch_id": "018f1e2a-0002-7d8e-9f0a-1b2c3d4e5f6a",
      "user_id": "user_123",
      "event_name": "purchase",
      "properties": { "amount": 49.99 },
      "occurred_at": "2026-07-05T18:00:00Z",
      "created_at": "2026-07-05T18:00:00Z",
      "branch_key": "control",
      "branch_name": "Control"
    }
  ]
}
```

**Error responses**

| Status | Condition |
|--------|-----------|
| `404`  | Experiment not found or does not belong to this application |
| `422`  | `limit` or `offset` is invalid |
| `500`  | Database error |

---

### `GET /api/v1/applications/{appID}/experiments/{id}/dashboard`

Returns assignment distribution for an experiment. When `event_name` is provided, each branch also includes event and conversion metrics for that event.

**Path parameters**

| Parameter | Description      |
|-----------|------------------|
| `appID`   | Application UUID |
| `id`      | Experiment UUID  |

**Query parameters**

| Parameter    | Required | Description |
|--------------|----------|-------------|
| `event_name` | no       | Include per-branch event counts and conversion rate for this event |

**Response `200`**
```json
{
  "experiment_id": "018f1e2a-0001-7d8e-9f0a-1b2c3d4e5f6a",
  "experiment_key": "checkout-button-color",
  "experiment_name": "Checkout Button Color",
  "experiment_status": "active",
  "event_name": "purchase",
  "total_assignments": 4,
  "branch_count": 2,
  "branches": [
    {
      "branch_id": "018f1e2a-0002-7d8e-9f0a-1b2c3d4e5f6a",
      "branch_key": "control",
      "branch_name": "Control",
      "configured_weight": 50,
      "assignment_count": 2,
      "assignment_share": 50,
      "event_count": 3,
      "unique_event_users": 1,
      "conversion_rate": 50
    }
  ]
}
```

When `event_name` is omitted, the response matches the assignment-only dashboard shape and omits `event_name`, `event_count`, `unique_event_users`, and `conversion_rate`.

**Error responses**

| Status | Condition |
|--------|-----------|
| `404`  | Experiment not found or does not belong to this application |
| `500`  | Database error |

---

## Branches

A branch represents a variant within an experiment. Each branch has a `key` (unique within the experiment), a display `name`, a `weight` (0–1 decimal), and an optional `metadata_json` payload for arbitrary variant configuration.

> Branches cannot be created at the same time as an experiment via the experiment `POST` endpoint. Use the branch endpoints below to manage them independently.

### Branch object

| Field           | Type     | Nullable | Description |
|-----------------|----------|----------|-------------|
| `id`            | `string` | no  | UUID |
| `experiment_id` | `string` | no  | Parent experiment UUID |
| `key`           | `string` | no  | URL-safe identifier, unique per experiment. Immutable after creation. |
| `name`          | `string` | no  | Human-readable label |
| `weight`        | `number` | no  | Traffic fraction 0–1 |
| `metadata_json` | `object` | yes | Arbitrary JSON payload |

---

### `POST /api/v1/applications/{appID}/experiments/{experimentID}/branches`

Creates a new branch on an experiment.

**Path parameters**

| Parameter      | Description          |
|----------------|----------------------|
| `appID`        | Application UUID     |
| `experimentID` | Experiment UUID      |

**Request body**
```json
{
  "key": "variant-a",
  "name": "Green Button",
  "weight": 0.5,
  "metadata_json": { "color": "#22c55e" }
}
```

| Field          | Required | Description |
|----------------|----------|-------------|
| `key`          | yes      | URL-safe identifier, unique within the experiment |
| `name`         | yes      | Non-empty string |
| `weight`       | yes      | Decimal 0–1 |
| `metadata_json`| no       | Arbitrary JSON object |

**Response `201`**
```json
{
  "id": "018f1e2a-0002-7d8e-9f0a-1b2c3d4e5f6a",
  "experiment_id": "018f1e2a-0001-7d8e-9f0a-1b2c3d4e5f6a",
  "key": "variant-a",
  "name": "Green Button",
  "weight": 0.5,
  "metadata_json": { "color": "#22c55e" }
}
```

**Error responses**

| Status | Condition |
|--------|-----------|
| `400`  | Request body is not valid JSON |
| `404`  | Experiment not found or does not belong to this application |
| `409`  | A branch with that `key` already exists for this experiment |
| `422`  | `key` or `name` is blank, or `weight` is outside 0–1 |
| `500`  | Database error |

---

### `PUT /api/v1/applications/{appID}/experiments/{experimentID}/branches/{id}`

Updates a branch. `key` cannot be changed.

**Path parameters**

| Parameter      | Description          |
|----------------|----------------------|
| `appID`        | Application UUID     |
| `experimentID` | Experiment UUID      |
| `id`           | Branch UUID          |

**Request body**
```json
{
  "name": "Blue Button",
  "weight": 0.4,
  "metadata_json": { "color": "#3b82f6" }
}
```

| Field          | Required | Description |
|----------------|----------|-------------|
| `name`         | yes      | Non-empty string |
| `weight`       | yes      | Decimal 0–1 |
| `metadata_json`| no       | Pass `null` to clear |

**Response `200`** — same shape as branch object above.

**Error responses**

| Status | Condition |
|--------|-----------|
| `400`  | Request body is not valid JSON |
| `404`  | Branch or experiment not found |
| `422`  | `name` is blank or `weight` is outside 0–1 |
| `500`  | Database error |

---

### `DELETE /api/v1/applications/{appID}/experiments/{experimentID}/branches/{id}`

Soft-deletes a branch.

**Path parameters**

| Parameter      | Description          |
|----------------|----------------------|
| `appID`        | Application UUID     |
| `experimentID` | Experiment UUID      |
| `id`           | Branch UUID          |

**Response `204`** — no body.

**Error responses**

| Status | Condition |
|--------|-----------|
| `404`  | Branch or experiment not found |
| `500`  | Database error |
