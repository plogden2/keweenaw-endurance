/**
 * All You Can East Bluffet 2026 event identity helper.
 *
 * Mirrors `backend/internal/eventpolicy.IsBluffetEventID`. The API exposes
 * event/race ids as a six-character `PublicUUID` suffix (see
 * `backend/internal/uuidutil`), so this must match the short form, not just
 * the full UUID.
 */

const BLUFFET_EVENT_ID_FULL = '1441674d-a011-471a-a601-722b88b117f5'
const BLUFFET_EVENT_ID_SHORT = 'b117f5'

/** Reports whether `id` refers to the All You Can East Bluffet 2026 event. */
export function isBluffetEventId(id: string | null | undefined): boolean {
  if (!id) return false
  const trimmed = id.trim()
  if (!trimmed) return false
  const lower = trimmed.toLowerCase()
  const fullLower = BLUFFET_EVENT_ID_FULL.toLowerCase()
  if (lower === fullLower || lower === BLUFFET_EVENT_ID_SHORT) {
    return true
  }
  if (lower.length === BLUFFET_EVENT_ID_SHORT.length && fullLower.endsWith(lower)) {
    return true
  }
  return false
}
