import { useEffect, useState } from "react"
import { Link, useNavigate, useParams } from "react-router-dom"
import { applicationsApi, type Application } from "@/api/applications"
import { experimentsApi, type Experiment } from "@/api/experiments"
import { ApiError } from "@/api/client"
import { ApplicationStatusToggle } from "@/components/ApplicationStatusToggle"
import { ExperimentStatusToggle } from "@/components/ExperimentStatusToggle"
import { StatusBadge } from "@/components/StatusBadge"
import { Button } from "@/components/ui/button"
import { APPLICATION_NAME_MAX_LENGTH, validateApplicationName } from "@/lib/applicationName"

function formatBranchCount(count: number) {
  return `${count} branch${count === 1 ? "" : "es"}`
}

export default function ApplicationDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const [app, setApp] = useState<Application | null>(null)
  const [experiments, setExperiments] = useState<Experiment[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const [editing, setEditing] = useState(false)
  const [editName, setEditName] = useState("")
  const [editLoading, setEditLoading] = useState(false)
  const [editError, setEditError] = useState<string | null>(null)

  const [statusLoading, setStatusLoading] = useState(false)
  const [statusError, setStatusError] = useState<string | null>(null)
  const [togglingExperimentId, setTogglingExperimentId] = useState<string | null>(null)
  const [toggleError, setToggleError] = useState<string | null>(null)

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
      const updated = await applicationsApi.update(app.id, {
        name: editName.trim(),
      })
      setApp(updated)
      setEditing(false)
    } catch (err) {
      setEditError(err instanceof ApiError ? err.message : "Failed to update application")
    } finally {
      setEditLoading(false)
    }
  }

  async function handleStatusToggle() {
    if (!app || statusLoading) return

    const previousStatus = app.status
    const nextStatus = previousStatus === "active" ? "inactive" : "active"

    setStatusLoading(true)
    setStatusError(null)
    setApp({ ...app, status: nextStatus })

    try {
      const updated = await applicationsApi.update(app.id, {
        name: app.name,
        status: nextStatus,
      })
      setApp(updated)
    } catch (err) {
      setApp({ ...app, status: previousStatus })
      setStatusError(err instanceof ApiError ? err.message : "Failed to update status")
    } finally {
      setStatusLoading(false)
    }
  }

  async function handleExperimentToggle(experiment: Experiment) {
    if (!id || togglingExperimentId || experiment.status === "completed") return

    const previousStatus = experiment.status
    const nextStatus = previousStatus === "active" ? "paused" : "active"

    setTogglingExperimentId(experiment.id)
    setToggleError(null)
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
    } catch (err) {
      setExperiments((current) =>
        current.map((item) =>
          item.id === experiment.id ? { ...item, status: previousStatus } : item,
        ),
      )
      setToggleError(
        err instanceof ApiError ? err.message : "Failed to update experiment status",
      )
    } finally {
      setTogglingExperimentId(null)
    }
  }

  function handleCopyKey() {
    if (!app?.api_key) return

    navigator.clipboard.writeText(app.api_key).then(() => {
      setKeyCopied(true)
      setTimeout(() => setKeyCopied(false), 2000)
    })
  }

  async function handleDelete() {
    if (!app) return
    if (!window.confirm(`Delete "${app.name}"? This will also remove its experiments and branches.`)) {
      return
    }

    setDeleteLoading(true)
    setDeleteError(null)
    try {
      await applicationsApi.delete(app.id)
      navigate("/applications")
    } catch (err) {
      setDeleteError(err instanceof ApiError ? err.message : "Failed to delete application")
      setDeleteLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-slate-50">
      <div className="mx-auto max-w-4xl px-6 py-12">
        <Link
          to="/applications"
          className="inline-flex items-center gap-1.5 text-sm text-slate-500 hover:text-slate-900"
        >
          ← Applications
        </Link>

        {loading && (
          <div className="mt-12 flex items-center justify-center text-sm text-slate-400">
            Loading…
          </div>
        )}

        {!loading && error && (
          <div className="mt-6 rounded-xl border border-red-200 bg-red-50 p-6 text-sm text-red-700">
            {error}
          </div>
        )}

        {!loading && !error && app && (
          <div className="mt-6 space-y-6">
            <div className="rounded-xl border border-slate-200 bg-white p-8 shadow-sm">
              <div className="flex flex-col gap-6 lg:flex-row lg:items-start lg:justify-between">
                <div className="min-w-0 flex-1">
                  {editing ? (
                    <form onSubmit={handleSave} className="space-y-3">
                      <div>
                        <label className="text-xs font-medium uppercase tracking-wide text-slate-400">
                          Name
                        </label>
                        <input
                          autoFocus
                          type="text"
                          value={editName}
                          onChange={(e) => setEditName(e.target.value)}
                          maxLength={APPLICATION_NAME_MAX_LENGTH}
                          className="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 placeholder:text-slate-400 focus:border-slate-500 focus:outline-none focus:ring-2 focus:ring-slate-200"
                          disabled={editLoading}
                        />
                      </div>

                      <div className="flex flex-wrap gap-3">
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
                    </form>
                  ) : (
                    <div className="flex flex-wrap items-start justify-between gap-4">
                      <div className="min-w-0 flex-1">
                        <h1 className="wrap-break-word text-2xl font-semibold tracking-tight text-slate-900">
                          {app.name}
                        </h1>
                      </div>
                      <Button
                        variant="outline"
                        onClick={() => setEditing(true)}
                        className="shrink-0"
                        disabled={statusLoading}
                      >
                        Rename
                      </Button>
                    </div>
                  )}
                </div>

                <ApplicationStatusToggle
                  status={app.status}
                  disabled={statusLoading}
                  onToggle={handleStatusToggle}
                />
              </div>

              {editError && (
                <p className="mt-2 text-sm text-red-600">{editError}</p>
              )}
              {statusError && (
                <p className="mt-2 text-sm text-red-600">{statusError}</p>
              )}

              <dl className="mt-6 grid gap-4 sm:grid-cols-2">
                <div>
                  <dt className="text-xs font-medium uppercase tracking-wide text-slate-400">ID</dt>
                  <dd className="mt-1 font-mono text-sm text-slate-700">{app.id}</dd>
                </div>
                <div>
                  <dt className="text-xs font-medium uppercase tracking-wide text-slate-400">
                    Created
                  </dt>
                  <dd className="mt-1 text-sm text-slate-700">
                    {new Date(app.created_at).toLocaleString()}
                  </dd>
                </div>
                <div>
                  <dt className="text-xs font-medium uppercase tracking-wide text-slate-400">
                    Last updated
                  </dt>
                  <dd className="mt-1 text-sm text-slate-700">
                    {new Date(app.updated_at).toLocaleString()}
                  </dd>
                </div>
              </dl>
            </div>

            <div className="rounded-xl border border-slate-200 bg-white p-8 shadow-sm">
              <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
                <div>
                  <h2 className="text-base font-medium text-slate-900">Experiments</h2>
                  <p className="mt-1 text-sm text-slate-500">
                    {app.status === "inactive"
                      ? "View experiments here. New experiments are disabled while the application is inactive."
                      : "Manage A/B tests for this application without leaving the page."}
                  </p>
                </div>
                <div className="flex flex-wrap gap-3">
                  <Button
                    type="button"
                    onClick={() => navigate(`/applications/${app.id}/experiments`)}
                    disabled={app.status === "inactive"}
                  >
                    New experiment
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => navigate(`/applications/${app.id}/experiments`)}
                  >
                    Manage
                  </Button>
                </div>
              </div>

              {toggleError && (
                <div className="mt-4 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">
                  {toggleError}
                </div>
              )}

              {experiments.length === 0 ? (
                <div className="mt-6 rounded-lg border border-dashed border-slate-200 bg-slate-50 px-4 py-12 text-center">
                  <p className="text-sm text-slate-500">No experiments yet.</p>
                  <Link
                    to={`/applications/${app.id}/experiments`}
                    className="mt-3 inline-flex h-8 items-center justify-center rounded-lg border border-slate-300 bg-white px-3 text-sm font-medium text-slate-900 transition-colors hover:bg-slate-50"
                  >
                    {app.status === "inactive" ? "Open experiments" : "Create your first experiment"}
                  </Link>
                </div>
              ) : (
                <ul className="mt-6 space-y-3">
                  {experiments.map((experiment) => (
                    <li key={experiment.id}>
                      <div className="rounded-xl border border-slate-200 bg-slate-50 px-6 py-4">
                        <div className="flex items-center justify-between gap-4">
                          <Link
                            to={`/applications/${app.id}/experiments/${experiment.id}`}
                            className="min-w-0 flex-1 transition-colors hover:text-slate-700"
                          >
                            <div className="flex items-center gap-3">
                              <p className="font-medium text-slate-900">{experiment.name}</p>
                              <StatusBadge status={experiment.status} />
                              <span className="rounded-full bg-white px-2 py-0.5 text-xs text-slate-600">
                                {formatBranchCount(experiment.branches.length)}
                              </span>
                            </div>
                            <p className="mt-0.5 font-mono text-xs text-slate-400">
                              {experiment.key}
                            </p>
                            {experiment.description && (
                              <p className="mt-1 truncate text-sm text-slate-500">
                                {experiment.description}
                              </p>
                            )}
                          </Link>

                          <div className="flex shrink-0 items-center gap-4">
                            <span className="text-xs text-slate-400">
                              {new Date(experiment.created_at).toLocaleDateString()}
                            </span>
                            <ExperimentStatusToggle
                              status={experiment.status}
                              disabled={togglingExperimentId === experiment.id}
                              onToggle={() => handleExperimentToggle(experiment)}
                            />
                          </div>
                        </div>
                      </div>
                    </li>
                  ))}
                </ul>
              )}
            </div>

            <div className="rounded-xl border border-slate-200 bg-white p-8 shadow-sm">
              <div className="flex items-center justify-between">
                <div>
                  <h2 className="text-base font-medium text-slate-900">API Key</h2>
                  <p className="mt-1 text-sm text-slate-500">
                    Use this key to authenticate SDK and ingestion requests. It cannot be retrieved
                    again after leaving this page.
                  </p>
                </div>
                <Button variant="outline" onClick={handleCopyKey}>
                  {keyCopied ? "Copied!" : "Copy"}
                </Button>
              </div>
              <div className="mt-4 rounded-lg bg-slate-50 px-4 py-3 font-mono text-sm text-slate-700 break-all">
                {app.api_key}
              </div>
            </div>

            <div className="rounded-xl border border-red-200 bg-white p-8 shadow-sm">
              <div className="flex items-start justify-between gap-4">
                <div>
                  <h2 className="text-base font-medium text-slate-900">Delete Application</h2>
                  <p className="mt-1 text-sm text-slate-500">
                    This soft-deletes the application and cascades to its experiments and branches.
                  </p>
                </div>
                <Button variant="destructive" onClick={handleDelete} disabled={deleteLoading}>
                  {deleteLoading ? "Deleting…" : "Delete"}
                </Button>
              </div>
              {deleteError && (
                <p className="mt-3 text-sm text-red-600">{deleteError}</p>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
