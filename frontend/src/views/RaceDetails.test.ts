import { describe, it, expect, vi, beforeEach, type Mock } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import RaceDetails from './RaceDetails.vue'
import { setupPinia, createTestRouter } from '@/test/helpers'
import { useRacesStore } from '@/stores/races'
import { useEventsStore } from '@/stores/events'
import { usePinAuthStore } from '@/stores/pinAuth'
import { timingApi, participantsApi } from '@/services/api'

vi.mock('@/stores/races', async () => {
  const actual = await vi.importActual<typeof import('@/stores/races')>('@/stores/races')
  return { ...actual, useRacesStore: vi.fn() }
})

vi.mock('@/stores/events', async () => {
  const actual = await vi.importActual<typeof import('@/stores/events')>('@/stores/events')
  return { ...actual, useEventsStore: vi.fn() }
})

vi.mock('@/services/api', async () => {
  const actual = await vi.importActual<typeof import('@/services/api')>('@/services/api')
  return {
    ...actual,
    timingApi: {
      getLeaderboard: vi.fn(),
      getResults: vi.fn(),
      getLive: vi.fn(),
    },
    participantsApi: {
      get: vi.fn(),
    },
  }
})

describe('RaceDetails.vue', () => {
  let racesStore: {
    currentRace: Record<string, unknown> | null
    loading: boolean
    error: string | null
    fetchRace: Mock
  }
  let eventsStore: {
    currentEvent: Record<string, unknown> | null
    fetchEvent: Mock
  }

  beforeEach(() => {
    setupPinia()
    racesStore = {
      currentRace: null,
      loading: false,
      error: null,
      fetchRace: vi.fn(),
    }
    eventsStore = {
      currentEvent: null,
      fetchEvent: vi.fn(),
    }
    ;(useRacesStore as unknown as Mock).mockReturnValue(racesStore)
    ;(useEventsStore as unknown as Mock).mockReturnValue(eventsStore)
    vi.clearAllMocks()
  })

  it('hides Racers and Manual entry ops without PIN', async () => {
    racesStore.currentRace = {
      id: 'race-1',
      name: 'Marathon',
      race_type: 'time_based',
      status: 'active',
    }
    ;(timingApi.getLeaderboard as Mock).mockResolvedValue({ data: { data: [] } })
    ;(timingApi.getLive as Mock).mockResolvedValue({
      data: { race_id: 'race-1', records: [] },
    })

    const router = createTestRouter()
    await router.push('/timing/evt-1/race/race-1')
    await router.isReady()

    const wrapper = mount(RaceDetails, {
      global: { plugins: [router] },
    })
    await flushPromises()

    expect(wrapper.find('[data-testid="race-details-racers"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="race-details-manual"]').exists()).toBe(false)
  })

  it('shows switch-to-live-view banner linking to event live route', async () => {
    racesStore.currentRace = {
      id: 'race-1',
      name: 'Marathon',
      race_type: 'time_based',
      status: 'active',
    }
    ;(timingApi.getLeaderboard as Mock).mockResolvedValue({ data: { data: [] } })
    ;(timingApi.getLive as Mock).mockResolvedValue({
      data: { race_id: 'race-1', records: [] },
    })

    const router = createTestRouter()
    await router.push('/timing/evt-1/race/race-1')
    await router.isReady()

    const wrapper = mount(RaceDetails, {
      global: { plugins: [router] },
    })
    await flushPromises()

    const banner = wrapper.get('[data-testid="switch-to-live-view"]')
    expect(banner.text()).toMatch(/Switch to live view/)
    expect(banner.attributes('href')).toBe('/events/evt-1/live')
  })

  it('shows Racers and Manual entry ops when PIN unlocked', async () => {
    const pin = usePinAuthStore()
    pin.token = 'test-token'
    pin.role = 'admin'
    pin.expiresAt = Math.floor(Date.now() / 1000) + 3600

    racesStore.currentRace = {
      id: 'race-1',
      name: 'Marathon',
      race_type: 'time_based',
      status: 'active',
    }
    ;(timingApi.getLeaderboard as Mock).mockResolvedValue({ data: { data: [] } })
    ;(timingApi.getLive as Mock).mockResolvedValue({
      data: { race_id: 'race-1', records: [] },
    })

    const router = createTestRouter()
    await router.push('/timing/evt-1/race/race-1')
    await router.isReady()

    const wrapper = mount(RaceDetails, {
      global: { plugins: [router] },
    })
    await flushPromises()

    expect(wrapper.find('[data-testid="race-details-racers"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="race-details-manual"]').exists()).toBe(true)
  })

  it('loads race and leaderboard on mount', async () => {
    racesStore.currentRace = {
      id: 'race-1',
      name: 'Marathon',
      race_type: 'time_based',
      status: 'active',
    }
    ;(timingApi.getLeaderboard as Mock).mockResolvedValue({
      data: {
        data: [
          {
            position: 1,
            participant_id: 'p1',
            bib_number: '7',
            first_name: 'Alex',
            last_name: 'Runner',
            location: 'Houghton MI',
            total_time_seconds: 3661,
            status: 'finished',
          },
        ],
      },
    })
    ;(timingApi.getLive as Mock).mockResolvedValue({
      data: { race_id: 'race-1', records: [] },
    })
    ;(participantsApi.get as Mock).mockResolvedValue({
      data: {
        id: 'p1',
        race_id: 'race-1',
        bib_number: '7',
        first_name: 'Alex',
        last_name: 'Runner',
        gender: 'male',
        age: 27,
        status: 'finished',
      },
    })

    const router = createTestRouter()
    await router.push('/timing/evt-1/race/race-1')
    await router.isReady()

    mount(RaceDetails, {
      global: { plugins: [router] },
    })
    await flushPromises()

    expect(racesStore.fetchRace).toHaveBeenCalledWith('race-1')
    expect(timingApi.getLeaderboard).toHaveBeenCalledWith('race-1')
  })

  it('renders leaderboard tab with API data', async () => {
    racesStore.currentRace = {
      id: 'race-1',
      name: 'Marathon',
      race_type: 'time_based',
      status: 'finished',
    }
    ;(timingApi.getLeaderboard as Mock).mockResolvedValue({
      data: {
        data: [
          {
            position: 1,
            participant_id: 'p1',
            bib_number: '7',
            first_name: 'Alex',
            last_name: 'Runner',
            location: 'Houghton MI',
            total_time_seconds: 3661,
            status: 'finished',
          },
        ],
      },
    })
    ;(timingApi.getLive as Mock).mockResolvedValue({
      data: { race_id: 'race-1', records: [] },
    })
    ;(participantsApi.get as Mock).mockResolvedValue({
      data: {
        id: 'p1',
        race_id: 'race-1',
        bib_number: '7',
        first_name: 'Alex',
        last_name: 'Runner',
        gender: 'male',
        age: 27,
        status: 'finished',
      },
    })

    const router = createTestRouter()
    await router.push('/timing/evt-1/race/race-1')
    await router.isReady()

    const wrapper = mount(RaceDetails, {
      global: { plugins: [router] },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Leaderboard')
    expect(wrapper.text()).toContain('Race Flow')
    expect(wrapper.text()).toContain('Statistics')
    expect(wrapper.text()).toContain('Alex')
    expect(wrapper.text()).toContain('Houghton MI')
    expect(wrapper.text()).toContain('7')

    const scroll = wrapper.get('[data-testid="leaderboard-scroll"]')
    expect(scroll.classes()).toContain('table-scroll')
    expect(scroll.find('table.leaderboard-table').exists()).toBe(true)
  })

  it('shows certificate when a finished participant is selected', async () => {
    racesStore.currentRace = {
      id: 'race-1',
      name: 'Long XC',
      race_type: 'time_based',
      distance_km: 42,
      status: 'finished',
    }
    eventsStore.currentEvent = {
      id: 'evt-1',
      name: 'Copper Harbor Trails Fest',
      event_date: '2025-08-30',
      location: 'Copper Harbor, MI',
      logo_url: '/images/chtf-2025-logo.png',
    }
    ;(timingApi.getLeaderboard as Mock).mockResolvedValue({
      data: {
        data: [
          {
            position: 1,
            participant_id: 'p1',
            bib_number: '788',
            first_name: 'Peter',
            last_name: 'Karinen',
            location: 'Tucson AZ',
            total_time_seconds: 7829,
            status: 'finished',
          },
        ],
      },
    })
    ;(timingApi.getLive as Mock).mockResolvedValue({
      data: { race_id: 'race-1', records: [] },
    })
    ;(participantsApi.get as Mock).mockResolvedValue({
      data: {
        id: 'p1',
        race_id: 'race-1',
        bib_number: '788',
        first_name: 'Peter',
        last_name: 'Karinen',
        gender: 'male',
        age: 27,
        location: 'Tucson AZ',
        status: 'finished',
      },
    })

    const router = createTestRouter()
    await router.push('/timing/evt-1/race/race-1')
    await router.isReady()

    const wrapper = mount(RaceDetails, {
      global: { plugins: [router] },
    })
    await flushPromises()

    await wrapper.find('tbody tr').trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-testid="result-certificate"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('Preliminary Results:')
    expect(wrapper.text()).toContain('Tucson AZ')
    expect(wrapper.text()).toContain('Save image')
    expect(wrapper.text()).toContain('Save square image')
    expect(wrapper.find('[data-testid="save-certificate-image"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="save-social-square-image"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('Inferior Timing')
    expect(wrapper.text()).toContain('Compare in Race Flow')
    expect(wrapper.find('[data-testid="inferior-timing-link"]').exists()).toBe(true)
    expect(wrapper.find('.event-logo-image').exists()).toBe(true)
    expect(participantsApi.get).toHaveBeenCalledWith('p1')
  })

  it('binds highlightParticipantId as v-model on the race flow chart', async () => {
    racesStore.currentRace = {
      id: 'race-1',
      name: 'Marathon',
      race_type: 'time_based',
      status: 'active',
    }
    ;(timingApi.getLeaderboard as Mock).mockResolvedValue({ data: { data: [] } })
    ;(timingApi.getLive as Mock).mockResolvedValue({
      data: { race_id: 'race-1', records: [] },
    })

    const router = createTestRouter()
    await router.push('/timing/evt-1/race/race-1')
    await router.isReady()

    const wrapper = mount(RaceDetails, {
      global: {
        plugins: [router],
        stubs: {
          RaceFlowChart: {
            name: 'RaceFlowChart',
            props: [
              'raceId',
              'raceStatus',
              'raceStartTime',
              'raceType',
              'durationMinutes',
              'highlightParticipantId',
            ],
            template: '<div data-testid="race-flow-chart-stub" />',
          },
        },
      },
    })
    await flushPromises()

    const raceFlowTab = wrapper.findAll('.tab').find((tab) => tab.text() === 'Race Flow')
    await raceFlowTab?.trigger('click')
    await flushPromises()

    const chart = wrapper.findComponent({ name: 'RaceFlowChart' })
    expect(chart.exists()).toBe(true)

    await chart.vm.$emit('update:highlightParticipantId', 'p1')
    await flushPromises()
    expect(chart.props('highlightParticipantId')).toBe('p1')

    await chart.vm.$emit('update:highlightParticipantId', undefined)
    await flushPromises()
    expect(chart.props('highlightParticipantId')).toBeUndefined()
  })

  describe('leaderboard category filters', () => {
    const categorizedLeaderboard = [
      {
        position: 1,
        participant_id: 'p1',
        bib_number: '1',
        first_name: 'Alex',
        last_name: 'Expert',
        location: 'Houghton MI',
        category_key: 'expert_men',
        total_time_seconds: 3600,
        status: 'finished',
      },
      {
        position: 2,
        participant_id: 'p2',
        bib_number: '2',
        first_name: 'Blake',
        last_name: 'West',
        location: 'Calumet MI',
        category_key: 'expert_women',
        total_time_seconds: 3700,
        status: 'finished',
      },
      {
        position: 3,
        participant_id: 'p3',
        bib_number: '3',
        first_name: 'Casey',
        last_name: 'Int',
        location: 'Copper Harbor MI',
        category_key: 'intermediate_women',
        total_time_seconds: 3800,
        status: 'finished',
      },
    ]

    async function mountWithLeaderboard(
      entries: typeof categorizedLeaderboard,
    ) {
      racesStore.currentRace = {
        id: 'race-1',
        name: 'Marathon',
        race_type: 'time_based',
        status: 'finished',
      }
      ;(timingApi.getLeaderboard as Mock).mockResolvedValue({ data: { data: entries } })
      ;(timingApi.getLive as Mock).mockResolvedValue({
        data: { race_id: 'race-1', records: [] },
      })

      const router = createTestRouter()
      await router.push('/timing/evt-1/race/race-1')
      await router.isReady()

      const wrapper = mount(RaceDetails, {
        global: { plugins: [router] },
      })
      await flushPromises()
      return wrapper
    }

    it('renders filters on leaderboard tab', async () => {
      const wrapper = await mountWithLeaderboard(categorizedLeaderboard)

      expect(wrapper.find('[data-testid="leaderboard-category-filters"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="lb-filter-skill"]').exists()).toBe(true)
      expect(wrapper.find('[data-testid="lb-filter-gender"]').exists()).toBe(true)
    })

    it('filters to Expert Women with renumbered position', async () => {
      const wrapper = await mountWithLeaderboard(categorizedLeaderboard)

      await wrapper.find('[data-testid="lb-filter-skill-expert"]').trigger('click')
      await wrapper.find('[data-testid="lb-filter-gender-women"]').trigger('click')
      await flushPromises()

      const rows = wrapper.findAll('tbody tr')
      expect(rows).toHaveLength(1)
      expect(rows[0]!.text()).toContain('Blake')
      expect(rows[0]!.text()).toContain('West')
      expect(rows[0]!.find('td').text()).toBe('1')
    })

    it('shows empty message when filter matches no racers', async () => {
      const wrapper = await mountWithLeaderboard([
        {
          position: 1,
          participant_id: 'p1',
          bib_number: '1',
          first_name: 'Alex',
          last_name: 'Expert',
          location: 'Houghton MI',
          category_key: 'expert_men',
          total_time_seconds: 3600,
          status: 'finished',
        },
      ])

      await wrapper.find('[data-testid="lb-filter-skill-expert"]').trigger('click')
      await wrapper.find('[data-testid="lb-filter-gender-women"]').trigger('click')
      await flushPromises()

      expect(wrapper.text()).toContain('No racers match')
      expect(wrapper.find('[data-testid="leaderboard-scroll"]').exists()).toBe(false)
    })
  })
})
