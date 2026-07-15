import type { APIRequestContext } from '@playwright/test'
import { API_BASE, BLUFFET, pinToken } from '../../fixtures/rfid'
import type { Racer } from './roster'

/**
 * Overwrite the shared 30/15-minute start time to now+2m; kids (5-minute) race
 * starts at T+20m. Bib/category assignment is untouched — only start_time moves.
 */
export async function setCompressedStartTimes(request: APIRequestContext, tZero: Date) {
  const token = await pinToken(request)
  const kidsStart = new Date(tZero.getTime() + 20 * 60_000)
  const updates = [
    [BLUFFET.races.twelveHour.id, tZero],
    [BLUFFET.races.sixHour.id, tZero],
    [BLUFFET.races.kids.id, kidsStart],
  ] as const
  for (const [id, start] of updates) {
    const res = await request.put(`${API_BASE}/api/races/${id}`, {
      headers: { Authorization: `Bearer ${token}` },
      data: { start_time: start.toISOString() },
    })
    if (!res.ok()) throw new Error(`update race ${id}: ${res.status()} ${await res.text()}`)
  }
}

/** Resolve a race's first category id — late signups only need any valid category. */
export async function firstCategoryId(
  request: APIRequestContext,
  raceId: string,
): Promise<string> {
  const token = await pinToken(request)
  const res = await request.get(`${API_BASE}/api/races/${raceId}/categories`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok()) throw new Error(`categories ${raceId}: ${res.status()} ${await res.text()}`)
  const body = await res.json()
  const rows = body.data ?? body
  if (!Array.isArray(rows) || rows.length === 0) {
    throw new Error(`no categories found for race ${raceId}`)
  }
  return rows[0].id
}

const finishCheckpointCache = new Map<string, string>()

/** Resolve (and cache) a race's "finish" checkpoint id — needed for manual-entry recovery. */
export async function finishCheckpointId(
  request: APIRequestContext,
  raceId: string,
): Promise<string> {
  const cached = finishCheckpointCache.get(raceId)
  if (cached) return cached
  const token = await pinToken(request)
  const res = await request.get(`${API_BASE}/api/races/${raceId}/checkpoints`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok()) throw new Error(`checkpoints ${raceId}: ${res.status()} ${await res.text()}`)
  const body = await res.json()
  const rows = body.data ?? body
  const finish = (Array.isArray(rows) ? rows : []).find(
    (cp: { checkpoint_type?: string }) => cp.checkpoint_type === 'finish',
  )
  if (!finish) throw new Error(`no finish checkpoint found for race ${raceId}`)
  finishCheckpointCache.set(raceId, finish.id)
  return finish.id
}

export type LateSignup = {
  id: string
  raceId: string
  firstName: string
  lastName: string
}

/**
 * Creates a last-minute participant on top of the pre-registered 100.
 * The first lap for this racer is scored later via `programRacerAndAwaitLap`,
 * which also programs (writes) their permanent logical UUID onto the chip.
 */
export async function addLateSignup(
  request: APIRequestContext,
  raceId: string,
  categoryId: string,
  name: { first: string; last: string },
): Promise<LateSignup> {
  const token = await pinToken(request)
  const res = await request.post(`${API_BASE}/api/races/${raceId}/participants`, {
    headers: { Authorization: `Bearer ${token}` },
    data: { first_name: name.first, last_name: name.last, category_id: categoryId },
  })
  if (!res.ok()) throw new Error(`late signup: ${res.status()} ${await res.text()}`)
  const p = await res.json()
  return { id: p.id, raceId, firstName: name.first, lastName: name.last }
}

/**
 * Late signups are created with `bib: ''` locally; the API assigns a bib
 * asynchronously. Before manual-entry recovery (which needs bib_number),
 * fetch each race's participant list and fill in any missing bibs by id.
 */
export async function refreshEmptyBibs(request: APIRequestContext, racers: Racer[]): Promise<void> {
  const needsBib = racers.filter((r) => !r.bib)
  if (needsBib.length === 0) return

  const token = await pinToken(request)
  const byRace = new Map<string, Racer[]>()
  for (const r of needsBib) {
    const list = byRace.get(r.raceId) ?? []
    list.push(r)
    byRace.set(r.raceId, list)
  }

  for (const [raceId, raceRacers] of byRace) {
    const res = await request.get(`${API_BASE}/api/races/${raceId}/participants`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    if (!res.ok()) continue
    const body = await res.json()
    const rows = body.data ?? body
    if (!Array.isArray(rows)) continue
    const bibById = new Map(
      rows.map((p: { id: string; bib_number?: string; bibNumber?: string }) => [
        p.id,
        String(p.bib_number ?? p.bibNumber ?? ''),
      ]),
    )
    for (const racer of raceRacers) {
      const bib = bibById.get(racer.id)
      if (bib) racer.bib = bib
    }
  }
}
