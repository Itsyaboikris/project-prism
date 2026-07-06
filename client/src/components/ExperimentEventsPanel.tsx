import { useEffect, useState } from "react"
import { Eye, Pencil, Plus, Trash2, Zap } from "lucide-react"
import { toast } from "sonner"
import { ApiError } from "@/api/client"
import { eventsApi, type ExperimentEventListItem, type ExperimentEventsView } from "@/api/events"
import {
  trackedEventsApi,
  type TrackedEvent,
} from "@/api/trackedEvents"
import { ConfirmDeleteDialog } from "@/components/ConfirmDeleteDialog"
import { EmptyState } from "@/components/EmptyState"
import { ErrorState } from "@/components/ErrorState"
import { TableLoading } from "@/components/PageLoading"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
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
import { slugifyKey } from "@/lib/slugify"
import { cn } from "@/lib/utils"

const PAGE_SIZE = 100
const KEY_MAX_LENGTH = 64
const NAME_MAX_LENGTH = 64
const DESCRIPTION_MAX_LENGTH = 280

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
  const [trackedEvents, setTrackedEvents] = useState<TrackedEvent[]>([])
  const [trackedLoading, setTrackedLoading] = useState(true)
  const [trackedError, setTrackedError] = useState<string | null>(null)

  const [createKey, setCreateKey] = useState("")
  const [createName, setCreateName] = useState("")
  const [createDescription, setCreateDescription] = useState("")
  const [createKeyCustom, setCreateKeyCustom] = useState(false)
  const [createLoading, setCreateLoading] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)

  const [editingId, setEditingId] = useState<string | null>(null)
  const [editName, setEditName] = useState("")
  const [editDescription, setEditDescription] = useState("")
  const [editLoading, setEditLoading] = useState(false)
  const [editError, setEditError] = useState<string | null>(null)

  const [deleteTarget, setDeleteTarget] = useState<TrackedEvent | null>(null)
  const [deleteLoading, setDeleteLoading] = useState(false)

  const [view, setView] = useState<ExperimentEventsView | null>(null)
  const [occurrencesLoading, setOccurrencesLoading] = useState(true)
  const [occurrencesError, setOccurrencesError] = useState<string | null>(null)
  const [eventNameFilter, setEventNameFilter] = useState("")
  const [filterInput, setFilterInput] = useState("")
  const [offset, setOffset] = useState(0)
  const [selectedEvent, setSelectedEvent] = useState<ExperimentEventListItem | null>(null)

  const [recordUserId, setRecordUserId] = useState("")
  const [recordEventKey, setRecordEventKey] = useState("")
  const [recordPropertiesText, setRecordPropertiesText] = useState("")
  const [recordLoading, setRecordLoading] = useState(false)
  const [recordError, setRecordError] = useState<string | null>(null)

  function loadTrackedEvents() {
    setTrackedLoading(true)
    setTrackedError(null)

    trackedEventsApi
      .list(appId, experimentId)
      .then((data) => setTrackedEvents(data))
      .catch((err) =>
        setTrackedError(
          err instanceof ApiError && err.status === 404
            ? "Experiment not found."
            : err instanceof ApiError
              ? err.message
              : "Failed to load tracked events",
        ),
      )
      .finally(() => setTrackedLoading(false))
  }

  function loadOccurrences() {
    setOccurrencesLoading(true)
    setOccurrencesError(null)

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
        setOccurrencesError(
          err instanceof ApiError && err.status === 404
            ? "Experiment not found."
            : err instanceof ApiError
              ? err.message
              : "Failed to load events",
        ),
      )
      .finally(() => setOccurrencesLoading(false))
  }

  useEffect(() => {
    loadTrackedEvents()
  }, [appId, experimentId])

  useEffect(() => {
    loadOccurrences()
  }, [appId, experimentId, eventNameFilter, offset])

  useEffect(() => {
    if (recordEventKey && !trackedEvents.some((event) => event.key === recordEventKey)) {
      setRecordEventKey("")
    }
  }, [trackedEvents, recordEventKey])

  function handleCreateNameChange(name: string) {
    setCreateName(name)
    if (!createKeyCustom) {
      setCreateKey(slugifyKey(name))
    }
  }

  async function handleCreateTrackedEvent(e: React.FormEvent) {
    e.preventDefault()

    const key = createKey.trim()
    const name = createName.trim()
    const description = createDescription.trim()

    if (!key) {
      setCreateError("Key is required.")
      return
    }
    if (!name) {
      setCreateError("Name is required.")
      return
    }
    if (key.length > KEY_MAX_LENGTH || name.length > NAME_MAX_LENGTH) {
      setCreateError("Key and name must be 64 characters or fewer.")
      return
    }
    if (description.length > DESCRIPTION_MAX_LENGTH) {
      setCreateError("Description must be 280 characters or fewer.")
      return
    }

    setCreateLoading(true)
    setCreateError(null)

    try {
      await trackedEventsApi.create(appId, experimentId, {
        key,
        name,
        description: description || null,
      })
      toast.success("Tracked event created")
      setCreateKey("")
      setCreateName("")
      setCreateDescription("")
      setCreateKeyCustom(false)
      loadTrackedEvents()
    } catch (err) {
      const message = err instanceof ApiError ? err.message : "Failed to create tracked event"
      setCreateError(message)
      toast.error(message)
    } finally {
      setCreateLoading(false)
    }
  }

  function startEdit(event: TrackedEvent) {
    setEditingId(event.id)
    setEditName(event.name)
    setEditDescription(event.description ?? "")
    setEditError(null)
  }

  function cancelEdit() {
    setEditingId(null)
    setEditName("")
    setEditDescription("")
    setEditError(null)
  }

  async function handleSaveEdit(eventId: string) {
    const name = editName.trim()
    const description = editDescription.trim()

    if (!name) {
      setEditError("Name is required.")
      return
    }
    if (name.length > NAME_MAX_LENGTH) {
      setEditError("Name must be 64 characters or fewer.")
      return
    }
    if (description.length > DESCRIPTION_MAX_LENGTH) {
      setEditError("Description must be 280 characters or fewer.")
      return
    }

    setEditLoading(true)
    setEditError(null)

    try {
      await trackedEventsApi.update(appId, experimentId, eventId, {
        name,
        description: description || null,
      })
      toast.success("Tracked event updated")
      cancelEdit()
      loadTrackedEvents()
    } catch (err) {
      const message = err instanceof ApiError ? err.message : "Failed to update tracked event"
      setEditError(message)
      toast.error(message)
    } finally {
      setEditLoading(false)
    }
  }

  async function handleDeleteTrackedEvent() {
    if (!deleteTarget) return

    setDeleteLoading(true)
    try {
      await trackedEventsApi.delete(appId, experimentId, deleteTarget.id)
      toast.success("Tracked event deleted")
      if (eventNameFilter === deleteTarget.key) {
        setEventNameFilter("")
        setFilterInput("")
        setOffset(0)
      }
      setDeleteTarget(null)
      loadTrackedEvents()
      loadOccurrences()
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to delete tracked event")
    } finally {
      setDeleteLoading(false)
    }
  }

  async function handleRecordEvent(e: React.FormEvent) {
    e.preventDefault()
    if (!apiKey || !view) {
      setRecordError("Application API key is unavailable.")
      return
    }

    const userId = recordUserId.trim()
    if (!userId) {
      setRecordError("User ID is required.")
      return
    }
    if (!recordEventKey) {
      setRecordError("Select a tracked event.")
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
        event_name: recordEventKey,
        experiment_key: view.experiment_key,
        properties: propertiesResult.value,
      })
      toast.success("Event recorded")
      setRecordUserId("")
      setRecordPropertiesText("")
      setOffset(0)
      loadTrackedEvents()
      if (offset === 0) {
        loadOccurrences()
      }
    } catch (err) {
      const message = err instanceof ApiError ? err.message : "Failed to record event"
      setRecordError(message)
      toast.error(message)
    } finally {
      setRecordLoading(false)
    }
  }

  function filterByTrackedEvent(key: string) {
    setFilterInput(key)
    setEventNameFilter(key)
    setOffset(0)
  }

  const hasNextPage = (view?.events.length ?? 0) === PAGE_SIZE
  const initialLoading = trackedLoading && occurrencesLoading && !view && trackedEvents.length === 0

  if (initialLoading) return <TableLoading rows={6} />
  if (trackedError && !view) return <ErrorState message={trackedError} />

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>Tracked events</CardTitle>
          <CardDescription>
            Define the events this experiment measures. The key is sent as{" "}
            <code className="text-xs">event_name</code> when recording occurrences via the SDK.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          <form onSubmit={handleCreateTrackedEvent} className="space-y-4 rounded-lg border p-4">
            <div className="flex items-center gap-2 text-sm font-medium">
              <Plus className="size-4" />
              Add tracked event
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor={`${idPrefix}-create-name`}>Name</Label>
                <Input
                  id={`${idPrefix}-create-name`}
                  value={createName}
                  onChange={(e) => handleCreateNameChange(e.target.value)}
                  placeholder="Button click"
                  maxLength={NAME_MAX_LENGTH}
                  disabled={createLoading}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor={`${idPrefix}-create-key`}>Key</Label>
                <Input
                  id={`${idPrefix}-create-key`}
                  value={createKey}
                  onChange={(e) => {
                    setCreateKey(e.target.value)
                    setCreateKeyCustom(true)
                  }}
                  placeholder="button_click"
                  maxLength={KEY_MAX_LENGTH}
                  className="font-mono"
                  disabled={createLoading}
                />
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor={`${idPrefix}-create-description`}>Description (optional)</Label>
              <Textarea
                id={`${idPrefix}-create-description`}
                value={createDescription}
                onChange={(e) => setCreateDescription(e.target.value)}
                rows={2}
                maxLength={DESCRIPTION_MAX_LENGTH}
                disabled={createLoading}
              />
            </div>
            {createError && <p className="text-sm text-destructive">{createError}</p>}
            <Button type="submit" disabled={createLoading}>
              {createLoading ? "Creating…" : "Add event"}
            </Button>
          </form>

          {trackedLoading && trackedEvents.length === 0 ? (
            <TableLoading rows={3} />
          ) : trackedError ? (
            <ErrorState message={trackedError} />
          ) : trackedEvents.length === 0 ? (
            <EmptyState
              icon={Zap}
              title="No tracked events yet"
              description="Add an event definition above before recording occurrences."
            />
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead className="hidden sm:table-cell">Key</TableHead>
                  <TableHead className="hidden md:table-cell">Occurrences</TableHead>
                  <TableHead className="hidden lg:table-cell">Last seen</TableHead>
                  <TableHead className="w-24" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {trackedEvents.map((event) =>
                  editingId === event.id ? (
                    <TableRow key={event.id}>
                      <TableCell colSpan={5}>
                        <div className="space-y-3 py-1">
                          <div className="grid gap-3 sm:grid-cols-2">
                            <div className="space-y-1.5">
                              <Label htmlFor={`${idPrefix}-edit-name-${event.id}`}>Name</Label>
                              <Input
                                id={`${idPrefix}-edit-name-${event.id}`}
                                value={editName}
                                onChange={(e) => setEditName(e.target.value)}
                                maxLength={NAME_MAX_LENGTH}
                                disabled={editLoading}
                              />
                            </div>
                            <div className="space-y-1.5">
                              <Label>Key</Label>
                              <Input value={event.key} disabled className="font-mono" />
                            </div>
                          </div>
                          <div className="space-y-1.5">
                            <Label htmlFor={`${idPrefix}-edit-description-${event.id}`}>
                              Description
                            </Label>
                            <Textarea
                              id={`${idPrefix}-edit-description-${event.id}`}
                              value={editDescription}
                              onChange={(e) => setEditDescription(e.target.value)}
                              rows={2}
                              maxLength={DESCRIPTION_MAX_LENGTH}
                              disabled={editLoading}
                            />
                          </div>
                          {editError && <p className="text-sm text-destructive">{editError}</p>}
                          <div className="flex gap-2">
                            <Button
                              size="sm"
                              disabled={editLoading}
                              onClick={() => void handleSaveEdit(event.id)}
                            >
                              {editLoading ? "Saving…" : "Save"}
                            </Button>
                            <Button
                              size="sm"
                              variant="outline"
                              disabled={editLoading}
                              onClick={cancelEdit}
                            >
                              Cancel
                            </Button>
                          </div>
                        </div>
                      </TableCell>
                    </TableRow>
                  ) : (
                    <TableRow key={event.id}>
                      <TableCell>
                        <div>
                          <p className="font-medium">{event.name}</p>
                          {event.description && (
                            <p className="mt-0.5 text-sm text-muted-foreground">
                              {event.description}
                            </p>
                          )}
                        </div>
                      </TableCell>
                      <TableCell className="hidden sm:table-cell">
                        <Badge variant="outline" className="font-mono text-xs">
                          {event.key}
                        </Badge>
                      </TableCell>
                      <TableCell className="hidden md:table-cell">
                        <Button
                          variant="link"
                          className="h-auto p-0"
                          onClick={() => filterByTrackedEvent(event.key)}
                        >
                          {event.occurrence_count}
                        </Button>
                      </TableCell>
                      <TableCell className="hidden text-muted-foreground lg:table-cell">
                        {event.last_occurred_at
                          ? new Date(event.last_occurred_at).toLocaleString()
                          : "—"}
                      </TableCell>
                      <TableCell>
                        <div className="flex justify-end gap-1">
                          <Button
                            variant="ghost"
                            size="icon-sm"
                            aria-label={`Edit ${event.name}`}
                            onClick={() => startEdit(event)}
                          >
                            <Pencil className="size-4" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon-sm"
                            aria-label={`Delete ${event.name}`}
                            onClick={() => setDeleteTarget(event)}
                          >
                            <Trash2 className="size-4 text-destructive" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ),
                )}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Record test event</CardTitle>
          <CardDescription>
            Send a test occurrence through the SDK endpoint using a registered event key.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {trackedEvents.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              Add at least one tracked event before recording test occurrences.
            </p>
          ) : (
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
                  <Label htmlFor={`${idPrefix}-event-key`}>Tracked event</Label>
                  <Select
                    value={recordEventKey}
                    onValueChange={(value) => setRecordEventKey(value ?? "")}
                    disabled={recordLoading}
                  >
                    <SelectTrigger id={`${idPrefix}-event-key`} className="w-full">
                      <SelectValue placeholder="Select an event" />
                    </SelectTrigger>
                    <SelectContent>
                      {trackedEvents.map((event) => (
                        <SelectItem key={event.id} value={event.key}>
                          {event.name} ({event.key})
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
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
              <Button type="submit" disabled={recordLoading || !apiKey || !recordEventKey}>
                {recordLoading ? "Recording…" : "Record event"}
              </Button>
              {!apiKey && (
                <p className="text-sm text-muted-foreground">Loading API key…</p>
              )}
            </form>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="border-b">
          <div className="flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <CardTitle>Recorded occurrences</CardTitle>
              <CardDescription>
                {view ? `${view.events.length} on this page` : "Loading…"} · click a row for details
              </CardDescription>
            </div>
            <div className="flex flex-wrap items-end gap-2">
              <div className="space-y-1.5">
                <Label htmlFor={`${idPrefix}-filter`} className="text-xs">
                  Filter by key
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
                  className="h-8 w-40 font-mono text-xs sm:w-48"
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
          {occurrencesError ? (
            <div className="p-6">
              <ErrorState message={occurrencesError} />
            </div>
          ) : occurrencesLoading && !view ? (
            <TableLoading rows={4} />
          ) : !view || view.events.length === 0 ? (
            <div className="p-6">
              <EmptyState
                icon={Zap}
                title={eventNameFilter ? "No matching occurrences" : "No occurrences yet"}
                description={
                  eventNameFilter
                    ? `Nothing matched "${eventNameFilter}".`
                    : "Record a test event or send events from your application."
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

        {(offset > 0 || hasNextPage) && view && view.events.length > 0 && (
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
              Occurrence details
            </CardTitle>
          </CardHeader>
          <CardContent>
            <dl className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              <div>
                <dt className="text-xs text-muted-foreground">Event key</dt>
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

      <ConfirmDeleteDialog
        open={deleteTarget != null}
        onOpenChange={(open) => {
          if (!open) setDeleteTarget(null)
        }}
        title="Delete tracked event?"
        description={
          deleteTarget
            ? `Remove "${deleteTarget.name}" (${deleteTarget.key}). Existing occurrences are kept, but new SDK events with this key will be rejected.`
            : ""
        }
        loading={deleteLoading}
        onConfirm={handleDeleteTrackedEvent}
      />
    </div>
  )
}
