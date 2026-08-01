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

  it('coalesces duplicate RFID paths for the same tap within the nearby window', () => {
    const store = useEventTestModeStore()
    store.open('ev1', [
      makeParticipant({ id: 'p1', bib_number: '12', tag_uids: ['TAG-DUP'] }),
    ])
    const at = '2026-07-31T22:00:00.000Z'
    expect(store.recordTagTap('TAG-DUP', at).ok).toBe(true)
    // Second path (e.g. scan_result after tag_read) must not add another lap.
    const dup = store.recordTagTap('TAG-DUP', '2026-07-31T22:00:00.200Z')
    expect(dup.ok).toBe(true)
    expect(dup.lap_count).toBe(1)
    expect(store.taps).toHaveLength(1)

    // A later intentional retap still counts.
    expect(store.recordTagTap('TAG-DUP', '2026-07-31T22:00:01.000Z').ok).toBe(true)
    expect(store.taps).toHaveLength(2)
  })

  it('records taps for unassigned event bibs by tag uid and bib number', () => {
    const store = useEventTestModeStore()
    store.open(
      'ev1',
      [makeParticipant({ id: 'p1', bib_number: '1', tag_uids: ['TAG-P1'] })],
      [
        {
          id: 'bib-99',
          bib_number: '99',
          logical_uuid: 'aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee',
          tag_count: 1,
          tag_uids: ['aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee'],
        },
      ],
    )

    const byTag = store.recordTagTap('aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee')
    expect(byTag.ok).toBe(true)
    expect(byTag.bib_number).toBe('99')
    expect(byTag.participant_name).toMatch(/unassigned/i)
    expect(store.taps).toHaveLength(1)
    expect(store.leaderboard).toHaveLength(1)
    expect(store.leaderboard[0]?.bib_number).toBe('99')
    expect(store.leaderboard[0]?.name).toMatch(/unassigned/i)

    const byBib = store.recordBibTap('99')
    expect(byBib.ok).toBe(true)
    expect(store.taps).toHaveLength(2)
    expect(store.leaderboard[0]?.laps).toBe(2)
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

  it('recordBibTap rejects ambiguous bibs across races', () => {
    const store = useEventTestModeStore()
    store.open('ev1', [
      makeParticipant({ id: 'p1', bib_number: '1', race_id: 'r1' }),
      makeParticipant({ id: 'p2', bib_number: '1', race_id: 'r2' }),
    ])
    const result = store.recordBibTap('1')
    expect(result.ok).toBe(false)
    expect(result.message).toMatch(/Multiple/i)
    expect(store.taps).toHaveLength(0)
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

  it('ingestLocalBridgeTap records new bridge taps and ignores duplicates / baseline', () => {
    const store = useEventTestModeStore()
    store.open('ev1', [
      makeParticipant({
        id: 'p1',
        bib_number: '3',
        first_name: 'Benjamin',
        last_name: 'Ciavola',
        tag_uids: ['7db35ca0-fdfc-44b5-a220-6d322d867f6f'],
        race_id: 'r12',
      }),
    ])

    // Baseline: tap that happened before test mode opened must not count.
    store.noteBridgeTapBaseline({
      last_tap_uuid: '7db35ca0-fdfc-44b5-a220-6d322d867f6f',
      last_tap_at: '2026-07-31T21:25:00.000Z',
      last_tap_bib: '3',
      last_tap_race_id: 'r12',
    })
    expect(
      store.ingestLocalBridgeTap({
        last_tap_uuid: '7db35ca0-fdfc-44b5-a220-6d322d867f6f',
        last_tap_at: '2026-07-31T21:25:00.000Z',
        last_tap_bib: '3',
        last_tap_race_id: 'r12',
      }),
    ).toBeNull()
    expect(store.taps).toHaveLength(0)

    const first = store.ingestLocalBridgeTap({
      last_tap_uuid: '7db35ca0-fdfc-44b5-a220-6d322d867f6f',
      last_tap_at: '2026-07-31T21:26:10.000Z',
      last_tap_bib: '3',
      last_tap_race_id: 'r12',
    })
    expect(first?.ok).toBe(true)
    expect(first?.bib_number).toBe('3')
    expect(store.taps).toHaveLength(1)

    // Same snapshot again is a no-op.
    expect(
      store.ingestLocalBridgeTap({
        last_tap_uuid: '7db35ca0-fdfc-44b5-a220-6d322d867f6f',
        last_tap_at: '2026-07-31T21:26:10.000Z',
        last_tap_bib: '3',
        last_tap_race_id: 'r12',
      }),
    ).toBeNull()
    expect(store.taps).toHaveLength(1)
  })

  it('ingestLocalBridgeTap falls back to bib when tag uid is missing from roster', () => {
    const store = useEventTestModeStore()
    store.open('ev1', [
      makeParticipant({ id: 'p1', bib_number: '3', first_name: 'Ben', last_name: 'C', race_id: 'r12' }),
    ])
    const result = store.ingestLocalBridgeTap({
      last_tap_uuid: 'unknown-chip-uuid',
      last_tap_at: '2026-07-31T21:26:10.000Z',
      last_tap_bib: '3',
      last_tap_race_id: 'r12',
    })
    expect(result?.ok).toBe(true)
    expect(store.taps).toHaveLength(1)
    expect(store.taps[0]?.source).toBe('rfid')
  })
})
