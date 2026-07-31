import { describe, it, expect, beforeEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import type { Participant } from '@/types/models'
import { useEventTestModeStore } from './eventTestMode'

function makeParticipant(overrides: Partial<Participant> & Pick<Participant, 'id' | 'bib_number'>): Participant {
  return {
    race_id: 'race-12h',
    first_name: 'Alex',
    last_name: 'Rivera',
    status: 'registered',
    tag_uids: [],
    ...overrides,
  }
}

describe('eventTestMode store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('starts closed with empty taps', () => {
    const store = useEventTestModeStore()
    expect(store.isOpen).toBe(false)
    expect(store.eventId).toBeNull()
    expect(store.taps).toEqual([])
    expect(store.leaderboard).toEqual([])
  })

  it('open sets event and loads roster; close discards taps', () => {
    const store = useEventTestModeStore()
    const roster = [
      makeParticipant({
        id: 'p1',
        bib_number: '101',
        tag_uids: ['TAG-A'],
        race: { id: 'race-12h', event_id: 'ev1', name: '12 Hour', race_type: 'time_based', status: 'active' },
      }),
    ]

    store.open('ev1', roster)
    expect(store.isOpen).toBe(true)
    expect(store.eventId).toBe('ev1')
    expect(store.roster).toHaveLength(1)

    const result = store.recordTagTap('TAG-A')
    expect(result.ok).toBe(true)
    expect(store.taps).toHaveLength(1)
    expect(store.leaderboard[0]?.laps).toBe(1)

    store.close()
    expect(store.isOpen).toBe(false)
    expect(store.eventId).toBeNull()
    expect(store.taps).toEqual([])
    expect(store.leaderboard).toEqual([])
    expect(store.roster).toEqual([])
  })

  it('isActiveForEvent is true only while open for that event', () => {
    const store = useEventTestModeStore()
    expect(store.isActiveForEvent('ev1')).toBe(false)
    store.open('ev1', [])
    expect(store.isActiveForEvent('ev1')).toBe(true)
    expect(store.isActiveForEvent('ev2')).toBe(false)
  })

  it('recordTagTap resolves tag_uids and rfid_tag_uid; unknown returns error', () => {
    const store = useEventTestModeStore()
    store.open('ev1', [
      makeParticipant({ id: 'p1', bib_number: '1', tag_uids: ['UID-1'] }),
      makeParticipant({ id: 'p2', bib_number: '2', rfid_tag_uid: 'LEGACY-2', tag_uids: [] }),
    ])

    expect(store.recordTagTap('UID-1').ok).toBe(true)
    expect(store.recordTagTap('LEGACY-2').ok).toBe(true)
    const unknown = store.recordTagTap('NOPE')
    expect(unknown.ok).toBe(false)
    expect(unknown.message).toMatch(/unknown/i)
    expect(store.taps).toHaveLength(2)
  })

  it('recordBibTap requires exact bib match; allows rapid repeat taps', () => {
    const store = useEventTestModeStore()
    store.open('ev1', [
      makeParticipant({ id: 'p1', bib_number: '42', first_name: 'Sam', last_name: 'Lee' }),
      makeParticipant({ id: 'p2', bib_number: '43' }),
    ])

    expect(store.recordBibTap('999').ok).toBe(false)
    expect(store.recordBibTap('42').ok).toBe(true)
    expect(store.recordBibTap('42').ok).toBe(true)
    expect(store.leaderboard).toHaveLength(1)
    expect(store.leaderboard[0]?.laps).toBe(2)
    expect(store.leaderboard[0]?.name).toContain('Sam')
  })

  it('leaderboard merges across races and sorts by laps then earliest last tap', () => {
    const store = useEventTestModeStore()
    store.open('ev1', [
      makeParticipant({
        id: 'p-slow',
        bib_number: '10',
        first_name: 'Slow',
        last_name: 'A',
        race_id: 'r1',
        race: { id: 'r1', event_id: 'ev1', name: '12H', race_type: 'time_based', status: 'active' },
      }),
      makeParticipant({
        id: 'p-fast',
        bib_number: '20',
        first_name: 'Fast',
        last_name: 'B',
        race_id: 'r2',
        race: { id: 'r2', event_id: 'ev1', name: '6H', race_type: 'time_based', status: 'active' },
      }),
    ])

    const t0 = '2026-07-31T12:00:00.000Z'
    const t1 = '2026-07-31T12:01:00.000Z'
    const t2 = '2026-07-31T12:02:00.000Z'

    store.recordBibTap('10', t0)
    store.recordBibTap('20', t0)
    store.recordBibTap('10', t2)
    store.recordBibTap('20', t1)

    // Both have 2 laps; Fast finished second lap earlier → place 1
    expect(store.leaderboard.map((e) => e.participant_id)).toEqual(['p-fast', 'p-slow'])
    expect(store.leaderboard[0]?.place).toBe(1)
    expect(store.leaderboard[0]?.laps).toBe(2)
    expect(store.leaderboard[1]?.place).toBe(2)
  })

  it('lastFeedback tracks the most recent tap outcome', () => {
    const store = useEventTestModeStore()
    store.open('ev1', [makeParticipant({ id: 'p1', bib_number: '7', tag_uids: ['T7'] })])
    store.recordTagTap('T7')
    expect(store.lastFeedback?.ok).toBe(true)
    expect(store.lastFeedback?.bib_number).toBe('7')
    store.recordTagTap('MISSING')
    expect(store.lastFeedback?.ok).toBe(false)
  })
})
