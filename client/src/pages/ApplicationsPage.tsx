import { useEffect, useRef, useState } from "react"
import { useNavigate } from "react-router-dom"
import { ChevronRight, LayoutGrid, Plus } from "lucide-react"
import { toast } from "sonner"
import { applicationsApi, type Application } from "@/api/applications"
import { ApiError } from "@/api/client"
import { ApplicationStatusBadge } from "@/components/ApplicationStatusBadge"
import { EmptyState, EmptyStateButton } from "@/components/EmptyState"
import { ErrorState } from "@/components/ErrorState"
import { PageHeader } from "@/components/PageHeader"
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
import { APPLICATION_NAME_MAX_LENGTH, validateApplicationName } from "@/lib/applicationName"

export default function ApplicationsPage() {
  const navigate = useNavigate()
  const [apps, setApps] = useState<Application[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const [creating, setCreating] = useState(false)
  const [newName, setNewName] = useState("")
  const [createError, setCreateError] = useState<string | null>(null)
  const [createLoading, setCreateLoading] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    applicationsApi
      .list()
      .then(setApps)
      .catch((err) => setError(err instanceof ApiError ? err.message : "Failed to load applications"))
      .finally(() => setLoading(false))
  }, [])

  function openCreateForm() {
    setCreating(true)
    setNewName("")
    setCreateError(null)
    setTimeout(() => inputRef.current?.focus(), 0)
  }

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    const nameError = validateApplicationName(newName)
    if (nameError) {
      setCreateError(nameError)
      return
    }
    setCreateLoading(true)
    setCreateError(null)
    try {
      const app = await applicationsApi.create(newName.trim())
      setApps((prev) => [app, ...prev])
      setCreating(false)
      setNewName("")
      toast.success("Application created")
    } catch (err) {
      const message = err instanceof ApiError ? err.message : "Failed to create application"
      setCreateError(message)
      toast.error(message)
    } finally {
      setCreateLoading(false)
    }
  }

  return (
    <>
      <PageHeader
        title="Applications"
        description="Manage SDK integrations. Each application has a unique API key for ingestion."
        actions={
          !creating ? (
            <Button onClick={openCreateForm}>
              <Plus />
              New application
            </Button>
          ) : undefined
        }
      />

      {creating && (
        <Card className="mb-6">
          <CardHeader>
            <CardTitle>Create application</CardTitle>
            <CardDescription>Choose a name your team will recognize in the dashboard.</CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleCreate} className="flex flex-col gap-4 sm:flex-row sm:items-end">
              <div className="flex-1 space-y-2">
                <Label htmlFor="app-name">Name</Label>
                <Input
                  ref={inputRef}
                  id="app-name"
                  value={newName}
                  onChange={(e) => setNewName(e.target.value)}
                  placeholder="My product"
                  maxLength={APPLICATION_NAME_MAX_LENGTH}
                  disabled={createLoading}
                />
              </div>
              <div className="flex gap-2">
                <Button
                  type="submit"
                  disabled={createLoading || Boolean(validateApplicationName(newName))}
                >
                  {createLoading ? "Creating…" : "Create"}
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => setCreating(false)}
                  disabled={createLoading}
                >
                  Cancel
                </Button>
              </div>
            </form>
            {createError && <p className="mt-3 text-sm text-destructive">{createError}</p>}
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader className="border-b">
          <CardTitle>All applications</CardTitle>
          <CardDescription>{apps.length} registered</CardDescription>
        </CardHeader>
        <CardContent className="p-0">
          {loading && <TableLoading rows={4} />}

          {!loading && error && (
            <div className="p-6">
              <ErrorState message={error} />
            </div>
          )}

          {!loading && !error && apps.length === 0 && (
            <div className="p-6">
              <EmptyState
                icon={LayoutGrid}
                title="No applications yet"
                description="Create an application to get an API key for SDK and ingestion."
                action={<EmptyStateButton onClick={openCreateForm}>Create application</EmptyStateButton>}
              />
            </div>
          )}

          {!loading && !error && apps.length > 0 && (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="hidden md:table-cell">ID</TableHead>
                  <TableHead className="text-right">Created</TableHead>
                  <TableHead className="w-10" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {apps.map((app) => (
                  <TableRow
                    key={app.id}
                    className="cursor-pointer"
                    onClick={() => navigate(`/applications/${app.id}`)}
                  >
                    <TableCell>
                      <span className="font-medium">{app.name}</span>
                    </TableCell>
                    <TableCell>
                      <ApplicationStatusBadge status={app.status} />
                    </TableCell>
                    <TableCell className="hidden font-mono text-xs text-muted-foreground md:table-cell">
                      {app.id}
                    </TableCell>
                    <TableCell className="text-right text-muted-foreground">
                      {new Date(app.created_at).toLocaleDateString()}
                    </TableCell>
                    <TableCell>
                      <ChevronRight className="size-4 text-muted-foreground" />
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </>
  )
}
