/** Last six characters of a UUID string (with or without hyphens). */
export function formatShortId(id: string): string {
  const normalized = id.replace(/-/g, '').toLowerCase()
  if (normalized.length <= 6) {
    return normalized
  }
  return normalized.slice(-6)
}
