import { api } from "./client"
import type { Branch, CreateBranchInput } from "./branches"

export type ExperimentStatus = "draft" | "active" | "paused" | "completed"

export const EXPERIMENT_STATUSES: ExperimentStatus[] = ["draft", "active", "paused", "completed"]

export interface Experiment {
  id: string
  application_id: string
  key: string
  name: string
  description: string | null
  status: ExperimentStatus
  start_date: string | null
  end_date: string | null
  created_at: string
  updated_at: string
  branches: Branch[]
}

export interface CreateExperimentInput {
  key: string
  name: string
  description?: string | null
  start_date?: string | null
  end_date?: string | null
  branches?: CreateBranchInput[]
}

export interface UpdateExperimentInput {
  name: string
  description?: string | null
  status?: ExperimentStatus
  start_date?: string | null
  end_date?: string | null
}

export const experimentsApi = {
  list: (appId: string) =>
    api.get<Experiment[]>(`/api/v1/applications/${appId}/experiments`),

  get: (appId: string, id: string) =>
    api.get<Experiment>(`/api/v1/applications/${appId}/experiments/${id}`),

  create: (appId: string, input: CreateExperimentInput) =>
    api.post<Experiment>(`/api/v1/applications/${appId}/experiments`, input),

  update: (appId: string, id: string, input: UpdateExperimentInput) =>
    api.put<Experiment>(`/api/v1/applications/${appId}/experiments/${id}`, input),
}
