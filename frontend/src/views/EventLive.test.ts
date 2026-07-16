import { describe, it, expect, vi, beforeEach, type Mock } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { nextTick } from 'vue'
import EventLive from '@/views/EventLive.vue'
import { setupPinia, createTestRouter } from '@/test/helpers'
import { eventsLiveApi } from '@/services/api'

vi.mock('@/services/api', async () => {
  const actual = await vi.importActual<typeof import('@/services/api')>('@/services/api')
  return {
    ...actual,
    eventsLiveApi: {
      getLive: vi.fn(),
    },
    rfidApi: {
      getSyncStatus: vi.fn().mockResolvedValue({
        data: { pending_count: 0, failed_count: 0, synced_count: 0 },
      }),
    },
  }
})

vi.mock('@/services/offlineQueue', () => ({
  getLocalPendingCount: vi.fn().mockResolvedValue(0),
  onOnline: vi.fn(() => () => {}),
  onPendingChange: vi.fn(() => () => {}),
  syncAll: vi.fn().mockResolvedValue({ synced: 0, failed: 0 }),
}))

vi.mock('@/services/timingStorage', () => ({
  setDisplayCache: vi.fn().mockResolvedValue(undefined),
}))

const livePayload = {
  event: { id: 'evt-1', name: 'All You Can East Bluffet' },
  category_legend: [
    { key: 'advanced_men', label: 'Advanced Men', color: '#1a5276' },
    { key: 'advanced_women', label: 'Advanced Women', color: '#8e44ad' },
  ],
  races: [
    {
      id: 'r-12',
      name: '12 Hour',
      race_type: 'lap_based',
      status: 'scheduled',
      start_time: '2026-08-01T08:00:00-04:00',
      countdown_seconds: 3600,
      leaderboard_overall: [
        {
          place: 1,
          participant_id: 'p1',
          bib_number: '12',
          name: 'Alex Rivera',
          category_key: 'advanced_men',
          laps: 14,
          last_lap_at: '2026-08-01T11:02:41-04:00',
        },
      ],
      flow_series: [],
    },
    {
      id: 'r-6',
      name: '6 Hour',
      race_type: 'lap_based',
      status: 'scheduled',
      start_time: '2026-08-01T08:00:00-04:00',
      countdown_seconds: 3600,
      leaderboard_overall: [],
      flow_series: [],
    },
    {
      id: 'r-90',
      name: '90-Minute Kids',
      race_type: 'lap_based',
      status: 'scheduled',
      start_time: '2026-08-01T15:00:00-04:00',
      countdown_seconds: 25200,
      leaderboard_overall: [],
      flow_series: [],
    },
  ],
}

describe('EventLive.vue', () => {
  beforeEach(() => {
    setupPinia()
    vi.clearAllMocks()
    ;(eventsLiveApi.getLive as Mock).mockResolvedValue({ data: livePayload })
  })

  async function mountLive() {
    const router = createTestRouter([
      {
        path: '/events/:eventId/live',
        name: 'event-live',
        component: EventLive,
      },
    ])
    await router.push('/events/evt-1/live')
    await router.isReady()

    const wrapper = mount(EventLive, {
      global: {
        plugins: [router],
        stubs: {
          ScanPopup: true,
          RaceFlowChart: {
            name: 'RaceFlowChart',
            props: ['raceId', 'raceStatus', 'raceStartTime', 'raceType'],
            template: '<div data-testid="race-flow-chart-stub" />',
          },
        },
      },
    })
    await flushPromises()
    return wrapper
  }

  it('renders live-view with countdown, overall board, and category legend', async () => {
    const wrapper = await mountLive()

    expect(wrapper.find('[data-testid="live-view"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="live-countdown"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="leaderboard-overall"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="category-legend"]').exists()).toBe(true)
    // Spectator / unlocked browsers do not show station sync chrome
    expect(wrapper.find('[data-testid="sync-status"]').exists()).toBe(false)
  })

  it('hides countdown after it reaches zero', async () => {
    ;(eventsLiveApi.getLive as Mock).mockResolvedValue({
      data: {
        ...livePayload,
        races: livePayload.races.map((race) => ({
          ...race,
          status: 'active',
          countdown_seconds: 0,
        })),
      },
    })

    const wrapper = await mountLive()

    expect(wrapper.find('[data-testid="live-countdown"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('Countdown')
  })

  it('passes race type and start time into race flow charts', async () => {
    const wrapper = await mountLive()
    const chart = wrapper.findComponent({ name: 'RaceFlowChart' })

    expect(chart.exists()).toBe(true)
    expect(chart.props('raceId')).toBe('r-12')
    expect(chart.props('raceType')).toBe('lap_based')
    expect(chart.props('raceStatus')).toBe('scheduled')
    expect(chart.props('raceStartTime')).toBe('2026-08-01T08:00:00-04:00')
  })

  it('shows sync status when PIN-unlocked as a reader session', async () => {
    const { usePinAuthStore } = await import('@/stores/pinAuth')
    const pin = usePinAuthStore()
    pin.token = 'test-token'
    pin.role = 'organizer'
    pin.expiresAt = Math.floor(Date.now() / 1000) + 3600

    const wrapper = await mountLive()
    expect(wrapper.find('[data-testid="sync-status"]').exists()).toBe(true)
  })

  it('switches race tabs 12h / 6h / 90m', async () => {
    const wrapper = await mountLive()

    await wrapper.find('[data-testid="race-tab-12h"]').trigger('click')
    await nextTick()
    expect(wrapper.find('[data-testid="race-panel-12h"]').isVisible()).toBe(true)

    await wrapper.find('[data-testid="race-tab-6h"]').trigger('click')
    await nextTick()
    expect(wrapper.find('[data-testid="race-panel-6h"]').isVisible()).toBe(true)

    await wrapper.find('[data-testid="race-tab-90m"]').trigger('click')
    await nextTick()
    expect(wrapper.find('[data-testid="race-panel-90m"]').isVisible()).toBe(true)
  })

  it('toggles overlap chart and fullscreen rotator', async () => {
    const wrapper = await mountLive()

    await wrapper.find('[data-testid="overlap-chart-toggle"]').trigger('click')
    await nextTick()
    expect(wrapper.find('[data-testid="overlap-chart"]').isVisible()).toBe(true)

    await wrapper.find('[data-testid="fullscreen-rotator-toggle"]').trigger('click')
    await nextTick()
    expect(wrapper.find('[data-testid="fullscreen-rotator"]').isVisible()).toBe(true)
    expect(wrapper.find('[data-testid="rotator-flow"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="rotator-leaderboard"]').exists()).toBe(true)
  })
})
