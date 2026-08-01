/** Absolute URL for the public event live page. */
export function liveEventUrl(origin: string, eventId: string): string {
  const base = origin.replace(/\/+$/, '')
  const id = String(eventId || '').trim()
  return `${base}/events/${id}/live`
}
