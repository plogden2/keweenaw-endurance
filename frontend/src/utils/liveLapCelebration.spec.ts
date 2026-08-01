import { describe, expect, it } from 'vitest'
import type { EventLiveRace } from '@/services/api'
import { detectLapIncreases, lapSnapshotKey } from './liveLapCelebration'

function race(
  id: string,
  rows: Array<{ id: string; name: string; laps: number; bib?: string }>,
): EventLiveRace {
  return {
    id,
    name: id,
    race_type: 'lap_based',
    status: 'active',
    start_time: '2026-08-01T08:00:00Z',
    countdown_seconds: 0,
    leaderboard_overall: rows.map((r, i) => ({
      place: i + 1,
      participant_id: r.id,
      bib_number: r.bib ?? '1',
      name: r.name,
      category_key: 'open',
      laps: r.laps,
    })),
    flow_series: [],
  }
}

describe('detectLapIncreases', () => {
  it('seeds snapshot without celebrating on first observation', () => {
    const prev = new Map<string, number>()
    const { events, next } = detectLapIncreases(
      [race('r12', [{ id: 'p1', name: 'Alex Rivera', laps: 3 }])],
      prev,
    )
    expect(events).toEqual([])
    expect(next.get(lapSnapshotKey('r12', 'p1'))).toBe(3)
  })

  it('emits celebration when laps increase after seed', () => {
    const prev = new Map([[lapSnapshotKey('r12', 'p1'), 3]])
    const { events, next } = detectLapIncreases(
      [race('r12', [{ id: 'p1', name: 'Alex Rivera', laps: 4, bib: '12' }])],
      prev,
    )
    expect(events).toHaveLength(1)
    expect(events[0]?.participant_name).toBe('Alex Rivera')
    expect(events[0]?.race_id).toBe('r12')
    expect(events[0]?.lap_count).toBe(4)
    expect(events[0]?.bib_number).toBe('12')
    expect(next.get(lapSnapshotKey('r12', 'p1'))).toBe(4)
  })

  it('ignores unchanged and decreased lap counts', () => {
    const prev = new Map([
      [lapSnapshotKey('r12', 'p1'), 5],
      [lapSnapshotKey('r12', 'p2'), 2],
    ])
    const { events } = detectLapIncreases(
      [
        race('r12', [
          { id: 'p1', name: 'Alex', laps: 5 },
          { id: 'p2', name: 'Jordan', laps: 1 },
        ]),
      ],
      prev,
    )
    expect(events).toEqual([])
  })
})
