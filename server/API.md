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

## Error response shape

All error responses use the same structure:

```json
{
  "error": "description of the problem"
}
```
