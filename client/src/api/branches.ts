import { api } from "./client"

export interface Branch {
  id: string
  experiment_id: string
  key: string
  name: string
  weight: number
  metadata_json: unknown | null
}

export interface CreateBranchInput {
  key: string
  name: string
  weight: number
  metadata_json?: unknown | null
}

export interface UpdateBranchInput {
  name: string
  weight: number
  metadata_json?: unknown | null
}

export const branchesApi = {
  create: (appId: string, experimentId: string, input: CreateBranchInput) =>
    api.post<Branch>(`/api/v1/applications/${appId}/experiments/${experimentId}/branches`, input),

  update: (appId: string, experimentId: string, id: string, input: UpdateBranchInput) =>
    api.put<Branch>(`/api/v1/applications/${appId}/experiments/${experimentId}/branches/${id}`, input),

  delete: (appId: string, experimentId: string, id: string) =>
    api.delete<void>(`/api/v1/applications/${appId}/experiments/${experimentId}/branches/${id}`),
}
