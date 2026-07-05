import { useEffect, useState } from "react"
import { Link, useNavigate, useParams } from "react-router-dom"
import {
  AlertTriangle,
  Check,
  Copy,
  FlaskConical,
  Key,
  Pencil,
  Plus,
  Settings,
  Trash2,
} from "lucide-react"
import { toast } from "sonner"
import { applicationsApi, type Application } from "@/api/applications"
import { experimentsApi, type Experiment } from "@/api/experiments"
import { ApiError } from "@/api/client"
import { ApplicationStatusToggle } from "@/components/ApplicationStatusToggle"
import { ConfirmDeleteDialog } from "@/components/ConfirmDeleteDialog"
import { EmptyState, EmptyStateButton } from "@/components/EmptyState"
import { ErrorState } from "@/components/ErrorState"
import { ExperimentStatusToggle } from "@/components/ExperimentStatusToggle"
import { PageHeader } from "@/components/PageHeader"
import { PageLoading } from "@/components/PageLoading"
import { StatusBadge } from "@/components/StatusBadge"
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { APPLICATION_NAME_MAX_LENGTH, validateApplicationName } from "@/lib/applicationName"

export default function ApplicationDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const [app, setApp] = useState<Application | null>(null)
  const [experiments, setExperiments] = useState<Experiment[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState("overview")

  const [editing, setEditing] = useState(false)
  const [editName, setEditName] = useState("")
  const [editLoading, setEditLoading] = useState(false)
  const [editError, setEditError] = useState<string | null>(null)

  const [statusLoading, setStatusLoading] = useState(false)
  const [togglingExperimentId, setTogglingExperimentId] = useState<string | null>(null)

  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [deleteLoading, setDeleteLoading] = useState(false)
  const [deleteError, setDeleteError] = useState<string | null>(null)
  const [keyCopied, setKeyCopied] = useState(false)

  useEffect(() => {
    if (!id) return

    Promise.all([applicationsApi.get(id), experimentsApi.list(id)])
      .then(([appData, experimentData]) => {
        setApp(appData)
        setEditName(appData.name)
        setExperiments(experimentData)
      })
      .catch((err) => {
        setError(
          err instanceof ApiError && err.status === 404
            ? "Application not found."
            : err instanceof ApiError
              ? err.message
              : "Failed to load application",
        )
      })
      .finally(() => setLoading(false))
  }, [id])

  async function handleSave(e: React.FormEvent) {
    e.preventDefault()
    if (!app) return
    const nameError = validateApplicationName(editName)
    if (nameError) {
      setEditError(nameError)
      return
    }

    setEditLoading(true)
    setEditError(null)
    try {
      const updated = await applicationsApi.update(app.id, { name: editName.trim() })
      setApp(updated)
      setEditing(false)
      toast.success("Changes saved")
    } catch (err) {
      const message = err instanceof ApiError ? err.message : "Failed to update application"
      setEditError(message)
      toast.error(message)
    } finally {
      setEditLoading(false)
    }
  }

  async function handleStatusToggle() {
    if (!app || statusLoading) return

    const previousStatus = app.status
    const nextStatus = previousStatus === "active" ? "inactive" : "active"

    setStatusLoading(true)
    setApp({ ...app, status: nextStatus })

    try {
      const updated = await applicationsApi.update(app.id, {
        name: app.name,
        status: nextStatus,
      })
      setApp(updated)
      toast.success("Status updated")
    } catch (err) {
      setApp({ ...app, status: previousStatus })
      toast.error(err instanceof ApiError ? err.message : "Failed to update status")
    } finally {
      setStatusLoading(false)
    }
  }

  async function handleExperimentToggle(experiment: Experiment) {
    if (!id || togglingExperimentId || experiment.status === "completed") return

    const previousStatus = experiment.status
    const nextStatus = previousStatus === "active" ? "paused" : "active"

    setTogglingExperimentId(experiment.id)
    setExperiments((current) =>
      current.map((item) =>
        item.id === experiment.id ? { ...item, status: nextStatus } : item,
      ),
    )

    try {
      const updated = await experimentsApi.update(id, experiment.id, {
        name: experiment.name,
        description: experiment.description,
        status: nextStatus,
        start_date: experiment.start_date,
        end_date: experiment.end_date,
      })
      setExperiments((current) =>
        current.map((item) => (item.id === updated.id ? updated : item)),
      )
      toast.success("Status updated")
    } catch (err) {
      setExperiments((current) =>
        current.map((item) =>
          item.id === experiment.id ? { ...item, status: previousStatus } : item,
        ),
      )
      toast.error(err instanceof ApiError ? err.message : "Failed to update experiment status")
    } finally {
      setTogglingExperimentId(null)
    }
  }

  function handleCopyKey() {
    if (!app?.api_key) return
    navigator.clipboard.writeText(app.api_key).then(() => {
      setKeyCopied(true)
      toast.success("API key copied")
      setTimeout(() => setKeyCopied(false), 2000)
    })
  }

  async function handleDelete() {
    if (!app) return

    setDeleteLoading(true)
    setDeleteError(null)
    try {
      await applicationsApi.delete(app.id)
      toast.success(`Deleted ${app.name}`)
      navigate("/applications")
    } catch (err) {
      const message = err instanceof ApiError ? err.message : "Failed to delete application"
      setDeleteError(message)
      toast.error(message)
      setDeleteLoading(false)
    }
  }

  function openCreateExperiment() {
    navigate(`/applications/${app!.id}/experiments/new`)
  }

  return (
    <>
      {loading && <PageLoading rows={4} />}
      {!loading && error && <ErrorState message={error} />}

      {!loading && !error && app && (
        <>
          <PageHeader
            title={app.name}
            description="Application settings, experiments, and API credentials."
            breadcrumbs={[
              { label: "Applications", href: "/applications" },
              { label: app.name },
            ]}
            actions={
              !editing ? (
                <div className="flex items-center gap-3">
                  <ApplicationStatusToggle
                    status={app.status}
                    disabled={statusLoading}
                    onToggle={handleStatusToggle}
                  />
                  <Button
                    variant="outline"
                    onClick={() => setEditing(true)}
                    disabled={statusLoading}
                  >
                    <Pencil />
                    Rename
                  </Button>
                </div>
              ) : undefined
            }
          />

          <Tabs value={activeTab} onValueChange={setActiveTab}>
            <TabsList>
              <TabsTrigger value="overview">
                <Settings className="size-4" />
                Overview
              </TabsTrigger>
              <TabsTrigger value="experiments">
                <FlaskConical className="size-4" />
                Experiments ({experiments.length})
              </TabsTrigger>
              <TabsTrigger value="api-key">
                <Key className="size-4" />
                API key
              </TabsTrigger>
              <TabsTrigger value="danger">
                <AlertTriangle className="size-4" />
                Danger zone
              </TabsTrigger>
            </TabsList>

            <TabsContent value="overview" className="mt-4">
              <Card>
                <CardHeader>
                  <CardTitle>Details</CardTitle>
                  <CardDescription>Application metadata and identity.</CardDescription>
                </CardHeader>
                <CardContent>
                  {editing ? (
                    <form onSubmit={handleSave} className="space-y-4">
                      <div className="space-y-2">
                        <Label htmlFor="app-rename">Name</Label>
                        <Input
                          id="app-rename"
                          autoFocus
                          value={editName}
                          onChange={(e) => setEditName(e.target.value)}
                          maxLength={APPLICATION_NAME_MAX_LENGTH}
                          disabled={editLoading}
                        />
                      </div>
                      <div className="flex gap-2">
                        <Button
                          type="submit"
                          disabled={editLoading || Boolean(validateApplicationName(editName))}
                        >
                          {editLoading ? "Saving…" : "Save"}
                        </Button>
                        <Button
                          type="button"
                          variant="outline"
                          onClick={() => {
                            setEditing(false)
                            setEditName(app.name)
                          }}
                          disabled={editLoading}
                        >
                          Cancel
                        </Button>
                      </div>
                      {editError && <p className="text-sm text-destructive">{editError}</p>}
                    </form>
                  ) : (
                    <dl className="grid gap-4 sm:grid-cols-3">
                      <div>
                        <dt className="text-xs text-muted-foreground">ID</dt>
                        <dd className="mt-1 font-mono text-sm">{app.id}</dd>
                      </div>
                      <div>
                        <dt className="text-xs text-muted-foreground">Status</dt>
                        <dd className="mt-1">
                          <Badge variant="outline" className="capitalize">
                            {app.status}
                          </Badge>
                        </dd>
                      </div>
                      <div>
                        <dt className="text-xs text-muted-foreground">Created</dt>
                        <dd className="mt-1 text-sm">{new Date(app.created_at).toLocaleString()}</dd>
                      </div>
                      <div>
                        <dt className="text-xs text-muted-foreground">Last updated</dt>
                        <dd className="mt-1 text-sm">{new Date(app.updated_at).toLocaleString()}</dd>
                      </div>
                    </dl>
                  )}
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="experiments" className="mt-4">
              <Card>
                <CardHeader className="flex-row items-start justify-between space-y-0">
                  <div>
                    <CardTitle>Experiments</CardTitle>
                    <CardDescription>
                      {app.status === "inactive"
                        ? "New experiments are disabled while the application is inactive."
                        : "A/B tests running under this application."}
                    </CardDescription>
                  </div>
                  <div className="flex gap-2">
                    <Button size="sm" onClick={openCreateExperiment} disabled={app.status === "inactive"}>
                      <Plus />
                      New experiment
                    </Button>
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => navigate(`/applications/${app.id}/experiments`)}
                    >
                      Manage all
                    </Button>
                  </div>
                </CardHeader>
                <CardContent className="p-0">
                  {experiments.length === 0 ? (
                    <div className="p-6">
                      <EmptyState
                        icon={FlaskConical}
                        title="No experiments yet"
                        description={
                          app.status === "inactive"
                            ? "Reactivate the application to create experiments."
                            : "Create an A/B test to start assigning users to variants."
                        }
                        action={
                          <EmptyStateButton
                            onClick={openCreateExperiment}
                            disabled={app.status === "inactive"}
                          >
                            Create experiment
                          </EmptyStateButton>
                        }
                      />
                    </div>
                  ) : (
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>Name</TableHead>
                          <TableHead>Status</TableHead>
                          <TableHead className="hidden md:table-cell">Key</TableHead>
                          <TableHead className="hidden sm:table-cell">Created</TableHead>
                          <TableHead className="text-right">Active</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {experiments.map((experiment) => (
                          <TableRow key={experiment.id}>
                            <TableCell>
                              <Link
                                to={`/applications/${app.id}/experiments/${experiment.id}`}
                                className="font-medium hover:text-primary"
                              >
                                {experiment.name}
                              </Link>
                              <p className="mt-0.5 text-xs text-muted-foreground md:hidden font-mono">
                                {experiment.key}
                              </p>
                            </TableCell>
                            <TableCell>
                              <StatusBadge status={experiment.status} />
                            </TableCell>
                            <TableCell className="hidden font-mono text-xs text-muted-foreground md:table-cell">
                              {experiment.key}
                            </TableCell>
                            <TableCell className="hidden text-muted-foreground sm:table-cell">
                              {new Date(experiment.created_at).toLocaleDateString()}
                            </TableCell>
                            <TableCell className="text-right">
                              <ExperimentStatusToggle
                                status={experiment.status}
                                disabled={togglingExperimentId === experiment.id}
                                onToggle={() => handleExperimentToggle(experiment)}
                              />
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  )}
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="api-key" className="mt-4">
              <Card>
                <CardHeader>
                  <CardTitle>API key</CardTitle>
                  <CardDescription>
                    Authenticate SDK and ingestion requests. Copy now — it won't be shown again.
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="flex items-center gap-2 rounded-lg bg-muted/50 px-4 py-3">
                    <code className="min-w-0 flex-1 font-mono text-sm break-all">{app.api_key}</code>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      onClick={handleCopyKey}
                      aria-label={keyCopied ? "API key copied" : "Copy API key"}
                      className="shrink-0"
                    >
                      {keyCopied ? <Check /> : <Copy />}
                    </Button>
                  </div>
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="danger" className="mt-4">
              <Card className="border-destructive/30">
                <CardHeader className="flex-row items-start justify-between space-y-0">
                  <div>
                    <CardTitle>Delete application</CardTitle>
                    <CardDescription>
                      Deletesthis application and all its experiments and branches.
                    </CardDescription>
                  </div>
                  <Button
                    variant="destructive"
                    size="sm"
                    onClick={() => setDeleteDialogOpen(true)}
                    disabled={deleteLoading}
                  >
                    <Trash2 />
                    Delete
                  </Button>
                </CardHeader>
                {deleteError && (
                  <CardContent className="pt-0">
                    <p className="text-sm text-destructive">{deleteError}</p>
                  </CardContent>
                )}
              </Card>
            </TabsContent>
          </Tabs>

          <ConfirmDeleteDialog
            open={deleteDialogOpen}
            onOpenChange={setDeleteDialogOpen}
            title={`Delete ${app.name}?`}
            description="This will Delete the application and cascade to its experiments and branches. This action cannot be undone."
            loading={deleteLoading}
            onConfirm={handleDelete}
          />
        </>
      )}
    </>
  )
}
