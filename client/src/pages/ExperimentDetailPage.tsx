import { useEffect, useState } from "react"
import { useNavigate, useParams } from "react-router-dom"
import { AlertTriangle, GitBranch, Pencil, Settings, Trash2, Zap } from "lucide-react"
import { toast } from "sonner"
import {
  experimentsApi,
  EXPERIMENT_STATUSES,
  type Experiment,
  type ExperimentStatus,
  type UpdateExperimentInput,
} from "@/api/experiments"
import { branchesApi, type Branch, type SaveBranchInput } from "@/api/branches"
import { ApiError } from "@/api/client"
import { ConfirmDeleteDialog } from "@/components/ConfirmDeleteDialog"
import { EmptyState } from "@/components/EmptyState"
import { ErrorState } from "@/components/ErrorState"
import { useApplication } from "@/hooks/useApplication"
import { ExperimentPageHeader } from "@/components/ExperimentPageHeader"
import { ExperimentEventsPanel } from "@/components/ExperimentEventsPanel"
import { ExperimentStatusToggle } from "@/components/ExperimentStatusToggle"
import { PageLoading } from "@/components/PageLoading"
import { StatusBadge } from "@/components/StatusBadge"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"
import { slugifyKey } from "@/lib/slugify"
import {
  BRANCH_WEIGHT_PERCENT_TOTAL,
  formatBranchWeightPercent,
  formatBranchWeightValue,
  inferStoredBranchWeightScale,
  sumWeights,
  toDisplayBranchWeight,
  toStoredBranchWeight,
  validateDisplayBranchWeights,
} from "@/lib/branchWeights"
import {
  BRANCH_KEY_MAX_LENGTH,
  BRANCH_NAME_MAX_LENGTH,
  parseBranchMetadataText,
  validateBranchKey,
  validateBranchName,
} from "@/lib/branchFields"
import { validateExperimentDateRange } from "@/lib/experimentDates"
import {
  EXPERIMENT_DESCRIPTION_MAX_LENGTH,
  EXPERIMENT_NAME_MAX_LENGTH,
  validateExperimentDescription,
  validateExperimentName,
} from "@/lib/experimentFields"


interface BranchDraft {
  id?: string
  key: string
  name: string
  weight: string
  metadataText: string
  isKeyCustom: boolean
}

function toDatetimeLocal(iso: string | null): string {
  if (!iso) return ""
  return iso.slice(0, 16)
}

function fromDatetimeLocal(value: string): string | null {
  return value ? new Date(value).toISOString() : null
}

function createEmptyBranchDraft(weight = "0"): BranchDraft {
  return {
    key: "",
    name: "",
    weight,
    metadataText: "",
    isKeyCustom: false,
  }
}

function createBranchDraftFromBranch(branch: Branch, branchWeightScale: "percent" | "fraction"): BranchDraft {
  return {
    id: branch.id,
    key: branch.key,
    name: branch.name,
    weight: formatBranchWeightValue(toDisplayBranchWeight(branch.weight, branchWeightScale)),
    metadataText: branch.metadata_json == null ? "" : JSON.stringify(branch.metadata_json, null, 2),
    isKeyCustom: true,
  }
}

function getRemainingWeight(weights: number[]): string {
  return formatBranchWeightValue(Math.max(0, BRANCH_WEIGHT_PERCENT_TOTAL - sumWeights(weights)))
}

function formatBranchCount(count: number) {
  return `${count} branch${count === 1 ? "" : "es"}`
}

function formatMetadata(metadata: unknown | null) {
  if (metadata == null) return "No metadata"
  return JSON.stringify(metadata, null, 2)
}

function validateBranchDraft(branch: BranchDraft): string | null {
  return (
    validateBranchName(branch.name) ??
    validateBranchKey(branch.key) ??
    parseBranchMetadataText(branch.metadataText).error
  )
}

export default function ExperimentDetailPage() {
  const { appId, id } = useParams<{ appId: string; id: string }>()
  const navigate = useNavigate()

  const [experiment, setExperiment] = useState<Experiment | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const [editing, setEditing] = useState(false)
  const [form, setForm] = useState<UpdateExperimentInput>({ name: "" })
  const [editLoading, setEditLoading] = useState(false)
  const [editError, setEditError] = useState<string | null>(null)
  const [deleteLoading, setDeleteLoading] = useState(false)
  const [deleteError, setDeleteError] = useState<string | null>(null)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [statusLoading, setStatusLoading] = useState(false)

  const { app } = useApplication(appId)

  const [editingBranches, setEditingBranches] = useState(false)
  const [branchDrafts, setBranchDrafts] = useState<BranchDraft[]>([])
  const [branchLoading, setBranchLoading] = useState(false)
  const [branchError, setBranchError] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState("settings")

  const branchWeightScale = inferStoredBranchWeightScale(
    experiment?.branches.map((branch) => branch.weight) ?? [],
  )
  const branchDisplayWeights = experiment
    ? experiment.branches.map((branch) => toDisplayBranchWeight(branch.weight, branchWeightScale))
    : []
  const currentBranchWeightTotal = sumWeights(branchDisplayWeights)
  const currentBranchWeightError = validateDisplayBranchWeights(branchDisplayWeights)

  useEffect(() => {
    if (!appId || !id) return
    experimentsApi
      .get(appId, id)
      .then((data) => {
        setExperiment(data)
        initForm(data)
      })
      .catch((err) =>
        setError(
          err instanceof ApiError && err.status === 404
            ? "Experiment not found."
            : err instanceof ApiError
              ? err.message
              : "Failed to load experiment",
        ),
      )
      .finally(() => setLoading(false))
  }, [appId, id])

  function initForm(exp: Experiment) {
    setForm({
      name: exp.name,
      description: exp.description,
      status: exp.status,
      start_date: exp.start_date,
      end_date: exp.end_date,
    })
  }

  function openBranchEditor() {
    if (!experiment) return
    setActiveTab("branches")
    setEditingBranches(true)
    setBranchDrafts(
      experiment.branches.map((branch) => createBranchDraftFromBranch(branch, branchWeightScale)),
    )
    setBranchError(null)
  }

  function cancelBranchEditor() {
    setEditingBranches(false)
    setBranchDrafts([])
    setBranchError(null)
  }

  async function handleSave(e: React.FormEvent) {
    e.preventDefault()
    if (!appId || !id || !experiment) return
    const nameError = validateExperimentName(form.name)
    if (nameError) {
      setEditError(nameError)
      return
    }
    const descriptionError = validateExperimentDescription(form.description)
    if (descriptionError) {
      setEditError(descriptionError)
      return
    }
    const dateError = validateExperimentDateRange(form.start_date, form.end_date)
    if (dateError) {
      setEditError(dateError)
      return
    }
    setEditLoading(true)
    setEditError(null)
    try {
      const updated = await experimentsApi.update(appId, id, {
        name: form.name.trim(),
        description: form.description?.trim() || null,
        status: form.status,
        start_date: form.start_date,
        end_date: form.end_date,
      })
      setExperiment(updated)
      setEditing(false)
      toast.success("Changes saved")
    } catch (err) {
      const message = err instanceof ApiError ? err.message : "Failed to update experiment"
      setEditError(message)
      toast.error(message)
    } finally {
      setEditLoading(false)
    }
  }

  function handleBranchNameChange(index: number, name: string) {
    setBranchDrafts((current) =>
      current.map((branch, branchIndex) => {
        if (branchIndex !== index) return branch
        const generatedKey = slugifyKey(name)
        return {
          ...branch,
          name,
          key: branch.isKeyCustom ? branch.key : generatedKey,
        }
      }),
    )
  }

  function handleBranchKeyChange(index: number, key: string) {
    setBranchDrafts((current) =>
      current.map((branch, branchIndex) =>
        branchIndex === index
          ? {
              ...branch,
              key,
              isKeyCustom: key !== slugifyKey(branch.name),
            }
          : branch,
      ),
    )
  }

  function addBranchDraft() {
    setBranchDrafts((current) => [
      ...current,
      createEmptyBranchDraft(
        getRemainingWeight(
          current
            .map((branch) => Number(branch.weight))
            .filter((weight) => !Number.isNaN(weight)),
        ),
      ),
    ])
  }

  function removeBranchDraft(index: number) {
    setBranchDrafts((current) => current.filter((_, branchIndex) => branchIndex !== index))
  }

  async function handleSaveBranches(e: React.FormEvent) {
    e.preventDefault()
    if (!appId || !id || !experiment) return

    const payload: SaveBranchInput[] = []
    const displayWeights: number[] = []

    for (const branch of branchDrafts) {
      const branchFieldError = validateBranchDraft(branch)
      if (branchFieldError) {
        setBranchError(branchFieldError)
        return
      }

      const displayWeight = Number(branch.weight)
      displayWeights.push(displayWeight)

      const metadataResult = parseBranchMetadataText(branch.metadataText)
      if (metadataResult.error) {
        setBranchError(metadataResult.error)
        return
      }

      payload.push({
        id: branch.id,
        key: branch.key.trim(),
        name: branch.name.trim(),
        weight: toStoredBranchWeight(displayWeight, branchWeightScale),
        metadata_json: metadataResult.value,
      })
    }

    const weightError = validateDisplayBranchWeights(displayWeights)
    if (weightError) {
      setBranchError(weightError)
      return
    }

    setBranchLoading(true)
    setBranchError(null)

    try {
      const updatedBranches = await branchesApi.saveAll(appId, id, {
        branches: payload,
      })
      setExperiment((current) =>
        current
          ? {
              ...current,
              branches: updatedBranches,
            }
          : current,
      )
      cancelBranchEditor()
      toast.success("Changes saved")
    } catch (err) {
      const message = err instanceof ApiError ? err.message : "Failed to save branches"
      setBranchError(message)
      toast.error(message)
    } finally {
      setBranchLoading(false)
    }
  }

  function handleCancel() {
    if (experiment) initForm(experiment)
    setEditing(false)
    setEditError(null)
  }

  async function handleStatusToggle() {
    if (!appId || !id || !experiment || statusLoading || experiment.status === "completed") return

    const previousStatus = experiment.status
    const nextStatus = previousStatus === "active" ? "paused" : "active"

    setStatusLoading(true)
    setExperiment({ ...experiment, status: nextStatus })

    try {
      const updated = await experimentsApi.update(appId, id, {
        name: experiment.name,
        description: experiment.description,
        status: nextStatus,
        start_date: experiment.start_date,
        end_date: experiment.end_date,
      })
      setExperiment(updated)
      setForm((current) => ({ ...current, status: updated.status }))
      toast.success("Status updated")
    } catch (err) {
      setExperiment({ ...experiment, status: previousStatus })
      toast.error(err instanceof ApiError ? err.message : "Failed to update experiment status")
    } finally {
      setStatusLoading(false)
    }
  }

  async function handleDelete() {
    if (!appId || !id || !experiment) return

    setDeleteLoading(true)
    setDeleteError(null)
    try {
      await experimentsApi.delete(appId, id)
      toast.success(`Deleted ${experiment.name}`)
      navigate(`/applications/${appId}/experiments`)
    } catch (err) {
      const message = err instanceof ApiError ? err.message : "Failed to delete experiment"
      setDeleteError(message)
      toast.error(message)
      setDeleteLoading(false)
    }
  }

  const draftBranchWeights = branchDrafts.map((branch) => Number(branch.weight))
  const draftBranchWeightTotal = sumWeights(
    draftBranchWeights.filter((weight) => !Number.isNaN(weight)),
  )
  const draftBranchWeightError = validateDisplayBranchWeights(draftBranchWeights)
  const draftBranchFieldError =
    branchDrafts.map((branch) => validateBranchDraft(branch)).find(Boolean) ?? null
  return (
    <>
      {loading && <PageLoading />}

      {!loading && error && <ErrorState message={error} />}

      {!loading && !error && experiment && appId && id && (
        <>
          <ExperimentPageHeader
            appId={appId}
            experimentId={id}
            appName={app?.name}
            title={experiment.name}
            description={experiment.key}
            actions={
              <div className="flex items-center gap-3">
                <StatusBadge status={experiment.status} />
                <ExperimentStatusToggle
                  status={experiment.status}
                  disabled={statusLoading}
                  onToggle={() => void handleStatusToggle()}
                />
                <Badge variant="secondary">{formatBranchCount(experiment.branches.length)}</Badge>
              </div>
            }
          />

          <Tabs value={activeTab} onValueChange={setActiveTab}>
            <TabsList>
              <TabsTrigger value="settings">
                <Settings className="size-4" />
                Settings
              </TabsTrigger>
              <TabsTrigger value="branches">
                <GitBranch className="size-4" />
                Branches ({experiment.branches.length})
              </TabsTrigger>
              <TabsTrigger value="events">
                <Zap className="size-4" />
                Events
              </TabsTrigger>
              <TabsTrigger value="danger">
                <AlertTriangle className="size-4" />
                Danger zone
              </TabsTrigger>
            </TabsList>

            <TabsContent value="settings" className="mt-4 space-y-4">
              <Card>
                <CardHeader className="flex-row items-start justify-between space-y-0">
                  <div>
                    <CardTitle>Experiment settings</CardTitle>
                    <CardDescription>Name, schedule, and status for this experiment.</CardDescription>
                  </div>
                  {!editing && (
                    <Button variant="outline" size="sm" onClick={() => setEditing(true)}>
                      <Pencil />
                      Edit
                    </Button>
                  )}
                </CardHeader>
                <CardContent>
                  {editing ? (
                    <form onSubmit={handleSave} className="space-y-4">
                      <div className="grid gap-4 sm:grid-cols-2">
                        <div className="space-y-2">
                          <Label htmlFor="exp-name">Name</Label>
                          <Input
                            id="exp-name"
                            autoFocus
                            value={form.name}
                            onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                            maxLength={EXPERIMENT_NAME_MAX_LENGTH}
                            disabled={editLoading}
                          />
                        </div>
                        <div className="space-y-2">
                          <Label htmlFor="exp-status">Status</Label>
                          <Select
                            value={form.status}
                            onValueChange={(value) =>
                              setForm((f) => ({ ...f, status: value as ExperimentStatus }))
                            }
                            disabled={editLoading}
                          >
                            <SelectTrigger id="exp-status" className="w-full">
                              <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                              {EXPERIMENT_STATUSES.map((s) => (
                                <SelectItem key={s} value={s}>
                                  {s}
                                </SelectItem>
                              ))}
                            </SelectContent>
                          </Select>
                        </div>
                      </div>
                      <div className="space-y-2">
                        <Label htmlFor="exp-desc">Description</Label>
                        <Textarea
                          id="exp-desc"
                          value={form.description ?? ""}
                          onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
                          rows={2}
                          maxLength={EXPERIMENT_DESCRIPTION_MAX_LENGTH}
                          disabled={editLoading}
                        />
                      </div>
                      <div className="grid gap-4 sm:grid-cols-2">
                        <div className="space-y-2">
                          <Label htmlFor="exp-start">Start date</Label>
                          <Input
                            id="exp-start"
                            type="datetime-local"
                            value={toDatetimeLocal(form.start_date ?? null)}
                            onChange={(e) =>
                              setForm((f) => ({ ...f, start_date: fromDatetimeLocal(e.target.value) }))
                            }
                            disabled={editLoading}
                          />
                        </div>
                        <div className="space-y-2">
                          <Label htmlFor="exp-end">End date</Label>
                          <Input
                            id="exp-end"
                            type="datetime-local"
                            value={toDatetimeLocal(form.end_date ?? null)}
                            onChange={(e) =>
                              setForm((f) => ({ ...f, end_date: fromDatetimeLocal(e.target.value) }))
                            }
                            disabled={editLoading}
                          />
                        </div>
                      </div>
                      {editError && <p className="text-sm text-destructive">{editError}</p>}
                      <div className="flex gap-2">
                        <Button
                          type="submit"
                          disabled={
                            editLoading ||
                            Boolean(validateExperimentName(form.name)) ||
                            Boolean(validateExperimentDescription(form.description))
                          }
                        >
                          {editLoading ? "Saving…" : "Save"}
                        </Button>
                        <Button type="button" variant="outline" onClick={handleCancel} disabled={editLoading}>
                          Cancel
                        </Button>
                      </div>
                    </form>
                  ) : (
                    <>
                      {experiment.description && (
                        <p className="mb-4 text-sm text-muted-foreground">{experiment.description}</p>
                      )}
                      <dl className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                        <div>
                          <dt className="text-xs text-muted-foreground">Key</dt>
                          <dd className="mt-1 font-mono text-sm">{experiment.key}</dd>
                        </div>
                        <div>
                          <dt className="text-xs text-muted-foreground">ID</dt>
                          <dd className="mt-1 font-mono text-sm">{experiment.id}</dd>
                        </div>
                        <div>
                          <dt className="text-xs text-muted-foreground">Status</dt>
                          <dd className="mt-1"><StatusBadge status={experiment.status} /></dd>
                        </div>
                        <div>
                          <dt className="text-xs text-muted-foreground">Start date</dt>
                          <dd className="mt-1 text-sm">
                            {experiment.start_date
                              ? new Date(experiment.start_date).toLocaleString()
                              : "—"}
                          </dd>
                        </div>
                        <div>
                          <dt className="text-xs text-muted-foreground">End date</dt>
                          <dd className="mt-1 text-sm">
                            {experiment.end_date
                              ? new Date(experiment.end_date).toLocaleString()
                              : "—"}
                          </dd>
                        </div>
                        <div>
                          <dt className="text-xs text-muted-foreground">Created</dt>
                          <dd className="mt-1 text-sm">
                            {new Date(experiment.created_at).toLocaleString()}
                          </dd>
                        </div>
                      </dl>
                    </>
                  )}
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="events" className="mt-4">
              <ExperimentEventsPanel
                appId={appId}
                experimentId={id}
                apiKey={app?.api_key}
                idPrefix="detail-events"
              />
            </TabsContent>

            <TabsContent value="branches" className="mt-4">
              <Card>
                <CardHeader className="flex-row items-start justify-between space-y-0">
                  <div>
                    <CardTitle>Branches</CardTitle>
                    <CardDescription>
                      Variants and traffic weights. Allocation should total 100%.
                    </CardDescription>
                    {experiment.branches.length > 0 && !editingBranches && (
                      <p
                        className={`mt-2 text-sm ${currentBranchWeightError ? "text-destructive" : "text-muted-foreground"}`}
                      >
                        Current allocation: {formatBranchWeightValue(currentBranchWeightTotal)}%.
                      </p>
                    )}
                  </div>
                  {!editingBranches && (
                    <Button variant="outline" size="sm" onClick={openBranchEditor}>
                      {experiment.branches.length === 0 ? "Add branches" : "Edit branches"}
                    </Button>
                  )}
                </CardHeader>
                <CardContent>
                  {editingBranches ? (
                    <form onSubmit={handleSaveBranches} className="space-y-4">
                      <div className="flex items-center justify-between">
                        <p
                          className={`text-sm ${draftBranchWeightError ? "text-destructive" : "text-muted-foreground"}`}
                        >
                          Draft total: {formatBranchWeightValue(draftBranchWeightTotal)}%.
                        </p>
                        <div className="flex gap-2">
                          <Button type="button" variant="outline" size="sm" onClick={addBranchDraft} disabled={branchLoading}>
                            Add branch
                          </Button>
                          <Button type="button" variant="outline" size="sm" onClick={cancelBranchEditor} disabled={branchLoading}>
                            Cancel
                          </Button>
                        </div>
                      </div>

                      {branchDrafts.length === 0 && (
                        <EmptyState title="No branches added" description="Add at least one branch to save." />
                      )}

                      {branchDrafts.map((branch, index) => (
                        <div key={branch.id ?? `new-${index}`} className="space-y-3 rounded-lg border p-4">
                          <div className="grid gap-3 sm:grid-cols-[1.2fr_1.2fr_0.8fr_auto]">
                            <div className="space-y-2">
                              <Label>Name</Label>
                              <Input
                                autoFocus={index === 0}
                                value={branch.name}
                                onChange={(e) => handleBranchNameChange(index, e.target.value)}
                                placeholder="Control"
                                maxLength={BRANCH_NAME_MAX_LENGTH}
                                disabled={branchLoading}
                              />
                            </div>
                            <div className="space-y-2">
                              <Label>Key</Label>
                              <Input
                                value={branch.key}
                                onChange={(e) => handleBranchKeyChange(index, e.target.value)}
                                placeholder="control"
                                maxLength={BRANCH_KEY_MAX_LENGTH}
                                className="font-mono"
                                disabled={branchLoading || Boolean(branch.id)}
                              />
                            </div>
                            <div className="space-y-2">
                              <Label>Weight</Label>
                              <Input
                                type="number"
                                min="0"
                                max="100"
                                step="0.01"
                                value={branch.weight}
                                onChange={(e) =>
                                  setBranchDrafts((current) =>
                                    current.map((b, i) =>
                                      i === index ? { ...b, weight: e.target.value } : b,
                                    ),
                                  )
                                }
                                disabled={branchLoading}
                              />
                            </div>
                            <div className="flex items-end">
                              <Button
                                type="button"
                                variant="outline"
                                size="sm"
                                onClick={() => removeBranchDraft(index)}
                                disabled={branchLoading}
                              >
                                Remove
                              </Button>
                            </div>
                          </div>
                          <div className="space-y-2">
                            <Label>Metadata JSON</Label>
                            <Textarea
                              value={branch.metadataText}
                              onChange={(e) =>
                                setBranchDrafts((current) =>
                                  current.map((b, i) =>
                                    i === index ? { ...b, metadataText: e.target.value } : b,
                                  ),
                                )
                              }
                              rows={3}
                              placeholder='{"color":"#22c55e"}'
                              className="font-mono text-xs"
                              disabled={branchLoading}
                            />
                          </div>
                        </div>
                      ))}

                      {branchError && <p className="text-sm text-destructive">{branchError}</p>}
                      <Button
                        type="submit"
                        disabled={
                          branchLoading ||
                          Boolean(draftBranchFieldError) ||
                          Boolean(draftBranchWeightError)
                        }
                      >
                        {branchLoading ? "Saving…" : "Save branch changes"}
                      </Button>
                    </form>
                  ) : experiment.branches.length === 0 ? (
                    <EmptyState
                      title="No branches yet"
                      description="Add branches to define experiment variants and traffic split."
                      action={
                        <Button variant="outline" size="sm" onClick={openBranchEditor}>
                          Add branches
                        </Button>
                      }
                    />
                  ) : (
                    <div className="space-y-3">
                      {experiment.branches.map((branch) => (
                        <div key={branch.id} className="rounded-lg border bg-muted/30 p-4">
                          <div className="flex flex-wrap items-center gap-3">
                            <h3 className="font-medium">{branch.name}</h3>
                            <Badge variant="outline">
                              {formatBranchWeightPercent(branch.weight, branchWeightScale)}
                            </Badge>
                            <span className="font-mono text-xs text-muted-foreground">{branch.key}</span>
                          </div>
                          <pre className="mt-3 overflow-x-auto rounded-md bg-background p-3 font-mono text-xs">
                            {formatMetadata(branch.metadata_json)}
                          </pre>
                        </div>
                      ))}
                    </div>
                  )}
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="danger" className="mt-4">
              <Card className="border-destructive/30">
                <CardHeader className="flex-row items-start justify-between space-y-0">
                  <div>
                    <CardTitle>Delete experiment</CardTitle>
                    <CardDescription>
                      Deletes this experiment and cascades to its branches.
                    </CardDescription>
                  </div>
                  <Button
                    variant="destructive"
                    size="sm"
                    onClick={() => setDeleteDialogOpen(true)}
                    disabled={deleteLoading}
                  >
                    <Trash2 />
                    Delete
                  </Button>
                </CardHeader>
                {deleteError && (
                  <CardContent className="pt-0">
                    <p className="text-sm text-destructive">{deleteError}</p>
                  </CardContent>
                )}
              </Card>
            </TabsContent>
          </Tabs>

          <ConfirmDeleteDialog
            open={deleteDialogOpen}
            onOpenChange={setDeleteDialogOpen}
            title={`Delete ${experiment.name}?`}
            description="This will Delete the experiment and cascade to its branches. This action cannot be undone."
            loading={deleteLoading}
            onConfirm={handleDelete}
          />
        </>
      )}
    </>
  )
}
