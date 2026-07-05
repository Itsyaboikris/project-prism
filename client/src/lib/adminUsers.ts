export const MIN_PASSWORD_LENGTH = 12

export function validateAdminEmail(email: string) {
  if (!email.trim()) return "Email is required."

  const normalized = email.trim()
  const simpleEmailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
  if (!simpleEmailPattern.test(normalized)) {
    return "Enter a valid email address."
  }

  return null
}

export function validateAdminPassword(password: string) {
  if (!password) return "Password is required."
  if (password.length < MIN_PASSWORD_LENGTH) {
    return `Password must be at least ${MIN_PASSWORD_LENGTH} characters.`
  }

  return null
}
