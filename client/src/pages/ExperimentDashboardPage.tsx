import { useEffect, useMemo, useState } from "react"
import { useParams, useSearchParams } from "react-router-dom"
import { BarChart3, Filter } from "lucide-react"
import { assignmentsApi, type ExperimentDashboard } from "@/api/assignments"
import { ApiError } from "@/api/client"
import { EmptyState } from "@/components/EmptyState"
import { ErrorState } from "@/components/ErrorState"
import { ExperimentPageHeader } from "@/components/ExperimentPageHeader"
import { useApplication } from "@/hooks/useApplication"
import { PageLoading } from "@/components/PageLoading"
import { StatusBadge } from "@/components/StatusBadge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { formatBranchWeightValue } from "@/lib/branchWeights"

function formatPercent(value: number) {
  return `${formatBranchWeightValue(value)}%`
}

export default function ExperimentDashboardPage() {
  const { appId, id } = useParams<{ appId: string; id: string }>()
  const { app } = useApplication(appId)
  const [searchParams, setSearchParams] = useSearchParams()

  const eventNameFilter = searchParams.get("event_name") ?? ""

  const [dashboard, setDashboard] = useState<ExperimentDashboard | null>(null)
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

    assignmentsApi
      .getExperimentDashboard(appId, id, eventNameFilter || undefined)
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
  }, [appId, id, eventNameFilter])

  const topBranch = useMemo(() => {
    if (!dashboard?.branches.length) return null
    const branch = [...dashboard.branches].sort((a, b) => b.assignment_count - a.assignment_count)[0]
    return branch.assignment_count > 0 ? branch : null
  }, [dashboard])

  const topConversionBranch = useMemo(() => {
    if (!dashboard?.event_name) return null
    const branch = [...dashboard.branches].sort(
      (a, b) => (b.conversion_rate ?? 0) - (a.conversion_rate ?? 0),
    )[0]
    return (branch.conversion_rate ?? 0) > 0 ? branch : null
  }, [dashboard])

  function applyEventFilter() {
    const trimmed = filterInput.trim()
    if (trimmed) {
      setSearchParams({ event_name: trimmed })
    } else {
      setSearchParams({})
    }
  }

  function clearEventFilter() {
    setFilterInput("")
    setSearchParams({})
  }

  const showingConversion = Boolean(dashboard?.event_name)

  return (
    <>
      {loading && <PageLoading rows={4} />}

      {!loading && error && <ErrorState message={error} />}

      {!loading && !error && dashboard && appId && id && (
        <>
          <ExperimentPageHeader
            appId={appId}
            experimentId={id}
            appName={app?.name}
            title={dashboard.experiment_name}
            description={dashboard.experiment_key}
            actions={<StatusBadge status={dashboard.experiment_status} />}
          />

          <div className="space-y-6">
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Filter className="size-4" />
                  Conversion filter
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-6">
              <div className="flex flex-wrap items-end gap-2">
                <div className="space-y-1.5">
                  <Label htmlFor="conversion-event">Conversion event</Label>
                  <Input
                    id="conversion-event"
                    value={filterInput}
                    onChange={(e) => setFilterInput(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter") applyEventFilter()
                    }}
                    placeholder="purchase"
                    className="w-48"
                  />
                </div>
                <Button size="sm" onClick={applyEventFilter}>
                  <Filter />
                  Apply
                </Button>
                {eventNameFilter && (
                  <Button size="sm" variant="outline" onClick={clearEventFilter}>
                    Clear
                  </Button>
                )}
              </div>

              <div className="grid gap-4 sm:grid-cols-3">
                <div className="rounded-lg border border-border bg-muted/40 p-4">
                  <p className="flex items-center gap-1.5 text-xs font-medium uppercase tracking-wide text-muted-foreground">
                    <BarChart3 className="size-3.5" />
                    Total assignments
                  </p>
                  <p className="mt-2 text-2xl font-semibold text-foreground">
                    {dashboard.total_assignments}
                  </p>
                </div>
                <div className="rounded-lg border border-border bg-muted/40 p-4">
                  <p className="flex items-center gap-1.5 text-xs font-medium uppercase tracking-wide text-muted-foreground">
                    <BarChart3 className="size-3.5" />
                    Branches tracked
                  </p>
                  <p className="mt-2 text-2xl font-semibold text-foreground">
                    {dashboard.branch_count}
                  </p>
                </div>
                <div className="rounded-lg border border-border bg-muted/40 p-4">
                  <p className="flex items-center gap-1.5 text-xs font-medium uppercase tracking-wide text-muted-foreground">
                    <BarChart3 className="size-3.5" />
                    {showingConversion ? "Top conversion" : "Leading branch"}
                  </p>
                  {showingConversion ? (
                    <>
                      <p className="mt-2 text-lg font-semibold text-foreground">
                        {topConversionBranch ? topConversionBranch.branch_name : "No conversions yet"}
                      </p>
                      {topConversionBranch && (
                        <p className="mt-1 text-sm text-muted-foreground">
                          {formatPercent(topConversionBranch.conversion_rate ?? 0)} for{" "}
                          {dashboard.event_name}
                        </p>
                      )}
                    </>
                  ) : (
                    <>
                      <p className="mt-2 text-lg font-semibold text-foreground">
                        {topBranch ? topBranch.branch_name : "No assignments yet"}
                      </p>
                      {topBranch && (
                        <p className="mt-1 text-sm text-muted-foreground">
                          {topBranch.assignment_count} users, {formatPercent(topBranch.assignment_share)}
                        </p>
                      )}
                    </>
                  )}
                </div>
              </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Branch distribution</CardTitle>
              </CardHeader>
              <CardContent>
              <p className="mb-4 text-sm text-muted-foreground">
                {showingConversion
                  ? `Assignment share and ${dashboard.event_name} conversion by branch.`
                  : "Compare configured traffic with actual assignment share."}
              </p>

              {dashboard.branches.length === 0 ? (
                <EmptyState
                  title="No branches configured"
                  description="Add branches on the experiment overview page to see distribution here."
                />
              ) : (
                <div className="mt-6 grid gap-4 md:grid-cols-2">
                  {dashboard.branches.map((branch) => (
                    <div
                      key={branch.branch_id}
                      className="rounded-lg border border-border bg-muted/40 p-4"
                    >
                      <div className="flex flex-wrap items-center justify-between gap-3">
                        <div>
                          <h3 className="font-medium text-foreground">{branch.branch_name}</h3>
                          <p className="font-mono text-xs text-muted-foreground">{branch.branch_key}</p>
                        </div>
                        <span className="rounded-full bg-card px-2 py-0.5 text-xs text-muted-foreground">
                          {branch.assignment_count} users
                        </span>
                      </div>

                      <div className="mt-4 space-y-3">
                        <div>
                          <div className="flex items-center justify-between text-sm text-muted-foreground">
                            <span>Configured weight</span>
                            <span>{formatPercent(branch.configured_weight)}</span>
                          </div>
                          <div className="mt-1 h-2 rounded-full bg-muted">
                            <div
                              className="h-2 rounded-full bg-muted/400"
                              style={{ width: `${Math.min(branch.configured_weight, 100)}%` }}
                            />
                          </div>
                        </div>

                        <div>
                          <div className="flex items-center justify-between text-sm text-muted-foreground">
                            <span>Actual assignment share</span>
                            <span>{formatPercent(branch.assignment_share)}</span>
                          </div>
                          <div className="mt-1 h-2 rounded-full bg-muted">
                            <div
                              className="h-2 rounded-full bg-primary"
                              style={{ width: `${Math.min(branch.assignment_share, 100)}%` }}
                            />
                          </div>
                        </div>

                        {showingConversion && (
                          <div>
                            <div className="flex items-center justify-between text-sm text-muted-foreground">
                              <span>Conversion rate</span>
                              <span>{formatPercent(branch.conversion_rate ?? 0)}</span>
                            </div>
                            <div className="mt-1 h-2 rounded-full bg-muted">
                              <div
                                className="h-2 rounded-full bg-emerald-600"
                                style={{ width: `${Math.min(branch.conversion_rate ?? 0, 100)}%` }}
                              />
                            </div>
                            <p className="mt-2 text-xs text-muted-foreground">
                              {branch.unique_event_users ?? 0} unique users, {branch.event_count ?? 0}{" "}
                              total events
                            </p>
                          </div>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              )}
              </CardContent>
            </Card>
          </div>
        </>
      )}
    </>
  )
}
