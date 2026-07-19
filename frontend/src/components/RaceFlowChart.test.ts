import { describe, it, expect, vi, beforeEach, type Mock } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { mount, flushPromises } from '@vue/test-utils'
import { Chart } from 'chart.js'
import RaceFlowChart from './RaceFlowChart.vue'
import { timingApi } from '@/services/api'
import { resolveCategoryColor } from '@/themes/defaultLegend'
import { buildParticipantFlowTooltip, buildParticipantFlows, buildRaceStatistics, buildExtrapolationPoint, clampElapsedToDuration, expandSteppedLapPoints, formatAverageResult, formatDuration, getAverageResultLabel, getCurrentElapsedMinutes, getParticipantAgeGroupKey, getParticipantAgeGroupLabel, getParticipantGenderKey, resolveRaceFlowAxisMaxMinutes, resolveRaceFlowXAxisMax, resolveRaceStartMs } from '@/utils/raceFlowData'
import { convertDistanceFromKm, KM_TO_MILES } from '@/utils/units'
import { setupPinia } from '@/test/helpers'
import type { TimingRecord } from '@/types/models'

vi.mock('chart.js', () => ({
  Chart: Object.assign(
    vi.fn((_canvas, config) => {
      const instance = {
        destroy: vi.fn(),
        update: vi.fn(),
        data: config?.data ?? { datasets: [] },
        options: config?.options ?? {},
        canvas: { style: { cursor: 'default' } as { cursor: string } },
        getElementsAtEventForMode: vi.fn().mockReturnValue([]),
      }
      return instance
    }),
    { register: vi.fn() },
  ),
  LineController: vi.fn(),
  LineElement: vi.fn(),
  PointElement: vi.fn(),
  LinearScale: vi.fn(),
  CategoryScale: vi.fn(),
  Title: vi.fn(),
  Tooltip: vi.fn(),
  Legend: vi.fn(),
}))

vi.mock('@/services/api', async () => {
  const actual = await vi.importActual<typeof import('@/services/api')>('@/services/api')
  return {
    ...actual,
    timingApi: {
      getLive: vi.fn(),
      getLeaderboard: vi.fn(),
      getResults: vi.fn(),
    },
    raceParticipantsApi: {
      list: vi.fn().mockResolvedValue({ data: { data: [], total: 0 } }),
    },
  }
})

const sampleRecords: TimingRecord[] = [
  {
    id: 'r1',
    participant_id: 'p1',
    checkpoint_id: 'cp-finish',
    timestamp: '2024-06-01T11:00:00.000Z',
    local_timestamp: '2024-06-01T11:00:00.000Z',
    sync_status: 'synced',
    participant: {
      id: 'p1',
      race_id: 'race-1',
      bib_number: '7',
      first_name: 'Alex',
      last_name: 'Runner',
      gender: 'male',
      age: 32,
      status: 'finished',
    },
    checkpoint: {
      id: 'cp-finish',
      race_id: 'race-1',
      name: 'Finish',
      checkpoint_type: 'finish',
      distance_from_start_km: 21.1,
      is_active: true,
    },
  },
  {
    id: 'r2',
    participant_id: 'p2',
    checkpoint_id: 'cp-finish',
    timestamp: '2024-06-01T11:30:00.000Z',
    local_timestamp: '2024-06-01T11:30:00.000Z',
    sync_status: 'synced',
    participant: {
      id: 'p2',
      race_id: 'race-1',
      bib_number: '12',
      first_name: 'Sam',
      last_name: 'Trail',
      gender: 'female',
      age: 28,
      status: 'finished',
    },
    checkpoint: {
      id: 'cp-finish',
      race_id: 'race-1',
      name: 'Finish',
      checkpoint_type: 'finish',
      distance_from_start_km: 21.1,
      is_active: true,
    },
  },
]

describe('raceFlowData', () => {
  it('builds distance flows from checkpoint distances for time-based races', () => {
    const flows = buildParticipantFlows(sampleRecords, undefined, 'time_based', 'metric')

    expect(flows).toHaveLength(2)
    expect(flows[0].label).toBe('#7 Alex Runner')
    expect(flows[0].points[0].value).toBe(21.1)
    expect(flows[1].points[0].value).toBe(21.1)
  })

  it('includes first and last name in flow legend labels', () => {
    const flows = buildParticipantFlows(sampleRecords, undefined, 'lap_based')

    expect(flows[0].label).toBe('#7 Alex Runner')
    expect(flows[1].label).toBe('#12 Sam Trail')
  })

  it('keeps an explicit (0, 0) origin even when the first lap is inside the cooldown window', () => {
    const earlyLap: TimingRecord[] = [
      {
        id: 'lap-1',
        participant_id: 'p1',
        checkpoint_id: 'cp-finish',
        timestamp: '2024-06-01T10:00:30.000Z',
        local_timestamp: '2024-06-01T10:00:30.000Z',
        sync_status: 'synced',
        record_type: 'rfid_lap',
        participant: {
          id: 'p1',
          race_id: 'race-1',
          bib_number: '7',
          first_name: 'Alex',
          last_name: 'Runner',
          status: 'started',
        },
        checkpoint: {
          id: 'cp-finish',
          race_id: 'race-1',
          name: 'Finish',
          checkpoint_type: 'finish',
          is_active: true,
        },
      },
    ]

    const flows = buildParticipantFlows(earlyLap, '2024-06-01T10:00:00.000Z', 'lap_based')
    expect(flows[0].points[0]).toEqual({ elapsedMinutes: 0, value: 0 })
    expect(flows[0].points[1]).toMatchObject({ elapsedMinutes: 0.5, value: 1, kind: 'rfid' })
    expect(expandSteppedLapPoints([
      { x: 0, y: 0 },
      { x: 2.791, y: 1 },
    ])).toEqual([
      { x: 0, y: 0 },
      { x: 2.791, y: 0 },
      { x: 2.791, y: 1 },
    ])
  })

  it('seeds registered racers at 0 laps at race start even without timing records', () => {
    const registered = [
      {
        id: 'p-zero',
        race_id: 'race-1',
        bib_number: '99',
        first_name: 'Zero',
        last_name: 'Start',
        status: 'registered' as const,
      },
      {
        id: 'p1',
        race_id: 'race-1',
        bib_number: '7',
        first_name: 'Alex',
        last_name: 'Runner',
        status: 'started' as const,
      },
    ]
    const flows = buildParticipantFlows(
      sampleRecords.filter((r) => r.participant?.id === 'p1'),
      '2024-06-01T10:00:00.000Z',
      'lap_based',
      'imperial',
      registered,
    )

    expect(flows).toHaveLength(2)
    const zero = flows.find((f) => f.participantId === 'p-zero')
    expect(zero?.points[0]).toEqual({ elapsedMinutes: 0, value: 0 })
    const alex = flows.find((f) => f.participantId === 'p1')
    expect(alex?.points[0]).toEqual({ elapsedMinutes: 0, value: 0 })
    expect(alex?.points.length).toBeGreaterThan(1)
  })

  it('converts distance flows to miles by default', () => {
    const flows = buildParticipantFlows(sampleRecords, undefined, 'time_based')

    expect(flows[0].points[0].value).toBeCloseTo(21.1 * KM_TO_MILES, 5)
  })

  it('builds distance flows across start, intermediate, and finish checkpoints', () => {
    const distanceRecords: TimingRecord[] = [
      {
        id: 'start-1',
        participant_id: 'p1',
        checkpoint_id: 'cp-start',
        timestamp: '2024-06-01T10:30:00.000Z',
        local_timestamp: '2024-06-01T10:30:00.000Z',
        sync_status: 'synced',
        participant: {
          id: 'p1',
          race_id: 'race-1',
          bib_number: '7',
          first_name: 'Alex',
          last_name: 'Runner',
          status: 'finished',
        },
        checkpoint: {
          id: 'cp-start',
          race_id: 'race-1',
          name: 'Start',
          checkpoint_type: 'start',
          distance_from_start_km: 0,
          is_active: true,
        },
      },
      {
        id: 'mid-1',
        participant_id: 'p1',
        checkpoint_id: 'cp-10k',
        timestamp: '2024-06-01T11:00:00.000Z',
        local_timestamp: '2024-06-01T11:00:00.000Z',
        sync_status: 'synced',
        participant: {
          id: 'p1',
          race_id: 'race-1',
          bib_number: '7',
          first_name: 'Alex',
          last_name: 'Runner',
          status: 'finished',
        },
        checkpoint: {
          id: 'cp-10k',
          race_id: 'race-1',
          name: '10K',
          checkpoint_type: 'intermediate',
          distance_from_start_km: 10,
          is_active: true,
        },
      },
      {
        id: 'finish-1',
        participant_id: 'p1',
        checkpoint_id: 'cp-finish',
        timestamp: '2024-06-01T11:45:00.000Z',
        local_timestamp: '2024-06-01T11:45:00.000Z',
        sync_status: 'synced',
        participant: {
          id: 'p1',
          race_id: 'race-1',
          bib_number: '7',
          first_name: 'Alex',
          last_name: 'Runner',
          status: 'finished',
        },
        checkpoint: {
          id: 'cp-finish',
          race_id: 'race-1',
          name: 'Finish',
          checkpoint_type: 'finish',
          distance_from_start_km: 21.1,
          is_active: true,
        },
      },
    ]

    const flows = buildParticipantFlows(distanceRecords, '2024-06-01T10:30:00.000Z', 'time_based', 'metric')

    expect(flows).toHaveLength(1)
    expect(flows[0].points.map((point) => point.value)).toEqual([0, 10, 21.1])
  })

  it('converts multi-checkpoint distance flows to miles by default', () => {
    const distanceRecords: TimingRecord[] = [
      {
        id: 'finish-1',
        participant_id: 'p1',
        checkpoint_id: 'cp-finish',
        timestamp: '2024-06-01T11:45:00.000Z',
        local_timestamp: '2024-06-01T11:45:00.000Z',
        sync_status: 'synced',
        participant: {
          id: 'p1',
          race_id: 'race-1',
          bib_number: '7',
          first_name: 'Alex',
          last_name: 'Runner',
          status: 'finished',
        },
        checkpoint: {
          id: 'cp-finish',
          race_id: 'race-1',
          name: 'Finish',
          checkpoint_type: 'finish',
          distance_from_start_km: 21.1,
          is_active: true,
        },
      },
    ]

    const flows = buildParticipantFlows(distanceRecords, '2024-06-01T10:30:00.000Z', 'time_based')

    expect(flows[0].points[0].value).toBeCloseTo(convertDistanceFromKm(21.1, 'imperial'), 5)
  })

  it('builds multi-lap flows with lap counts over time', () => {
    const lapRecords: TimingRecord[] = [
      {
        id: 'lap-1',
        participant_id: 'p1',
        checkpoint_id: 'cp-finish',
        timestamp: '2024-06-01T11:00:00.000Z',
        local_timestamp: '2024-06-01T11:00:00.000Z',
        sync_status: 'synced',
        participant: {
          id: 'p1',
          race_id: 'race-1',
          bib_number: '201',
          first_name: 'Pat',
          last_name: 'Runner',
          status: 'started',
        },
        checkpoint: {
          id: 'cp-finish',
          race_id: 'race-1',
          name: 'Loop',
          checkpoint_type: 'finish',
          is_active: true,
        },
      },
      {
        id: 'lap-2',
        participant_id: 'p2',
        checkpoint_id: 'cp-finish',
        timestamp: '2024-06-01T11:30:00.000Z',
        local_timestamp: '2024-06-01T11:30:00.000Z',
        sync_status: 'synced',
        participant: {
          id: 'p2',
          race_id: 'race-1',
          bib_number: '202',
          first_name: 'Dana',
          last_name: 'Endure',
          status: 'started',
        },
        checkpoint: {
          id: 'cp-finish',
          race_id: 'race-1',
          name: 'Loop',
          checkpoint_type: 'finish',
          is_active: true,
        },
      },
      {
        id: 'lap-3',
        participant_id: 'p1',
        checkpoint_id: 'cp-finish',
        timestamp: '2024-06-01T12:00:00.000Z',
        local_timestamp: '2024-06-01T12:00:00.000Z',
        sync_status: 'synced',
        participant: {
          id: 'p1',
          race_id: 'race-1',
          bib_number: '201',
          first_name: 'Pat',
          last_name: 'Runner',
          status: 'started',
        },
        checkpoint: {
          id: 'cp-finish',
          race_id: 'race-1',
          name: 'Loop',
          checkpoint_type: 'finish',
          is_active: true,
        },
      },
    ]

    const flows = buildParticipantFlows(lapRecords, '2024-06-01T10:00:00.000Z', 'lap_based')
    const pat = flows.find((flow) => flow.participantId === 'p1')

    expect(flows).toHaveLength(2)
    expect(pat?.points).toHaveLength(3)
    expect(pat?.points[0]).toEqual({ elapsedMinutes: 0, value: 0 })
    expect(pat?.points[1].value).toBe(1)
    expect(pat?.points[2].value).toBe(2)
  })

  it('computes race statistics from timing records', () => {
    const stats = buildRaceStatistics(sampleRecords, '2024-06-01T10:30:00.000Z', 'time_based')

    expect(stats.totalParticipants).toBe(2)
    expect(stats.finished).toBe(2)
    expect(stats.averageFinishSeconds).toBe(2700)
    expect(formatDuration(stats.averageFinishSeconds!)).toBe('45m 0s')
    expect(formatAverageResult('time_based', stats)).toBe('45m 0s')
    expect(getAverageResultLabel('lap_based')).toBe('Avg laps')
  })

  it('computes average laps for lap-based races', () => {
    const lapRecords: TimingRecord[] = [
      {
        id: 'lap-1',
        participant_id: 'p1',
        checkpoint_id: 'cp-finish',
        timestamp: '2024-06-01T11:00:00.000Z',
        local_timestamp: '2024-06-01T11:00:00.000Z',
        sync_status: 'synced',
        participant: {
          id: 'p1',
          race_id: 'race-1',
          bib_number: '201',
          first_name: 'Pat',
          last_name: 'Runner',
          status: 'started',
        },
        checkpoint: {
          id: 'cp-finish',
          race_id: 'race-1',
          name: 'Loop',
          checkpoint_type: 'finish',
          is_active: true,
        },
      },
      {
        id: 'lap-2',
        participant_id: 'p1',
        checkpoint_id: 'cp-finish',
        timestamp: '2024-06-01T12:00:00.000Z',
        local_timestamp: '2024-06-01T12:00:00.000Z',
        sync_status: 'synced',
        participant: {
          id: 'p1',
          race_id: 'race-1',
          bib_number: '201',
          first_name: 'Pat',
          last_name: 'Runner',
          status: 'started',
        },
        checkpoint: {
          id: 'cp-finish',
          race_id: 'race-1',
          name: 'Loop',
          checkpoint_type: 'finish',
          is_active: true,
        },
      },
      {
        id: 'lap-3',
        participant_id: 'p2',
        checkpoint_id: 'cp-finish',
        timestamp: '2024-06-01T11:30:00.000Z',
        local_timestamp: '2024-06-01T11:30:00.000Z',
        sync_status: 'synced',
        participant: {
          id: 'p2',
          race_id: 'race-1',
          bib_number: '202',
          first_name: 'Dana',
          last_name: 'Endure',
          status: 'started',
        },
        checkpoint: {
          id: 'cp-finish',
          race_id: 'race-1',
          name: 'Loop',
          checkpoint_type: 'finish',
          is_active: true,
        },
      },
    ]

    const stats = buildRaceStatistics(lapRecords, undefined, 'lap_based')

    expect(stats.averageLaps).toBe(1.5)
    expect(formatAverageResult('lap_based', stats)).toBe('1.5')
  })

  it('resolves race start from official start time when provided', () => {
    const startMs = resolveRaceStartMs(sampleRecords, '2024-06-01T10:30:00.000Z')

    expect(startMs).toBe(new Date('2024-06-01T10:30:00.000Z').getTime())
    expect(getCurrentElapsedMinutes(startMs!, new Date('2024-06-01T11:15:00.000Z').getTime())).toBe(45)
  })

  it('builds extrapolation points from the last tap to current time', () => {
    const flows = buildParticipantFlows(sampleRecords, '2024-06-01T10:30:00.000Z', 'time_based', 'metric')
    const extrapolation = buildExtrapolationPoint(flows[0], 45)

    expect(extrapolation).toEqual({ elapsedMinutes: 45, value: 21.1 })
  })

  it('does not project zero-lap racers forward in time', () => {
    const flows = buildParticipantFlows(
      [],
      '2024-06-01T10:00:00.000Z',
      'lap_based',
      'imperial',
      [
        {
          id: 'p-zero',
          bib_number: '99',
          first_name: 'Zero',
          last_name: 'Start',
          status: 'registered',
        },
      ],
    )

    expect(buildExtrapolationPoint(flows[0], 12)).toBeNull()
  })

  it('coalesces near-simultaneous finish taps into one lap step', () => {
    const closeTaps: TimingRecord[] = [
      {
        id: 'lap-1',
        participant_id: 'p1',
        checkpoint_id: 'cp-finish',
        timestamp: '2024-06-01T10:05:00.000Z',
        local_timestamp: '2024-06-01T10:05:00.000Z',
        sync_status: 'synced',
        record_type: 'rfid_lap',
        participant: {
          id: 'p1',
          race_id: 'race-1',
          bib_number: '7',
          first_name: 'Alex',
          last_name: 'Runner',
          status: 'started',
        },
        checkpoint: {
          id: 'cp-finish',
          race_id: 'race-1',
          name: 'Finish',
          checkpoint_type: 'finish',
          is_active: true,
        },
      },
      {
        id: 'dup',
        participant_id: 'p1',
        checkpoint_id: 'cp-finish',
        timestamp: '2024-06-01T10:05:30.000Z',
        local_timestamp: '2024-06-01T10:05:30.000Z',
        sync_status: 'synced',
        record_type: 'rfid_lap',
        participant: {
          id: 'p1',
          race_id: 'race-1',
          bib_number: '7',
          first_name: 'Alex',
          last_name: 'Runner',
          status: 'started',
        },
        checkpoint: {
          id: 'cp-finish',
          race_id: 'race-1',
          name: 'Finish',
          checkpoint_type: 'finish',
          is_active: true,
        },
      },
    ]

    const flows = buildParticipantFlows(closeTaps, '2024-06-01T10:00:00.000Z', 'lap_based')
    const alex = flows.find((flow) => flow.participantId === 'p1')
    const afterStart = alex?.points.filter((point) => point.value > 0) ?? []

    expect(afterStart).toHaveLength(1)
    expect(afterStart[0].value).toBe(2)
  })

  it('plots karaoke bonus laps as distinct music-note points', () => {
    const withKaraoke: TimingRecord[] = [
      {
        id: 'lap-1',
        participant_id: 'p1',
        checkpoint_id: 'cp-finish',
        timestamp: '2024-06-01T10:05:00.000Z',
        local_timestamp: '2024-06-01T10:05:00.000Z',
        sync_status: 'synced',
        record_type: 'rfid_lap',
        participant: {
          id: 'p1',
          race_id: 'race-1',
          bib_number: '7',
          first_name: 'Alex',
          last_name: 'Runner',
          status: 'started',
        },
        checkpoint: {
          id: 'cp-finish',
          race_id: 'race-1',
          name: 'Finish',
          checkpoint_type: 'finish',
          is_active: true,
        },
      },
      {
        id: 'bonus',
        participant_id: 'p1',
        checkpoint_id: 'cp-finish',
        timestamp: '2024-06-01T10:05:20.000Z',
        local_timestamp: '2024-06-01T10:05:20.000Z',
        sync_status: 'synced',
        record_type: 'karaoke_bonus',
        participant: {
          id: 'p1',
          race_id: 'race-1',
          bib_number: '7',
          first_name: 'Alex',
          last_name: 'Runner',
          status: 'started',
        },
        checkpoint: {
          id: 'cp-finish',
          race_id: 'race-1',
          name: 'Finish',
          checkpoint_type: 'finish',
          is_active: true,
        },
      },
    ]

    const flows = buildParticipantFlows(withKaraoke, '2024-06-01T10:00:00.000Z', 'lap_based')
    const alex = flows.find((flow) => flow.participantId === 'p1')
    const afterStart = alex?.points.filter((point) => point.value > 0) ?? []

    expect(afterStart).toHaveLength(2)
    expect(afterStart[0]).toMatchObject({ value: 1, kind: 'rfid' })
    expect(afterStart[1]).toMatchObject({ value: 2, kind: 'karaoke' })
  })

  it('expands lap points into axis-aligned steps that never move backwards', () => {
    const stepped = expandSteppedLapPoints([
      { x: 0, y: 0 },
      { x: 2.275, y: 1 },
      { x: 4.5, y: 2 },
    ])

    expect(stepped).toEqual([
      { x: 0, y: 0 },
      { x: 2.275, y: 0 },
      { x: 2.275, y: 1 },
      { x: 4.5, y: 1 },
      { x: 4.5, y: 2 },
    ])
    for (let index = 1; index < stepped.length; index += 1) {
      expect(stepped[index].x).toBeGreaterThanOrEqual(stepped[index - 1].x)
    }
  })

  it('builds participant tooltip details from flow data', () => {
    const flows = buildParticipantFlows(sampleRecords, '2024-06-01T10:30:00.000Z', 'time_based', 'metric')
    const tooltip = buildParticipantFlowTooltip(flows[0], 'time_based', 'metric')

    expect(tooltip.fullName).toBe('Alex Runner')
    expect(tooltip.bibNumber).toBe('7')
    expect(tooltip.status).toBe('finished')
    expect(tooltip.progress).toContain('21.1 km at')
  })

  it('assigns maximally spaced hues for selected participants', () => {
    const colors = new Map<string, string>()
    for (const participantId of ['p-alpha', 'p-beta']) {
      colors.set(participantId, resolveCategoryColor(participantId))
    }

    expect(colors.get('p-alpha')).not.toBe(colors.get('p-beta'))
    expect(['#1a3f3d', '#2f6b5a', '#9b654e', '#a1b383', '#6b7a76']).toContain(colors.get('p-alpha'))
    expect(['#1a3f3d', '#2f6b5a', '#9b654e', '#a1b383', '#6b7a76']).toContain(colors.get('p-beta'))
  })

  it('derives gender and age group filter keys from participant data', () => {
    expect(getParticipantGenderKey('female')).toBe('female')
    expect(getParticipantGenderKey(undefined)).toBe('unknown')
    expect(getParticipantAgeGroupKey(32)).toBe('30-34')
    expect(getParticipantAgeGroupKey(17)).toBe('under-20')
    expect(getParticipantAgeGroupLabel('30-34')).toBe('30–34')
  })

  it('clamps wall-clock elapsed to duration', () => {
    expect(clampElapsedToDuration(2559, 720)).toBe(720)
    expect(clampElapsedToDuration(100, 720)).toBe(100)
    expect(clampElapsedToDuration(100, undefined)).toBe(100)
  })

  it('uses duration as axis max when present', () => {
    expect(resolveRaceFlowAxisMaxMinutes(720, 12, 2559)).toBe(720)
    expect(resolveRaceFlowAxisMaxMinutes(360, 5, 40)).toBe(360)
  })

  it('falls back to recorded/live max when duration missing', () => {
    expect(resolveRaceFlowAxisMaxMinutes(undefined, 45, 50)).toBe(50)
    expect(resolveRaceFlowAxisMaxMinutes(0, 45, null)).toBe(45)
  })

  it('resolves chart x-axis max with optional padding', () => {
    expect(resolveRaceFlowXAxisMax(720, 12, 2559, true)).toBe(720)
    expect(resolveRaceFlowXAxisMax(undefined, 100, 100, true)).toBe(105)
    expect(resolveRaceFlowXAxisMax(undefined, 100, 100, false)).toBeUndefined()
  })

  it('excludes timing records before race start so elapsed never goes negative', () => {
    const earlyAndOnTime: TimingRecord[] = [
      {
        id: 'early',
        participant_id: 'p1',
        checkpoint_id: 'cp-finish',
        timestamp: '2024-05-31T12:00:00.000Z',
        local_timestamp: '2024-05-31T12:00:00.000Z',
        sync_status: 'synced',
        participant: {
          id: 'p1',
          race_id: 'race-1',
          bib_number: '7',
          first_name: 'Alex',
          last_name: 'Runner',
          status: 'started',
        },
        checkpoint: {
          id: 'cp-finish',
          race_id: 'race-1',
          name: 'Finish',
          checkpoint_type: 'finish',
          is_active: true,
        },
      },
      {
        id: 'on-time',
        participant_id: 'p1',
        checkpoint_id: 'cp-finish',
        timestamp: '2024-06-01T11:00:00.000Z',
        local_timestamp: '2024-06-01T11:00:00.000Z',
        sync_status: 'synced',
        participant: {
          id: 'p1',
          race_id: 'race-1',
          bib_number: '7',
          first_name: 'Alex',
          last_name: 'Runner',
          status: 'started',
        },
        checkpoint: {
          id: 'cp-finish',
          race_id: 'race-1',
          name: 'Finish',
          checkpoint_type: 'finish',
          is_active: true,
        },
      },
    ]

    const flows = buildParticipantFlows(
      earlyAndOnTime,
      '2024-06-01T10:30:00.000Z',
      'lap_based',
    )

    expect(flows).toHaveLength(1)
    expect(flows[0].points).toHaveLength(2)
    expect(flows[0].points[0]).toEqual({ elapsedMinutes: 0, value: 0 })
    expect(flows[0].points[1].elapsedMinutes).toBe(30)
    expect(flows[0].points[1].value).toBe(1)
    expect(flows[0].points.every((point) => point.elapsedMinutes >= 0)).toBe(true)
  })

  it('keeps each participant flow monotonic in elapsed time', () => {
    const flows = buildParticipantFlows(sampleRecords, '2024-06-01T10:30:00.000Z', 'lap_based')

    for (const flow of flows) {
      for (let index = 1; index < flow.points.length; index += 1) {
        expect(flow.points[index].elapsedMinutes).toBeGreaterThanOrEqual(
          flow.points[index - 1].elapsedMinutes,
        )
      }
    }
  })
})

describe('RaceFlowChart.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    setupPinia()
  })

  it('does not hardcode legacy signal or blue chrome', () => {
    const src = readFileSync(join(process.cwd(), 'src/components/RaceFlowChart.vue'), 'utf8')
    expect(src).not.toMatch(/#e74c3c|#1a5276/i)
    const style = src.split('<style')[1] ?? ''
    expect(style).not.toMatch(/#2c3e50|#3498db|#2980b9|#1f4f82/i)
    expect(src).toMatch(/#c45c38|SIGNAL_COLOR/)
  })

  it('scales legend typography with live display scale', () => {
    const src = readFileSync(join(process.cwd(), 'src/components/RaceFlowChart.vue'), 'utf8')
    const style = src.split('<style scoped>')[1]?.split('</style>')[0] ?? ''
    expect(style).toMatch(/\.legend-panel[\s\S]*--live-display-scale/)
  })

  it('hosts the canvas in a dedicated relatively positioned container', () => {
    const src = readFileSync(join(process.cwd(), 'src/components/RaceFlowChart.vue'), 'utf8')
    const style = src.split('<style scoped>')[1]?.split('</style>')[0] ?? ''

    // Chart.js measures the parent for bitmap size; legend must not share that parent,
    // and CSS must not force canvas width/height (causes squished axis/tooltip text).
    expect(src).toMatch(
      /class="chart-canvas-host"[\s\S]*data-testid="race-flow-canvas"[\s\S]*class="legend-panel"/,
    )
    expect(style).toMatch(
      /\.chart-canvas-host\s*\{[^}]*position:\s*relative;[^}]*height:\s*320px/s,
    )
    expect(style).not.toMatch(/canvas\s*\{[^}]*width:\s*100%\s*!important/s)
    expect(style).not.toMatch(/canvas\s*\{[^}]*height:\s*\d+px\s*!important/s)
  })

  it('loads live timing data and renders chart canvas', async () => {
    ;(timingApi.getLive as Mock).mockResolvedValue({
      data: { race_id: 'race-1', records: sampleRecords },
    })

    const wrapper = mount(RaceFlowChart, {
      props: { raceId: 'race-1' },
    })
    await flushPromises()

    expect(timingApi.getLive).toHaveBeenCalledWith('race-1')
    expect(wrapper.find('[data-testid="race-flow-canvas"]').exists()).toBe(true)
  })

  it('assigns distinct colors to each participant line', async () => {
    ;(timingApi.getLive as Mock).mockResolvedValue({
      data: { race_id: 'race-1', records: sampleRecords },
    })

    mount(RaceFlowChart, {
      props: { raceId: 'race-1' },
    })
    await flushPromises()

    const chartConfig = (Chart as unknown as Mock).mock.calls.at(-1)?.[1] as {
      data: { datasets: Array<{ borderColor: string }> }
    }
    const colors = chartConfig.data.datasets.map((dataset) => dataset.borderColor)

    expect(colors).toHaveLength(2)
    expect(colors[0]).not.toBe(colors[1])
    expect(colors).toEqual([
      resolveCategoryColor('p1'),
      resolveCategoryColor('p2'),
    ])
  })

  it('draws current time line and dotted extrapolations for active races', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2024-06-01T12:00:00.000Z'))

    ;(timingApi.getLive as Mock).mockResolvedValue({
      data: { race_id: 'race-1', records: sampleRecords },
    })

    mount(RaceFlowChart, {
      props: {
        raceId: 'race-1',
        raceStatus: 'active',
        raceStartTime: '2024-06-01T10:30:00.000Z',
        raceType: 'time_based',
      },
    })
    await flushPromises()

    const chartConfig = (Chart as unknown as Mock).mock.calls.at(-1)?.[1] as {
      data: {
        datasets: Array<{
          hasExtrapolation: boolean
          segment?: { borderDash: (ctx: { p1DataIndex: number }) => number[] | undefined }
        }>
      }
      options: {
        plugins: { currentTimeLine: { xMinutes: number | null } }
        scales: { y: { title: { text: string } } }
      }
    }

    expect(chartConfig.options.plugins.currentTimeLine.xMinutes).toBe(90)
    expect(chartConfig.options.scales.y.title.text).toBe('Distance (mi)')
    expect(chartConfig.data.datasets.every((dataset) => dataset.hasExtrapolation)).toBe(true)
    expect(
      chartConfig.data.datasets[0].segment?.borderDash({ p1DataIndex: 1 }),
    ).toEqual([6, 6])

    vi.useRealTimers()
  })

  it('caps x-axis and extrapolations to duration_minutes for active races', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2024-06-03T12:00:00.000Z'))

    ;(timingApi.getLive as Mock).mockResolvedValue({
      data: { race_id: 'race-1', records: sampleRecords },
    })

    mount(RaceFlowChart, {
      props: {
        raceId: 'race-1',
        raceStatus: 'active',
        raceStartTime: '2024-06-01T10:30:00.000Z',
        raceType: 'lap_based',
        durationMinutes: 720,
      },
    })
    await flushPromises()

    const chartConfig = (Chart as unknown as Mock).mock.calls.at(-1)?.[1] as {
      options: {
        scales: { x: { max?: number } }
        plugins: { currentTimeLine: { xMinutes: number | null } }
      }
      data: { datasets: Array<{ data: Array<{ x: number }> }> }
    }

    expect(chartConfig.options.scales.x.max).toBe(720)
    expect(chartConfig.options.plugins.currentTimeLine.xMinutes).toBe(720)

    const xs = chartConfig.data.datasets.flatMap((d) => d.data.map((p) => p.x))
    expect(Math.max(...xs)).toBeLessThanOrEqual(720)

    vi.useRealTimers()
  })

  it('draws straight stepped lap lines from x=0 with readable axis type', async () => {
    ;(timingApi.getLive as Mock).mockResolvedValue({
      data: { race_id: 'race-1', records: sampleRecords },
    })

    mount(RaceFlowChart, {
      props: {
        raceId: 'race-1',
        raceStartTime: '2024-06-01T10:30:00.000Z',
        raceType: 'lap_based',
        durationMinutes: 720,
      },
    })
    await flushPromises()

    const chartConfig = (Chart as unknown as Mock).mock.calls.at(-1)?.[1] as {
      options: {
        scales: {
          x: {
            min?: number
            max?: number
            ticks?: { font?: { size?: number }; color?: string }
            title?: { font?: { size?: number }; color?: string }
          }
          y: {
            ticks?: { font?: { size?: number }; color?: string }
            title?: { font?: { size?: number }; color?: string }
          }
        }
        plugins: {
          title?: { display?: boolean }
        }
      }
      data: {
        datasets: Array<{
          tension: number
          stepped?: boolean | string
          data: Array<{ x: number; y: number }>
        }>
      }
    }

    expect(chartConfig.options.scales.x.min).toBe(0)
    expect(chartConfig.data.datasets.every((dataset) => dataset.tension === 0)).toBe(true)
    expect(chartConfig.data.datasets.every((dataset) => !dataset.stepped)).toBe(true)
    expect(
      chartConfig.data.datasets.every((dataset) =>
        dataset.data.every((point) => point.x >= 0),
      ),
    ).toBe(true)
    for (const dataset of chartConfig.data.datasets) {
      for (let index = 1; index < dataset.data.length; index += 1) {
        expect(dataset.data[index].x).toBeGreaterThanOrEqual(dataset.data[index - 1].x)
      }
    }
    expect(chartConfig.options.scales.x.ticks?.font?.size).toBe(10)
    expect(chartConfig.options.scales.y.ticks?.font?.size).toBe(10)
    expect(chartConfig.options.plugins.title?.display).toBe(false)
    expect(chartConfig.options.scales.x.ticks?.color).toBe('#1a3f3d')
    expect(chartConfig.options.scales.y.title?.color).toBe('#1a3f3d')
  })

  it('renders karaoke lap points with a music-note marker', async () => {
    const karaokeRecords: TimingRecord[] = [
      {
        id: 'lap-1',
        participant_id: 'p1',
        checkpoint_id: 'cp-finish',
        timestamp: '2024-06-01T10:35:00.000Z',
        local_timestamp: '2024-06-01T10:35:00.000Z',
        sync_status: 'synced',
        record_type: 'rfid_lap',
        participant: {
          id: 'p1',
          race_id: 'race-1',
          bib_number: '7',
          first_name: 'Alex',
          last_name: 'Runner',
          status: 'started',
        },
        checkpoint: {
          id: 'cp-finish',
          race_id: 'race-1',
          name: 'Finish',
          checkpoint_type: 'finish',
          is_active: true,
        },
      },
      {
        id: 'bonus',
        participant_id: 'p1',
        checkpoint_id: 'cp-finish',
        timestamp: '2024-06-01T10:35:20.000Z',
        local_timestamp: '2024-06-01T10:35:20.000Z',
        sync_status: 'synced',
        record_type: 'karaoke_bonus',
        participant: {
          id: 'p1',
          race_id: 'race-1',
          bib_number: '7',
          first_name: 'Alex',
          last_name: 'Runner',
          status: 'started',
        },
        checkpoint: {
          id: 'cp-finish',
          race_id: 'race-1',
          name: 'Finish',
          checkpoint_type: 'finish',
          is_active: true,
        },
      },
    ]

    ;(timingApi.getLive as Mock).mockResolvedValue({
      data: { race_id: 'race-1', records: karaokeRecords },
    })

    mount(RaceFlowChart, {
      props: {
        raceId: 'race-1',
        raceStartTime: '2024-06-01T10:30:00.000Z',
        raceType: 'lap_based',
        durationMinutes: 720,
      },
    })
    await flushPromises()

    const chartConfig = (Chart as unknown as Mock).mock.calls.at(-1)?.[1] as {
      data: {
        datasets: Array<{
          pointStyle?: Array<string | HTMLCanvasElement | HTMLImageElement | false>
          data: Array<{ x: number; y: number; kind?: string }>
        }>
      }
    }

    const styles = chartConfig.data.datasets[0].pointStyle ?? []
    const karaokeStyles = styles.filter(
      (style) => typeof style === 'object' && style !== null && 'src' in style,
    )
    expect(karaokeStyles.length).toBeGreaterThanOrEqual(1)
  })

  it('shows empty state when no finish records exist', async () => {
    ;(timingApi.getLive as Mock).mockResolvedValue({
      data: { race_id: 'race-1', records: [] },
    })

    const wrapper = mount(RaceFlowChart, {
      props: { raceId: 'race-1' },
    })
    await flushPromises()

    expect(wrapper.find('[data-testid="race-flow-empty"]').exists()).toBe(true)
  })

  it('shows error message when API request fails', async () => {
    ;(timingApi.getLive as Mock).mockRejectedValue(new Error('Network error'))

    const wrapper = mount(RaceFlowChart, {
      props: { raceId: 'race-1' },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Network error')
  })

  it('renders custom legend with search, status filters, and select all', async () => {
    ;(timingApi.getLive as Mock).mockResolvedValue({
      data: { race_id: 'race-1', records: sampleRecords },
    })

    const wrapper = mount(RaceFlowChart, {
      props: { raceId: 'race-1' },
    })
    await flushPromises()

    expect(wrapper.find('[data-testid="race-flow-legend"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="race-flow-legend-search"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="race-flow-filters"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="race-flow-status-filters"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="race-flow-gender-filters"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="race-flow-age-group-filters"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="race-flow-select-all"]').exists()).toBe(true)
    expect(wrapper.find('.filter-dropdown-trigger').exists()).toBe(true)
    expect(wrapper.text()).toContain('Gender')
    expect(wrapper.text()).toContain('Age group')
    expect(wrapper.text()).toContain('#7 Alex')
    expect(wrapper.text()).toContain('#12 Sam')
  })

  it('shows All for status when every available status is selected', async () => {
    ;(timingApi.getLive as Mock).mockResolvedValue({
      data: { race_id: 'race-1', records: sampleRecords },
    })

    const wrapper = mount(RaceFlowChart, {
      props: { raceId: 'race-1' },
    })
    await flushPromises()

    expect(
      wrapper.find('[data-testid="race-flow-status-filters"] .filter-dropdown-value').text(),
    ).toBe('All')
  })

  it('filters legend items by gender', async () => {
    ;(timingApi.getLive as Mock).mockResolvedValue({
      data: { race_id: 'race-1', records: sampleRecords },
    })

    const wrapper = mount(RaceFlowChart, {
      props: { raceId: 'race-1' },
    })
    await flushPromises()

    const genderDropdown = wrapper.find('[data-testid="race-flow-gender-filters"]')
    await genderDropdown.find('.filter-dropdown-trigger').trigger('click')
    const femaleOption = genderDropdown
      .findAll('.filter-dropdown-option')
      .find((option) => option.text().includes('Female'))
    await femaleOption?.trigger('click')
    await flushPromises()
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('#7 Alex')
    expect(wrapper.text()).not.toContain('#12 Sam')

    const chartConfig = (Chart as unknown as Mock).mock.calls.at(-1)?.[1] as {
      data: { datasets: Array<{ label: string }> }
    }
    expect(chartConfig.data.datasets).toHaveLength(1)
    expect(chartConfig.data.datasets[0].label).toContain('Alex')
  })

  it('filters legend items by search query', async () => {
    ;(timingApi.getLive as Mock).mockResolvedValue({
      data: { race_id: 'race-1', records: sampleRecords },
    })

    const wrapper = mount(RaceFlowChart, {
      props: { raceId: 'race-1' },
    })
    await flushPromises()

    await wrapper.find('[data-testid="race-flow-legend-search"]').setValue('Alex')

    expect(wrapper.text()).toContain('#7 Alex')
    expect(wrapper.text()).not.toContain('#12 Sam')
  })

  it('select all toggles visibility for filtered participants', async () => {
    ;(timingApi.getLive as Mock).mockResolvedValue({
      data: { race_id: 'race-1', records: sampleRecords },
    })

    const wrapper = mount(RaceFlowChart, {
      props: { raceId: 'race-1' },
    })
    await flushPromises()

    const selectAllButton = wrapper.find('[data-testid="race-flow-select-all"]')
    expect(selectAllButton.text()).toBe('Deselect all')

    await selectAllButton.trigger('click')
    await flushPromises()
    await wrapper.vm.$nextTick()

    const chartConfig = (Chart as unknown as Mock).mock.calls.at(-1)?.[1] as {
      data: { datasets: unknown[] }
    }
    expect(chartConfig.data.datasets).toHaveLength(0)
    expect(selectAllButton.text()).toBe('Select all')
  })

  it('shows participant tooltip when hovering a legend item', async () => {
    ;(timingApi.getLive as Mock).mockResolvedValue({
      data: { race_id: 'race-1', records: sampleRecords },
    })

    const wrapper = mount(RaceFlowChart, {
      props: { raceId: 'race-1' },
      attachTo: document.body,
    })
    await flushPromises()

    const legendItem = wrapper.find('.legend-item')
    await legendItem.trigger('mouseenter', { clientX: 100, clientY: 200 })
    await wrapper.vm.$nextTick()

    const tooltip = wrapper.find('[data-testid="race-flow-legend-tooltip"]')
    expect(tooltip.exists()).toBe(true)
    expect(tooltip.text()).toContain('Alex Runner')
    expect(tooltip.text()).toContain('Bib 7')
    expect(tooltip.text()).toContain('FINISHED')

    await legendItem.trigger('mouseleave')
    await wrapper.vm.$nextTick()

    expect(wrapper.find('[data-testid="race-flow-legend-tooltip"]').exists()).toBe(false)
    wrapper.unmount()
  })

  it('highlights participant line when hovering a legend item', async () => {
    ;(timingApi.getLive as Mock).mockResolvedValue({
      data: { race_id: 'race-1', records: sampleRecords },
    })

    const wrapper = mount(RaceFlowChart, {
      props: { raceId: 'race-1' },
    })
    await flushPromises()

    const chartInstance = (Chart as unknown as Mock).mock.results.at(-1)?.value as {
      update: Mock
      data: { datasets: Array<{ participantId?: string; borderWidth: number }> }
    }
    const initialWidths = chartInstance.data.datasets.map((dataset) => dataset.borderWidth)
    expect(initialWidths.every((width) => width === 2)).toBe(true)

    await wrapper.find('.legend-item').trigger('mouseenter', { clientX: 100, clientY: 200 })
    await flushPromises()

    expect(wrapper.vm.hoveredParticipantId).toBe('p1')
    expect(chartInstance.update).toHaveBeenCalled()

    const highlightedDataset = chartInstance.data.datasets.find(
      (dataset) => dataset.participantId === 'p1',
    )
    const dimmedDataset = chartInstance.data.datasets.find(
      (dataset) => dataset.participantId === 'p2',
    )
    expect(highlightedDataset?.borderWidth).toBe(4)
    expect(dimmedDataset?.borderWidth).toBe(1)
  })

  describe('sticky highlight selection', () => {
    async function mountWithData(props: Record<string, unknown> = {}) {
      ;(timingApi.getLive as Mock).mockResolvedValue({
        data: { race_id: 'race-1', records: sampleRecords },
      })
      const wrapper = mount(RaceFlowChart, {
        props: { raceId: 'race-1', ...props },
      })
      await flushPromises()
      return wrapper
    }

    function lastChartInstance() {
      return (Chart as unknown as Mock).mock.results.at(-1)?.value as {
        update: Mock
        data: { datasets: Array<{ participantId?: string; borderWidth: number }> }
        options: { onClick?: Function; onHover?: Function }
        canvas: { style: { cursor: string } }
        getElementsAtEventForMode: Mock
      }
    }

    it('emits sticky select on plot click and keeps highlight after hover clears', async () => {
      const wrapper = await mountWithData()
      const chart = lastChartInstance()
      chart.getElementsAtEventForMode.mockReturnValue([{ datasetIndex: 0 }])

      chart.options.onClick?.(
        { native: new MouseEvent('click') },
        [{ datasetIndex: 0 }],
        chart,
      )
      await flushPromises()

      expect(wrapper.emitted('update:highlightParticipantId')?.at(-1)).toEqual(['p1'])

      await wrapper.setProps({ highlightParticipantId: 'p1' })
      await flushPromises()

      chart.options.onHover?.(
        { native: new MouseEvent('mousemove') },
        [],
        chart,
      )
      await flushPromises()

      const highlighted = chart.data.datasets.find((d) => d.participantId === 'p1')
      expect(highlighted?.borderWidth).toBe(4)
    })

    it('does not let hover change styling while sticky is active', async () => {
      const wrapper = await mountWithData({ highlightParticipantId: 'p1' })
      await flushPromises()
      const chart = lastChartInstance()

      chart.options.onHover?.(
        { native: new MouseEvent('mousemove') },
        [{ datasetIndex: 1 }],
        chart,
      )
      await flushPromises()

      expect(wrapper.vm.hoveredParticipantId).toBeNull()
      const p1 = chart.data.datasets.find((d) => d.participantId === 'p1')
      const p2 = chart.data.datasets.find((d) => d.participantId === 'p2')
      expect(p1?.borderWidth).toBe(4)
      expect(p2?.borderWidth).toBe(1)
    })

    it('emits clear when clicking empty plot area', async () => {
      const wrapper = await mountWithData({ highlightParticipantId: 'p1' })
      const chart = lastChartInstance()
      chart.getElementsAtEventForMode.mockReturnValue([])

      chart.options.onClick?.(
        { native: new MouseEvent('click') },
        [],
        chart,
      )
      await flushPromises()

      expect(wrapper.emitted('update:highlightParticipantId')?.at(-1)).toEqual([undefined])
    })

    it('emits clear when re-clicking the selected line', async () => {
      const wrapper = await mountWithData({ highlightParticipantId: 'p1' })
      const chart = lastChartInstance()
      chart.getElementsAtEventForMode.mockReturnValue([{ datasetIndex: 0 }])

      chart.options.onClick?.(
        { native: new MouseEvent('click') },
        [{ datasetIndex: 0 }],
        chart,
      )
      await flushPromises()

      expect(wrapper.emitted('update:highlightParticipantId')?.at(-1)).toEqual([undefined])
    })

    it('emits clear on document click outside all race-flow charts', async () => {
      const wrapper = await mountWithData({ highlightParticipantId: 'p1' })
      document.body.dispatchEvent(new MouseEvent('click', { bubbles: true }))
      await flushPromises()

      expect(wrapper.emitted('update:highlightParticipantId')?.at(-1)).toEqual([undefined])
      wrapper.unmount()
    })

    it('sets pointer cursor when hovering a line with no sticky selection', async () => {
      await mountWithData()
      const chart = lastChartInstance()

      chart.options.onHover?.(
        { native: new MouseEvent('mousemove') },
        [{ datasetIndex: 0 }],
        chart,
      )
      expect(chart.canvas.style.cursor).toBe('pointer')

      chart.options.onHover?.(
        { native: new MouseEvent('mousemove') },
        [],
        chart,
      )
      expect(chart.canvas.style.cursor).toBe('default')
    })

    it('legend name button sticky-selects; checkbox only toggles visibility', async () => {
      const wrapper = await mountWithData()
      const selectBtn = wrapper.find('[data-testid="race-flow-legend-select"]')
      expect(selectBtn.exists()).toBe(true)

      await selectBtn.trigger('click')
      await flushPromises()
      expect(wrapper.emitted('update:highlightParticipantId')?.at(-1)).toEqual(['p1'])

      const checkbox = wrapper.find('.legend-item input[type="checkbox"]')
      const before = wrapper.vm.visibleParticipantIds.has('p1')
      await checkbox.setValue(false)
      await flushPromises()
      expect(wrapper.vm.visibleParticipantIds.has('p1')).toBe(!before)
      // checkbox click must not emit a second select
      const selectEmits = wrapper.emitted('update:highlightParticipantId') ?? []
      expect(selectEmits.filter((e) => e[0] === 'p1').length).toBe(1)
    })
  })

  describe('isLegendBusy', () => {
    async function mountChartWithData() {
      ;(timingApi.getLive as Mock).mockResolvedValue({
        data: { race_id: 'race-1', records: sampleRecords },
      })

      const wrapper = mount(RaceFlowChart, {
        props: { raceId: 'race-1' },
      })
      await flushPromises()
      return wrapper
    }

    it('is true when search query is set', async () => {
      const wrapper = await mountChartWithData()

      expect(wrapper.vm.isLegendBusy).toBe(false)

      await wrapper.find('[data-testid="race-flow-legend-search"]').setValue('Alex')

      expect(wrapper.vm.isLegendBusy).toBe(true)
    })

    it('is true when a status filter is deselected', async () => {
      const wrapper = await mountChartWithData()

      await wrapper.find('[data-testid="race-flow-legend-search"]').setValue('')
      expect(wrapper.vm.isLegendBusy).toBe(false)

      const statusDropdown = wrapper.find('[data-testid="race-flow-status-filters"]')
      await statusDropdown.find('.filter-dropdown-trigger').trigger('click')
      const finishedOption = statusDropdown
        .findAll('.filter-dropdown-option')
        .find((option) => option.text().includes('FINISHED'))
      await finishedOption?.trigger('click')
      await flushPromises()
      await wrapper.vm.$nextTick()

      expect(wrapper.vm.isLegendBusy).toBe(true)
    })

    it('is true when legend-items is scrolled', async () => {
      const wrapper = await mountChartWithData()

      await wrapper.find('[data-testid="race-flow-legend-search"]').setValue('')
      expect(wrapper.vm.isLegendBusy).toBe(false)

      const legendItems = wrapper.find('.legend-items')
      ;(legendItems.element as HTMLElement).scrollTop = 10
      await legendItems.trigger('scroll')
      await wrapper.vm.$nextTick()

      expect(wrapper.vm.isLegendBusy).toBe(true)
    })
  })
})
