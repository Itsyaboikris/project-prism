import { useEffect, useState } from "react"
import { Link, useParams } from "react-router-dom"
import { assignmentsApi, type ExperimentAssignmentsView } from "@/api/assignments"
import { ApiError } from "@/api/client"
import { StatusBadge } from "@/components/StatusBadge"

function formatCount(count: number) {
  return `${count} assignment${count === 1 ? "" : "s"}`
}

export default function ExperimentAssignmentsPage() {
  const { appId, id } = useParams<{ appId: string; id: string }>()

  const [view, setView] = useState<ExperimentAssignmentsView | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!appId || !id) return

    assignmentsApi
      .listByExperiment(appId, id)
      .then(setView)
      .catch((err) =>
        setError(
          err instanceof ApiError && err.status === 404
            ? "Experiment not found."
            : err instanceof ApiError
              ? err.message
              : "Failed to load assignments",
        ),
      )
      .finally(() => setLoading(false))
  }, [appId, id])

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
                  <p className="text-sm text-slate-500">Experiment assignments</p>
                  <div className="mt-2 flex flex-wrap items-center gap-3">
                    <h1 className="text-2xl font-semibold tracking-tight text-slate-900">
                      {view.experiment_name}
                    </h1>
                    <StatusBadge status={view.experiment_status} />
                    <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-600">
                      {formatCount(view.assignments.length)}
                    </span>
                  </div>
                  <p className="mt-2 font-mono text-sm text-slate-500">{view.experiment_key}</p>
                </div>

                <Link
                  to={`/applications/${appId}/experiments/${id}/dashboard`}
                  className="inline-flex h-8 items-center justify-center rounded-lg border border-slate-300 bg-white px-3 text-sm font-medium text-slate-900 transition-colors hover:bg-slate-50"
                >
                  Open dashboard
                </Link>
              </div>
            </div>

            <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
              <div className="flex items-center justify-between gap-4">
                <h2 className="text-lg font-medium text-slate-900">Assignments</h2>
                <p className="text-sm text-slate-500">Newest assignments first</p>
              </div>

              {view.assignments.length === 0 ? (
                <div className="mt-6 rounded-lg border border-dashed border-slate-200 bg-slate-50 px-4 py-12 text-center text-sm text-slate-500">
                  No assignments yet.
                </div>
              ) : (
                <div className="mt-6 overflow-x-auto">
                  <table className="min-w-full divide-y divide-slate-200 text-sm">
                    <thead>
                      <tr className="text-left text-slate-500">
                        <th className="pb-3 pr-4 font-medium">User</th>
                        <th className="pb-3 pr-4 font-medium">Branch</th>
                        <th className="pb-3 pr-4 font-medium">Assigned</th>
                        <th className="pb-3 font-medium">Assignment ID</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100">
                      {view.assignments.map((assignment) => (
                        <tr key={assignment.id}>
                          <td className="py-3 pr-4 font-mono text-slate-700">
                            {assignment.user_id}
                          </td>
                          <td className="py-3 pr-4">
                            <div className="flex flex-wrap items-center gap-2">
                              <span className="font-medium text-slate-900">
                                {assignment.branch_name}
                              </span>
                              <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-600">
                                {assignment.branch_key}
                              </span>
                            </div>
                          </td>
                          <td className="py-3 pr-4 text-slate-600">
                            {new Date(assignment.assigned_at).toLocaleString()}
                          </td>
                          <td className="py-3 font-mono text-xs text-slate-500">
                            {assignment.id}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
