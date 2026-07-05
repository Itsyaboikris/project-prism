import { api } from "./client"
import type { AuthUser, UserStatus } from "./auth"

export const usersApi = {
  list: () => api.get<AuthUser[]>("/api/v1/users"),

  create: (email: string) =>
    api.post<AuthUser>("/api/v1/users", { email }),

  updateStatus: (id: string, status: UserStatus) =>
    api.patch<AuthUser>(`/api/v1/users/${id}`, { status }),
}
