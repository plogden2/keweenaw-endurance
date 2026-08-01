import type { LiveLeaderboardEntry } from '@/services/api'
import type { BibListItem, Checkpoint, Participant, TimingRecord } from '@/types/models'

export type TestModeTapSource = 'rfid' | 'manual'

export interface TestModeTap {
  participant_id: string
  recorded_at: string
  source: TestModeTapSource
}

export function unassignedBibParticipantId(bibId: string): string {
  return `unassigned-bib:${bibId}`
}

/** Synthetic roster row so unassigned event bibs can appear on the test board. */
export function participantFromUnassignedBib(bib: BibListItem): Participant {
  const uids = [...(bib.tag_uids ?? [])]
  if (bib.logical_uuid) uids.push(bib.logical_uuid)
  const unique = [...new Set(uids.map((u) => u.trim()).filter(Boolean))]
  return {
    id: unassignedBibParticipantId(bib.id),
    race_id: bib.race_id || 'unassigned',
    bib_number: bib.bib_number,
    first_name: 'Unassigned',
    last_name: '',
    status: 'registered',
    tag_uids: unique,
    race: {
      id: bib.race_id || 'unassigned',
      event_id: '',
      name: 'Unassigned bib',
      race_type: 'time_based',
      status: 'scheduled',
    },
  }
}

export function findBibByTag(bibs: BibListItem[], tagUid: string): BibListItem | undefined {
  const needle = tagUid.trim().toLowerCase()
  if (!needle) return undefined
  return bibs.find((bib) => {
    const uids = [...(bib.tag_uids ?? [])]
    if (bib.logical_uuid) uids.push(bib.logical_uuid)
    return uids.some((uid) => uid.trim().toLowerCase() === needle)
  })
}

export function findBibByNumber(bibs: BibListItem[], bibNumber: string): BibListItem | undefined {
  const needle = bibNumber.trim()
  if (!needle) return undefined
  return bibs.find((bib) => String(bib.bib_number) === needle)
}

function participantLabel(p: Participant): string {
  const name = `${p.first_name} ${p.last_name}`.trim()
  const raceName = p.race?.name?.trim()
  return raceName ? `${name} · ${raceName}` : name
}

function categoryKey(p: Participant): string {
  return p.category?.name?.toLowerCase().replace(/\s+/g, '_') || 'uncategorized'
}

export function buildTestModeTimingRecords(
  taps: TestModeTap[],
  roster: Participant[],
): TimingRecord[] {
  const byId = new Map(roster.map((p) => [p.id, p]))
  return taps.map((tap, index): TimingRecord => {
    const participant = byId.get(tap.participant_id)
    const checkpoint: Checkpoint = {
      id: 'test-mode-finish',
      race_id: participant?.race_id || 'test-mode',
      name: 'Finish',
      checkpoint_type: 'finish',
      is_active: true,
    }
    return {
      id: `test-tap-${index}-${tap.participant_id}`,
      participant_id: tap.participant_id,
      checkpoint_id: checkpoint.id,
      timestamp: tap.recorded_at,
      local_timestamp: tap.recorded_at,
      sync_status: 'synced',
      record_type: 'rfid_lap',
      participant,
      checkpoint,
    }
  })
}

export function buildTestModeLeaderboard(
  taps: TestModeTap[],
  roster: Participant[],
): LiveLeaderboardEntry[] {
  const byId = new Map(roster.map((p) => [p.id, p]))
  const aggregates = new Map<
    string,
    { laps: number; last_lap_at: string; participant: Participant }
  >()

  for (const tap of taps) {
    const participant = byId.get(tap.participant_id)
    if (!participant) continue
    const existing = aggregates.get(tap.participant_id)
    if (!existing) {
      aggregates.set(tap.participant_id, {
        laps: 1,
        last_lap_at: tap.recorded_at,
        participant,
      })
      continue
    }
    existing.laps += 1
    if (tap.recorded_at > existing.last_lap_at) {
      existing.last_lap_at = tap.recorded_at
    }
  }

  const rows = [...aggregates.values()].sort((a, b) => {
    if (b.laps !== a.laps) return b.laps - a.laps
    return a.last_lap_at.localeCompare(b.last_lap_at)
  })

  return rows.map((row, index) => ({
    place: index + 1,
    participant_id: row.participant.id,
    bib_number: row.participant.bib_number,
    name: participantLabel(row.participant),
    category_key: categoryKey(row.participant),
    laps: row.laps,
    last_lap_at: row.last_lap_at,
  }))
}

export function findParticipantByTag(
  roster: Participant[],
  tagUid: string,
): Participant | undefined {
  const needle = tagUid.trim().toLowerCase()
  if (!needle) return undefined
  return roster.find((p) => {
    const uids = [...(p.tag_uids ?? [])]
    if (p.rfid_tag_uid) uids.push(p.rfid_tag_uid)
    return uids.some((uid) => uid.trim().toLowerCase() === needle)
  })
}

export function findParticipantsByBib(roster: Participant[], bib: string): Participant[] {
  const needle = bib.trim()
  if (!needle) return []
  return roster.filter((p) => String(p.bib_number) === needle)
}

export function findParticipantByBib(
  roster: Participant[],
  bib: string,
): Participant | undefined {
  const matches = findParticipantsByBib(roster, bib)
  return matches.length === 1 ? matches[0] : undefined
}
