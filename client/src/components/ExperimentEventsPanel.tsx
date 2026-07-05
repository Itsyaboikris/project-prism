import { useEffect, useState } from "react"
import { Eye, Zap } from "lucide-react"
import { toast } from "sonner"
import { ApiError } from "@/api/client"
import { eventsApi, type ExperimentEventListItem, type ExperimentEventsView } from "@/api/events"
import { EmptyState } from "@/components/EmptyState"
import { ErrorState } from "@/components/ErrorState"
import { TableLoading } from "@/components/PageLoading"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { parseBranchMetadataText } from "@/lib/branchFields"
import { cn } from "@/lib/utils"

const PAGE_SIZE = 100
const EVENT_NAME_MAX_LENGTH = 64

function formatProperties(properties: unknown | null) {
  if (properties == null) return "—"
  try {
    return JSON.stringify(properties, null, 2)
  } catch {
    return "—"
  }
}

interface ExperimentEventsPanelProps {
  appId: string
  experimentId: string
  apiKey?: string
  idPrefix?: string
}

export function ExperimentEventsPanel({
  appId,
  experimentId,
  apiKey,
  idPrefix = "events",
}: ExperimentEventsPanelProps) {
  const [view, setView] = useState<ExperimentEventsView | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [eventNameFilter, setEventNameFilter] = useState("")
  const [filterInput, setFilterInput] = useState("")
  const [offset, setOffset] = useState(0)
  const [selectedEvent, setSelectedEvent] = useState<ExperimentEventListItem | null>(null)

  const [recordUserId, setRecordUserId] = useState("")
  const [recordEventName, setRecordEventName] = useState("")
  const [recordPropertiesText, setRecordPropertiesText] = useState("")
  const [recordLoading, setRecordLoading] = useState(false)
  const [recordError, setRecordError] = useState<string | null>(null)

  function loadEvents() {
    setLoading(true)
    setError(null)

    eventsApi
      .listByExperiment(appId, experimentId, {
        event_name: eventNameFilter || undefined,
        limit: PAGE_SIZE,
        offset,
      })
      .then((data) => {
        setView(data)
        setSelectedEvent((current) =>
          current ? data.events.find((event) => event.id === current.id) ?? null : null,
        )
      })
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
  }

  useEffect(() => {
    loadEvents()
  }, [appId, experimentId, eventNameFilter, offset])

  async function handleRecordEvent(e: React.FormEvent) {
    e.preventDefault()
    if (!apiKey || !view) {
      setRecordError("Application API key is unavailable.")
      return
    }

    const userId = recordUserId.trim()
    const eventName = recordEventName.trim()
    if (!userId) {
      setRecordError("User ID is required.")
      return
    }
    if (!eventName) {
      setRecordError("Event name is required.")
      return
    }
    if (eventName.length > EVENT_NAME_MAX_LENGTH) {
      setRecordError("Event name must be 64 characters or fewer.")
      return
    }

    const propertiesResult = parseBranchMetadataText(recordPropertiesText)
    if (propertiesResult.error) {
      setRecordError(propertiesResult.error)
      return
    }

    setRecordLoading(true)
    setRecordError(null)

    try {
      await eventsApi.create(apiKey, {
        user_id: userId,
        event_name: eventName,
        experiment_key: view.experiment_key,
        properties: propertiesResult.value,
      })
      toast.success("Event recorded")
      setRecordUserId("")
      setRecordEventName("")
      setRecordPropertiesText("")
      setOffset(0)
      if (offset === 0) {
        loadEvents()
      }
    } catch (err) {
      const message = err instanceof ApiError ? err.message : "Failed to record event"
      setRecordError(message)
      toast.error(message)
    } finally {
      setRecordLoading(false)
    }
  }

  const hasNextPage = (view?.events.length ?? 0) === PAGE_SIZE

  if (loading && !view) return <TableLoading rows={6} />
  if (error) return <ErrorState message={error} />
  if (!view) return null

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>Record an event</CardTitle>
          <CardDescription>
            Fill in user ID and event name, then click Record. Use this to test tracking before
            wiring up your SDK.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleRecordEvent} className="space-y-4">
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor={`${idPrefix}-user-id`}>User ID</Label>
                <Input
                  id={`${idPrefix}-user-id`}
                  value={recordUserId}
                  onChange={(e) => setRecordUserId(e.target.value)}
                  placeholder="user_123"
                  disabled={recordLoading}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor={`${idPrefix}-event-name`}>Event name</Label>
                <Input
                  id={`${idPrefix}-event-name`}
                  value={recordEventName}
                  onChange={(e) => setRecordEventName(e.target.value)}
                  placeholder="purchase"
                  maxLength={EVENT_NAME_MAX_LENGTH}
                  disabled={recordLoading}
                />
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor={`${idPrefix}-properties`}>Properties JSON (optional)</Label>
              <Textarea
                id={`${idPrefix}-properties`}
                value={recordPropertiesText}
                onChange={(e) => setRecordPropertiesText(e.target.value)}
                rows={2}
                placeholder='{"amount": 49.99}'
                className="font-mono text-xs"
                disabled={recordLoading}
              />
            </div>
            {recordError && <p className="text-sm text-destructive">{recordError}</p>}
            <Button type="submit" disabled={recordLoading || !apiKey}>
              {recordLoading ? "Recording…" : "Record event"}
            </Button>
            {!apiKey && (
              <p className="text-sm text-muted-foreground">
                Loading API key…
              </p>
            )}
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="border-b">
          <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <CardTitle>Recorded events</CardTitle>
              <CardDescription>
                {view.events.length} on this page · click a row for details
              </CardDescription>
            </div>
            <div className="flex flex-wrap items-end gap-2">
              <div className="space-y-1.5">
                <Label htmlFor={`${idPrefix}-filter`} className="text-xs">
                  Filter by name
                </Label>
                <Input
                  id={`${idPrefix}-filter`}
                  value={filterInput}
                  onChange={(e) => setFilterInput(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") {
                      setEventNameFilter(filterInput.trim())
                      setOffset(0)
                    }
                  }}
                  placeholder="purchase"
                  className="h-8 w-40 sm:w-48"
                />
              </div>
              <Button
                size="sm"
                onClick={() => {
                  setEventNameFilter(filterInput.trim())
                  setOffset(0)
                }}
              >
                Apply
              </Button>
              {eventNameFilter && (
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => {
                    setFilterInput("")
                    setEventNameFilter("")
                    setOffset(0)
                  }}
                >
                  Clear
                </Button>
              )}
            </div>
          </div>
        </CardHeader>
        <CardContent className="p-0">
          {loading && view.events.length === 0 ? (
            <TableLoading rows={4} />
          ) : view.events.length === 0 ? (
            <div className="p-6">
              <EmptyState
                icon={Zap}
                title={eventNameFilter ? "No matching events" : "No events yet"}
                description={
                  eventNameFilter
                    ? `Nothing matched "${eventNameFilter}".`
                    : "Use the form above to record your first test event."
                }
              />
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>User</TableHead>
                  <TableHead>Event</TableHead>
                  <TableHead className="hidden sm:table-cell">Branch</TableHead>
                  <TableHead className="hidden md:table-cell">Occurred</TableHead>
                  <TableHead className="w-10" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {view.events.map((event) => (
                  <TableRow
                    key={event.id}
                    className={cn(
                      "cursor-pointer",
                      selectedEvent?.id === event.id && "bg-muted/50",
                    )}
                    onClick={() => setSelectedEvent(event)}
                  >
                    <TableCell className="font-mono">{event.user_id}</TableCell>
                    <TableCell className="font-medium">{event.event_name}</TableCell>
                    <TableCell className="hidden sm:table-cell">
                      {event.branch_name ? (
                        <div className="flex flex-wrap items-center gap-2">
                          <span>{event.branch_name}</span>
                          {event.branch_key && (
                            <Badge variant="outline" className="font-mono text-xs">
                              {event.branch_key}
                            </Badge>
                          )}
                        </div>
                      ) : (
                        <span className="text-muted-foreground">—</span>
                      )}
                    </TableCell>
                    <TableCell className="hidden text-muted-foreground md:table-cell">
                      {new Date(event.occurred_at).toLocaleString()}
                    </TableCell>
                    <TableCell>
                      <Eye className="size-4 text-muted-foreground" />
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>

        {(offset > 0 || hasNextPage) && view.events.length > 0 && (
          <div className="flex items-center justify-between border-t px-6 py-4">
            <Button
              variant="outline"
              size="sm"
              disabled={offset === 0}
              onClick={() => setOffset(Math.max(0, offset - PAGE_SIZE))}
            >
              Previous
            </Button>
            <p className="text-sm text-muted-foreground">
              {offset + 1}–{offset + view.events.length}
            </p>
            <Button
              variant="outline"
              size="sm"
              disabled={!hasNextPage}
              onClick={() => setOffset(offset + PAGE_SIZE)}
            >
              Next
            </Button>
          </div>
        )}
      </Card>

      {selectedEvent && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Eye className="size-4" />
              Event details
            </CardTitle>
          </CardHeader>
          <CardContent>
            <dl className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              <div>
                <dt className="text-xs text-muted-foreground">Event name</dt>
                <dd className="mt-1 font-medium">{selectedEvent.event_name}</dd>
              </div>
              <div>
                <dt className="text-xs text-muted-foreground">User ID</dt>
                <dd className="mt-1 font-mono text-sm">{selectedEvent.user_id}</dd>
              </div>
              <div>
                <dt className="text-xs text-muted-foreground">Branch</dt>
                <dd className="mt-1 text-sm">{selectedEvent.branch_name ?? "—"}</dd>
              </div>
              <div>
                <dt className="text-xs text-muted-foreground">Occurred</dt>
                <dd className="mt-1 text-sm">
                  {new Date(selectedEvent.occurred_at).toLocaleString()}
                </dd>
              </div>
            </dl>
            <div className="mt-4 space-y-2">
              <Label>Properties</Label>
              <pre className="overflow-x-auto rounded-lg bg-muted/50 p-4 font-mono text-xs">
                {formatProperties(selectedEvent.properties)}
              </pre>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
