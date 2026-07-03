import { useEffect, useState } from "react"
import { Link, useParams } from "react-router-dom"
import { applicationsApi, type Application } from "@/api/applications"
import { ApiError } from "@/api/client"
import { Button } from "@/components/ui/button"

export default function ApplicationDetailPage() {
  const { id } = useParams<{ id: string }>()

  const [app, setApp] = useState<Application | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const [editing, setEditing] = useState(false)
  const [editName, setEditName] = useState("")
  const [editLoading, setEditLoading] = useState(false)
  const [editError, setEditError] = useState<string | null>(null)

  const [keyCopied, setKeyCopied] = useState(false)

  useEffect(() => {
    if (!id) return
    applicationsApi
      .get(id)
      .then((data) => {
        setApp(data)
        setEditName(data.name)
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
    setEditLoading(true)
    setEditError(null)
    try {
      const updated = await applicationsApi.update(app.id, editName.trim())
      setApp(updated)
      setEditing(false)
    } catch (err) {
      setEditError(err instanceof ApiError ? err.message : "Failed to update application")
    } finally {
      setEditLoading(false)
    }
  }

  function handleCopyKey() {
    if (!app?.api_key) return
    navigator.clipboard.writeText(app.api_key).then(() => {
      setKeyCopied(true)
      setTimeout(() => setKeyCopied(false), 2000)
    })
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
              <div className="flex items-start justify-between gap-4">
                {editing ? (
                  <form onSubmit={handleSave} className="flex flex-1 items-center gap-3">
                    <input
                      autoFocus
                      type="text"
                      value={editName}
                      onChange={(e) => setEditName(e.target.value)}
                      className="flex-1 rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 placeholder:text-slate-400 focus:border-slate-500 focus:outline-none focus:ring-2 focus:ring-slate-200"
                      disabled={editLoading}
                    />
                    <Button type="submit" disabled={editLoading || !editName.trim()}>
                      {editLoading ? "Saving…" : "Save"}
                    </Button>
                    <Button
                      type="button"
                      variant="outline"
                      onClick={() => { setEditing(false); setEditName(app.name) }}
                      disabled={editLoading}
                    >
                      Cancel
                    </Button>
                  </form>
                ) : (
                  <>
                    <h1 className="text-2xl font-semibold tracking-tight text-slate-900">
                      {app.name}
                    </h1>
                    <Button variant="outline" onClick={() => setEditing(true)}>
                      Rename
                    </Button>
                  </>
                )}
              </div>

              {editError && (
                <p className="mt-2 text-sm text-red-600">{editError}</p>
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
          </div>
        )}
      </div>
    </div>
  )
}
