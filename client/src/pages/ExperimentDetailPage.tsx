import { useEffect, useState } from "react"
import { Link, useNavigate, useParams } from "react-router-dom"
import {
  experimentsApi,
  EXPERIMENT_STATUSES,
  type Experiment,
  type ExperimentStatus,
  type UpdateExperimentInput,
} from "@/api/experiments"
import { branchesApi, type Branch } from "@/api/branches"
import { ApiError } from "@/api/client"
import { Button } from "@/components/ui/button"
import { StatusBadge } from "@/components/StatusBadge"
import { slugifyKey } from "@/lib/slugify"

interface BranchFormState {
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

function createEmptyBranchForm(): BranchFormState {
  return {
    key: "",
    name: "",
    weight: "0.5",
    metadataText: "",
    isKeyCustom: false,
  }
}

function formatBranchCount(count: number) {
  return `${count} branch${count === 1 ? "" : "es"}`
}

function formatMetadata(metadata: unknown | null) {
  if (metadata == null) return "No metadata"
  return JSON.stringify(metadata, null, 2)
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

  const [branchMode, setBranchMode] = useState<"create" | "edit" | null>(null)
  const [editingBranchId, setEditingBranchId] = useState<string | null>(null)
  const [branchForm, setBranchForm] = useState<BranchFormState>(createEmptyBranchForm())
  const [branchLoading, setBranchLoading] = useState(false)
  const [branchError, setBranchError] = useState<string | null>(null)
  const [deletingBranchId, setDeletingBranchId] = useState<string | null>(null)

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

  function openCreateBranchForm() {
    setBranchMode("create")
    setEditingBranchId(null)
    setBranchForm(createEmptyBranchForm())
    setBranchError(null)
  }

  function openEditBranchForm(branch: Branch) {
    setBranchMode("edit")
    setEditingBranchId(branch.id)
    setBranchForm({
      key: branch.key,
      name: branch.name,
      weight: String(branch.weight),
      metadataText: branch.metadata_json == null ? "" : JSON.stringify(branch.metadata_json, null, 2),
      isKeyCustom: true,
    })
    setBranchError(null)
  }

  function closeBranchForm() {
    setBranchMode(null)
    setEditingBranchId(null)
    setBranchForm(createEmptyBranchForm())
    setBranchError(null)
  }

  async function handleSave(e: React.FormEvent) {
    e.preventDefault()
    if (!appId || !id || !experiment) return
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

  async function handleBranchSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!appId || !id) return

    const weight = Number(branchForm.weight)
    if (!branchForm.name.trim()) {
      setBranchError("Branch name is required.")
      return
    }
    if (branchMode === "create" && !branchForm.key.trim()) {
      setBranchError("Branch key is required.")
      return
    }
    if (Number.isNaN(weight) || weight < 0 || weight > 1) {
      setBranchError("Branch weight must be a number between 0 and 1.")
      return
    }

    let metadataJson: unknown | null = null
    if (branchForm.metadataText.trim()) {
      try {
        metadataJson = JSON.parse(branchForm.metadataText)
      } catch {
        setBranchError("Metadata must be valid JSON.")
        return
      }

      if (metadataJson === null || Array.isArray(metadataJson) || typeof metadataJson !== "object") {
        setBranchError("Metadata must be a JSON object.")
        return
      }
    }

    setBranchLoading(true)
    setBranchError(null)

    try {
      if (branchMode === "create") {
        const branch = await branchesApi.create(appId, id, {
          key: branchForm.key.trim(),
          name: branchForm.name.trim(),
          weight,
          metadata_json: metadataJson,
        })
        setExperiment((current) =>
          current
            ? {
                ...current,
                branches: [...current.branches, branch],
              }
            : current,
        )
      } else if (branchMode === "edit" && editingBranchId) {
        const updated = await branchesApi.update(appId, id, editingBranchId, {
          name: branchForm.name.trim(),
          weight,
          metadata_json: metadataJson,
        })
        setExperiment((current) =>
          current
            ? {
                ...current,
                branches: current.branches.map((branch) =>
                  branch.id === updated.id ? updated : branch,
                ),
              }
            : current,
        )
      }

      closeBranchForm()
    } catch (err) {
      setBranchError(err instanceof ApiError ? err.message : "Failed to save branch")
    } finally {
      setBranchLoading(false)
    }
  }

  async function handleDeleteBranch(branchId: string) {
    if (!appId || !id) return
    if (!window.confirm("Delete this branch?")) return

    setDeletingBranchId(branchId)
    try {
      await branchesApi.delete(appId, id, branchId)
      setExperiment((current) =>
        current
          ? {
              ...current,
              branches: current.branches.filter((branch) => branch.id !== branchId),
            }
          : current,
      )
      if (editingBranchId === branchId) {
        closeBranchForm()
      }
    } catch (err) {
      setBranchError(err instanceof ApiError ? err.message : "Failed to delete branch")
    } finally {
      setDeletingBranchId(null)
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
                    <Button type="submit" disabled={editLoading || !form.name.trim()}>
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
                </>
              )}
            </div>

            <div className="rounded-xl border border-slate-200 bg-white p-8 shadow-sm">
              <div className="flex items-start justify-between gap-4">
                <div>
                  <h2 className="text-lg font-medium text-slate-900">Branches</h2>
                  <p className="mt-1 text-sm text-slate-500">
                    Manage the variants that participate in this experiment.
                  </p>
                </div>
                {branchMode !== "create" && (
                  <Button variant="outline" onClick={openCreateBranchForm}>
                    Add branch
                  </Button>
                )}
              </div>

              {branchMode && (
                <form
                  onSubmit={handleBranchSubmit}
                  className="mt-6 space-y-4 rounded-lg border border-slate-200 bg-slate-50 p-4"
                >
                  <div className="flex items-center justify-between">
                    <h3 className="text-sm font-medium text-slate-900">
                      {branchMode === "create" ? "New branch" : "Edit branch"}
                    </h3>
                    <Button type="button" variant="outline" size="sm" onClick={closeBranchForm} disabled={branchLoading}>
                      Cancel
                    </Button>
                  </div>

                  <div className="grid gap-4 sm:grid-cols-2">
                    <div className="space-y-1">
                      <label className="text-xs font-medium uppercase tracking-wide text-slate-500">
                        Name <span className="text-red-500">*</span>
                      </label>
                      <input
                        autoFocus
                        type="text"
                        value={branchForm.name}
                        onChange={(e) => {
                          const name = e.target.value
                          const generatedKey = slugifyKey(name)
                          setBranchForm((current) => ({
                            ...current,
                            name,
                            key:
                              branchMode === "edit" || current.isKeyCustom
                                ? current.key
                                : generatedKey,
                          }))
                        }}
                        className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 placeholder:text-slate-400 focus:border-slate-500 focus:outline-none focus:ring-2 focus:ring-slate-200"
                        disabled={branchLoading}
                      />
                    </div>

                    <div className="space-y-1">
                      <label className="text-xs font-medium uppercase tracking-wide text-slate-500">
                        Key {branchMode === "create" && <span className="text-red-500">*</span>}
                      </label>
                      <input
                        type="text"
                        value={branchForm.key}
                        onChange={(e) =>
                          setBranchForm((current) => ({
                            ...current,
                            key: e.target.value,
                            isKeyCustom: e.target.value !== slugifyKey(current.name),
                          }))
                        }
                        className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-sm text-slate-900 placeholder:text-slate-400 focus:border-slate-500 focus:outline-none focus:ring-2 focus:ring-slate-200 disabled:bg-slate-100 disabled:text-slate-500"
                        disabled={branchLoading || branchMode === "edit"}
                      />
                    </div>
                  </div>

                  <div className="grid gap-4 sm:grid-cols-2">
                    <div className="space-y-1">
                      <label className="text-xs font-medium uppercase tracking-wide text-slate-500">
                        Weight <span className="text-red-500">*</span>
                      </label>
                      <input
                        type="number"
                        min="0"
                        max="1"
                        step="0.01"
                        value={branchForm.weight}
                        onChange={(e) =>
                          setBranchForm((current) => ({ ...current, weight: e.target.value }))
                        }
                        className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-slate-500 focus:outline-none focus:ring-2 focus:ring-slate-200"
                        disabled={branchLoading}
                      />
                    </div>
                  </div>

                  <div className="space-y-1">
                    <label className="text-xs font-medium uppercase tracking-wide text-slate-500">
                      Metadata JSON
                    </label>
                    <textarea
                      value={branchForm.metadataText}
                      onChange={(e) =>
                        setBranchForm((current) => ({ ...current, metadataText: e.target.value }))
                      }
                      rows={5}
                      placeholder='{"color":"#22c55e"}'
                      className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-sm text-slate-900 placeholder:text-slate-400 focus:border-slate-500 focus:outline-none focus:ring-2 focus:ring-slate-200"
                      disabled={branchLoading}
                    />
                    <p className="text-xs text-slate-500">
                      Optional JSON object for branch-specific configuration.
                    </p>
                  </div>

                  {branchError && <p className="text-sm text-red-600">{branchError}</p>}

                  <div className="flex gap-3">
                    <Button
                      type="submit"
                      disabled={
                        branchLoading ||
                        !branchForm.name.trim() ||
                        (branchMode === "create" && !branchForm.key.trim())
                      }
                    >
                      {branchLoading ? "Saving…" : branchMode === "create" ? "Create branch" : "Save branch"}
                    </Button>
                  </div>
                </form>
              )}

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
                            weight {branch.weight}
                          </span>
                        </div>
                        <p className="mt-1 font-mono text-xs text-slate-500">{branch.key}</p>
                      </div>

                      <div className="flex gap-2">
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => openEditBranchForm(branch)}
                          disabled={branchLoading || deletingBranchId === branch.id}
                        >
                          Edit
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleDeleteBranch(branch.id)}
                          disabled={branchLoading || deletingBranchId === branch.id}
                        >
                          {deletingBranchId === branch.id ? "Deleting…" : "Delete"}
                        </Button>
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
