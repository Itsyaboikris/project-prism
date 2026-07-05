import { useEffect, useRef, useState } from "react"
import { Link, useNavigate, useParams } from "react-router-dom"
import { toast } from "sonner"
import { applicationsApi, type Application } from "@/api/applications"
import { experimentsApi } from "@/api/experiments"
import { ApiError } from "@/api/client"
import { ErrorState } from "@/components/ErrorState"
import { PageHeader } from "@/components/PageHeader"
import { PageLoading } from "@/components/PageLoading"
import { StatusBadge } from "@/components/StatusBadge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import {
  BRANCH_WEIGHT_PERCENT_TOTAL,
  formatBranchWeightValue,
  sumWeights,
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
  EXPERIMENT_KEY_MAX_LENGTH,
  EXPERIMENT_NAME_MAX_LENGTH,
  validateExperimentDescription,
  validateExperimentKey,
  validateExperimentName,
} from "@/lib/experimentFields"
import { slugifyKey } from "@/lib/slugify"

interface BranchDraft {
  key: string
  name: string
  weight: string
  metadataText: string
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

function createEmptyBranchDraft(weight = "0"): BranchDraft {
  return {
    key: "",
    name: "",
    weight,
    metadataText: "",
    isKeyCustom: false,
  }
}

function getRemainingWeight(weights: number[]): string {
  return formatBranchWeightValue(Math.max(0, BRANCH_WEIGHT_PERCENT_TOTAL - sumWeights(weights)))
}

function validateBranchDraft(branch: BranchDraft): string | null {
  return (
    validateBranchName(branch.name) ??
    validateBranchKey(branch.key) ??
    parseBranchMetadataText(branch.metadataText).error
  )
}

export default function CreateExperimentPage() {
  const { appId } = useParams<{ appId: string }>()
  const navigate = useNavigate()

  const [app, setApp] = useState<Application | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

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

  const populatedBranches = form.branches.filter(
    (branch) =>
      branch.name.trim() ||
      branch.key.trim() ||
      branch.weight.trim() ||
      branch.metadataText.trim(),
  )
  const populatedBranchWeights = populatedBranches.map((branch) => Number(branch.weight))
  const populatedBranchWeightTotal = sumWeights(
    populatedBranchWeights.filter((weight) => !Number.isNaN(weight)),
  )
  const populatedBranchWeightError = validateDisplayBranchWeights(populatedBranchWeights)
  const populatedBranchFieldError =
    populatedBranches.map((branch) => validateBranchDraft(branch)).find(Boolean) ?? null

  useEffect(() => {
    if (!appId) return
    applicationsApi
      .get(appId)
      .then(setApp)
      .catch((err) =>
        setError(err instanceof ApiError ? err.message : "Failed to load application"),
      )
      .finally(() => setLoading(false))
  }, [appId])

  useEffect(() => {
    if (!loading && app?.status === "inactive") {
      navigate(`/applications/${appId}/experiments`, { replace: true })
    }
  }, [loading, app, appId, navigate])

  useEffect(() => {
    if (!loading && !error && app?.status !== "inactive") {
      firstInputRef.current?.focus()
    }
  }, [loading, error, app?.status])

  function handleNameChange(name: string) {
    const generatedKey = slugifyKey(name)
    setForm((current) => ({
      ...current,
      name,
      key: isKeyCustom ? current.key : generatedKey,
    }))
  }

  function handleKeyChange(key: string) {
    setForm((current) => {
      setIsKeyCustom(key !== slugifyKey(current.name))
      return { ...current, key }
    })
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
    if (app?.status === "inactive") {
      setCreateError("Inactive applications cannot create new experiments.")
      return
    }

    const nameError = validateExperimentName(form.name)
    if (nameError) {
      setCreateError(nameError)
      return
    }
    const keyError = validateExperimentKey(form.key)
    if (keyError) {
      setCreateError(keyError)
      return
    }
    const descriptionError = validateExperimentDescription(form.description)
    if (descriptionError) {
      setCreateError(descriptionError)
      return
    }
    const dateError = validateExperimentDateRange(form.start_date, form.end_date)
    if (dateError) {
      setCreateError(dateError)
      return
    }

    const branchWeights: number[] = []
    for (const branch of populatedBranches) {
      const branchFieldError = validateBranchDraft(branch)
      if (branchFieldError) {
        setCreateError(branchFieldError)
        return
      }
      branchWeights.push(Number(branch.weight))
    }

    const branchWeightError = validateDisplayBranchWeights(branchWeights)
    if (branchWeightError) {
      setCreateError(branchWeightError)
      return
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
          metadata_json: parseBranchMetadataText(branch.metadataText).value,
        })),
      })
      toast.success("Experiment created")
      navigate(`/applications/${appId}/experiments/${exp.id}`)
    } catch (err) {
      const message = err instanceof ApiError ? err.message : "Failed to create experiment"
      setCreateError(message)
      toast.error(message)
    } finally {
      setCreateLoading(false)
    }
  }

  if (loading) return <PageLoading rows={6} />
  if (error) return <ErrorState message={error} />
  if (!app || !appId) return null

  return (
    <>
      <PageHeader
        title="New experiment"
        description={`Create an A/B test for ${app.name}.`}
        breadcrumbs={[
          { label: "Applications", href: "/applications" },
          { label: app.name, href: `/applications/${appId}` },
          { label: "Experiments", href: `/applications/${appId}/experiments` },
          { label: "New" },
        ]}
      />

      <Card>
        <CardHeader>
          <CardTitle>Experiment details</CardTitle>
          <CardDescription>
            The key is generated from the name. You can edit it before saving.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleCreate} className="space-y-5">
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="exp-name">Name</Label>
                <Input
                  ref={firstInputRef}
                  id="exp-name"
                  value={form.name}
                  onChange={(e) => handleNameChange(e.target.value)}
                  placeholder="Checkout Button Color"
                  maxLength={EXPERIMENT_NAME_MAX_LENGTH}
                  disabled={createLoading}
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="exp-key">Key</Label>
                <Input
                  id="exp-key"
                  value={form.key}
                  onChange={(e) => handleKeyChange(e.target.value)}
                  placeholder="checkout-button-color"
                  maxLength={EXPERIMENT_KEY_MAX_LENGTH}
                  className="font-mono"
                  disabled={createLoading}
                />
                {!isKeyCustom && form.name.trim() && (
                  <p className="text-xs text-muted-foreground">Auto-generated from name</p>
                )}
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="exp-desc">Description</Label>
              <Textarea
                id="exp-desc"
                value={form.description ?? ""}
                onChange={(e) =>
                  setForm((current) => ({ ...current, description: e.target.value }))
                }
                placeholder="Optional description"
                rows={2}
                maxLength={EXPERIMENT_DESCRIPTION_MAX_LENGTH}
                disabled={createLoading}
              />
            </div>

            <div className="rounded-lg border border-border bg-muted/40 px-4 py-3">
              <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                Status
              </p>
              <div className="mt-2 flex items-center gap-2">
                <StatusBadge status="draft" />
                <p className="text-sm text-muted-foreground">
                  New experiments are created as <span className="font-medium">draft</span>.
                </p>
              </div>
            </div>

            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="exp-start">Start date</Label>
                <Input
                  id="exp-start"
                  type="datetime-local"
                  value={form.start_date ? form.start_date.slice(0, 16) : ""}
                  onChange={(e) =>
                    setForm((current) => ({
                      ...current,
                      start_date: e.target.value ? new Date(e.target.value).toISOString() : null,
                    }))
                  }
                  disabled={createLoading}
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="exp-end">End date</Label>
                <Input
                  id="exp-end"
                  type="datetime-local"
                  value={form.end_date ? form.end_date.slice(0, 16) : ""}
                  onChange={(e) =>
                    setForm((current) => ({
                      ...current,
                      end_date: e.target.value ? new Date(e.target.value).toISOString() : null,
                    }))
                  }
                  disabled={createLoading}
                />
              </div>
            </div>

            <div className="space-y-3 rounded-lg border border-border bg-muted/40 p-4">
              <div className="flex items-center justify-between gap-4">
                <div>
                  <h3 className="text-sm font-medium text-foreground">Initial branches</h3>
                  <p className="text-xs text-muted-foreground">
                    Optional variants. Branch keys are generated from names. Weights should total
                    100%.
                  </p>
                  {form.branches.length > 0 && (
                    <p
                      className={`mt-1 text-xs ${populatedBranchWeightError ? "text-destructive" : "text-muted-foreground"}`}
                    >
                      Current total: {formatBranchWeightValue(populatedBranchWeightTotal)}%.
                    </p>
                  )}
                </div>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() =>
                    setForm((current) => ({
                      ...current,
                      branches: [
                        ...current.branches,
                        createEmptyBranchDraft(
                          getRemainingWeight(
                            current.branches
                              .map((branch) => Number(branch.weight))
                              .filter((weight) => !Number.isNaN(weight)),
                          ),
                        ),
                      ],
                    }))
                  }
                  disabled={createLoading}
                >
                  Add branch
                </Button>
              </div>

              {form.branches.length === 0 && (
                <p className="text-sm text-muted-foreground">No branches added yet.</p>
              )}

              {form.branches.map((branch, index) => (
                <div
                  key={index}
                  className="space-y-3 rounded-lg border border-border bg-card p-4"
                >
                  <div className="grid gap-3 sm:grid-cols-[1.2fr_1.2fr_0.8fr_auto]">
                    <div className="space-y-2">
                      <Label>Name</Label>
                      <Input
                        value={branch.name}
                        onChange={(e) => handleBranchNameChange(index, e.target.value)}
                        placeholder="Control"
                        maxLength={BRANCH_NAME_MAX_LENGTH}
                        disabled={createLoading}
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
                        disabled={createLoading}
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
                          setForm((current) => ({
                            ...current,
                            branches: current.branches.map((currentBranch, branchIndex) =>
                              branchIndex === index
                                ? { ...currentBranch, weight: e.target.value }
                                : currentBranch,
                            ),
                          }))
                        }
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

                  <div className="space-y-2">
                    <Label>Metadata JSON</Label>
                    <Textarea
                      value={branch.metadataText}
                      onChange={(e) =>
                        setForm((current) => ({
                          ...current,
                          branches: current.branches.map((currentBranch, branchIndex) =>
                            branchIndex === index
                              ? { ...currentBranch, metadataText: e.target.value }
                              : currentBranch,
                          ),
                        }))
                      }
                      rows={4}
                      placeholder='{"color":"#22c55e"}'
                      className="font-mono"
                      disabled={createLoading}
                    />
                  </div>
                </div>
              ))}
            </div>

            {createError && <p className="text-sm text-destructive">{createError}</p>}

            <div className="flex gap-3">
              <Button
                type="submit"
                disabled={
                  createLoading ||
                  Boolean(validateExperimentKey(form.key)) ||
                  Boolean(validateExperimentName(form.name)) ||
                  Boolean(validateExperimentDescription(form.description)) ||
                  Boolean(populatedBranchFieldError) ||
                  Boolean(populatedBranchWeightError)
                }
              >
                {createLoading ? "Creating…" : "Create experiment"}
              </Button>
              <Button
                type="button"
                variant="outline"
                disabled={createLoading}
                render={<Link to={`/applications/${appId}/experiments`} />}
              >
                Cancel
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </>
  )
}
