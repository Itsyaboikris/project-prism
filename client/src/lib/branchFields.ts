export const BRANCH_NAME_MAX_LENGTH = 64
export const BRANCH_KEY_MAX_LENGTH = 64
export const BRANCH_METADATA_MAX_BYTES = 4096

const metadataEncoder = new TextEncoder()

export function validateBranchName(name: string): string | null {
  const trimmed = name.trim()
  if (!trimmed) return "Branch name is required."
  if (trimmed.length > BRANCH_NAME_MAX_LENGTH) {
    return `Branch name must be ${BRANCH_NAME_MAX_LENGTH} characters or fewer.`
  }
  return null
}

export function validateBranchKey(key: string): string | null {
  const trimmed = key.trim()
  if (!trimmed) return "Branch key is required."
  if (trimmed.length > BRANCH_KEY_MAX_LENGTH) {
    return `Branch key must be ${BRANCH_KEY_MAX_LENGTH} characters or fewer.`
  }
  return null
}

export function validateBranchMetadataValue(metadata: unknown | null | undefined): string | null {
  if (metadata == null) return null
  if (Array.isArray(metadata) || typeof metadata !== "object") {
    return "Branch metadata must be a JSON object."
  }

  const serialized = JSON.stringify(metadata)
  if (metadataEncoder.encode(serialized).length > BRANCH_METADATA_MAX_BYTES) {
    return `Branch metadata must be ${BRANCH_METADATA_MAX_BYTES} bytes or fewer.`
  }

  return null
}

export function parseBranchMetadataText(metadataText: string): {
  value: Record<string, unknown> | null
  error: string | null
} {
  if (!metadataText.trim()) {
    return { value: null, error: null }
  }

  try {
    const parsed = JSON.parse(metadataText) as unknown
    const error = validateBranchMetadataValue(parsed)
    if (error) {
      return { value: null, error }
    }

    return {
      value: parsed as Record<string, unknown>,
      error: null,
    }
  } catch {
    return { value: null, error: "Branch metadata must be valid JSON." }
  }
}
