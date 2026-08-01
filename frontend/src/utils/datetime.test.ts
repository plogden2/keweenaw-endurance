import { describe, expect, it } from 'vitest'
import {
  formatDateTime,
  formatTimeHHMM,
  getDisplayTimeZone,
  isoToDateInputValue,
  isoToTimeInputValue,
  wallTimeToRFC3339,
} from './datetime'

describe('getDisplayTimeZone', () => {
  it('returns a non-empty IANA timezone string', () => {
    const tz = getDisplayTimeZone()
    expect(typeof tz).toBe('string')
    expect(tz.length).toBeGreaterThan(0)
  })

  it('falls back to America/Detroit when resolved timezone is empty', () => {
    const original = Intl.DateTimeFormat
    // @ts-expect-error test stub
    Intl.DateTimeFormat = function () {
      return { resolvedOptions: () => ({ timeZone: '' }) }
    }
    try {
      expect(getDisplayTimeZone()).toBe('America/Detroit')
    } finally {
      Intl.DateTimeFormat = original
    }
  })
})

describe('formatDateTime', () => {
  it('returns em dash for invalid ISO', () => {
    expect(formatDateTime('not-a-date')).toBe('—')
  })

  it('formats a valid ISO in the display timezone', () => {
    const iso = wallTimeToRFC3339('2026-08-01', '09:15')
    const formatted = formatDateTime(iso)
    expect(formatted).not.toBe('—')
    expect(formatted.length).toBeGreaterThan(0)
  })
})

describe('formatTimeHHMM / isoToTimeInputValue', () => {
  it('round-trips wall time through ISO as HH:mm', () => {
    const iso = wallTimeToRFC3339('2026-08-01', '14:30')
    expect(formatTimeHHMM(iso)).toBe('14:30')
    expect(isoToTimeInputValue(iso)).toBe('14:30')
  })

  it('returns em dash / empty for invalid ISO', () => {
    expect(formatTimeHHMM('bad')).toBe('—')
    expect(isoToTimeInputValue('bad')).toBe('')
  })
})

describe('isoToDateInputValue', () => {
  it('round-trips wall date through ISO as YYYY-MM-DD', () => {
    const iso = wallTimeToRFC3339('2026-07-31', '21:30')
    expect(isoToDateInputValue(iso)).toBe('2026-07-31')
  })

  it('returns empty for invalid ISO', () => {
    expect(isoToDateInputValue('bad')).toBe('')
  })
})

describe('wallTimeToRFC3339', () => {
  it('emits UTC Z and preserves device-local wall clock', () => {
    const iso = wallTimeToRFC3339('2026-08-01', '08:00')
    expect(iso.endsWith('Z')).toBe(true)
    const d = new Date(iso)
    expect(d.getFullYear()).toBe(2026)
    expect(d.getMonth()).toBe(7)
    expect(d.getDate()).toBe(1)
    expect(d.getHours()).toBe(8)
    expect(d.getMinutes()).toBe(0)
  })

  it('does not hardcode a fixed UTC offset string', () => {
    const iso = wallTimeToRFC3339('2026-08-01', '08:00')
    expect(iso).not.toContain('-04:00')
    expect(iso).not.toContain('-05:00')
  })
})
