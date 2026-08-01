const FALLBACK_TIME_ZONE = 'America/Detroit'

/** Device IANA timezone, or America/Detroit if unavailable. */
export function getDisplayTimeZone(): string {
  const tz = Intl.DateTimeFormat().resolvedOptions().timeZone
  return tz && tz.trim() ? tz : FALLBACK_TIME_ZONE
}

/** Format an ISO timestamp in the display timezone; invalid → '—'. */
export function formatDateTime(
  iso: string,
  opts?: Intl.DateTimeFormatOptions,
): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return '—'
  return new Intl.DateTimeFormat(undefined, {
    ...opts,
    timeZone: getDisplayTimeZone(),
  }).format(d)
}

/** 24h HH:mm in the display timezone via formatToParts. */
export function formatTimeHHMM(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return '—'
  const parts = new Intl.DateTimeFormat('en-US', {
    timeZone: getDisplayTimeZone(),
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  }).formatToParts(d)
  const hour = parts.find((p) => p.type === 'hour')?.value ?? ''
  const minute = parts.find((p) => p.type === 'minute')?.value ?? ''
  return `${hour.padStart(2, '0')}:${minute.padStart(2, '0')}`
}

/**
 * Interpret YYYY-MM-DD + HH:mm as device-local wall time and return UTC ISO (Z).
 */
export function wallTimeToRFC3339(
  dateYYYYMMDD: string,
  timeHHMM: string,
): string {
  const [y, m, d] = dateYYYYMMDD.split('-').map(Number)
  const [hh, mm] = timeHHMM.split(':').map(Number)
  const date = new Date(y, m - 1, d, hh, mm, 0, 0)
  return date.toISOString()
}

/** HH:mm for `<input type="time">` in the display timezone. */
export function isoToTimeInputValue(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  return formatTimeHHMM(iso)
}

/** YYYY-MM-DD for `<input type="date">` in the display timezone. */
export function isoToDateInputValue(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  const parts = new Intl.DateTimeFormat('en-US', {
    timeZone: getDisplayTimeZone(),
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).formatToParts(d)
  const year = parts.find((p) => p.type === 'year')?.value
  const month = parts.find((p) => p.type === 'month')?.value
  const day = parts.find((p) => p.type === 'day')?.value
  if (!year || !month || !day) return ''
  return `${year}-${month}-${day}`
}
