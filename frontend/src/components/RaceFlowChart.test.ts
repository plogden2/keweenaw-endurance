import { describe, it, expect, vi, beforeEach, type Mock } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { Chart } from 'chart.js'
import RaceFlowChart from './RaceFlowChart.vue'
import { timingApi } from '@/services/api'
import { assignContrastFlowColors, buildParticipantFlowTooltip, buildParticipantFlows, buildRaceStatistics, buildExtrapolationPoint, formatAverageResult, formatDuration, getAverageResultLabel, getCurrentElapsedMinutes, resolveRaceStartMs } from '@/utils/raceFlowData'
import { convertDistanceFromKm, KM_TO_MILES } from '@/utils/units'
import { setupPinia } from '@/test/helpers'
import type { TimingRecord } from '@/types/models'

vi.mock('chart.js', () => ({
  Chart: Object.assign(
    vi.fn(() => ({ destroy: vi.fn() })),
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
    expect(flows[0].label).toContain('#7')
    expect(flows[0].points[0].value).toBe(21.1)
    expect(flows[1].points[0].value).toBe(21.1)
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

    const flows = buildParticipantFlows(lapRecords, undefined, 'lap_based')
    const pat = flows.find((flow) => flow.participantId === 'p1')

    expect(flows).toHaveLength(2)
    expect(pat?.points).toHaveLength(2)
    expect(pat?.points[0].value).toBe(1)
    expect(pat?.points[1].value).toBe(2)
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

  it('builds participant tooltip details from flow data', () => {
    const flows = buildParticipantFlows(sampleRecords, '2024-06-01T10:30:00.000Z', 'time_based', 'metric')
    const tooltip = buildParticipantFlowTooltip(flows[0], 'time_based', 'metric')

    expect(tooltip.fullName).toBe('Alex Runner')
    expect(tooltip.bibNumber).toBe('7')
    expect(tooltip.status).toBe('finished')
    expect(tooltip.progress).toContain('21.1 km at')
  })

  it('assigns maximally spaced hues for selected participants', () => {
    const colors = assignContrastFlowColors(['p-alpha', 'p-beta'])

    expect(colors.get('p-alpha')).toBe('hsl(0, 70%, 45%)')
    expect(colors.get('p-beta')).toBe('hsl(180, 70%, 45%)')
    expect(colors.get('p-alpha')).not.toBe(colors.get('p-beta'))
  })
})

describe('RaceFlowChart.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    setupPinia()
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
    expect(colors).toEqual(['hsl(0, 70%, 45%)', 'hsl(180, 70%, 45%)'])
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
    expect(wrapper.find('[data-testid="race-flow-status-filters"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="race-flow-select-all"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('#7 Alex')
    expect(wrapper.text()).toContain('#12 Sam')
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
})
