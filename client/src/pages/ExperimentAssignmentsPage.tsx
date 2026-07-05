import { useEffect, useState } from "react"
import { useParams } from "react-router-dom"
import { ListTree } from "lucide-react"
import { assignmentsApi, type ExperimentAssignmentsView } from "@/api/assignments"
import { ApiError } from "@/api/client"
import { EmptyState } from "@/components/EmptyState"
import { ErrorState } from "@/components/ErrorState"
import { ExperimentPageHeader } from "@/components/ExperimentPageHeader"
import { useApplication } from "@/hooks/useApplication"
import { TableLoading } from "@/components/PageLoading"
import { StatusBadge } from "@/components/StatusBadge"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"

export default function ExperimentAssignmentsPage() {
  const { appId, id } = useParams<{ appId: string; id: string }>()
  const { app } = useApplication(appId)

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
    <>
      {loading && <TableLoading />}

      {!loading && error && <ErrorState message={error} />}

      {!loading && !error && view && appId && id && (
        <>
          <ExperimentPageHeader
            appId={appId}
            experimentId={id}
            appName={app?.name}
            title={view.experiment_name}
            description={view.experiment_key}
            actions={
              <div className="flex items-center gap-2">
                <StatusBadge status={view.experiment_status} />
                <Badge variant="secondary">{view.assignments.length} assignments</Badge>
              </div>
            }
          />

          <Card>
            <CardHeader className="border-b">
              <CardTitle>Assignments</CardTitle>
              <CardDescription>Newest assignments first</CardDescription>
            </CardHeader>
            <CardContent className="p-0">
              {view.assignments.length === 0 ? (
                <div className="p-6">
                  <EmptyState
                    icon={ListTree}
                    title="No assignments yet"
                    description="User assignments will appear here once the experiment is running."
                  />
                </div>
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>User</TableHead>
                      <TableHead>Branch</TableHead>
                      <TableHead className="hidden sm:table-cell">Assigned</TableHead>
                      <TableHead className="hidden md:table-cell">Assignment ID</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {view.assignments.map((assignment) => (
                      <TableRow key={assignment.id}>
                        <TableCell className="font-mono">{assignment.user_id}</TableCell>
                        <TableCell>
                          <div className="flex flex-wrap items-center gap-2">
                            <span className="font-medium">{assignment.branch_name}</span>
                            <Badge variant="outline" className="font-mono text-xs">
                              {assignment.branch_key}
                            </Badge>
                          </div>
                        </TableCell>
                        <TableCell className="hidden text-muted-foreground sm:table-cell">
                          {new Date(assignment.assigned_at).toLocaleString()}
                        </TableCell>
                        <TableCell className="hidden font-mono text-xs text-muted-foreground md:table-cell">
                          {assignment.id}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              )}
            </CardContent>
          </Card>
        </>
      )}
    </>
  )
}
