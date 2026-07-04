export const EXPERIMENT_NAME_MAX_LENGTH = 64
export const EXPERIMENT_KEY_MAX_LENGTH = 64
export const EXPERIMENT_DESCRIPTION_MAX_LENGTH = 280

export function validateExperimentName(name: string): string | null {
  const trimmed = name.trim()
  if (!trimmed) return "Name is required."
  if (trimmed.length > EXPERIMENT_NAME_MAX_LENGTH) {
    return `Name must be ${EXPERIMENT_NAME_MAX_LENGTH} characters or fewer.`
  }
  return null
}

export function validateExperimentKey(key: string): string | null {
  const trimmed = key.trim()
  if (!trimmed) return "Key is required."
  if (trimmed.length > EXPERIMENT_KEY_MAX_LENGTH) {
    return `Key must be ${EXPERIMENT_KEY_MAX_LENGTH} characters or fewer.`
  }
  return null
}

export function validateExperimentDescription(description: string | null | undefined): string | null {
  if ((description ?? "").length > EXPERIMENT_DESCRIPTION_MAX_LENGTH) {
    return `Description must be ${EXPERIMENT_DESCRIPTION_MAX_LENGTH} characters or fewer.`
  }
  return null
}
