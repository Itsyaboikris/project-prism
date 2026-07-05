import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react"
import { authApi, type AuthSession, type AuthUser } from "@/api/auth"
import { configureApiAuth, setAccessToken } from "@/api/client"
import { AuthContext, type AuthContextValue } from "./AuthContext"

let bootstrapRefreshPromise: Promise<boolean> | null = null

function runBootstrapRefresh(refreshSession: () => Promise<boolean>) {
  if (!bootstrapRefreshPromise) {
    bootstrapRefreshPromise = refreshSession().finally(() => {
      bootstrapRefreshPromise = null
    })
  }

  return bootstrapRefreshPromise
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(null)
  const [loading, setLoading] = useState(true)

  const applySession = useCallback((session: AuthSession) => {
    setAccessToken(session.access_token)
    setUser(session.user)
  }, [])

  const clearSession = useCallback(() => {
    setAccessToken(null)
    setUser(null)
  }, [])

  const refreshSession = useCallback(async () => {
    try {
      const session = await authApi.refresh()
      applySession(session)
      return true
    } catch {
      clearSession()
      return false
    }
  }, [applySession, clearSession])

  useEffect(() => {
    configureApiAuth({
      refreshAccessToken: refreshSession,
      onAuthFailure: clearSession,
    })

    return () => {
      configureApiAuth({
        refreshAccessToken: null,
        onAuthFailure: null,
      })
    }
  }, [clearSession, refreshSession])

  useEffect(() => {
    let cancelled = false

    async function bootstrap() {
      try {
        const refreshed = await runBootstrapRefresh(refreshSession)
        if (!refreshed) return

        const currentUser = await authApi.me()
        if (!cancelled) {
          setUser(currentUser)
        }
      } catch {
        if (!cancelled) {
          clearSession()
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    bootstrap()

    return () => {
      cancelled = true
    }
  }, [clearSession, refreshSession])

  const login = useCallback(async (email: string, password: string) => {
    const session = await authApi.login(email, password)
    applySession(session)
  }, [applySession])

  const activateInvitation = useCallback(async (token: string, password: string) => {
    const session = await authApi.activateInvitation(token, password)
    applySession(session)
  }, [applySession])

  const logout = useCallback(async () => {
    try {
      await authApi.logout()
    } finally {
      clearSession()
    }
  }, [clearSession])

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      loading,
      isAuthenticated: Boolean(user),
      login,
      activateInvitation,
      logout,
    }),
    [user, loading, login, activateInvitation, logout],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
