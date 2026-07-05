import { useEffect, useMemo, useState } from "react"
import { Navigate, useNavigate, useSearchParams } from "react-router-dom"
import { authApi } from "@/api/auth"
import { ApiError } from "@/api/client"
import { useAuth } from "@/auth/AuthContext"
import { Button } from "@/components/ui/button"
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
    <div className="flex min-h-screen items-center justify-center bg-slate-50 px-6 py-12">
      <div className="w-full max-w-md rounded-2xl border border-slate-200 bg-white p-8 shadow-sm">
        <h1 className="text-2xl font-semibold tracking-tight text-slate-900">Activate admin account</h1>
        <p className="mt-2 text-sm text-slate-500">
          Set your password to finish joining the Prism admin console.
        </p>

        {inviteLoading && <p className="mt-6 text-sm text-slate-400">Loading invitation…</p>}

        {!inviteLoading && inviteError && (
          <div className="mt-6 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
            {inviteError}
          </div>
        )}

        {!inviteLoading && !inviteError && (
          <>
            <div className="mt-6 rounded-lg border border-slate-200 bg-slate-50 px-3 py-2 text-sm text-slate-600">
              Invited email: <span className="font-medium text-slate-900">{email}</span>
            </div>

            <form onSubmit={handleSubmit} className="mt-6 space-y-4">
              <div>
                <label className="text-sm font-medium text-slate-700">Password</label>
                <input
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  autoComplete="new-password"
                  className="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 placeholder:text-slate-400 focus:border-slate-500 focus:outline-none focus:ring-2 focus:ring-slate-200"
                  disabled={submitting}
                />
              </div>

              <div>
                <label className="text-sm font-medium text-slate-700">Confirm password</label>
                <input
                  type="password"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  autoComplete="new-password"
                  className="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 placeholder:text-slate-400 focus:border-slate-500 focus:outline-none focus:ring-2 focus:ring-slate-200"
                  disabled={submitting}
                />
              </div>

              {submitError && (
                <div className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
                  {submitError}
                </div>
              )}

              <Button
                type="submit"
                size="lg"
                className="w-full"
                disabled={submitting || Boolean(passwordError) || Boolean(confirmPasswordError)}
              >
                {submitting ? "Activating…" : "Activate account"}
              </Button>
            </form>
          </>
        )}
      </div>
    </div>
  )
}
