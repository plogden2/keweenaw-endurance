import { describe, it, expect } from 'vitest'
import type { Participant } from '@/types/models'
import {
  buildTestModeLeaderboard,
  buildTestModeTimingRecords,
  type TestModeTap,
} from './eventTestModeFlow'

const roster: Participant[] = [
  {
    id: 'p1',
    race_id: 'r1',
    bib_number: '101',
    first_name: 'Alex',
    last_name: 'Rivera',
    status: 'registered',
    category: { id: 'c1', race_id: 'r1', name: 'Open', category_type: 'open' },
    race: { id: 'r1', event_id: 'ev', name: '12 Hour', race_type: 'time_based', status: 'active' },
  },
  {
    id: 'p2',
    race_id: 'r2',
    bib_number: '202',
    first_name: 'Jamie',
    last_name: 'Stone',
    status: 'registered',
    race: { id: 'r2', event_id: 'ev', name: '6 Hour', race_type: 'time_based', status: 'active' },
  },
]

const taps: TestModeTap[] = [
  { participant_id: 'p1', recorded_at: '2026-07-31T12:00:00.000Z', source: 'rfid' },
  { participant_id: 'p2', recorded_at: '2026-07-31T12:00:30.000Z', source: 'manual' },
  { participant_id: 'p1', recorded_at: '2026-07-31T12:01:00.000Z', source: 'rfid' },
]

describe('eventTestModeFlow', () => {
  it('buildTestModeTimingRecords creates rfid_lap rows with participant embeds', () => {
    const records = buildTestModeTimingRecords(taps, roster)
    expect(records).toHaveLength(3)
    expect(records.every((r) => r.record_type === 'rfid_lap')).toBe(true)
    expect(records[0]?.participant_id).toBe('p1')
    expect(records[0]?.participant?.first_name).toBe('Alex')
    expect(records[0]?.local_timestamp).toBe('2026-07-31T12:00:00.000Z')
    expect(records[0]?.checkpoint?.checkpoint_type).toBe('finish')
  })

  it('buildTestModeLeaderboard ranks by laps then earliest last tap and includes race label in name', () => {
    const board = buildTestModeLeaderboard(taps, roster)
    expect(board).toHaveLength(2)
    expect(board[0]?.participant_id).toBe('p1')
    expect(board[0]?.laps).toBe(2)
    expect(board[0]?.place).toBe(1)
    expect(board[0]?.name).toMatch(/Alex Rivera/)
    expect(board[0]?.name).toMatch(/12 Hour/)
    expect(board[1]?.participant_id).toBe('p2')
    expect(board[1]?.laps).toBe(1)
  })
})
