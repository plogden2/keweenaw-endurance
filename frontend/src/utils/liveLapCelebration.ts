import type { EventLiveRace, LapRecordedEvent } from '@/services/api'

export type LapCountSnapshot = Map<string, number>

/** Key for per-race participant lap tracking. */
export function lapSnapshotKey(raceId: string, participantId: string): string {
  return `${raceId}|${participantId}`
}

/**
 * Diff the latest live boards against a prior lap snapshot and emit celebration
 * events for any individual whose scored laps increased. First observation of a
 * participant only seeds the snapshot (no celebration on initial page load).
 */
export function detectLapIncreases(
  races: EventLiveRace[],
  previous: LapCountSnapshot,
): { events: LapRecordedEvent[]; next: LapCountSnapshot } {
  const next: LapCountSnapshot = new Map(previous)
  const events: LapRecordedEvent[] = []
  const now = new Date().toISOString()

  for (const race of races) {
    if (!race?.id) continue
    for (const entry of race.leaderboard_overall ?? []) {
      if (!entry?.participant_id) continue
      const key = lapSnapshotKey(race.id, entry.participant_id)
      const laps = Number(entry.laps) || 0
      const prior = previous.get(key)
      if (prior != null && laps > prior) {
        events.push({
          type: 'lap_recorded',
          event_id: '',
          race_id: race.id,
          participant_id: entry.participant_id,
          participant_name: entry.name,
          bib_number: entry.bib_number,
          lap_count: laps,
          recorded_at: entry.last_lap_at || now,
        })
      }
      next.set(key, laps)
    }
  }

  return { events, next }
}
