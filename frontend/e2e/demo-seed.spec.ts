/**
 * T022 / T067 / US8 — All You Can East Bluffet 2026 demo seed acceptance.
 * Relies on deterministic BLUFFET UUIDs from generate_bluffet_seed.py.
 */
import { expect, test } from '@playwright/test'
import { API_BASE, BLUFFET } from './fixtures/rfid'

test.describe('US8 demo seed — All You Can East Bluffet 2026', () => {
  test('event list shows All You Can East Bluffet on 2026-08-01', async ({ page, request }) => {
    const res = await request.get(`${API_BASE}/api/events`)
    expect(res.ok()).toBeTruthy()
    const payload = await res.json()
    const events = Array.isArray(payload) ? payload : (payload.data ?? payload.events ?? [])
    const bluffet = events.find(
      (e: { name?: string; id?: string }) =>
        e.name === BLUFFET.eventName || e.id === BLUFFET.eventId,
    )
    expect(bluffet).toBeTruthy()
    expect(String(bluffet.event_date ?? bluffet.date ?? '')).toContain(BLUFFET.eventDate)

    // At least one page assertion (US8 still needs UI coverage even when seed is backend-first).
    await page.goto('/timing')
    await expect(page.getByText(BLUFFET.eventName)).toBeVisible()
    await expect(
      page.getByText(/2026-08-01|08\/01\/2026|August 1,? 2026|Aug 1,? 2026/i),
    ).toBeVisible()
  })

  test('three lap races with start times 08:00 / 08:00 / 15:00', async ({ request }) => {
    const res = await request.get(`${API_BASE}/api/races`, {
      params: { event_id: BLUFFET.eventId, limit: 50 },
    })
    expect(res.ok()).toBeTruthy()
    const payload = await res.json()
    const races = Array.isArray(payload) ? payload : (payload.data ?? payload.races ?? [])
    expect(races).toHaveLength(3)

    const byName = Object.fromEntries(
      races.map((r: { name: string }) => [r.name, r]),
    ) as Record<
      string,
      { name: string; race_type?: string; start_time?: string; duration_minutes?: number }
    >

    expect(byName[BLUFFET.races.twelveHour.name]).toBeTruthy()
    expect(byName[BLUFFET.races.sixHour.name]).toBeTruthy()
    expect(byName[BLUFFET.races.kids.name]).toBeTruthy()

    for (const r of races) {
      expect(r.race_type === 'lap_based' || r.race_type === 'lap').toBeTruthy()
    }

    const startLocal = (iso: string) => {
      // America/Detroit wall time on seed day (EDT = UTC-4).
      const d = new Date(iso)
      const parts = new Intl.DateTimeFormat('en-US', {
        timeZone: 'America/Detroit',
        hour: '2-digit',
        minute: '2-digit',
        hour12: false,
      }).formatToParts(d)
      const hour = parts.find((p) => p.type === 'hour')?.value ?? ''
      const minute = parts.find((p) => p.type === 'minute')?.value ?? ''
      return `${hour.padStart(2, '0')}:${minute.padStart(2, '0')}`
    }

    expect(startLocal(byName[BLUFFET.races.twelveHour.name].start_time!)).toBe('08:00')
    expect(startLocal(byName[BLUFFET.races.sixHour.name].start_time!)).toBe('08:00')
    expect(startLocal(byName[BLUFFET.races.kids.name].start_time!)).toBe('15:00')
  })

  test('category matrix: 4 per adult race, 2 for kids', async ({ request }) => {
    const adult = [BLUFFET.races.twelveHour, BLUFFET.races.sixHour]
    for (const race of adult) {
      const res = await request.get(`${API_BASE}/api/races/${race.id}/categories`)
      expect(res.ok()).toBeTruthy()
      const payload = await res.json()
      const categories = Array.isArray(payload)
        ? payload
        : (payload.data ?? payload.categories ?? [])
      expect(categories).toHaveLength(4)
      const names = categories.map((c: { name: string }) => c.name)
      for (const expected of race.categoryNames) {
        expect(names).toContain(expected)
      }
    }

    const kidsRes = await request.get(
      `${API_BASE}/api/races/${BLUFFET.races.kids.id}/categories`,
    )
    expect(kidsRes.ok()).toBeTruthy()
    const kidsPayload = await kidsRes.json()
    const kidsCats = Array.isArray(kidsPayload)
      ? kidsPayload
      : (kidsPayload.data ?? kidsPayload.categories ?? [])
    expect(kidsCats).toHaveLength(2)
    const kidsNames = kidsCats.map((c: { name: string }) => c.name)
    expect(kidsNames).toEqual(expect.arrayContaining(['Men', 'Women']))
    expect(kidsNames.some((n: string) => /intermediate|advanced/i.test(n))).toBeFalsy()
  })

  test('100 racers total across the demo event', async ({ request }) => {
    const raceIds = [
      BLUFFET.races.twelveHour.id,
      BLUFFET.races.sixHour.id,
      BLUFFET.races.kids.id,
    ]
    let total = 0
    for (const raceId of raceIds) {
      const res = await request.get(`${API_BASE}/api/races/${raceId}/participants`, {
        params: { limit: 200 },
      })
      expect(res.ok()).toBeTruthy()
      const payload = await res.json()
      const participants = Array.isArray(payload)
        ? payload
        : (payload.data ?? payload.participants ?? [])
      total += participants.length
    }
    expect(total).toBe(BLUFFET.expectedRacerCount)
  })
})
