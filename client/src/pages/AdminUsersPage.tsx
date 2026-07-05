import { useEffect, useMemo, useState } from "react"
import { Mail, Send, UserPlus } from "lucide-react"
import { toast } from "sonner"
import { ApiError } from "@/api/client"
import { type AuthUser } from "@/api/auth"
import { usersApi } from "@/api/users"
import { useAuth } from "@/auth/AuthContext"
import { EmptyState } from "@/components/EmptyState"
import { ErrorState } from "@/components/ErrorState"
import { PageHeader } from "@/components/PageHeader"
import { Badge } from "@/components/ui/badge"
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
import { TableLoading } from "@/components/PageLoading"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { validateAdminEmail } from "@/lib/adminUsers"
import { cn } from "@/lib/utils"

function userStatusBadge(status: AuthUser["status"]) {
  const styles = {
    active: "border-emerald-500/20 bg-emerald-500/10 text-emerald-400",
    invited: "border-amber-500/20 bg-amber-500/10 text-amber-400",
    inactive: "border-border bg-muted/50 text-muted-foreground",
  } as const

  return (
    <Badge variant="outline" className={cn("capitalize", styles[status])}>
      {status}
    </Badge>
  )
}

export default function AdminUsersPage() {
  const { user: currentUser } = useAuth()

  const [users, setUsers] = useState<AuthUser[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const [email, setEmail] = useState("")
  const [createLoading, setCreateLoading] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)
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

    try {
      const created = await usersApi.create(email.trim())
      setUsers((prev) => [created, ...prev])
      toast.success(`Invite sent to ${created.email}`)
      setEmail("")
    } catch (err) {
      const message = err instanceof ApiError ? err.message : "Failed to send invite"
      setCreateError(message)
      toast.error(message)
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
      toast.success("Status updated")
    } catch (err) {
      const message = err instanceof ApiError ? err.message : "Failed to update user"
      setError(message)
      toast.error(message)
    } finally {
      setUpdatingUserId(null)
    }
  }

  return (
    <>
      <PageHeader
        title="Team"
        description="Invite admins and manage access to the Prism console."
      />

      <Card className="mb-6">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <UserPlus className="size-4" />
            Invite admin
          </CardTitle>
          <CardDescription>
            Prism emails a one-time activation link so the invited admin can set their password.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleCreate} className="flex flex-col gap-4 sm:flex-row sm:items-end">
            <div className="flex-1 space-y-2">
              <Label htmlFor="invite-email">Email</Label>
              <Input
                id="invite-email"
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="admin@example.com"
                disabled={createLoading}
              />
            </div>
            <Button type="submit" disabled={createLoading || Boolean(emailError)}>
              <Send />
              {createLoading ? "Sending…" : "Send invite"}
            </Button>
          </form>
          {createError && <p className="mt-3 text-sm text-destructive">{createError}</p>}
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="border-b">
          <CardTitle>Members</CardTitle>
          <CardDescription>{users.length} admin accounts</CardDescription>
        </CardHeader>
        <CardContent className="p-0">
          {loading && <TableLoading rows={3} />}

          {!loading && error && (
            <div className="p-6">
              <ErrorState message={error} />
            </div>
          )}

          {!loading && !error && users.length === 0 && (
            <div className="p-6">
              <EmptyState
                icon={Mail}
                title="No admin users"
                description="Invite your first team member using the form above."
              />
            </div>
          )}

          {!loading && !error && users.length > 0 && (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Email</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="hidden md:table-cell">Last login</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {users.map((user) => {
                  const isCurrentUser = user.id === currentUser?.id
                  return (
                    <TableRow key={user.id}>
                      <TableCell>
                        <div className="flex flex-wrap items-center gap-2">
                          <span className="font-medium">{user.email}</span>
                          {isCurrentUser && (
                            <Badge variant="secondary" className="text-xs">
                              You
                            </Badge>
                          )}
                        </div>
                      </TableCell>
                      <TableCell>{userStatusBadge(user.status)}</TableCell>
                      <TableCell className="hidden text-muted-foreground md:table-cell">
                        {user.status === "invited"
                          ? "Pending activation"
                          : user.last_login_at
                            ? new Date(user.last_login_at).toLocaleString()
                            : "Never"}
                      </TableCell>
                      <TableCell className="text-right">
                        {user.status !== "invited" ? (
                          <Button
                            type="button"
                            variant={user.status === "active" ? "outline" : "default"}
                            size="sm"
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
                          <span className="text-sm text-muted-foreground">Awaiting activation</span>
                        )}
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </>
  )
}
