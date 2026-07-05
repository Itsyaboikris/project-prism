import { Navigate, Outlet, useLocation } from "react-router-dom"
import { useAuth } from "./AuthContext"

export default function RequireAdmin() {
  const { user, loading, isAuthenticated } = useAuth()
  const location = useLocation()

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background text-sm text-muted-foreground">
        Restoring session…
      </div>
    )
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace state={{ from: location }} />
  }

  if (user?.role !== "admin") {
    return (
      <div className="auth-shell">
        <div className="auth-card text-center">
          <h1 className="text-xl font-semibold text-foreground">Access denied</h1>
          <p className="mt-2 text-sm text-muted-foreground">
            Your account does not have permission to access the Prism admin console.
          </p>
        </div>
      </div>
    )
  }

  return <Outlet />
}
