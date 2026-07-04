import { useEffect, useMemo, useState } from "react"
import { Link, useParams } from "react-router-dom"
import { assignmentsApi, type ExperimentDashboard } from "@/api/assignments"
import { ApiError } from "@/api/client"
import { StatusBadge } from "@/components/StatusBadge"
import { formatBranchWeightValue } from "@/lib/branchWeights"

function formatPercent(value: number) {
  return `${formatBranchWeightValue(value)}%`
}

export default function ExperimentDashboardPage() {
  const { appId, id } = useParams<{ appId: string; id: string }>()

  const [dashboard, setDashboard] = useState<ExperimentDashboard | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!appId || !id) return

    assignmentsApi
      .getExperimentDashboard(appId, id)
      .then(setDashboard)
      .catch((err) =>
        setError(
          err instanceof ApiError && err.status === 404
            ? "Experiment not found."
            : err instanceof ApiError
              ? err.message
              : "Failed to load experiment dashboard",
        ),
      )
      .finally(() => setLoading(false))
  }, [appId, id])

  const topBranch = useMemo(() => {
    if (!dashboard?.branches.length) return null
    const branch = [...dashboard.branches].sort((a, b) => b.assignment_count - a.assignment_count)[0]
    return branch.assignment_count > 0 ? branch : null
  }, [dashboard])

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

        {!loading && !error && dashboard && (
          <div className="mt-6 space-y-6">
            <div className="rounded-xl border border-slate-200 bg-white p-8 shadow-sm">
              <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
                <div className="min-w-0 flex-1">
                  <p className="text-sm text-slate-500">Experiment dashboard</p>
                  <div className="mt-2 flex flex-wrap items-center gap-3">
                    <h1 className="text-2xl font-semibold tracking-tight text-slate-900">
                      {dashboard.experiment_name}
                    </h1>
                    <StatusBadge status={dashboard.experiment_status} />
                  </div>
                  <p className="mt-2 font-mono text-sm text-slate-500">
                    {dashboard.experiment_key}
                  </p>
                </div>

                <Link
                  to={`/applications/${appId}/experiments/${id}/assignments`}
                  className="inline-flex h-8 items-center justify-center rounded-lg border border-slate-300 bg-white px-3 text-sm font-medium text-slate-900 transition-colors hover:bg-slate-50"
                >
                  View assignments
                </Link>
              </div>

              <div className="mt-6 grid gap-4 sm:grid-cols-3">
                <div className="rounded-lg border border-slate-200 bg-slate-50 p-4">
                  <p className="text-xs font-medium uppercase tracking-wide text-slate-500">
                    Total assignments
                  </p>
                  <p className="mt-2 text-2xl font-semibold text-slate-900">
                    {dashboard.total_assignments}
                  </p>
                </div>
                <div className="rounded-lg border border-slate-200 bg-slate-50 p-4">
                  <p className="text-xs font-medium uppercase tracking-wide text-slate-500">
                    Branches tracked
                  </p>
                  <p className="mt-2 text-2xl font-semibold text-slate-900">
                    {dashboard.branch_count}
                  </p>
                </div>
                <div className="rounded-lg border border-slate-200 bg-slate-50 p-4">
                  <p className="text-xs font-medium uppercase tracking-wide text-slate-500">
                    Leading branch
                  </p>
                  <p className="mt-2 text-lg font-semibold text-slate-900">
                    {topBranch ? topBranch.branch_name : "No assignments yet"}
                  </p>
                  {topBranch && (
                    <p className="mt-1 text-sm text-slate-500">
                      {topBranch.assignment_count} users, {formatPercent(topBranch.assignment_share)}
                    </p>
                  )}
                </div>
              </div>
            </div>

            <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
              <div className="flex items-center justify-between gap-4">
                <h2 className="text-lg font-medium text-slate-900">Branch distribution</h2>
                <p className="text-sm text-slate-500">
                  Compare configured traffic with actual assignment share.
                </p>
              </div>

              {dashboard.branches.length === 0 ? (
                <div className="mt-6 rounded-lg border border-dashed border-slate-200 bg-slate-50 px-4 py-12 text-center text-sm text-slate-500">
                  No branches configured for this experiment yet.
                </div>
              ) : (
                <div className="mt-6 grid gap-4 md:grid-cols-2">
                  {dashboard.branches.map((branch) => (
                    <div
                      key={branch.branch_id}
                      className="rounded-lg border border-slate-200 bg-slate-50 p-4"
                    >
                      <div className="flex flex-wrap items-center justify-between gap-3">
                        <div>
                          <h3 className="font-medium text-slate-900">{branch.branch_name}</h3>
                          <p className="font-mono text-xs text-slate-500">{branch.branch_key}</p>
                        </div>
                        <span className="rounded-full bg-white px-2 py-0.5 text-xs text-slate-600">
                          {branch.assignment_count} users
                        </span>
                      </div>

                      <div className="mt-4 space-y-3">
                        <div>
                          <div className="flex items-center justify-between text-sm text-slate-600">
                            <span>Configured weight</span>
                            <span>{formatPercent(branch.configured_weight)}</span>
                          </div>
                          <div className="mt-1 h-2 rounded-full bg-slate-200">
                            <div
                              className="h-2 rounded-full bg-slate-500"
                              style={{ width: `${Math.min(branch.configured_weight, 100)}%` }}
                            />
                          </div>
                        </div>

                        <div>
                          <div className="flex items-center justify-between text-sm text-slate-600">
                            <span>Actual assignment share</span>
                            <span>{formatPercent(branch.assignment_share)}</span>
                          </div>
                          <div className="mt-1 h-2 rounded-full bg-slate-200">
                            <div
                              className="h-2 rounded-full bg-slate-900"
                              style={{ width: `${Math.min(branch.assignment_share, 100)}%` }}
                            />
                          </div>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
