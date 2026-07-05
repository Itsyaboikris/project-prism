import { useParams } from "react-router-dom"
import { ExperimentEventsPanel } from "@/components/ExperimentEventsPanel"
import { ExperimentPageHeader } from "@/components/ExperimentPageHeader"
import { ExperimentStatusToggle } from "@/components/ExperimentStatusToggle"
import { useApplication } from "@/hooks/useApplication"
import { StatusBadge } from "@/components/StatusBadge"
import { Badge } from "@/components/ui/badge"
import { useEffect, useState } from "react"
import { experimentsApi } from "@/api/experiments"
import { toast } from "sonner"
import { ApiError } from "@/api/client"

export default function ExperimentEventsPage() {
  const { appId, id } = useParams<{ appId: string; id: string }>()
  const { app } = useApplication(appId)
  const [status, setStatus] = useState<string | null>(null)
  const [title, setTitle] = useState("")
  const [experimentKey, setExperimentKey] = useState("")
  const [statusLoading, setStatusLoading] = useState(false)

  useEffect(() => {
    if (!appId || !id) return
    experimentsApi.get(appId, id).then((exp) => {
      setStatus(exp.status)
      setTitle(exp.name)
      setExperimentKey(exp.key)
    })
  }, [appId, id])

  async function handleStatusToggle() {
    if (!appId || !id || !status || statusLoading || status === "completed") return

    const previousStatus = status
    const nextStatus = previousStatus === "active" ? "paused" : "active"

    setStatusLoading(true)
    setStatus(nextStatus)

    try {
      const experiment = await experimentsApi.get(appId, id)
      const updated = await experimentsApi.update(appId, id, {
        name: experiment.name,
        description: experiment.description,
        status: nextStatus,
        start_date: experiment.start_date,
        end_date: experiment.end_date,
      })
      setStatus(updated.status)
      toast.success("Status updated")
    } catch (err) {
      setStatus(previousStatus)
      toast.error(err instanceof ApiError ? err.message : "Failed to update experiment status")
    } finally {
      setStatusLoading(false)
    }
  }

  if (!appId || !id) return null

  return (
    <>
      <ExperimentPageHeader
        appId={appId}
        experimentId={id}
        appName={app?.name}
        title={title || "Experiment"}
        description={experimentKey || undefined}
        actions={
          status ? (
            <div className="flex items-center gap-3">
              <StatusBadge status={status as "draft"} />
              <ExperimentStatusToggle
                status={status as "draft"}
                disabled={statusLoading}
                onToggle={() => void handleStatusToggle()}
              />
              <Badge variant="secondary">Events</Badge>
            </div>
          ) : undefined
        }
      />

      <ExperimentEventsPanel
        appId={appId}
        experimentId={id}
        apiKey={app?.api_key}
        idPrefix="events-page"
      />
    </>
  )
}
