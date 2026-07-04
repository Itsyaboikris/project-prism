export const BRANCH_WEIGHT_PERCENT_TOTAL = 100

const BRANCH_WEIGHT_TOLERANCE = 0.0001

export type BranchWeightScale = "percent" | "fraction"

export function inferStoredBranchWeightScale(weights: number[]): BranchWeightScale {
  if (weights.length === 0) return "percent"
  return weights.every((weight) => weight <= 1 + BRANCH_WEIGHT_TOLERANCE) ? "fraction" : "percent"
}

export function toDisplayBranchWeight(weight: number, scale: BranchWeightScale): number {
  const value = scale === "fraction" ? weight * BRANCH_WEIGHT_PERCENT_TOTAL : weight
  return roundBranchWeight(value)
}

export function toStoredBranchWeight(weight: number, scale: BranchWeightScale): number {
  return scale === "fraction" ? weight / BRANCH_WEIGHT_PERCENT_TOTAL : weight
}

export function formatBranchWeightValue(weight: number): string {
  return roundBranchWeight(weight).toFixed(2).replace(/\.?0+$/, "")
}

export function formatBranchWeightPercent(weight: number, scale: BranchWeightScale): string {
  return `${formatBranchWeightValue(toDisplayBranchWeight(weight, scale))}%`
}

export function validateDisplayBranchWeights(weights: number[]): string | null {
  if (weights.length === 0) return null
  if (weights.some((weight) => Number.isNaN(weight) || weight < 0 || weight > BRANCH_WEIGHT_PERCENT_TOTAL)) {
    return "Branch weights must be numbers between 0 and 100."
  }

  const total = sumWeights(weights)
  if (!nearlyEqual(total, BRANCH_WEIGHT_PERCENT_TOTAL)) {
    return `Branch weights must add up to 100%. Current total: ${formatBranchWeightValue(total)}%.`
  }

  return null
}

export function sumWeights(weights: number[]): number {
  return roundBranchWeight(weights.reduce((total, weight) => total + weight, 0))
}

function roundBranchWeight(weight: number): number {
  return Math.round(weight * 100) / 100
}

function nearlyEqual(a: number, b: number): boolean {
  return Math.abs(a - b) <= BRANCH_WEIGHT_TOLERANCE
}
