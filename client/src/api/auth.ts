import { api } from "./client"

export type UserRole = "admin"
export type UserStatus = "invited" | "active" | "inactive"

export interface AuthUser {
  id: string
  email: string
  role: UserRole
  status: UserStatus
  created_at: string
  updated_at: string
  last_login_at: string | null
}

export interface AuthSession {
  user: AuthUser
  access_token: string
  access_token_expires_at: string
}

export interface InvitationPreview {
  email: string
  expires_at: string
}

export const authApi = {
  login: (email: string, password: string) =>
    api.post<AuthSession>("/api/v1/auth/login", { email, password }, { auth: false, retryOn401: false }),

  refresh: () =>
    api.post<AuthSession>("/api/v1/auth/refresh", {}, { auth: false, retryOn401: false }),

  logout: () =>
    api.post<void>("/api/v1/auth/logout", {}, { auth: false, retryOn401: false }),

  getInvitation: (token: string) =>
    api.get<InvitationPreview>(`/api/v1/auth/invitations/${encodeURIComponent(token)}`, {
      auth: false,
      retryOn401: false,
    }),

  activateInvitation: (token: string, password: string) =>
    api.post<AuthSession>(
      "/api/v1/auth/invitations/activate",
      { token, password },
      { auth: false, retryOn401: false },
    ),

  me: () => api.get<AuthUser>("/api/v1/auth/me"),
}
