import type { LiveLeaderboardEntry } from '@/services/api'
import type { Participant, TimingRecord } from '@/types/models'

export type TestModeTapSource = 'rfid' | 'manual'

export interface TestModeTap {
  participant_id: string
  recorded_at: string
  source: TestModeTapSource
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
  return taps.map((tap, index) => {
    const participant = byId.get(tap.participant_id)
    return {
      id: `test-tap-${index}-${tap.participant_id}`,
      participant_id: tap.participant_id,
      checkpoint_id: 'test-mode-finish',
      timestamp: tap.recorded_at,
      local_timestamp: tap.recorded_at,
      sync_status: 'synced',
      record_type: 'rfid_lap',
      participant,
      checkpoint: {
        id: 'test-mode-finish',
        race_id: participant?.race_id || 'test-mode',
        name: 'Finish',
        checkpoint_type: 'finish' as const,
      },
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

export function findParticipantByBib(
  roster: Participant[],
  bib: string,
): Participant | undefined {
  const needle = bib.trim()
  if (!needle) return undefined
  const matches = roster.filter((p) => String(p.bib_number) === needle)
  return matches.length === 1 ? matches[0] : undefined
}
