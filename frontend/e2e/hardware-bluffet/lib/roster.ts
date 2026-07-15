import type { APIRequestContext } from '@playwright/test'
import { API_BASE, BLUFFET, pinToken } from '../../fixtures/rfid'

export type Racer = {
  id: string
  raceId: string
  bib: string
  firstName: string
  lastName: string
  logicalTagUuid: string // permanent racer RFID UUID (from seed/association)
}

export type LoadSeededRacersResult = {
  racers: Racer[]
  /** Racers whose participant record has no logical tag UUID yet (never scored). */
  skipped: Racer[]
}

/**
 * Loads all seeded participants across the three Bluffet races. Any racer
 * with no logical tag UUID (association missing/not yet written) is split
 * into `skipped` rather than included in `racers` — a lap engine that tried
 * to program an empty UUID onto the chip would silently corrupt the run.
 */
export async function loadSeededRacers(request: APIRequestContext): Promise<LoadSeededRacersResult> {
  const token = await pinToken(request)
  const racers: Racer[] = []
  const skipped: Racer[] = []
  for (const race of Object.values(BLUFFET.races)) {
    const res = await request.get(`${API_BASE}/api/races/${race.id}/participants`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    if (!res.ok()) throw new Error(`participants ${res.status()}`)
    const body = await res.json()
    const rows = body.data ?? body
    for (const p of rows) {
      const racer: Racer = {
        id: p.id,
        raceId: race.id,
        bib: String(p.bib_number ?? p.bibNumber ?? ''),
        firstName: p.first_name ?? p.firstName,
        lastName: p.last_name ?? p.lastName,
        logicalTagUuid: p.rfid_tag_uid ?? p.rfidTagUid ?? (p.tag_uids?.[0] ?? ''),
      }
      if (racer.logicalTagUuid) {
        racers.push(racer)
      } else {
        skipped.push(racer)
      }
    }
  }
  return { racers, skipped }
}

export function pickNoShows(racers: Racer[], n = 9): Set<string> {
  const sorted = [...racers].sort((a, b) => a.id.localeCompare(b.id))
  return new Set(sorted.slice(0, n).map((r) => r.id))
}

export function pickDnfs(activeIds: string[], n = 10, seed = 42): Set<string> {
  const arr = [...activeIds].sort()
  const out = new Set<string>()
  let x = seed
  while (out.size < Math.min(n, arr.length)) {
    x = (x * 1103515245 + 12345) & 0x7fffffff
    out.add(arr[x % arr.length])
  }
  return out
}
