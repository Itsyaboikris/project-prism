const BASE_URL = import.meta.env.VITE_API_URL ?? "http://localhost:8080"

export class ApiError extends Error {
  status: number

  constructor(status: number, message: string) {
    super(message)
    this.status = status
    this.name = "ApiError"
  }
}

export interface RequestOptions {
  auth?: boolean
  retryOn401?: boolean
  apiKey?: string
}

type RefreshHandler = () => Promise<boolean>

let accessToken: string | null = null
let refreshHandler: RefreshHandler | null = null
let authFailureHandler: (() => void) | null = null
let inFlightRefresh: Promise<boolean> | null = null

export function setAccessToken(token: string | null) {
  accessToken = token
}

export function configureApiAuth(options: {
  refreshAccessToken?: RefreshHandler | null
  onAuthFailure?: (() => void) | null
}) {
  refreshHandler = options.refreshAccessToken ?? null
  authFailureHandler = options.onAuthFailure ?? null
}

async function request<T>(path: string, init?: RequestInit, options?: RequestOptions): Promise<T> {
  const headers = new Headers(init?.headers)
  if (init?.body !== undefined && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json")
  }
  if (options?.apiKey) {
    headers.set("X-API-Key", options.apiKey)
  } else if (options?.auth !== false && accessToken) {
    headers.set("Authorization", `Bearer ${accessToken}`)
  }

  const res = await fetch(`${BASE_URL}${path}`, {
    credentials: "include",
    ...init,
    headers,
  })

  if (res.status === 401 && options?.auth !== false && options?.retryOn401 !== false) {
    const refreshed = await refreshAccessToken()
    if (refreshed) {
      return request<T>(path, init, { ...options, retryOn401: false })
    }
    authFailureHandler?.()
  }

  if (!res.ok) {
    let message = res.statusText
    try {
      const body = await res.json()
      if (body?.error) message = body.error
    } catch {
      // ignore parse errors
    }
    throw new ApiError(res.status, message)
  }

  // 204 or empty body
  const text = await res.text()
  return (text ? JSON.parse(text) : undefined) as T
}

async function refreshAccessToken() {
  if (!refreshHandler) return false
  if (!inFlightRefresh) {
    inFlightRefresh = refreshHandler().finally(() => {
      inFlightRefresh = null
    })
  }

  return inFlightRefresh
}

export const api = {
  get: <T>(path: string, options?: RequestOptions) => request<T>(path, undefined, options),
  post: <T>(path: string, body: unknown, options?: RequestOptions) =>
    request<T>(path, { method: "POST", body: JSON.stringify(body) }, options),
  put: <T>(path: string, body: unknown, options?: RequestOptions) =>
    request<T>(path, { method: "PUT", body: JSON.stringify(body) }, options),
  patch: <T>(path: string, body: unknown, options?: RequestOptions) =>
    request<T>(path, { method: "PATCH", body: JSON.stringify(body) }, options),
  delete: <T>(path: string, options?: RequestOptions) =>
    request<T>(path, { method: "DELETE" }, options),
}
