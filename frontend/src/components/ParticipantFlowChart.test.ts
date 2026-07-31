import { describe, it, expect, vi, beforeEach, type Mock } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { Chart } from 'chart.js'
import ParticipantFlowChart from './ParticipantFlowChart.vue'
import { timingApi } from '@/services/api'
import { setupPinia } from '@/test/helpers'
import type { TimingRecord } from '@/types/models'

vi.mock('chart.js', () => ({
  Chart: Object.assign(
    vi.fn((_canvas, config) => {
      const instance = {
        destroy: vi.fn(),
        update: vi.fn(),
        resize: vi.fn(),
        data: config?.data ?? { datasets: [] },
        options: config?.options ?? {},
        canvas: { style: { cursor: 'default' } as { cursor: string } },
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
      gender: 'male',
      age: 32,
      status: 'started',
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

describe('ParticipantFlowChart', () => {
  beforeEach(() => {
    setupPinia()
    vi.clearAllMocks()
  })

  it('updates live progress in place instead of destroying Chart.js every tick', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2024-06-01T12:00:00.000Z'))

    ;(timingApi.getLive as Mock).mockResolvedValue({
      data: { race_id: 'race-1', records: sampleRecords },
    })

    const wrapper = mount(ParticipantFlowChart, {
      props: {
        raceId: 'race-1',
        participantId: 'p1',
        raceStatus: 'active',
        raceStartTime: '2024-06-01T10:30:00.000Z',
        raceType: 'time_based',
      },
    })
    await flushPromises()

    const chartMock = Chart as unknown as Mock
    const constructsAfterMount = chartMock.mock.calls.length
    const chartInstance = chartMock.mock.results.at(-1)?.value as {
      update: Mock
      destroy: Mock
      data: { datasets: Array<{ data: Array<{ x: number }> }> }
    }
    expect(constructsAfterMount).toBeGreaterThan(0)
    chartInstance.update.mockClear()
    chartInstance.destroy.mockClear()

    await vi.advanceTimersByTimeAsync(30_000)
    await flushPromises()
    await wrapper.vm.$nextTick()

    expect(chartMock.mock.calls.length).toBe(constructsAfterMount)
    expect(chartInstance.destroy).not.toHaveBeenCalled()
    expect(chartInstance.update).toHaveBeenCalled()

    wrapper.unmount()
    vi.useRealTimers()
  })

  it('disables animation and caps devicePixelRatio', async () => {
    ;(timingApi.getLive as Mock).mockResolvedValue({
      data: { race_id: 'race-1', records: sampleRecords },
    })

    mount(ParticipantFlowChart, {
      props: {
        raceId: 'race-1',
        participantId: 'p1',
        raceType: 'time_based',
      },
    })
    await flushPromises()

    const chartConfig = (Chart as unknown as Mock).mock.calls.at(-1)?.[1] as {
      options: { animation: boolean; devicePixelRatio: number }
    }
    expect(chartConfig.options.animation).toBe(false)
    expect(chartConfig.options.devicePixelRatio).toBeLessThanOrEqual(2)
  })
})
