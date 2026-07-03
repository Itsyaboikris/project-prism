# Prism API Reference

Base URL: `http://localhost:8080`

All request and response bodies use `application/json`.

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

## Applications

An application is the top-level entity in Prism. Each application has a unique API key used to authenticate SDK and ingestion requests.

Records are soft-deleted: `DELETE` sets `deleted_at` rather than removing the row. Soft-deleted applications are excluded from all reads and cascade soft-delete to their experiments and branches. Assignment history is preserved.

### Application object

| Field        | Type     | Description                                      |
|--------------|----------|--------------------------------------------------|
| `id`         | `string` | UUID                                             |
| `name`       | `string` | Human-readable name                              |
| `api_key`    | `string` | Prefixed API key (`prism_...`). Returned on create. |
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

Updates the name of an existing application. The `api_key` cannot be changed.

**Path parameters**

| Parameter | Description         |
|-----------|---------------------|
| `id`      | Application UUID    |

**Request body**
```json
{
  "name": "Renamed App"
}
```

| Field  | Required | Description       |
|--------|----------|-------------------|
| `name` | yes      | Non-empty string  |

**Response `200`**
```json
{
  "id": "018f1e2a-3b4c-7d8e-9f0a-1b2c3d4e5f6a",
  "name": "Renamed App",
  "api_key": "prism_abc123...",
  "created_at": "2026-07-03T20:00:00Z",
  "updated_at": "2026-07-03T20:01:00Z"
}
```

**Error responses**

| Status | Condition |
|--------|-----------|
| `400`  | Request body is not valid JSON |
| `404`  | No application with that ID |
| `422`  | `name` is missing or blank |
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

Soft-deleted experiments are excluded from reads. Deleting an experiment also soft-deletes its branches. Experiment keys can be reused after deletion.

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
| `409`  | An experiment with that `key` already exists for this application |
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
