import { useEffect, useState } from "react"
import { Link, useNavigate, useParams } from "react-router-dom"
import {
  experimentsApi,
  EXPERIMENT_STATUSES,
  type Experiment,
  type ExperimentStatus,
  type UpdateExperimentInput,
} from "@/api/experiments"
import { branchesApi, type Branch, type SaveBranchInput } from "@/api/branches"
import { ApiError } from "@/api/client"
import { Button } from "@/components/ui/button"
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
import { StatusBadge } from "@/components/StatusBadge"
import { slugifyKey } from "@/lib/slugify"

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

  const [editingBranches, setEditingBranches] = useState(false)
  const [branchDrafts, setBranchDrafts] = useState<BranchDraft[]>([])
  const [branchLoading, setBranchLoading] = useState(false)
  const [branchError, setBranchError] = useState<string | null>(null)

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
    } catch (err) {
      setEditError(
        err instanceof ApiError ? err.message : "Failed to update experiment",
      )
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
    } catch (err) {
      setBranchError(err instanceof ApiError ? err.message : "Failed to save branches")
    } finally {
      setBranchLoading(false)
    }
  }

  function handleCancel() {
    if (experiment) initForm(experiment)
    setEditing(false)
    setEditError(null)
  }

  async function handleDelete() {
    if (!appId || !id || !experiment) return
    if (!window.confirm(`Delete "${experiment.name}"? This will also remove its branches.`)) {
      return
    }

    setDeleteLoading(true)
    setDeleteError(null)
    try {
      await experimentsApi.delete(appId, id)
      navigate(`/applications/${appId}/experiments`)
    } catch (err) {
      setDeleteError(err instanceof ApiError ? err.message : "Failed to delete experiment")
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
    <div className="min-h-screen bg-slate-50">
      <div className="mx-auto max-w-4xl px-6 py-12">
        <Link
          to={`/applications/${appId}/experiments`}
          className="inline-flex items-center gap-1.5 text-sm text-slate-500 hover:text-slate-900"
        >
          ← Experiments
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

        {!loading && !error && experiment && (
          <div className="mt-6 space-y-6">
            <div className="rounded-xl border border-slate-200 bg-white p-8 shadow-sm">
              {editing ? (
                <form onSubmit={handleSave} className="space-y-4">
                  <div className="grid gap-4 sm:grid-cols-2">
                    <div className="space-y-1">
                      <label className="text-xs font-medium uppercase tracking-wide text-slate-400">
                        Name <span className="text-red-500">*</span>
                      </label>
                      <input
                        autoFocus
                        type="text"
                        value={form.name}
                        onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                        maxLength={EXPERIMENT_NAME_MAX_LENGTH}
                        className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 placeholder:text-slate-400 focus:border-slate-500 focus:outline-none focus:ring-2 focus:ring-slate-200"
                        disabled={editLoading}
                      />
                    </div>

                    <div className="space-y-1">
                      <label className="text-xs font-medium uppercase tracking-wide text-slate-400">
                        Status
                      </label>
                      <select
                        value={form.status}
                        onChange={(e) =>
                          setForm((f) => ({ ...f, status: e.target.value as ExperimentStatus }))
                        }
                        className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-slate-500 focus:outline-none focus:ring-2 focus:ring-slate-200"
                        disabled={editLoading}
                      >
                        {EXPERIMENT_STATUSES.map((s) => (
                          <option key={s} value={s}>
                            {s}
                          </option>
                        ))}
                      </select>
                    </div>
                  </div>

                  <div className="space-y-1">
                    <label className="text-xs font-medium uppercase tracking-wide text-slate-400">
                      Description
                    </label>
                    <textarea
                      value={form.description ?? ""}
                      onChange={(e) =>
                        setForm((f) => ({ ...f, description: e.target.value }))
                      }
                      rows={2}
                      maxLength={EXPERIMENT_DESCRIPTION_MAX_LENGTH}
                      className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 placeholder:text-slate-400 focus:border-slate-500 focus:outline-none focus:ring-2 focus:ring-slate-200"
                      disabled={editLoading}
                    />
                  </div>

                  <div className="grid gap-4 sm:grid-cols-2">
                    <div className="space-y-1">
                      <label className="text-xs font-medium uppercase tracking-wide text-slate-400">
                        Start date
                      </label>
                      <input
                        type="datetime-local"
                        value={toDatetimeLocal(form.start_date ?? null)}
                        onChange={(e) =>
                          setForm((f) => ({ ...f, start_date: fromDatetimeLocal(e.target.value) }))
                        }
                        className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-slate-500 focus:outline-none focus:ring-2 focus:ring-slate-200"
                        disabled={editLoading}
                      />
                    </div>

                    <div className="space-y-1">
                      <label className="text-xs font-medium uppercase tracking-wide text-slate-400">
                        End date
                      </label>
                      <input
                        type="datetime-local"
                        value={toDatetimeLocal(form.end_date ?? null)}
                        onChange={(e) =>
                          setForm((f) => ({ ...f, end_date: fromDatetimeLocal(e.target.value) }))
                        }
                        className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-slate-500 focus:outline-none focus:ring-2 focus:ring-slate-200"
                        disabled={editLoading}
                      />
                    </div>
                  </div>

                  {editError && <p className="text-sm text-red-600">{editError}</p>}

                  <div className="flex gap-3">
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
                  <div className="flex items-start justify-between gap-4">
                    <div>
                      <div className="flex flex-wrap items-center gap-3">
                        <h1 className="text-2xl font-semibold tracking-tight text-slate-900">
                          {experiment.name}
                        </h1>
                        <StatusBadge status={experiment.status} />
                        <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-600">
                          {formatBranchCount(experiment.branches.length)}
                        </span>
                      </div>
                      {experiment.description && (
                        <p className="mt-2 text-sm text-slate-600">{experiment.description}</p>
                      )}
                    </div>
                    <Button variant="outline" onClick={() => setEditing(true)}>
                      Edit
                    </Button>
                  </div>

                  <dl className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                    <div>
                      <dt className="text-xs font-medium uppercase tracking-wide text-slate-400">
                        Key
                      </dt>
                      <dd className="mt-1 font-mono text-sm text-slate-700">{experiment.key}</dd>
                    </div>
                    <div>
                      <dt className="text-xs font-medium uppercase tracking-wide text-slate-400">
                        ID
                      </dt>
                      <dd className="mt-1 font-mono text-sm text-slate-700">{experiment.id}</dd>
                    </div>
                    <div>
                      <dt className="text-xs font-medium uppercase tracking-wide text-slate-400">
                        Status
                      </dt>
                      <dd className="mt-1">
                        <StatusBadge status={experiment.status} />
                      </dd>
                    </div>
                    <div>
                      <dt className="text-xs font-medium uppercase tracking-wide text-slate-400">
                        Start date
                      </dt>
                      <dd className="mt-1 text-sm text-slate-700">
                        {experiment.start_date
                          ? new Date(experiment.start_date).toLocaleString()
                          : "—"}
                      </dd>
                    </div>
                    <div>
                      <dt className="text-xs font-medium uppercase tracking-wide text-slate-400">
                        End date
                      </dt>
                      <dd className="mt-1 text-sm text-slate-700">
                        {experiment.end_date
                          ? new Date(experiment.end_date).toLocaleString()
                          : "—"}
                      </dd>
                    </div>
                    <div>
                      <dt className="text-xs font-medium uppercase tracking-wide text-slate-400">
                        Created
                      </dt>
                      <dd className="mt-1 text-sm text-slate-700">
                        {new Date(experiment.created_at).toLocaleString()}
                      </dd>
                    </div>
                    <div>
                      <dt className="text-xs font-medium uppercase tracking-wide text-slate-400">
                        Last updated
                      </dt>
                      <dd className="mt-1 text-sm text-slate-700">
                        {new Date(experiment.updated_at).toLocaleString()}
                      </dd>
                    </div>
                  </dl>

                  <div className="mt-6 flex flex-wrap gap-3">
                    <Link
                      to={`/applications/${appId}/experiments/${id}/assignments`}
                      className="inline-flex h-8 items-center justify-center rounded-lg border border-slate-300 bg-white px-3 text-sm font-medium text-slate-900 transition-colors hover:bg-slate-50"
                    >
                      View assignments
                    </Link>
                    <Link
                      to={`/applications/${appId}/experiments/${id}/events`}
                      className="inline-flex h-8 items-center justify-center rounded-lg border border-slate-300 bg-white px-3 text-sm font-medium text-slate-900 transition-colors hover:bg-slate-50"
                    >
                      View events
                    </Link>
                    <Link
                      to={`/applications/${appId}/experiments/${id}/dashboard`}
                      className="inline-flex h-8 items-center justify-center rounded-lg border border-slate-300 bg-white px-3 text-sm font-medium text-slate-900 transition-colors hover:bg-slate-50"
                    >
                      Open dashboard
                    </Link>
                  </div>
                </>
              )}
            </div>

            <div className="rounded-xl border border-slate-200 bg-white p-8 shadow-sm">
              <div className="flex items-start justify-between gap-4">
                <div>
                  <h2 className="text-lg font-medium text-slate-900">Branches</h2>
                  <p className="mt-1 text-sm text-slate-500">
                    Manage the variants that participate in this experiment. Weights should add up
                    to 100%.
                  </p>
                  {experiment.branches.length > 0 && (
                    <p
                      className={`mt-2 text-sm ${currentBranchWeightError ? "text-red-600" : "text-slate-500"}`}
                    >
                      Current allocation: {formatBranchWeightValue(currentBranchWeightTotal)}%.
                    </p>
                  )}
                </div>
                {!editingBranches && (
                  <Button variant="outline" onClick={openBranchEditor}>
                    {experiment.branches.length === 0 ? "Add branches" : "Edit branches"}
                  </Button>
                )}
              </div>

              {editingBranches && (
                <form
                  onSubmit={handleSaveBranches}
                  className="mt-6 space-y-5 rounded-lg border border-slate-200 bg-slate-50 p-4"
                >
                  <div className="flex items-center justify-between">
                    <h3 className="text-sm font-medium text-slate-900">
                      Bulk edit branches
                    </h3>
                    <div className="flex gap-2">
                      <Button type="button" variant="outline" size="sm" onClick={addBranchDraft} disabled={branchLoading}>
                        Add branch
                      </Button>
                      <Button type="button" variant="outline" size="sm" onClick={cancelBranchEditor} disabled={branchLoading}>
                        Cancel
                      </Button>
                    </div>
                  </div>

                  <p className={`text-sm ${draftBranchWeightError ? "text-red-600" : "text-slate-500"}`}>
                    Draft total: {formatBranchWeightValue(draftBranchWeightTotal)}%.
                  </p>

                  {branchDrafts.length === 0 && (
                    <p className="text-sm text-slate-500">No branches added yet.</p>
                  )}

                  {branchDrafts.map((branch, index) => (
                    <div
                      key={branch.id ?? `new-${index}`}
                      className="space-y-3 rounded-lg border border-slate-200 bg-white p-4"
                    >
                      <div className="grid gap-3 sm:grid-cols-[1.2fr_1.2fr_0.8fr_auto]">
                        <div className="space-y-1">
                          <label className="text-xs font-medium uppercase tracking-wide text-slate-500">
                            Name
                          </label>
                          <input
                            autoFocus={index === 0}
                            type="text"
                            value={branch.name}
                            onChange={(e) => handleBranchNameChange(index, e.target.value)}
                            placeholder="Control"
                            maxLength={BRANCH_NAME_MAX_LENGTH}
                            className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 placeholder:text-slate-400 focus:border-slate-500 focus:outline-none focus:ring-2 focus:ring-slate-200"
                            disabled={branchLoading}
                          />
                        </div>

                        <div className="space-y-1">
                          <label className="text-xs font-medium uppercase tracking-wide text-slate-500">
                            Key
                          </label>
                          <input
                            type="text"
                            value={branch.key}
                            onChange={(e) => handleBranchKeyChange(index, e.target.value)}
                            placeholder="control"
                            maxLength={BRANCH_KEY_MAX_LENGTH}
                            className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-sm text-slate-900 placeholder:text-slate-400 focus:border-slate-500 focus:outline-none focus:ring-2 focus:ring-slate-200 disabled:bg-slate-100 disabled:text-slate-500"
                            disabled={branchLoading || Boolean(branch.id)}
                          />
                        </div>

                        <div className="space-y-1">
                          <label className="text-xs font-medium uppercase tracking-wide text-slate-500">
                            Weight
                          </label>
                          <input
                            type="number"
                            min="0"
                            max="100"
                            step="0.01"
                            value={branch.weight}
                            onChange={(e) =>
                              setBranchDrafts((current) =>
                                current.map((currentBranch, branchIndex) =>
                                  branchIndex === index
                                    ? { ...currentBranch, weight: e.target.value }
                                    : currentBranch,
                                ),
                              )
                            }
                            className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-slate-500 focus:outline-none focus:ring-2 focus:ring-slate-200"
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

                      <div className="space-y-1">
                        <label className="text-xs font-medium uppercase tracking-wide text-slate-500">
                          Metadata JSON
                        </label>
                        <textarea
                          value={branch.metadataText}
                          onChange={(e) =>
                            setBranchDrafts((current) =>
                              current.map((currentBranch, branchIndex) =>
                                branchIndex === index
                                  ? { ...currentBranch, metadataText: e.target.value }
                                  : currentBranch,
                              ),
                            )
                          }
                          rows={4}
                          placeholder='{"color":"#22c55e"}'
                          className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-sm text-slate-900 placeholder:text-slate-400 focus:border-slate-500 focus:outline-none focus:ring-2 focus:ring-slate-200"
                          disabled={branchLoading}
                        />
                        <p className="text-xs text-slate-500">
                          Optional JSON object for branch-specific configuration.
                        </p>
                      </div>
                    </div>
                  ))}

                  {branchError && <p className="text-sm text-red-600">{branchError}</p>}

                  <div className="flex gap-3">
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
                  </div>
                </form>
              )}

              {!editingBranches && (
                <div className="mt-6 space-y-3">
                  {experiment.branches.length === 0 && (
                    <div className="rounded-lg border border-dashed border-slate-200 bg-slate-50 px-4 py-10 text-center text-sm text-slate-500">
                      No branches yet.
                    </div>
                  )}

                  {experiment.branches.map((branch) => (
                    <div
                      key={branch.id}
                      className="rounded-lg border border-slate-200 bg-slate-50 p-4"
                    >
                      <div className="flex items-start justify-between gap-4">
                        <div className="min-w-0">
                          <div className="flex flex-wrap items-center gap-3">
                            <h3 className="font-medium text-slate-900">{branch.name}</h3>
                            <span className="rounded-full bg-white px-2 py-0.5 text-xs text-slate-600">
                              {formatBranchWeightPercent(branch.weight, branchWeightScale)}
                            </span>
                          </div>
                          <p className="mt-1 font-mono text-xs text-slate-500">{branch.key}</p>
                        </div>
                      </div>

                      <div className="mt-4">
                        <p className="mb-2 text-xs font-medium uppercase tracking-wide text-slate-500">
                          Metadata
                        </p>
                        <pre className="overflow-x-auto rounded-lg bg-white p-3 text-xs text-slate-700">
{formatMetadata(branch.metadata_json)}
                        </pre>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>

            <div className="rounded-xl border border-red-200 bg-white p-8 shadow-sm">
              <div className="flex items-start justify-between gap-4">
                <div>
                  <h2 className="text-base font-medium text-slate-900">Delete Experiment</h2>
                  <p className="mt-1 text-sm text-slate-500">
                    This soft-deletes the experiment and cascades to its branches.
                  </p>
                </div>
                <Button variant="destructive" onClick={handleDelete} disabled={deleteLoading}>
                  {deleteLoading ? "Deleting…" : "Delete"}
                </Button>
              </div>
              {deleteError && (
                <p className="mt-3 text-sm text-red-600">{deleteError}</p>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
