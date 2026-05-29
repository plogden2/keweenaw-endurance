import { describe, it, expect, vi, beforeEach, type Mock } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import RaceFlowChart from './RaceFlowChart.vue'
import { timingApi } from '@/services/api'
import { buildParticipantFlows, buildRaceStatistics } from '@/utils/raceFlowData'
import type { TimingRecord } from '@/types/models'

vi.mock('chart.js', () => ({
  Chart: Object.assign(vi.fn(), { register: vi.fn() }),
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
      is_active: true,
    },
  },
]

describe('raceFlowData', () => {
  it('builds participant flows from finish records', () => {
    const flows = buildParticipantFlows(sampleRecords)

    expect(flows).toHaveLength(2)
    expect(flows[0].label).toContain('#7')
    expect(flows[0].points[0].position).toBe(1)
    expect(flows[1].points[0].position).toBe(2)
  })

  it('computes race statistics from timing records', () => {
    const stats = buildRaceStatistics(sampleRecords)

    expect(stats.totalParticipants).toBe(2)
    expect(stats.finished).toBe(2)
    expect(stats.averageFinishMinutes).not.toBeNull()
  })
})

describe('RaceFlowChart.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
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
})
