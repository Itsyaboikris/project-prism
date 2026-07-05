import { useEffect, useState } from "react"
import { Link, useParams, useSearchParams } from "react-router-dom"
import { eventsApi, type ExperimentEventsView } from "@/api/events"
import { ApiError } from "@/api/client"
import { StatusBadge } from "@/components/StatusBadge"

const PAGE_SIZE = 100

function formatCount(count: number) {
  return `${count} event${count === 1 ? "" : "s"}`
}

function formatProperties(properties: unknown | null) {
  if (properties == null) return "—"
  try {
    return JSON.stringify(properties)
  } catch {
    return "—"
  }
}

export default function ExperimentEventsPage() {
  const { appId, id } = useParams<{ appId: string; id: string }>()
  const [searchParams, setSearchParams] = useSearchParams()

  const eventNameFilter = searchParams.get("event_name") ?? ""
  const offset = Number(searchParams.get("offset") ?? "0")

  const [view, setView] = useState<ExperimentEventsView | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [filterInput, setFilterInput] = useState(eventNameFilter)

  useEffect(() => {
    setFilterInput(eventNameFilter)
  }, [eventNameFilter])

  useEffect(() => {
    if (!appId || !id) return

    setLoading(true)
    setError(null)

    eventsApi
      .listByExperiment(appId, id, {
        event_name: eventNameFilter || undefined,
        limit: PAGE_SIZE,
        offset: Number.isFinite(offset) && offset >= 0 ? offset : 0,
      })
      .then(setView)
      .catch((err) =>
        setError(
          err instanceof ApiError && err.status === 404
            ? "Experiment not found."
            : err instanceof ApiError
              ? err.message
              : "Failed to load events",
        ),
      )
      .finally(() => setLoading(false))
  }, [appId, id, eventNameFilter, offset])

  function applyFilter() {
    const next = new URLSearchParams()
    const trimmed = filterInput.trim()
    if (trimmed) next.set("event_name", trimmed)
    setSearchParams(next)
  }

  function clearFilter() {
    setFilterInput("")
    setSearchParams(new URLSearchParams())
  }

  function goToPage(nextOffset: number) {
    const next = new URLSearchParams(searchParams)
    if (nextOffset <= 0) {
      next.delete("offset")
    } else {
      next.set("offset", String(nextOffset))
    }
    setSearchParams(next)
  }

  const currentOffset = Number.isFinite(offset) && offset >= 0 ? offset : 0
  const hasNextPage = (view?.events.length ?? 0) === PAGE_SIZE

  return (
    <div className="min-h-screen bg-slate-50">
      <div className="mx-auto max-w-5xl px-6 py-12">
        <Link
          to={`/applications/${appId}/experiments/${id}`}
          className="inline-flex items-center gap-1.5 text-sm text-slate-500 hover:text-slate-900"
        >
          ← Experiment
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

        {!loading && !error && view && (
          <div className="mt-6 space-y-6">
            <div className="rounded-xl border border-slate-200 bg-white p-8 shadow-sm">
              <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
                <div className="min-w-0 flex-1">
                  <p className="text-sm text-slate-500">Experiment events</p>
                  <div className="mt-2 flex flex-wrap items-center gap-3">
                    <h1 className="text-2xl font-semibold tracking-tight text-slate-900">
                      {view.experiment_name}
                    </h1>
                    <StatusBadge status={view.experiment_status} />
                    <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-600">
                      {formatCount(view.events.length)}
                      {currentOffset > 0 ? ` (from ${currentOffset + 1})` : ""}
                    </span>
                  </div>
                  <p className="mt-2 font-mono text-sm text-slate-500">{view.experiment_key}</p>
                </div>

                <div className="flex flex-wrap gap-2">
                  <Link
                    to={`/applications/${appId}/experiments/${id}/assignments`}
                    className="inline-flex h-8 items-center justify-center rounded-lg border border-slate-300 bg-white px-3 text-sm font-medium text-slate-900 transition-colors hover:bg-slate-50"
                  >
                    View assignments
                  </Link>
                  <Link
                    to={`/applications/${appId}/experiments/${id}/dashboard${
                      eventNameFilter ? `?event_name=${encodeURIComponent(eventNameFilter)}` : ""
                    }`}
                    className="inline-flex h-8 items-center justify-center rounded-lg border border-slate-300 bg-white px-3 text-sm font-medium text-slate-900 transition-colors hover:bg-slate-50"
                  >
                    Open dashboard
                  </Link>
                </div>
              </div>
            </div>

            <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
              <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
                <div>
                  <h2 className="text-lg font-medium text-slate-900">Events</h2>
                  <p className="mt-1 text-sm text-slate-500">Newest events first</p>
                </div>

                <div className="flex flex-wrap items-end gap-2">
                  <label className="flex flex-col gap-1 text-sm text-slate-600">
                    <span className="text-xs font-medium uppercase tracking-wide text-slate-500">
                      Event name
                    </span>
                    <input
                      type="text"
                      value={filterInput}
                      onChange={(e) => setFilterInput(e.target.value)}
                      onKeyDown={(e) => {
                        if (e.key === "Enter") applyFilter()
                      }}
                      placeholder="purchase"
                      className="h-8 rounded-lg border border-slate-300 bg-white px-3 text-sm text-slate-900"
                    />
                  </label>
                  <button
                    type="button"
                    onClick={applyFilter}
                    className="inline-flex h-8 items-center justify-center rounded-lg bg-slate-900 px-3 text-sm font-medium text-white transition-colors hover:bg-slate-800"
                  >
                    Apply
                  </button>
                  {eventNameFilter && (
                    <button
                      type="button"
                      onClick={clearFilter}
                      className="inline-flex h-8 items-center justify-center rounded-lg border border-slate-300 bg-white px-3 text-sm font-medium text-slate-900 transition-colors hover:bg-slate-50"
                    >
                      Clear
                    </button>
                  )}
                </div>
              </div>

              {view.events.length === 0 ? (
                <div className="mt-6 rounded-lg border border-dashed border-slate-200 bg-slate-50 px-4 py-12 text-center text-sm text-slate-500">
                  {eventNameFilter
                    ? `No events found for "${eventNameFilter}".`
                    : "No events recorded yet."}
                </div>
              ) : (
                <div className="mt-6 overflow-x-auto">
                  <table className="min-w-full divide-y divide-slate-200 text-sm">
                    <thead>
                      <tr className="text-left text-slate-500">
                        <th className="pb-3 pr-4 font-medium">User</th>
                        <th className="pb-3 pr-4 font-medium">Event</th>
                        <th className="pb-3 pr-4 font-medium">Branch</th>
                        <th className="pb-3 pr-4 font-medium">Occurred</th>
                        <th className="pb-3 pr-4 font-medium">Properties</th>
                        <th className="pb-3 font-medium">Event ID</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100">
                      {view.events.map((event) => (
                        <tr key={event.id}>
                          <td className="py-3 pr-4 font-mono text-slate-700">{event.user_id}</td>
                          <td className="py-3 pr-4 font-medium text-slate-900">{event.event_name}</td>
                          <td className="py-3 pr-4">
                            {event.branch_name ? (
                              <div className="flex flex-wrap items-center gap-2">
                                <span className="font-medium text-slate-900">{event.branch_name}</span>
                                {event.branch_key && (
                                  <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-600">
                                    {event.branch_key}
                                  </span>
                                )}
                              </div>
                            ) : (
                              <span className="text-slate-400">—</span>
                            )}
                          </td>
                          <td className="py-3 pr-4 text-slate-600">
                            {new Date(event.occurred_at).toLocaleString()}
                          </td>
                          <td className="max-w-xs truncate py-3 pr-4 font-mono text-xs text-slate-500">
                            {formatProperties(event.properties)}
                          </td>
                          <td className="py-3 font-mono text-xs text-slate-500">{event.id}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}

              {(currentOffset > 0 || hasNextPage) && (
                <div className="mt-6 flex items-center justify-between gap-4">
                  <button
                    type="button"
                    disabled={currentOffset === 0}
                    onClick={() => goToPage(Math.max(0, currentOffset - PAGE_SIZE))}
                    className="inline-flex h-8 items-center justify-center rounded-lg border border-slate-300 bg-white px-3 text-sm font-medium text-slate-900 transition-colors hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    Previous
                  </button>
                  <p className="text-sm text-slate-500">
                    Showing {currentOffset + 1}–{currentOffset + view.events.length}
                  </p>
                  <button
                    type="button"
                    disabled={!hasNextPage}
                    onClick={() => goToPage(currentOffset + PAGE_SIZE)}
                    className="inline-flex h-8 items-center justify-center rounded-lg border border-slate-300 bg-white px-3 text-sm font-medium text-slate-900 transition-colors hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    Next
                  </button>
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
