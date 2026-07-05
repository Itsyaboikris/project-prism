# Prism Frontend Integration Guide

This guide is for engineers integrating Prism into their own frontend or application stack.

It is intentionally implementation-focused. For the full backend contract, see `server/API.md`.

## Overview

Prism v1 exposes one intended SDK-facing endpoint:

- `POST /api/v1/assign`

This endpoint returns the branch a user should see for a specific experiment.

The assignment is:

- authenticated by application API key
- deterministic for a given `user_id`
- sticky across repeated requests for the same user and experiment

## When To Call Prism

Call Prism when your application needs to decide which branch of an experiment a user should receive.

Common patterns:

- on page load before rendering experiment-controlled UI
- when mounting a feature or route that is experiment-gated
- on the server side during SSR or edge rendering
- in a backend-for-frontend or edge function that your frontend calls

## Prerequisites

Before another team can test integration, make sure you provide them:

- the Prism base URL, for example `http://localhost:8080`
- an application API key
- the experiment key they should request
- at least one active experiment with active branches configured in Prism

## Endpoint

### `POST /api/v1/assign`

Assigns a user to a branch for a single experiment.

### Headers

Provide one of:

- `X-API-Key: <api_key>`
- `Authorization: Bearer <api_key>`

### Request Body

```json
{
  "user_id": "user_123",
  "experiment_key": "checkout-button-color"
}
```

Fields:

- `user_id`: required stable unique identifier for the user
- `experiment_key`: required experiment key configured in Prism

## Response

Success returns a branch object:

```json
{
  "id": "018f1e2a-0002-7d8e-9f0a-1b2c3d4e5f6a",
  "experiment_id": "018f1e2a-0001-7d8e-9f0a-1b2c3d4e5f6a",
  "key": "variant-a",
  "name": "Green Button",
  "weight": 0.5,
  "metadata_json": {
    "color": "#22c55e"
  }
}
```

Important fields:

- `key`: the stable branch key to use in application logic
- `name`: display-friendly branch name
- `metadata_json`: optional structured payload that can drive branch-specific behavior

## Sticky Assignment Behavior

Prism persists assignments.

That means:

- the first request for a given `user_id` and `experiment_key` creates an assignment
- later requests for the same user and experiment return the same branch

This is why choosing a stable `user_id` matters.

Good `user_id` examples:

- authenticated internal user id
- stable customer id
- durable anonymous id stored in cookie or local storage

Avoid:

- request ids
- session ids that rotate frequently
- timestamps
- random ids generated on every page load

## Assignment Rules

Prism will only assign a branch when:

- the API key belongs to a valid application
- the application is `active`
- the experiment exists under that application
- the experiment is `active`
- the experiment is within its start and end date window, if configured
- the experiment has active branches with valid weights

## Error Handling

Expected responses:

- `400`: invalid JSON body
- `401`: missing or invalid API key
- `403`: application is inactive
- `404`: experiment not found for that application
- `409`: experiment is not eligible for assignment, or branches are misconfigured
- `422`: `user_id` or `experiment_key` missing
- `500`: server or database error

## Minimal Browser Example

```ts
const response = await fetch("http://localhost:8080/api/v1/assign", {
  method: "POST",
  headers: {
    "Content-Type": "application/json",
    "X-API-Key": PRISM_API_KEY,
  },
  body: JSON.stringify({
    user_id: userId,
    experiment_key: "checkout-button-color",
  }),
})

if (!response.ok) {
  throw new Error(`Prism assignment failed: ${response.status}`)
}

const branch = await response.json()
```

## React Example

```ts
type Branch = {
  id: string
  experiment_id: string
  key: string
  name: string
  weight: number
  metadata_json: unknown | null
}

export async function assignBranch(userId: string): Promise<Branch> {
  const response = await fetch("http://localhost:8080/api/v1/assign", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-API-Key": PRISM_API_KEY,
    },
    body: JSON.stringify({
      user_id: userId,
      experiment_key: "checkout-button-color",
    }),
  })

  if (!response.ok) {
    throw new Error(`Prism assignment failed: ${response.status}`)
  }

  return response.json()
}
```

Then use the branch result to:

- choose a component variant
- switch copy or styling
- enable or disable a feature
- pass `metadata_json` into rendering logic

## Recommended Integration Pattern

For testing, direct browser calls are acceptable.

For production, prefer one of these:

- call Prism from your backend
- call Prism from an edge function
- call Prism from a backend-for-frontend

Why:

- exposing the API key directly in browser code makes it visible to end users
- a backend proxy gives you better control over auth, caching, retries, and observability

## Local Seed Project

Migration `000011_seed_sample_project` inserts a demo application for local integration testing.

| Field | Value |
|-------|-------|
| Application | Demo Store |
| API key | `prism_demo_api_key` |
| Experiment key | `checkout-button-color` |
| Branches | `control` (50%), `variant-a` (50%) |
| Control users | `user_001` through `user_005` |
| Variant-a users | `user_006` through `user_008` |

After running `make migrate-up`, you can use these values immediately without creating anything in the admin UI.

## Browser Integration Test Page

The repo includes a small browser test harness:

- `test.html`
- `test.js`

Prerequisites:

1. Start Postgres and apply migrations: `make db-up && make migrate-up`
2. Start the Prism server: `make server-dev`
3. Serve the test page from an allowed CORS origin

```bash
# From the repo root, serve test.html on an allowed origin
python3 -m http.server 5500 --bind 127.0.0.1
```

Then open `http://127.0.0.1:5500/test.html`.

The page is pre-filled with the seeded demo project values. You can:

- click **Assign Branch** to manually test a single user
- use the seed user shortcuts for `user_001` (control) and `user_006` (variant-a)
- click **Run Seed Tests** to execute the automated checklist below against the demo data

If you assign `user_006`, the page background should switch to the green theme from `metadata_json`.

## Testing Checklist

Use this checklist when handing Prism to another team:

1. Confirm the team has the correct base URL and API key.
2. Confirm the experiment exists and is `active`.
3. Call `POST /api/v1/assign` with a known `user_id`.
4. Verify the response returns a branch.
5. Call the same request again with the same `user_id`.
6. Verify the same branch is returned again.
7. Try a different `user_id` and verify assignment still succeeds.
8. Test invalid API key handling and confirm `401`.
9. Test inactive application handling and confirm `403`.
10. Test a missing experiment key and confirm `404`.

For local development, `test.html` automates items 3 through 8 against the seeded demo project.

## Suggested Handoff Package

When giving this to another engineer or team, send them:

- this file: `INTEGRATION_GUIDE.md`
- the contract reference: `server/API.md`
- the browser test harness: `test.html` and `test.js`
- the base URL for the target environment
- the API key for their application
- the experiment key they should use
- one or two expected branch keys, so they know what success looks like

## Current v1 Scope

In Prism v1, `POST /api/v1/assign` is the intended public integration surface.

Administrative CRUD endpoints for applications, experiments, and branches also exist on the server, but they are part of the Prism admin surface rather than the recommended external integration path.
