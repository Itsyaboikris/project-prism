export function validateExperimentDateRange(
  startDate: string | null | undefined,
  endDate: string | null | undefined,
): string | null {
  if (!startDate || !endDate) return null

  const startTime = new Date(startDate).getTime()
  const endTime = new Date(endDate).getTime()
  if (Number.isNaN(startTime) || Number.isNaN(endTime)) return null

  if (endTime < startTime) {
    return "End date must be on or after start date."
  }

  return null
}
