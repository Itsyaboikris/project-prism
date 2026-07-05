import { useEffect, useMemo, useState } from "react"
import { Navigate, useNavigate, useSearchParams } from "react-router-dom"
import { FlaskConical } from "lucide-react"
import { authApi } from "@/api/auth"
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
import { Skeleton } from "@/components/ui/skeleton"
import { validateAdminPassword } from "@/lib/adminUsers"

export default function ActivateInvitePage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const token = searchParams.get("token") ?? ""
  const { activateInvitation, isAuthenticated, loading } = useAuth()

  const [email, setEmail] = useState<string | null>(null)
  const [inviteLoading, setInviteLoading] = useState(true)
  const [inviteError, setInviteError] = useState<string | null>(null)

  const [password, setPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [submitting, setSubmitting] = useState(false)
  const [submitError, setSubmitError] = useState<string | null>(null)

  useEffect(() => {
    if (!token) {
      setInviteError("Invitation token is missing.")
      setInviteLoading(false)
      return
    }

    authApi
      .getInvitation(token)
      .then((invitation) => setEmail(invitation.email))
      .catch((err) =>
        setInviteError(err instanceof ApiError ? err.message : "Failed to load invitation"),
      )
      .finally(() => setInviteLoading(false))
  }, [token])

  const passwordError = useMemo(() => validateAdminPassword(password), [password])
  const confirmPasswordError = useMemo(() => {
    if (!confirmPassword) return "Please confirm your password."
    if (confirmPassword !== password) return "Passwords do not match."
    return null
  }, [confirmPassword, password])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (passwordError || confirmPasswordError) {
      setSubmitError(passwordError ?? confirmPasswordError)
      return
    }

    setSubmitting(true)
    setSubmitError(null)

    try {
      await activateInvitation(token, password)
      navigate("/applications", { replace: true })
    } catch (err) {
      setSubmitError(err instanceof ApiError ? err.message : "Failed to activate invitation")
    } finally {
      setSubmitting(false)
    }
  }

  if (!loading && isAuthenticated) {
    return <Navigate to="/applications" replace />
  }

  return (
    <div className="auth-shell">
      <div className="auth-card">
        <Card>
          <CardHeader className="text-center">
            <div className="mx-auto mb-2 flex size-10 items-center justify-center rounded-xl bg-primary text-primary-foreground">
              <FlaskConical className="size-5" />
            </div>
            <CardTitle className="text-xl">Activate account</CardTitle>
            <CardDescription>Set your password to join the Prism admin console.</CardDescription>
          </CardHeader>
          <CardContent>
            {inviteLoading && (
              <div className="space-y-3">
                <Skeleton className="h-10 w-full" />
                <Skeleton className="h-10 w-full" />
              </div>
            )}

            {!inviteLoading && inviteError && (
              <div className="rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
                {inviteError}
              </div>
            )}

            {!inviteLoading && !inviteError && (
              <>
                <div className="mb-4 rounded-lg border bg-muted/40 px-3 py-2 text-sm">
                  <span className="text-muted-foreground">Invited as </span>
                  <span className="font-medium">{email}</span>
                </div>

                <form onSubmit={handleSubmit} className="space-y-4">
                  <div className="space-y-2">
                    <Label htmlFor="password">Password</Label>
                    <Input
                      id="password"
                      type="password"
                      value={password}
                      onChange={(e) => setPassword(e.target.value)}
                      autoComplete="new-password"
                      disabled={submitting}
                    />
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="confirm-password">Confirm password</Label>
                    <Input
                      id="confirm-password"
                      type="password"
                      value={confirmPassword}
                      onChange={(e) => setConfirmPassword(e.target.value)}
                      autoComplete="new-password"
                      disabled={submitting}
                    />
                  </div>

                  {submitError && (
                    <div className="rounded-lg border border-destructive/30 bg-destructive/10 px-3 py-2 text-sm text-destructive">
                      {submitError}
                    </div>
                  )}

                  <Button
                    type="submit"
                    className="w-full"
                    size="lg"
                    disabled={submitting || Boolean(passwordError) || Boolean(confirmPasswordError)}
                  >
                    {submitting ? "Activating…" : "Activate account"}
                  </Button>
                </form>
              </>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
