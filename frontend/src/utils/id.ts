/** Last six characters of a UUID string (with or without hyphens). */
export function formatShortId(id: string): string {
  const normalized = id.replace(/-/g, '').toLowerCase()
  if (normalized.length <= 6) {
    return normalized
  }
  return normalized.slice(-6)
}

/** True when two API ids refer to the same row (short suffix or full UUID). */
export function publicIdsMatch(a: string | null | undefined, b: string | null | undefined): boolean {
  if (!a || !b) return false
  const left = a.trim().toLowerCase()
  const right = b.trim().toLowerCase()
  if (!left || !right) return false
  if (left === right) return true
  return formatShortId(left) === formatShortId(right)
}
