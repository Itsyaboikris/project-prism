import { useEffect, useRef, useState } from "react"
import { Link, useParams } from "react-router-dom"
import { applicationsApi, type Application } from "@/api/applications"
import { experimentsApi, type Experiment } from "@/api/experiments"
import { ApiError } from "@/api/client"
import { Button } from "@/components/ui/button"
import { slugifyKey } from "@/lib/slugify"
import { StatusBadge } from "../components/StatusBadge"

interface BranchDraft {
  key: string
  name: string
  weight: string
  isKeyCustom: boolean
}

interface CreateExperimentForm {
  key: string
  name: string
  description: string | null
  start_date: string | null
  end_date: string | null
  branches: BranchDraft[]
}

function createEmptyBranchDraft(): BranchDraft {
  return {
    key: "",
    name: "",
    weight: "0.5",
    isKeyCustom: false,
  }
}

function formatBranchCount(count: number) {
  return `${count} branch${count === 1 ? "" : "es"}`
}

export default function ExperimentsPage() {
  const { appId } = useParams<{ appId: string }>()

  const [app, setApp] = useState<Application | null>(null)
  const [experiments, setExperiments] = useState<Experiment[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const [creating, setCreating] = useState(false)
  const [form, setForm] = useState<CreateExperimentForm>({
    key: "",
    name: "",
    description: null,
    start_date: null,
    end_date: null,
    branches: [],
  })
  const [isKeyCustom, setIsKeyCustom] = useState(false)
  const [createError, setCreateError] = useState<string | null>(null)
  const [createLoading, setCreateLoading] = useState(false)
  const firstInputRef = useRef<HTMLInputElement>(null)

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
    setCreating(true)
    setForm({
      key: "",
      name: "",
      description: null,
      start_date: null,
      end_date: null,
      branches: [],
    })
    setIsKeyCustom(false)
    setCreateError(null)
    setTimeout(() => firstInputRef.current?.focus(), 0)
  }

  function handleBranchNameChange(index: number, name: string) {
    setForm((current) => ({
      ...current,
      branches: current.branches.map((branch, branchIndex) => {
        if (branchIndex !== index) return branch
        const generatedKey = slugifyKey(name)
        return {
          ...branch,
          name,
          key: branch.isKeyCustom ? branch.key : generatedKey,
        }
      }),
    }))
  }

  function handleBranchKeyChange(index: number, key: string) {
    setForm((current) => ({
      ...current,
      branches: current.branches.map((branch, branchIndex) =>
        branchIndex === index
          ? {
              ...branch,
              key,
              isKeyCustom: key !== slugifyKey(branch.name),
            }
          : branch,
      ),
    }))
  }

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    if (!appId) return

    const populatedBranches = form.branches.filter(
      (branch) => branch.name.trim() || branch.key.trim() || branch.weight.trim(),
    )

    for (const branch of populatedBranches) {
      if (!branch.name.trim() || !branch.key.trim()) {
        setCreateError("Each branch needs both a name and a key.")
        return
      }

      const weight = Number(branch.weight)
      if (Number.isNaN(weight) || weight < 0 || weight > 1) {
        setCreateError("Each branch weight must be a number between 0 and 1.")
        return
      }
    }

    setCreateLoading(true)
    setCreateError(null)
    try {
      const exp = await experimentsApi.create(appId, {
        key: form.key.trim(),
        name: form.name.trim(),
        description: form.description?.trim() || null,
        start_date: form.start_date,
        end_date: form.end_date,
        branches: populatedBranches.map((branch) => ({
          key: branch.key.trim(),
          name: branch.name.trim(),
          weight: Number(branch.weight),
        })),
      })
      setExperiments((prev) => [exp, ...prev])
      setCreating(false)
    } catch (err) {
      setCreateError(
        err instanceof ApiError ? err.message : "Failed to create experiment",
      )
    } finally {
      setCreateLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-slate-50">
      <div className="mx-auto max-w-4xl px-6 py-12">
        <Link
          to={`/applications/${appId}`}
          className="inline-flex items-center gap-1.5 text-sm text-slate-500 hover:text-slate-900"
        >
          ← {app ? app.name : "Application"}
        </Link>

        <div className="mt-6 flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-semibold tracking-tight text-slate-900">
              Experiments
            </h1>
            {app && <p className="mt-1 text-sm text-slate-500">{app.name}</p>}
          </div>
          {!creating && !loading && !error && (
            <Button onClick={openCreateForm}>New experiment</Button>
          )}
        </div>

        {creating && (
          <form
            onSubmit={handleCreate}
            className="mt-6 space-y-5 rounded-xl border border-slate-200 bg-white p-6 shadow-sm"
          >
            <h2 className="text-base font-medium text-slate-900">New experiment</h2>

            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-1">
                <label className="text-xs font-medium uppercase tracking-wide text-slate-500">
                  Name <span className="text-red-500">*</span>
                </label>
                <input
                  ref={firstInputRef}
                  type="text"
                  value={form.name}
                  onChange={(e) => {
                    const name = e.target.value
                    const generatedKey = slugifyKey(name)
                    setForm((current) => ({
                      ...current,
                      name,
                      key: isKeyCustom ? current.key : generatedKey,
                    }))
                  }}
                  placeholder="Checkout Button Color"
                  className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 placeholder:text-slate-400 focus:border-slate-500 focus:outline-none focus:ring-2 focus:ring-slate-200"
                  disabled={createLoading}
                />
              </div>

              <div className="space-y-1">
                <label className="text-xs font-medium uppercase tracking-wide text-slate-500">
                  Key <span className="text-red-500">*</span>
                </label>
                <input
                  type="text"
                  value={form.key}
                  onChange={(e) => {
                    const key = e.target.value
                    setForm((current) => ({ ...current, key }))
                    setIsKeyCustom(key !== slugifyKey(form.name))
                  }}
                  placeholder="checkout-button-color"
                  className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-sm text-slate-900 placeholder:text-slate-400 focus:border-slate-500 focus:outline-none focus:ring-2 focus:ring-slate-200"
                  disabled={createLoading}
                />
              </div>
            </div>

            <div className="space-y-1">
              <label className="text-xs font-medium uppercase tracking-wide text-slate-500">
                Description
              </label>
              <textarea
                value={form.description ?? ""}
                onChange={(e) =>
                  setForm((current) => ({ ...current, description: e.target.value }))
                }
                placeholder="Optional description"
                rows={2}
                className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 placeholder:text-slate-400 focus:border-slate-500 focus:outline-none focus:ring-2 focus:ring-slate-200"
                disabled={createLoading}
              />
            </div>

            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-1">
                <label className="text-xs font-medium uppercase tracking-wide text-slate-500">
                  Start date
                </label>
                <input
                  type="datetime-local"
                  value={form.start_date ? form.start_date.slice(0, 16) : ""}
                  onChange={(e) =>
                    setForm((current) => ({
                      ...current,
                      start_date: e.target.value ? new Date(e.target.value).toISOString() : null,
                    }))
                  }
                  className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-slate-500 focus:outline-none focus:ring-2 focus:ring-slate-200"
                  disabled={createLoading}
                />
              </div>

              <div className="space-y-1">
                <label className="text-xs font-medium uppercase tracking-wide text-slate-500">
                  End date
                </label>
                <input
                  type="datetime-local"
                  value={form.end_date ? form.end_date.slice(0, 16) : ""}
                  onChange={(e) =>
                    setForm((current) => ({
                      ...current,
                      end_date: e.target.value ? new Date(e.target.value).toISOString() : null,
                    }))
                  }
                  className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-slate-500 focus:outline-none focus:ring-2 focus:ring-slate-200"
                  disabled={createLoading}
                />
              </div>
            </div>

            <div className="space-y-3 rounded-lg border border-slate-200 bg-slate-50 p-4">
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="text-sm font-medium text-slate-900">Initial branches</h3>
                  <p className="text-xs text-slate-500">
                    Optional variants to create with the experiment.
                  </p>
                </div>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() =>
                    setForm((current) => ({
                      ...current,
                      branches: [...current.branches, createEmptyBranchDraft()],
                    }))
                  }
                  disabled={createLoading}
                >
                  Add branch
                </Button>
              </div>

              {form.branches.length === 0 && (
                <p className="text-sm text-slate-500">No branches added yet.</p>
              )}

              {form.branches.map((branch, index) => (
                <div
                  key={index}
                  className="space-y-3 rounded-lg border border-slate-200 bg-white p-4"
                >
                  <div className="grid gap-3 sm:grid-cols-[1.2fr_1.2fr_0.8fr_auto]">
                    <div className="space-y-1">
                      <label className="text-xs font-medium uppercase tracking-wide text-slate-500">
                        Name
                      </label>
                      <input
                        type="text"
                        value={branch.name}
                        onChange={(e) => handleBranchNameChange(index, e.target.value)}
                        placeholder="Control"
                        className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 placeholder:text-slate-400 focus:border-slate-500 focus:outline-none focus:ring-2 focus:ring-slate-200"
                        disabled={createLoading}
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
                        className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 font-mono text-sm text-slate-900 placeholder:text-slate-400 focus:border-slate-500 focus:outline-none focus:ring-2 focus:ring-slate-200"
                        disabled={createLoading}
                      />
                    </div>

                    <div className="space-y-1">
                      <label className="text-xs font-medium uppercase tracking-wide text-slate-500">
                        Weight
                      </label>
                      <input
                        type="number"
                        min="0"
                        max="1"
                        step="0.01"
                        value={branch.weight}
                        onChange={(e) =>
                          setForm((current) => ({
                            ...current,
                            branches: current.branches.map((currentBranch, branchIndex) =>
                              branchIndex === index
                                ? { ...currentBranch, weight: e.target.value }
                                : currentBranch,
                            ),
                          }))
                        }
                        className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:border-slate-500 focus:outline-none focus:ring-2 focus:ring-slate-200"
                        disabled={createLoading}
                      />
                    </div>

                    <div className="flex items-end">
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={() =>
                          setForm((current) => ({
                            ...current,
                            branches: current.branches.filter((_, branchIndex) => branchIndex !== index),
                          }))
                        }
                        disabled={createLoading}
                      >
                        Remove
                      </Button>
                    </div>
                  </div>
                </div>
              ))}
            </div>

            {createError && <p className="text-sm text-red-600">{createError}</p>}

            <div className="flex gap-3">
              <Button
                type="submit"
                disabled={createLoading || !form.key.trim() || !form.name.trim()}
              >
                {createLoading ? "Creating…" : "Create"}
              </Button>
              <Button
                type="button"
                variant="outline"
                onClick={() => setCreating(false)}
                disabled={createLoading}
              >
                Cancel
              </Button>
            </div>
          </form>
        )}

        <div className="mt-6">
          {loading && (
            <div className="flex items-center justify-center py-20 text-sm text-slate-400">
              Loading…
            </div>
          )}

          {!loading && error && (
            <div className="rounded-xl border border-red-200 bg-red-50 p-6 text-sm text-red-700">
              {error}
            </div>
          )}

          {!loading && !error && experiments.length === 0 && (
            <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-slate-200 bg-white py-20 text-center">
              <p className="text-sm text-slate-500">No experiments yet.</p>
              <button
                onClick={openCreateForm}
                className="mt-2 text-sm font-medium text-slate-900 underline-offset-4 hover:underline"
              >
                Create your first experiment
              </button>
            </div>
          )}

          {!loading && !error && experiments.length > 0 && (
            <ul className="space-y-3">
              {experiments.map((exp) => (
                <li key={exp.id}>
                  <Link
                    to={`/applications/${appId}/experiments/${exp.id}`}
                    className="flex items-center justify-between rounded-xl border border-slate-200 bg-white px-6 py-4 shadow-sm transition-colors hover:border-slate-300 hover:bg-slate-50"
                  >
                    <div className="min-w-0">
                      <div className="flex items-center gap-3">
                        <p className="font-medium text-slate-900">{exp.name}</p>
                        <StatusBadge status={exp.status} />
                        <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs text-slate-600">
                          {formatBranchCount(exp.branches.length)}
                        </span>
                      </div>
                      <p className="mt-0.5 font-mono text-xs text-slate-400">{exp.key}</p>
                      {exp.description && (
                        <p className="mt-1 truncate text-sm text-slate-500">{exp.description}</p>
                      )}
                    </div>
                    <span className="ml-4 shrink-0 text-xs text-slate-400">
                      {new Date(exp.created_at).toLocaleDateString()}
                    </span>
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </div>
  )
}
