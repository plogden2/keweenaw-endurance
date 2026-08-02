import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, it, expect, vi, beforeEach, afterEach, type Mock } from 'vitest'
import { mount, flushPromises, type VueWrapper } from '@vue/test-utils'
import { nextTick } from 'vue'
import EventLive from '@/views/EventLive.vue'
import { setupPinia, createTestRouter } from '@/test/helpers'
import {
  downloadEventResultsExcel,
  eventsLiveApi,
  rfidApi,
  type LapRecordedEvent,
} from '@/services/api'

const { lastLap, isBusyMock } = vi.hoisted(() => {
  const { ref } = require('vue') as typeof import('vue')
  return {
    lastLap: ref<LapRecordedEvent | null>(null),
    isBusyMock: ref(false),
  }
})

vi.mock('@/composables/useEventLiveStream', () => ({
  useEventLiveStream: () => ({
    lastLap,
    connected: require('vue').ref(false),
    start: vi.fn(),
    stop: vi.fn(),
  }),
}))

vi.mock('@/composables/useSpectatorIdle', () => ({
  useSpectatorIdle: () => ({
    isBusy: isBusyMock,
    noteInteraction: vi.fn(),
  }),
}))

vi.mock('@/services/api', async () => {
  const actual = await vi.importActual<typeof import('@/services/api')>('@/services/api')
  return {
    ...actual,
    eventsLiveApi: {
      getLive: vi.fn(),
    },
    eventParticipantsApi: {
      list: vi.fn().mockResolvedValue({
        data: {
          data: [
            { id: 'p1', team_id: 'team-a', first_name: 'Alex', last_name: 'Rivera' },
            { id: 'p6', team_id: null, first_name: 'Solo', last_name: 'Six' },
          ],
        },
      }),
    },
    raceParticipantsApi: {
      list: vi.fn().mockResolvedValue({ data: { data: [] } }),
    },
    rfidApi: {
      getSyncStatus: vi.fn().mockResolvedValue({
        data: { pending_count: 0, failed_count: 0, synced_count: 0 },
      }),
      getBridgeStatus: vi.fn().mockResolvedValue({
        data: { connected: true, pending_count: 0, syncing: false },
      }),
      getLocalBridgeStatus: vi.fn().mockResolvedValue(null),
      syncPending: vi.fn().mockResolvedValue({ data: { synced_count: 0 } }),
    },
    downloadEventResultsExcel: vi.fn().mockResolvedValue(undefined),
  }
})

vi.mock('@/services/offlineQueue', () => ({
  getLocalPendingCount: vi.fn().mockResolvedValue(0),
  onOnline: vi.fn(() => () => {}),
  onPendingChange: vi.fn(() => () => {}),
  syncAll: vi.fn().mockResolvedValue({ synced: 0, failed: 0 }),
}))

vi.mock('qrcode', () => ({
  default: { toCanvas: vi.fn(async () => undefined) },
}))

vi.mock('@/services/timingStorage', () => ({
  setDisplayCache: vi.fn().mockResolvedValue(undefined),
}))

const livePayload = {
  event: { id: 'evt-1', name: 'All You Can East Bluffet' },
  category_legend: [
    { key: 'expert_men', label: 'Expert Men', color: '#1a5276' },
    { key: 'expert_women', label: 'Expert Women', color: '#8e44ad' },
    { key: 'intermediate_women', label: 'Intermediate Women', color: '#27ae60' },
  ],
  races: [
    {
      id: 'r-12',
      name: '12 Hour',
      race_type: 'lap_based',
      status: 'scheduled',
      start_time: '2026-08-01T08:00:00-04:00',
      duration_minutes: 720,
      countdown_seconds: 3600,
      leaderboard_overall: [
        {
          place: 1,
          participant_id: 'p1',
          bib_number: '12',
          name: 'Alex Rivera',
          category_key: 'expert_men',
          laps: 14,
          last_lap_at: '2026-08-01T11:02:41-04:00',
        },
        {
          place: 2,
          participant_id: 'p2',
          bib_number: '22',
          name: 'Blake West',
          category_key: 'expert_women',
          laps: 12,
          last_lap_at: '2026-08-01T11:01:30-04:00',
        },
        {
          place: 3,
          participant_id: 'p3',
          bib_number: '33',
          name: 'Casey East',
          category_key: 'intermediate_women',
          laps: 10,
          last_lap_at: '2026-08-01T11:00:00-04:00',
        },
      ],
      leaderboard_teams: [
        {
          place: 1,
          team_id: 'team-a',
          name: 'East Bluff A',
          avg_laps: 12.5,
          member_count: 4,
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
      duration_minutes: 360,
      countdown_seconds: 3600,
      leaderboard_overall: [
        {
          place: 1,
          participant_id: 'p1',
          bib_number: '12',
          name: 'Alex Rivera',
          category_key: 'expert_men',
          laps: 8,
          last_lap_at: '2026-08-01T11:02:41-04:00',
        },
        {
          place: 2,
          participant_id: 'p6',
          bib_number: '60',
          name: 'Solo Six',
          category_key: 'expert_men',
          laps: 3,
          last_lap_at: '2026-08-01T11:01:00-04:00',
        },
      ],
      leaderboard_teams: [
        {
          place: 1,
          team_id: 'team-a',
          name: 'East Bluff A',
          avg_laps: 8,
          member_count: 4,
        },
      ],
      flow_series: [],
    },
    {
      id: 'r-90',
      name: '90-Minute Kids',
      race_type: 'lap_based',
      status: 'scheduled',
      start_time: '2026-08-01T15:00:00-04:00',
      duration_minutes: 90,
      countdown_seconds: 25200,
      leaderboard_overall: [
        {
          place: 1,
          participant_id: 'kid1',
          bib_number: '101',
          name: 'Kid One',
          category_key: 'men',
          laps: 2,
          last_lap_at: '2026-08-01T15:20:00-04:00',
        },
        {
          place: 2,
          participant_id: 'kid2',
          bib_number: '102',
          name: 'Kid Two',
          category_key: 'women',
          laps: 1,
          last_lap_at: '2026-08-01T15:18:00-04:00',
        },
      ],
      leaderboard_teams: [],
      flow_series: [],
    },
  ],
}

function lapEvent(
  overrides: Partial<LapRecordedEvent> & Pick<LapRecordedEvent, 'race_id' | 'participant_name'>,
): LapRecordedEvent {
  return {
    type: 'lap_recorded',
    event_id: 'evt-1',
    participant_id: 'p1',
    lap_count: 15,
    recorded_at: '2026-08-01T11:05:00-04:00',
    ...overrides,
  }
}

describe('EventLive.vue', () => {
  let activeWrapper: VueWrapper | null = null
  const originalScrollIntoView = Element.prototype.scrollIntoView

  beforeEach(() => {
    setupPinia()
    vi.clearAllMocks()
    lastLap.value = null
    isBusyMock.value = false
    ;(eventsLiveApi.getLive as Mock).mockResolvedValue({ data: livePayload })
    Element.prototype.scrollIntoView = vi.fn()
  })

  afterEach(() => {
    Element.prototype.scrollIntoView = originalScrollIntoView
    activeWrapper?.unmount()
    activeWrapper = null
  })

  async function mountLive() {
    activeWrapper?.unmount()
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
            props: [
              'raceId',
              'raceStatus',
              'raceStartTime',
              'raceType',
              'durationMinutes',
              'highlightParticipantId',
              'defaultPlotExpanded',
            ],
            template: '<div data-testid="race-flow-chart-stub" />',
            setup() {
              return {
                loadRecords: vi.fn().mockResolvedValue(undefined),
                isLegendBusy: false,
              }
            },
          },
        },
      },
    })
    activeWrapper = wrapper
    await flushPromises()
    return wrapper
  }

  it('sets currentEvent from live payload after load', async () => {
    const { useEventsStore } = await import('@/stores/events')
    const events = useEventsStore()
    expect(events.currentEvent).toBeNull()

    await mountLive()

    expect(events.currentEvent).toEqual(
      expect.objectContaining({
        id: livePayload.event.id,
        name: 'All You Can East Bluffet',
      }),
    )
  })

  it('does not clear currentEvent when live fetch fails', async () => {
    const { useEventsStore } = await import('@/stores/events')
    const events = useEventsStore()
    events.setCurrentEventSummary(livePayload.event)
    ;(eventsLiveApi.getLive as Mock).mockRejectedValueOnce(new Error('network'))

    await mountLive()

    expect(events.currentEvent?.name).toBe('All You Can East Bluffet')
    expect(events.currentEvent?.id).toBe(livePayload.event.id)
  })

  it('renders live-view with countdown, overall board, and category legend', async () => {
    const wrapper = await mountLive()

    expect(wrapper.find('[data-testid="live-view"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="live-countdown"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="leaderboard-overall"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="leaderboard-mode-toggle"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="leaderboard-teams"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="category-legend"]').exists()).toBe(true)
    // Spectator / unlocked browsers do not show station sync chrome
    expect(wrapper.find('[data-testid="sync-status"]').exists()).toBe(false)
  })

  it('toggles leaderboard between individuals and teams', async () => {
    const wrapper = await mountLive()

    const toggle = wrapper.find('[data-testid="leaderboard-mode-toggle"]')
    const teamsBtn = toggle.findAll('button').find((btn) => btn.text() === 'Teams')
    expect(teamsBtn).toBeTruthy()
    await teamsBtn!.trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-testid="leaderboard-overall"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="leaderboard-teams"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="leaderboard-category-filters"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('East Bluff A')
    expect(wrapper.text()).toContain('12.5')

    const individualsBtn = toggle.findAll('button').find((btn) => btn.text() === 'Individuals')
    await individualsBtn!.trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-testid="leaderboard-overall"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="leaderboard-teams"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="leaderboard-category-filters"]').exists()).toBe(true)
  })

  describe('leaderboard category filters', () => {
    it('renders filters in individuals mode', async () => {
      const wrapper = await mountLive()

      expect(wrapper.find('[data-testid="leaderboard-category-filters"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="lb-filter-skill"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="lb-filter-gender"]').exists()).toBe(true)
    })

    it('filters to Expert Women with renumbered place', async () => {
      const wrapper = await mountLive()

      await wrapper.find('[data-testid="lb-filter-skill-expert"]').trigger('click')
      await wrapper.find('[data-testid="lb-filter-gender-women"]').trigger('click')
      await flushPromises()

      const rows = wrapper.findAll('[data-testid="leaderboard-row"]')
      expect(rows).toHaveLength(1)
      expect(rows[0]!.attributes('data-participant-id')).toBe('p2')
      expect(rows[0]!.text()).toContain('Blake West')
      expect(rows[0]!.find('.place').text()).toBe('1')
    })

    it('hides filters when fullscreen rotator is open', async () => {
      const wrapper = await mountLive()

      expect(wrapper.find('[data-testid="leaderboard-category-filters"]').exists()).toBe(true)

      await wrapper.find('[data-testid="fullscreen-rotator-toggle"]').trigger('click')
      await nextTick()

      expect(wrapper.find('[data-testid="leaderboard-category-filters"]').exists()).toBe(false)
      const rotator = wrapper.find('[data-testid="fullscreen-rotator"]')
      expect(rotator.exists()).toBe(true)
      expect(rotator.find('[data-testid="leaderboard-category-filters"]').exists()).toBe(false)
    })

    it('hides filters in teams mode', async () => {
      const wrapper = await mountLive()

      const toggle = wrapper.find('[data-testid="leaderboard-mode-toggle"]')
      const teamsBtn = toggle.findAll('button').find((btn) => btn.text() === 'Teams')
      await teamsBtn!.trigger('click')
      await flushPromises()

      expect(wrapper.find('[data-testid="leaderboard-category-filters"]').exists()).toBe(false)
    })

    it('hides Class chips on kids tab even when event legend has Expert', async () => {
      const wrapper = await mountLive()

      await wrapper.find('[data-testid="race-tab-90m"]').trigger('click')
      await flushPromises()

      expect(wrapper.find('[data-testid="leaderboard-category-filters"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="lb-filter-skill"]').exists()).toBe(false)
      expect(wrapper.find('[data-testid="lb-filter-gender"]').exists()).toBe(true)
    })
  })

  it('retints category legend and leaderboard dots with brand colors', async () => {
    const wrapper = await mountLive()

    const legendSwatches = wrapper.findAll('.legend i')
    expect(legendSwatches[0]?.attributes('style')).toContain('rgb(26, 63, 61)')
    expect(legendSwatches[1]?.attributes('style')).toContain('rgb(47, 107, 90)')

    const dot = wrapper.find('.cat-dot')
    expect(dot.attributes('style')).toContain('rgb(26, 63, 61)')
    expect(dot.attributes('style')).not.toContain('#1a5276')
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

  it('ticks countdown every second between live polls', async () => {
    vi.useFakeTimers()
    try {
      const wrapper = await mountLive()
      const countdown = wrapper.find('[data-testid="live-countdown"]')
      expect(countdown.text()).toBe('01:00:00')

      await vi.advanceTimersByTimeAsync(1000)
      await nextTick()
      expect(countdown.text()).toBe('00:59:59')

      await vi.advanceTimersByTimeAsync(1000)
      await nextTick()
      expect(countdown.text()).toBe('00:59:58')
    } finally {
      vi.useRealTimers()
    }
  })

  it('passes race type and start time into race flow charts', async () => {
    const wrapper = await mountLive()
    const chart = wrapper.findComponent({ name: 'RaceFlowChart' })

    expect(chart.exists()).toBe(true)
    expect(chart.props('raceId')).toBe('r-12')
    expect(chart.props('raceType')).toBe('lap_based')
    expect(chart.props('raceStatus')).toBe('scheduled')
    expect(chart.props('raceStartTime')).toBe('2026-08-01T08:00:00-04:00')
    expect(chart.props('durationMinutes')).toBe(720)
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

  it('hides Export Excel without an authenticated PIN session', async () => {
    const wrapper = await mountLive()

    expect(wrapper.find('[data-testid="export-results-excel"]').exists()).toBe(false)
  })

  it('exports event results when PIN-unlocked', async () => {
    const wrapper = await mountReaderLive()
    const exportButton = wrapper.find('[data-testid="export-results-excel"]')

    expect(exportButton.exists()).toBe(true)
    await exportButton.trigger('click')

    expect(downloadEventResultsExcel).toHaveBeenCalledWith('evt-1', 'All You Can East Bluffet')
  })

  it('shows Bibs ops link when PIN-unlocked', async () => {
    const wrapper = await mountReaderLive()
    const bibsLink = wrapper.find('[data-testid="live-open-bibs"]')

    expect(bibsLink.exists()).toBe(true)
    expect(bibsLink.attributes('href')).toBe('/events/evt-1/bibs')
    expect(bibsLink.text()).toBe('Bibs')
  })

  it('links Racers ops to the event-wide racers page', async () => {
    const wrapper = await mountReaderLive()
    const racersLink = wrapper.find('[data-testid="live-open-racers"]')
    expect(racersLink.exists()).toBe(true)
    expect(racersLink.attributes('href')).toBe('/events/evt-1/racers')
  })

  async function mountReaderLive() {
    const { usePinAuthStore } = await import('@/stores/pinAuth')
    const pin = usePinAuthStore()
    pin.token = 'test-token'
    pin.role = 'organizer'
    pin.expiresAt = Math.floor(Date.now() / 1000) + 3600
    return mountLive()
  }

  it('shows Online · Synced chip when bridge is connected with no pending', async () => {
    ;(rfidApi.getBridgeStatus as Mock).mockResolvedValue({
      data: { connected: true, pending_count: 0, syncing: false },
    })

    const wrapper = await mountReaderLive()
    await flushPromises()

    expect(wrapper.find('[data-testid="sync-online"]').text()).toBe('Online · Synced')
    expect(wrapper.find('[data-testid="sync-offline"]').exists()).toBe(false)
  })

  it('shows Offline chip when navigator is offline', async () => {
    Object.defineProperty(navigator, 'onLine', { value: false, configurable: true })

    const wrapper = await mountReaderLive()
    await flushPromises()

    expect(wrapper.find('[data-testid="sync-offline"]').text()).toBe('Offline')

    Object.defineProperty(navigator, 'onLine', { value: true, configurable: true })
  })

  it('shows Syncing chip when bridge reports syncing', async () => {
    ;(rfidApi.getBridgeStatus as Mock).mockResolvedValue({
      data: { connected: true, pending_count: 2, syncing: true },
    })

    const wrapper = await mountReaderLive()
    await flushPromises()

    expect(wrapper.find('[data-testid="sync-syncing"]').text()).toBe('Syncing')
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

  it('keeps sync chip mounted while fullscreen rotator is open', async () => {
    const wrapper = await mountReaderLive()
    await wrapper.find('[data-testid="fullscreen-rotator-toggle"]').trigger('click')
    await nextTick()
    expect(wrapper.find('[data-testid="fullscreen-rotator"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="sync-status"]').exists()).toBe(true)
    expect(wrapper.find('.sync-bar--overlay').exists()).toBe(true)
  })

  it('defines baseline and fullscreen display type scales', () => {
    const src = readFileSync(join(process.cwd(), 'src/views/EventLive.vue'), 'utf8')
    expect(src).toMatch(/--live-display-scale:\s*1/)
    expect(src).toMatch(
      /fullscreen-rotator[\s\S]*--live-display-scale:\s*1\.35|--live-display-scale:\s*1\.25/,
    )
  })

  it('uses a light page background for the fullscreen rotator', () => {
    const src = readFileSync(join(process.cwd(), 'src/views/EventLive.vue'), 'utf8')
    const style = src.split('<style scoped>')[1]?.split('</style>')[0] ?? ''
    expect(style).toMatch(/\.fs-root\s*\{[^}]*background:\s*var\(--mist\)/s)
    expect(style).not.toMatch(/\.fs-root\s*\{[^}]*background:\s*var\(--ink-deep\)/s)
  })

  describe('lap celebration', () => {
    it('shows overlay when lap recorded for visible race', async () => {
      const wrapper = await mountLive()

      lastLap.value = lapEvent({ race_id: 'r-12', participant_name: 'Alex Rivera' })
      await nextTick()

      const celebration = wrapper.find('[data-testid="lap-celebration"]')
      expect(celebration.exists()).toBe(true)
      expect(celebration.text()).toContain('Alex Rivera')
      expect(celebration.text()).toContain('+1')
    })

    it('shows overlay but skips highlight when spectator is busy', async () => {
      isBusyMock.value = true
      const wrapper = await mountLive()

      lastLap.value = lapEvent({ race_id: 'r-12', participant_name: 'Alex Rivera' })
      await nextTick()

      expect(wrapper.find('[data-testid="lap-celebration"]').exists()).toBe(true)
      const chart = wrapper.findComponent({ name: 'RaceFlowChart' })
      expect(chart.props('highlightParticipantId')).toBeUndefined()
      const row = wrapper.find('[data-testid="leaderboard-row"]')
      expect(row.classes()).not.toContain('leaderboard-row--focus')
    })

    it('updates overlay name on rapid laps (latest-wins)', async () => {
      const wrapper = await mountLive()

      lastLap.value = lapEvent({ race_id: 'r-12', participant_name: 'Alex Rivera' })
      await nextTick()
      lastLap.value = lapEvent({
        race_id: 'r-12',
        participant_id: 'p2',
        participant_name: 'Jordan Lee',
      })
      await nextTick()

      const celebration = wrapper.find('[data-testid="lap-celebration"]')
      expect(celebration.text()).toContain('Jordan Lee')
      expect(celebration.text()).not.toContain('Alex Rivera')
    })

    it('ignores lap for non-visible race', async () => {
      const wrapper = await mountLive()

      lastLap.value = lapEvent({ race_id: 'r-90', participant_name: 'Kid Runner' })
      await nextTick()

      expect(wrapper.find('[data-testid="lap-celebration"]').exists()).toBe(false)
    })

    it('shows overlay when lap race_id is a full UUID whose suffix matches the visible race', async () => {
      ;(eventsLiveApi.getLive as Mock).mockResolvedValue({
        data: {
          ...livePayload,
          races: livePayload.races.map((race) =>
            race.id === 'r-12' ? { ...race, id: 'ab12cd' } : race,
          ),
        },
      })
      const wrapper = await mountLive()

      lastLap.value = lapEvent({
        race_id: '550e8400-e29b-41d4-a716-446655ab12cd',
        participant_name: 'Alex Rivera',
      })
      await nextTick()

      expect(wrapper.find('[data-testid="lap-celebration"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="lap-celebration"]').text()).toContain('Alex Rivera')
    })

    it('celebrates when a polled leaderboard lap count increases', async () => {
      vi.useFakeTimers()
      const wrapper = await mountLive()
      expect(wrapper.find('[data-testid="lap-celebration"]').exists()).toBe(false)

      const bumped = structuredClone(livePayload)
      const alex = bumped.races[0]?.leaderboard_overall[0]
      if (alex) alex.laps = (alex.laps ?? 0) + 1
      ;(eventsLiveApi.getLive as Mock).mockResolvedValue({ data: bumped })

      await vi.advanceTimersByTimeAsync(2000)
      await flushPromises()
      await nextTick()

      expect(wrapper.find('[data-testid="lap-celebration"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="lap-celebration"]').text()).toContain('Alex Rivera')
      vi.useRealTimers()
    })
  })

  describe('sticky highlight v-model wiring', () => {
    it('binds highlightParticipantId as v-model on race flow charts', async () => {
      const wrapper = await mountLive()
      // Only the active tab mounts a chart (v-if) so iOS does not keep 5 Chart.js instances alive.
      const chart = wrapper.findComponent({ name: 'RaceFlowChart' })

      await chart.vm.$emit('update:highlightParticipantId', 'racer-1')
      await flushPromises()
      expect(chart.props('highlightParticipantId')).toBe('racer-1')

      await wrapper.find('[data-testid="race-tab-6h"]').trigger('click')
      await nextTick()
      await flushPromises()
      const chart6h = wrapper.findComponent({ name: 'RaceFlowChart' })
      expect(chart6h.props('highlightParticipantId')).toBe('racer-1')

      await chart6h.vm.$emit('update:highlightParticipantId', undefined)
      await flushPromises()
      expect(chart6h.props('highlightParticipantId')).toBeUndefined()
    })

    it('clears focusParticipantId when highlightParticipantId is cleared', async () => {
      const wrapper = await mountLive()

      lastLap.value = lapEvent({ race_id: 'r-12', participant_name: 'Alex Rivera' })
      await nextTick()

      const focusedRow = wrapper.find(
        '[data-testid="leaderboard-row"][data-participant-id="p1"]',
      )
      expect(focusedRow.classes()).toContain('leaderboard-row--focus')

      const chart = wrapper.findComponent({ name: 'RaceFlowChart' })
      await chart.vm.$emit('update:highlightParticipantId', undefined)
      await flushPromises()

      const row = wrapper.find('[data-testid="leaderboard-row"][data-participant-id="p1"]')
      expect(row.classes()).not.toContain('leaderboard-row--focus')
    })
  })

  describe('fullscreen rotator split handle', () => {
    const FS_FLOW_WIDTH_KEY = 'event-live-fs-flow-width'

    beforeEach(() => {
      sessionStorage.clear()
    })

    async function openFullscreenRotator(wrapper: VueWrapper) {
      await wrapper.find('[data-testid="fullscreen-rotator-toggle"]').trigger('click')
      await nextTick()
    }

    function mockFsGridRect(wrapper: VueWrapper, width = 1000) {
      const grid = wrapper.find('.fs-grid')
      vi.spyOn(grid.element, 'getBoundingClientRect').mockReturnValue({
        left: 0,
        top: 0,
        width,
        height: 600,
        right: width,
        bottom: 600,
        x: 0,
        y: 0,
        toJSON: () => ({}),
      } as DOMRect)
      return grid
    }

    async function dragSplitHandle(wrapper: VueWrapper, clientXSequence: number[]) {
      const grid = mockFsGridRect(wrapper)
      const handle = wrapper.find('[data-testid="rotator-split-handle"]')
      const el = handle.element as HTMLElement
      el.setPointerCapture = vi.fn()
      el.releasePointerCapture = vi.fn()

      const dispatch = (type: string, clientX: number) => {
        el.dispatchEvent(
          new PointerEvent(type, {
            bubbles: true,
            clientX,
            pointerId: 1,
            buttons: type === 'pointerup' ? 0 : 1,
          }),
        )
      }

      dispatch('pointerdown', clientXSequence[0])
      await nextTick()
      for (const clientX of clientXSequence.slice(1, -1)) {
        dispatch('pointermove', clientX)
        await nextTick()
      }
      dispatch('pointerup', clientXSequence[clientXSequence.length - 1])
      await nextTick()
      return grid
    }

    it('renders rotator-split-handle between rotator-flow and rotator-leaderboard', async () => {
      const wrapper = await mountLive()
      await openFullscreenRotator(wrapper)

      const handle = wrapper.find('[data-testid="rotator-split-handle"]')
      expect(handle.exists()).toBe(true)

      const gridHtml = wrapper.find('.fs-grid').html()
      const flowPos = gridHtml.indexOf('data-testid="rotator-flow"')
      const handlePos = gridHtml.indexOf('data-testid="rotator-split-handle"')
      const leaderboardPos = gridHtml.indexOf('data-testid="rotator-leaderboard"')
      expect(flowPos).toBeGreaterThanOrEqual(0)
      expect(handlePos).toBeGreaterThan(flowPos)
      expect(leaderboardPos).toBeGreaterThan(handlePos)
    })

    it('fs-grid source uses --fs-flow-width for column sizing', () => {
      const src = readFileSync(join(process.cwd(), 'src/views/EventLive.vue'), 'utf8')
      expect(src).toMatch(/--fs-flow-width/)
      const style = src.split('<style scoped>')[1]?.split('</style>')[0] ?? ''
      expect(style).toMatch(/\.fs-grid[\s\S]*grid-template-columns:[^;]*var\(--fs-flow-width/)
      expect(src).toMatch(/'--fs-flow-width'|--fs-flow-width':/)
    })

    it('updates --fs-flow-width when dragging the split handle', async () => {
      const wrapper = await mountLive()
      await openFullscreenRotator(wrapper)

      const grid = await dragSplitHandle(wrapper, [520, 700, 700])
      expect((grid.element as HTMLElement).style.getPropertyValue('--fs-flow-width')).toBe('70%')
    })

    it('clamps --fs-flow-width between 25% and 75% when dragging', async () => {
      const wrapper = await mountLive()
      await openFullscreenRotator(wrapper)

      const gridLow = await dragSplitHandle(wrapper, [10, 10, 10])
      expect((gridLow.element as HTMLElement).style.getPropertyValue('--fs-flow-width')).toBe('25%')

      const gridHigh = await dragSplitHandle(wrapper, [990, 990, 990])
      expect((gridHigh.element as HTMLElement).style.getPropertyValue('--fs-flow-width')).toBe('75%')
    })

    it('restores flow width from sessionStorage when opening rotator', async () => {
      sessionStorage.setItem(FS_FLOW_WIDTH_KEY, '60')
      const wrapper = await mountLive()
      await openFullscreenRotator(wrapper)

      const grid = wrapper.find('.fs-grid')
      expect((grid.element as HTMLElement).style.getPropertyValue('--fs-flow-width')).toBe('60%')
    })

    it('persists flow width to sessionStorage after drag ends', async () => {
      const wrapper = await mountLive()
      await openFullscreenRotator(wrapper)

      await dragSplitHandle(wrapper, [520, 650, 650])
      expect(sessionStorage.getItem(FS_FLOW_WIDTH_KEY)).toBe('65')
    })

    it('source contract: uses event-live-fs-flow-width sessionStorage key', () => {
      const src = readFileSync(join(process.cwd(), 'src/views/EventLive.vue'), 'utf8')
      expect(src).toMatch(/event-live-fs-flow-width/)
    })
  })

  describe('rotator lap jump', () => {
    it('jumps to team page for a teammate lap on another race while playing', async () => {
      const wrapper = await mountLive()
      await flushPromises()
      await wrapper.find('[data-testid="fullscreen-rotator-toggle"]').trigger('click')
      await nextTick()

      expect(wrapper.find('.fs-meta').text()).toContain('12 Hour · Individual')

      lastLap.value = lapEvent({
        race_id: 'r-6',
        participant_id: 'p1',
        participant_name: 'Alex Rivera',
      })
      await nextTick()
      await flushPromises()

      expect(wrapper.find('[data-testid="lap-celebration"]').exists()).toBe(true)
      expect(wrapper.find('.fs-meta').text()).toContain('6 Hour · Team')
    })

    it('jumps to individual page when the racer has no team', async () => {
      const wrapper = await mountLive()
      await flushPromises()
      await wrapper.find('[data-testid="fullscreen-rotator-toggle"]').trigger('click')
      await nextTick()

      lastLap.value = lapEvent({
        race_id: 'r-6',
        participant_id: 'p6',
        participant_name: 'Solo Six',
      })
      await nextTick()
      await flushPromises()

      expect(wrapper.find('.fs-meta').text()).toContain('6 Hour · Individual')
    })

    it('leaves the team page when a solo racer in that race records a lap', async () => {
      const wrapper = await mountLive()
      await flushPromises()
      await wrapper.find('[data-testid="fullscreen-rotator-toggle"]').trigger('click')
      await nextTick()

      // Advance to 12h · Team via a teammate lap, then a solo racer should flip to Individual.
      lastLap.value = lapEvent({
        race_id: 'r-12',
        participant_id: 'p1',
        participant_name: 'Alex Rivera',
      })
      await nextTick()
      await flushPromises()
      expect(wrapper.find('.fs-meta').text()).toContain('12 Hour · Team')

      lastLap.value = lapEvent({
        race_id: 'r-12',
        participant_id: 'p6',
        participant_name: 'Solo Six',
        lap_count: 2,
      })
      await nextTick()
      await flushPromises()

      expect(wrapper.find('.fs-meta').text()).toContain('12 Hour · Individual')
    })

    it('does not jump when rotator is paused', async () => {
      const wrapper = await mountLive()
      await flushPromises()
      await wrapper.find('[data-testid="fullscreen-rotator-toggle"]').trigger('click')
      await nextTick()
      await wrapper.find('[data-testid="rotator-play-pause"]').trigger('click')
      await nextTick()

      lastLap.value = lapEvent({
        race_id: 'r-6',
        participant_id: 'p1',
        participant_name: 'Alex Rivera',
      })
      await nextTick()

      expect(wrapper.find('[data-testid="lap-celebration"]').exists()).toBe(false)
      expect(wrapper.find('.fs-meta').text()).toContain('12 Hour · Individual')
    })

    it('scrolls the visible rotator leaderboard row after jumping race', async () => {
      const scrollIntoView = vi.fn()
      const previous = Element.prototype.scrollIntoView
      Element.prototype.scrollIntoView = scrollIntoView
      try {
        const wrapper = await mountLive()
        await flushPromises()
        await wrapper.find('[data-testid="fullscreen-rotator-toggle"]').trigger('click')
        await nextTick()
        await flushPromises()

        lastLap.value = lapEvent({
          race_id: 'r-6',
          participant_id: 'p6',
          participant_name: 'Solo Six',
        })
        await flushPromises()
        await nextTick()
        await flushPromises()
        await vi.waitFor(() => expect(scrollIntoView).toHaveBeenCalled())
        await flushPromises()

        expect(wrapper.find('.fs-meta').text()).toContain('6 Hour · Individual')
        const scrolled = scrollIntoView.mock.instances[0] as HTMLElement
        const board = wrapper.find('[data-testid="rotator-leaderboard"]').element
        expect(board.contains(scrolled)).toBe(true)
        expect(scrolled.getAttribute('data-participant-id')).toBe('p6')
      } finally {
        Element.prototype.scrollIntoView = previous
      }
    })

    it('scrolls the team row on the rotator board for a teammate lap', async () => {
      const scrollIntoView = vi.fn()
      const previous = Element.prototype.scrollIntoView
      Element.prototype.scrollIntoView = scrollIntoView
      try {
        const wrapper = await mountLive()
        await flushPromises()
        await wrapper.find('[data-testid="fullscreen-rotator-toggle"]').trigger('click')
        await nextTick()
        await flushPromises()

        lastLap.value = lapEvent({
          race_id: 'r-6',
          participant_id: 'p1',
          participant_name: 'Alex Rivera',
        })
        await flushPromises()
        await nextTick()
        await flushPromises()
        await vi.waitFor(() => expect(scrollIntoView).toHaveBeenCalled())
        await flushPromises()

        expect(wrapper.find('.fs-meta').text()).toContain('6 Hour · Team')
        const scrolled = scrollIntoView.mock.instances[0] as HTMLElement
        const board = wrapper.find('[data-testid="rotator-leaderboard"]').element
        expect(board.contains(scrolled)).toBe(true)
        expect(scrolled.getAttribute('data-team-id')).toBe('team-a')
      } finally {
        Element.prototype.scrollIntoView = previous
      }
    })
  })

  describe('rotator live QR', () => {
    beforeEach(() => {
      sessionStorage.clear()
    })

    afterEach(() => {
      vi.useRealTimers()
    })

    it('does not show QR by default', async () => {
      const wrapper = await mountLive()
      await wrapper.find('[data-testid="fullscreen-rotator-toggle"]').trigger('click')
      await nextTick()
      expect(wrapper.find('[data-testid="rotator-live-qr"]').exists()).toBe(false)
    })

    it('shows QR with live url when enabled in settings', async () => {
      const wrapper = await mountLive()
      await wrapper.find('[data-testid="fullscreen-rotator-toggle"]').trigger('click')
      await nextTick()
      await wrapper.find('[data-testid="rotator-settings-open"]').trigger('click')
      await nextTick()
      const checkbox = wrapper.find('[data-testid="rotator-show-qr"]')
      expect(checkbox.exists()).toBe(true)
      await checkbox.setValue(true)
      await nextTick()
      await wrapper.find('[data-testid="rotator-settings-done"]').trigger('click')
      await nextTick()
      const qr = wrapper.find('[data-testid="rotator-live-qr"]')
      expect(qr.exists()).toBe(true)
      expect(qr.text()).toContain('view results at keweenawendurance.com')
    })

    it('hides controls after idle when QR is enabled', async () => {
      vi.useFakeTimers()
      sessionStorage.setItem(
        'event-live-fs-rotator-settings',
        JSON.stringify({
          dwellMs: 5000,
          showQrCode: true,
          pages: [
            { race: '12h', mode: 'individuals', enabled: true },
            { race: '12h', mode: 'teams', enabled: true },
            { race: '6h', mode: 'individuals', enabled: true },
            { race: '6h', mode: 'teams', enabled: true },
          ],
        }),
      )
      const wrapper = await mountLive()
      await wrapper.find('[data-testid="fullscreen-rotator-toggle"]').trigger('click')
      await nextTick()
      expect(wrapper.find('[data-testid="rotator-live-qr"]').exists()).toBe(true)
      const controls = wrapper.find('[data-testid="rotator-controls"]')
      const corner = wrapper.find('[data-testid="rotator-corner"]')
      expect(controls.classes()).not.toContain('fs-controls--idle')
      expect(corner.classes()).not.toContain('fs-corner--controls-idle')
      await vi.advanceTimersByTimeAsync(3000)
      await nextTick()
      expect(controls.classes()).toContain('fs-controls--idle')
      expect(corner.classes()).toContain('fs-corner--controls-idle')
      vi.useRealTimers()
    })

    it('does not idle-hide controls when QR is off', async () => {
      vi.useFakeTimers()
      const wrapper = await mountLive()
      await wrapper.find('[data-testid="fullscreen-rotator-toggle"]').trigger('click')
      await nextTick()
      await vi.advanceTimersByTimeAsync(5000)
      await nextTick()
      expect(wrapper.find('[data-testid="rotator-controls"]').classes()).not.toContain(
        'fs-controls--idle',
      )
      vi.useRealTimers()
    })
  })

  describe('fullscreen rotator cycle', () => {
    beforeEach(() => {
      sessionStorage.clear()
      vi.useFakeTimers()
    })

    afterEach(() => {
      vi.useRealTimers()
    })

    async function openFullscreenRotator(wrapper: VueWrapper) {
      await wrapper.find('[data-testid="fullscreen-rotator-toggle"]').trigger('click')
      await nextTick()
    }

    it('opens the rotator race-flow chart in expanded plot mode', async () => {
      const wrapper = await mountLive()
      await openFullscreenRotator(wrapper)
      const chart = wrapper
        .find('[data-testid="rotator-flow"]')
        .findComponent({ name: 'RaceFlowChart' })
      expect(chart.props('defaultPlotExpanded')).toBe(true)
    })

    it('shows play/pause and settings controls in the top right', async () => {
      const wrapper = await mountLive()
      await openFullscreenRotator(wrapper)

      expect(wrapper.find('[data-testid="rotator-play-pause"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="rotator-settings-open"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="rotator-exit"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="rotator-controls"]').exists()).toBe(true)
    })

    it('cycles pages on a 5s dwell and can be paused', async () => {
      const wrapper = await mountLive()
      await openFullscreenRotator(wrapper)

      expect(wrapper.find('.fs-meta').text()).toContain('12 Hour · Individual')
      await vi.advanceTimersByTimeAsync(5000)
      await nextTick()
      expect(wrapper.find('.fs-meta').text()).toContain('12 Hour · Team')

      await wrapper.find('[data-testid="rotator-play-pause"]').trigger('click')
      await nextTick()
      await vi.advanceTimersByTimeAsync(5000)
      await nextTick()
      expect(wrapper.find('.fs-meta').text()).toContain('12 Hour · Team')
    })

    it('opens settings dialog to adjust dwell and page enablement', async () => {
      const wrapper = await mountLive()
      await openFullscreenRotator(wrapper)

      await wrapper.find('[data-testid="rotator-settings-open"]').trigger('click')
      await nextTick()
      expect(wrapper.find('[data-testid="rotator-settings-dialog"]').exists()).toBe(true)

      const dwell = wrapper.find('[data-testid="rotator-dwell-seconds"]')
      await dwell.setValue('8')
      await dwell.trigger('change')
      await nextTick()
      expect(sessionStorage.getItem('event-live-fs-rotator-settings')).toContain('"dwellMs":8000')

      await wrapper.find('[data-testid="rotator-settings-done"]').trigger('click')
      await nextTick()
      expect(wrapper.find('[data-testid="rotator-settings-dialog"]').exists()).toBe(false)
    })

    it('removes 6 hour pages from the cycle after the race finishes', async () => {
      const wrapper = await mountLive()
      await openFullscreenRotator(wrapper)

      window.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowRight' }))
      await nextTick()
      window.dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowRight' }))
      await nextTick()
      expect(wrapper.find('.fs-meta').text()).toContain('6 Hour')

      const finished = structuredClone(livePayload)
      finished.races[1]!.status = 'finished'
      ;(eventsLiveApi.getLive as Mock).mockResolvedValue({ data: finished })
      await vi.advanceTimersByTimeAsync(2000)
      await flushPromises()
      await nextTick()

      expect(wrapper.find('.fs-meta').text()).not.toContain('6 Hour')
      expect(wrapper.find('.fs-meta').text()).toContain('12 Hour')
    })
  })
})
