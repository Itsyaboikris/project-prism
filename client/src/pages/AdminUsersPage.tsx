import { useEffect, useMemo, useState } from "react"
import { ApiError } from "@/api/client"
import { type AuthUser } from "@/api/auth"
import { usersApi } from "@/api/users"
import { useAuth } from "@/auth/AuthContext"
import { Button } from "@/components/ui/button"
import { validateAdminEmail } from "@/lib/adminUsers"

export default function AdminUsersPage() {
  const { user: currentUser } = useAuth()

  const [users, setUsers] = useState<AuthUser[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const [email, setEmail] = useState("")
  const [createLoading, setCreateLoading] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)
  const [inviteSentTo, setInviteSentTo] = useState<string | null>(null)
  const [updatingUserId, setUpdatingUserId] = useState<string | null>(null)

  useEffect(() => {
    usersApi
      .list()
      .then(setUsers)
      .catch((err) => setError(err instanceof ApiError ? err.message : "Failed to load users"))
      .finally(() => setLoading(false))
  }, [])

  const emailError = useMemo(() => validateAdminEmail(email), [email])

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    if (emailError) {
      setCreateError(emailError)
      return
    }

    setCreateLoading(true)
    setCreateError(null)
    setInviteSentTo(null)

    try {
      const created = await usersApi.create(email.trim())
      setUsers((prev) => [created, ...prev])
      setInviteSentTo(created.email)
      setEmail("")
    } catch (err) {
      setCreateError(err instanceof ApiError ? err.message : "Failed to send invite")
    } finally {
      setCreateLoading(false)
    }
  }

  async function handleToggleStatus(targetUser: AuthUser) {
    const nextStatus = targetUser.status === "active" ? "inactive" : "active"
    setUpdatingUserId(targetUser.id)

    try {
      const updated = await usersApi.updateStatus(targetUser.id, nextStatus)
      setUsers((prev) => prev.map((item) => (item.id === updated.id ? updated : item)))
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to update user")
    } finally {
      setUpdatingUserId(null)
    }
  }

  return (
    <div className="min-h-screen bg-slate-50">
      <div className="mx-auto max-w-4xl px-6 py-12">
        <div>
          <h1 className="text-3xl font-semibold tracking-tight text-slate-900">Admin users</h1>
          <p className="mt-1 text-sm text-slate-500">
            Bootstrap admins can create additional admin accounts. Public signup is disabled.
          </p>
        </div>

        <form
          onSubmit={handleCreate}
          className="mt-6 rounded-xl border border-slate-200 bg-white p-6 shadow-sm"
        >
          <h2 className="text-base font-medium text-slate-900">Send admin invite</h2>
          <p className="mt-1 text-sm text-slate-500">
            Prism will email a one-time activation link so the invited admin can set their own
            password.
          </p>
          <div className="mt-4 grid gap-3 md:grid-cols-[1fr_auto]">
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="admin@example.com"
              className="rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 placeholder:text-slate-400 focus:border-slate-500 focus:outline-none focus:ring-2 focus:ring-slate-200"
              disabled={createLoading}
            />
            <Button
              type="submit"
              disabled={createLoading || Boolean(emailError)}
            >
              {createLoading ? "Sending…" : "Send invite"}
            </Button>
          </div>
          {createError && <p className="mt-2 text-sm text-red-600">{createError}</p>}
          {inviteSentTo && (
            <p className="mt-2 text-sm text-emerald-700">
              Invite sent to {inviteSentTo}. They can activate their account from the email link.
            </p>
          )}
        </form>

        <div className="mt-6 rounded-xl border border-slate-200 bg-white shadow-sm">
          <div className="border-b border-slate-200 px-6 py-4">
            <h2 className="text-base font-medium text-slate-900">Users</h2>
          </div>

          {loading && (
            <div className="px-6 py-12 text-center text-sm text-slate-400">Loading…</div>
          )}

          {!loading && error && (
            <div className="px-6 py-6 text-sm text-red-700">{error}</div>
          )}

          {!loading && !error && users.length === 0 && (
            <div className="px-6 py-12 text-center text-sm text-slate-500">
              No admin users found.
            </div>
          )}

          {!loading && !error && users.length > 0 && (
            <ul className="divide-y divide-slate-200">
              {users.map((user) => {
                const isCurrentUser = user.id === currentUser?.id
                return (
                  <li
                    key={user.id}
                    className="flex flex-col gap-4 px-6 py-4 md:flex-row md:items-center md:justify-between"
                  >
                    <div>
                      <div className="flex flex-wrap items-center gap-2">
                        <p className="font-medium text-slate-900">{user.email}</p>
                        <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-600">
                          {user.role}
                        </span>
                        <span
                          className={`rounded-full px-2 py-0.5 text-xs ${
                            user.status === "active"
                              ? "bg-emerald-100 text-emerald-700"
                              : user.status === "invited"
                                ? "bg-amber-100 text-amber-700"
                                : "bg-slate-200 text-slate-600"
                          }`}
                        >
                          {user.status}
                        </span>
                        {isCurrentUser && (
                          <span className="rounded-full bg-blue-100 px-2 py-0.5 text-xs text-blue-700">
                            You
                          </span>
                        )}
                      </div>
                      <p className="mt-1 text-xs text-slate-400">
                        {user.status === "invited"
                          ? "Activation pending"
                          : `Last login: ${
                              user.last_login_at ? new Date(user.last_login_at).toLocaleString() : "Never"
                            }`}
                      </p>
                    </div>

                    {user.status !== "invited" ? (
                      <Button
                        type="button"
                        variant={user.status === "active" ? "outline" : "default"}
                        onClick={() => void handleToggleStatus(user)}
                        disabled={updatingUserId === user.id}
                      >
                        {updatingUserId === user.id
                          ? "Saving…"
                          : user.status === "active"
                            ? "Deactivate"
                            : "Reactivate"}
                      </Button>
                    ) : (
                      <span className="text-sm text-slate-400">Awaiting activation</span>
                    )}
                  </li>
                )
              })}
            </ul>
          )}
        </div>
      </div>
    </div>
  )
}
