import { useMemo, useState } from "react"
import { Navigate, useLocation, useNavigate } from "react-router-dom"
import { FlaskConical } from "lucide-react"
import { ApiError } from "@/api/client"
import { useAuth } from "@/auth/AuthContext"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { validateAdminEmail, validateAdminPassword } from "@/lib/adminUsers"

export default function LoginPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const { login, isAuthenticated, loading } = useAuth()

  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fromPath = useMemo(() => {
    const state = location.state as { from?: { pathname?: string } } | null
    return state?.from?.pathname || "/applications"
  }, [location.state])

  if (!loading && isAuthenticated) {
    return <Navigate to={fromPath} replace />
  }

  const emailError = validateAdminEmail(email)
  const passwordError = validateAdminPassword(password)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (emailError || passwordError) {
      setError(emailError ?? passwordError)
      return
    }

    setSubmitting(true)
    setError(null)

    try {
      await login(email.trim(), password)
      navigate(fromPath, { replace: true })
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to sign in")
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="auth-shell">
      <div className="auth-card">
        <Card>
          <CardHeader className="text-center">
            <div className="mx-auto mb-2 flex size-10 items-center justify-center rounded-xl bg-primary text-primary-foreground">
              <FlaskConical className="size-5" />
            </div>
            <CardTitle className="text-xl">Welcome back</CardTitle>
            <CardDescription>
              Sign in to manage experiments and applications.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="email">Email</Label>
                <Input
                  id="email"
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  autoComplete="email"
                  disabled={submitting}
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="password">Password</Label>
                <Input
                  id="password"
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  autoComplete="current-password"
                  disabled={submitting}
                />
              </div>

              {error && (
                <div className="rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
                  {error}
                </div>
              )}

              <Button
                type="submit"
                className="w-full"
                size="lg"
                disabled={submitting || Boolean(emailError) || Boolean(passwordError)}
              >
                {submitting ? "Signing in…" : "Sign in"}
              </Button>
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
