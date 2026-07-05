import { useEffect, useState } from "react"
import { Link, useNavigate, useParams } from "react-router-dom"
import { ChevronRight, FlaskConical, Plus } from "lucide-react"
import { toast } from "sonner"
import { applicationsApi, type Application } from "@/api/applications"
import { experimentsApi, type Experiment } from "@/api/experiments"
import { ApiError } from "@/api/client"
import { ExperimentStatusToggle } from "@/components/ExperimentStatusToggle"
import { EmptyState, EmptyStateButton } from "@/components/EmptyState"
import { ErrorState } from "@/components/ErrorState"
import { PageHeader } from "@/components/PageHeader"
import { PageLoading } from "@/components/PageLoading"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { StatusBadge } from "../components/StatusBadge"

function formatBranchCount(count: number) {
  return `${count} branch${count === 1 ? "" : "es"}`
}

export default function ExperimentsPage() {
  const { appId } = useParams<{ appId: string }>()
  const navigate = useNavigate()

  const [app, setApp] = useState<Application | null>(null)
  const [experiments, setExperiments] = useState<Experiment[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [togglingExperimentId, setTogglingExperimentId] = useState<string | null>(null)
  const [toggleError, setToggleError] = useState<string | null>(null)

  useEffect(() => {
    if (!appId) return
    Promise.all([applicationsApi.get(appId), experimentsApi.list(appId)])
      .then(([appData, expData]) => {
        setApp(appData)
        setExperiments(expData)
      })
      .catch((err) =>
        setError(err instanceof ApiError ? err.message : "Failed to load data"),
      )
      .finally(() => setLoading(false))
  }, [appId])

  function openCreateForm() {
    if (!appId || app?.status === "inactive") return
    navigate(`/applications/${appId}/experiments/new`)
  }

  async function handleExperimentToggle(experiment: Experiment) {
    if (!appId || togglingExperimentId || experiment.status === "completed") return

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
      const updated = await experimentsApi.update(appId, experiment.id, {
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
      const message = err instanceof ApiError ? err.message : "Failed to update experiment status"
      setToggleError(message)
      toast.error(message)
    } finally {
      setTogglingExperimentId(null)
    }
  }

  return (
    <>
      <PageHeader
        title="Experiments"
        description={app ? `Experiments for ${app.name}` : "Manage A/B tests for this application."}
        breadcrumbs={[
          { label: "Applications", href: "/applications" },
          { label: app?.name ?? "Application", href: appId ? `/applications/${appId}` : undefined },
          { label: "Experiments" },
        ]}
        actions={
          !loading && !error ? (
            <Button onClick={openCreateForm} disabled={app?.status === "inactive"}>
              <Plus />
              New experiment
            </Button>
          ) : undefined
        }
      />

      {app?.status === "inactive" && !loading && !error && (
        <div className="mb-6 rounded-lg border border-amber-500/20 bg-amber-500/10 p-4 text-sm text-amber-400">
          This application is inactive. Existing experiments remain visible, but creating new
          experiments is disabled until the application is reactivated.
        </div>
      )}

      {loading && <PageLoading rows={5} />}

      {!loading && error && <ErrorState message={error} className="mb-6" />}

      {!loading && !error && experiments.length === 0 && (
        <EmptyState
          icon={FlaskConical}
          title="No experiments yet"
          description={
            app?.status === "inactive"
              ? "Reactivate the application to create experiments."
              : "Create an A/B test to start assigning users to variants."
          }
          action={
            <EmptyStateButton onClick={openCreateForm} disabled={app?.status === "inactive"}>
              Create experiment
            </EmptyStateButton>
          }
        />
      )}

      {!loading && !error && experiments.length > 0 && (
        <div className="space-y-3">
          {toggleError && <ErrorState message={toggleError} />}
          <Card>
            <CardContent className="p-0">
              <ul className="divide-y">
                {experiments.map((exp) => (
                  <li
                    key={exp.id}
                    className="flex items-center justify-between gap-4 px-6 py-4 transition-colors hover:bg-muted/30"
                  >
                    <Link
                      to={`/applications/${appId}/experiments/${exp.id}`}
                      className="min-w-0 flex-1"
                    >
                      <div className="flex items-center gap-3">
                        <p className="font-medium text-foreground">{exp.name}</p>
                        <StatusBadge status={exp.status} />
                        <span className="rounded-full bg-muted px-2 py-0.5 text-xs text-muted-foreground">
                          {formatBranchCount(exp.branches.length)}
                        </span>
                      </div>
                      <p className="mt-0.5 font-mono text-xs text-muted-foreground">{exp.key}</p>
                      {exp.description && (
                        <p className="mt-1 truncate text-sm text-muted-foreground">
                          {exp.description}
                        </p>
                      )}
                    </Link>

                    <div className="flex shrink-0 items-center gap-4">
                      <span className="text-xs text-muted-foreground">
                        {new Date(exp.created_at).toLocaleDateString()}
                      </span>
                      <ExperimentStatusToggle
                        status={exp.status}
                        disabled={togglingExperimentId === exp.id}
                        onToggle={() => void handleExperimentToggle(exp)}
                      />
                      <Link
                        to={`/applications/${appId}/experiments/${exp.id}`}
                        className="text-muted-foreground hover:text-foreground"
                        aria-label={`Open ${exp.name}`}
                      >
                        <ChevronRight className="size-4" />
                      </Link>
                    </div>
                  </li>
                ))}
              </ul>
            </CardContent>
          </Card>
        </div>
      )}
    </>
  )
}
