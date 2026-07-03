import { api } from "./client"

export type ApplicationStatus = "active" | "inactive"

export const APPLICATION_STATUSES: ApplicationStatus[] = ["active", "inactive"]

export interface Application {
  id: string
  name: string
  api_key: string
  status: ApplicationStatus
  created_at: string
  updated_at: string
}

export interface UpdateApplicationInput {
  name: string
  status?: ApplicationStatus
}

export const applicationsApi = {
  list: () => api.get<Application[]>("/api/v1/applications"),

  get: (id: string) => api.get<Application>(`/api/v1/applications/${id}`),

  create: (name: string) =>
    api.post<Application>("/api/v1/applications", { name }),

  update: (id: string, input: UpdateApplicationInput) =>
    api.put<Application>(`/api/v1/applications/${id}`, input),

  delete: (id: string) =>
    api.delete<void>(`/api/v1/applications/${id}`),
}
