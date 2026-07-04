export const APPLICATION_NAME_MAX_LENGTH = 80

export function validateApplicationName(name: string): string | null {
  const trimmed = name.trim()
  if (!trimmed) {
    return "Name is required."
  }
  if (trimmed.length > APPLICATION_NAME_MAX_LENGTH) {
    return `Name must be ${APPLICATION_NAME_MAX_LENGTH} characters or fewer.`
  }

  return null
}
