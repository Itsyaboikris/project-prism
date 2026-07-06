import { api } from "./client"

export interface TrackedEvent {
  id: string
  experiment_id: string
  key: string
  name: string
  description: string | null
  occurrence_count: number
  last_occurred_at: string | null
  created_at: string
  updated_at: string
}

export interface CreateTrackedEventInput {
  key: string
  name: string
  description?: string | null
}

export interface UpdateTrackedEventInput {
  name: string
  description?: string | null
}

export const trackedEventsApi = {
  list: (appId: string, experimentId: string) =>
    api.get<TrackedEvent[]>(
      `/api/v1/applications/${appId}/experiments/${experimentId}/tracked-events`,
    ),

  create: (appId: string, experimentId: string, input: CreateTrackedEventInput) =>
    api.post<TrackedEvent>(
      `/api/v1/applications/${appId}/experiments/${experimentId}/tracked-events`,
      input,
    ),

  update: (appId: string, experimentId: string, id: string, input: UpdateTrackedEventInput) =>
    api.put<TrackedEvent>(
      `/api/v1/applications/${appId}/experiments/${experimentId}/tracked-events/${id}`,
      input,
    ),

  delete: (appId: string, experimentId: string, id: string) =>
    api.delete<void>(
      `/api/v1/applications/${appId}/experiments/${experimentId}/tracked-events/${id}`,
    ),
}
