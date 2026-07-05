import { api } from "./client"
import type { ExperimentStatus } from "./experiments"

export interface ExperimentEventListItem {
  id: string
  application_id: string
  experiment_id: string | null
  branch_id: string | null
  user_id: string
  event_name: string
  properties: unknown | null
  occurred_at: string
  created_at: string
  branch_key: string | null
  branch_name: string | null
}

export interface ExperimentEventsView {
  experiment_id: string
  experiment_key: string
  experiment_name: string
  experiment_status: ExperimentStatus
  events: ExperimentEventListItem[]
}

export interface ListEventsParams {
  event_name?: string
  limit?: number
  offset?: number
}

export interface CreateEventInput {
  user_id: string
  event_name: string
  experiment_key?: string
  properties?: Record<string, unknown> | null
}

export interface CreatedEvent {
  id: string
  user_id: string
  event_name: string
  experiment_id: string | null
  branch_id: string | null
  properties: unknown
  occurred_at: string
}

function buildEventsQuery(params?: ListEventsParams) {
  if (!params) return ""
  const qs = new URLSearchParams()
  if (params.event_name) qs.set("event_name", params.event_name)
  if (params.limit != null) qs.set("limit", String(params.limit))
  if (params.offset != null) qs.set("offset", String(params.offset))
  const query = qs.toString()
  return query ? `?${query}` : ""
}

export const eventsApi = {
  listByExperiment: (appId: string, experimentId: string, params?: ListEventsParams) =>
    api.get<ExperimentEventsView>(
      `/api/v1/applications/${appId}/experiments/${experimentId}/events${buildEventsQuery(params)}`,
    ),

  create: (apiKey: string, body: CreateEventInput) =>
    api.post<CreatedEvent>("/api/v1/events", body, { auth: false, apiKey }),
}
