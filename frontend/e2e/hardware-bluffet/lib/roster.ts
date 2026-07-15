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

export async function loadSeededRacers(request: APIRequestContext): Promise<Racer[]> {
  const token = await pinToken(request)
  const racers: Racer[] = []
  for (const race of Object.values(BLUFFET.races)) {
    const res = await request.get(`${API_BASE}/api/races/${race.id}/participants`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    if (!res.ok()) throw new Error(`participants ${res.status()}`)
    const body = await res.json()
    const rows = body.data ?? body
    for (const p of rows) {
      racers.push({
        id: p.id,
        raceId: race.id,
        bib: String(p.bib_number ?? p.bibNumber ?? ''),
        firstName: p.first_name ?? p.firstName,
        lastName: p.last_name ?? p.lastName,
        logicalTagUuid: p.rfid_tag_uid ?? p.rfidTagUid ?? (p.tag_uids?.[0] ?? ''),
      })
    }
  }
  return racers
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
