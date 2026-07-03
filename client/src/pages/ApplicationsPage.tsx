import { useEffect, useRef, useState } from "react"
import { Link } from "react-router-dom"
import { applicationsApi, type Application } from "@/api/applications"
import { ApiError } from "@/api/client"
import { ApplicationStatusBadge } from "@/components/ApplicationStatusBadge"
import { Button } from "@/components/ui/button"

export default function ApplicationsPage() {
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
    setCreateLoading(true)
    setCreateError(null)
    try {
      const app = await applicationsApi.create(newName.trim())
      setApps((prev) => [app, ...prev])
      setCreating(false)
      setNewName("")
    } catch (err) {
      setCreateError(err instanceof ApiError ? err.message : "Failed to create application")
    } finally {
      setCreateLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-slate-50">
      <div className="mx-auto max-w-4xl px-6 py-12">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-semibold tracking-tight text-slate-900">Applications</h1>
            <p className="mt-1 text-sm text-slate-500">
              Each application has a unique API key used for SDK and ingestion requests.
            </p>
          </div>
          {!creating && (
            <Button onClick={openCreateForm}>New application</Button>
          )}
        </div>

        {creating && (
          <form
            onSubmit={handleCreate}
            className="mt-6 rounded-xl border border-slate-200 bg-white p-6 shadow-sm"
          >
            <h2 className="text-base font-medium text-slate-900">New application</h2>
            <div className="mt-4 flex gap-3">
              <input
                ref={inputRef}
                type="text"
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                placeholder="Application name"
                className="flex-1 rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 placeholder:text-slate-400 focus:border-slate-500 focus:outline-none focus:ring-2 focus:ring-slate-200"
                disabled={createLoading}
              />
              <Button type="submit" disabled={createLoading || !newName.trim()}>
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
            {createError && (
              <p className="mt-2 text-sm text-red-600">{createError}</p>
            )}
          </form>
        )}

        <div className="mt-6">
          {loading && (
            <div className="flex items-center justify-center py-20 text-sm text-slate-400">
              Loading…
            </div>
          )}

          {!loading && error && (
            <div className="rounded-xl border border-red-200 bg-red-50 p-6 text-sm text-red-700">
              {error}
            </div>
          )}

          {!loading && !error && apps.length === 0 && (
            <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-slate-200 bg-white py-20 text-center">
              <p className="text-sm text-slate-500">No applications yet.</p>
              <button
                onClick={openCreateForm}
                className="mt-2 text-sm font-medium text-slate-900 underline-offset-4 hover:underline"
              >
                Create your first application
              </button>
            </div>
          )}

          {!loading && !error && apps.length > 0 && (
            <ul className="space-y-3">
              {apps.map((app) => (
                <li key={app.id}>
                  <Link
                    to={`/applications/${app.id}`}
                    className="flex items-center justify-between rounded-xl border border-slate-200 bg-white px-6 py-4 shadow-sm transition-colors hover:border-slate-300 hover:bg-slate-50"
                  >
                    <div>
                      <div className="flex items-center gap-3">
                        <p className="font-medium text-slate-900">{app.name}</p>
                        <ApplicationStatusBadge status={app.status} />
                      </div>
                      <p className="mt-0.5 font-mono text-xs text-slate-400">{app.id}</p>
                    </div>
                    <span className="text-xs text-slate-400">
                      {new Date(app.created_at).toLocaleDateString()}
                    </span>
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </div>
  )
}
