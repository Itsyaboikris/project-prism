import { api } from "./client"

export interface Application {
  id: string
  name: string
  api_key: string
  created_at: string
  updated_at: string
}

export const applicationsApi = {
  list: () => api.get<Application[]>("/api/v1/applications"),

  get: (id: string) => api.get<Application>(`/api/v1/applications/${id}`),

  create: (name: string) =>
    api.post<Application>("/api/v1/applications", { name }),

  update: (id: string, name: string) =>
    api.put<Application>(`/api/v1/applications/${id}`, { name }),
}
