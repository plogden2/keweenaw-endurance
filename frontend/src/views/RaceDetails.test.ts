import { describe, it, expect, vi, beforeEach, type Mock } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import RaceDetails from './RaceDetails.vue'
import { setupPinia, createTestRouter } from '@/test/helpers'
import { useRacesStore } from '@/stores/races'
import { useEventsStore } from '@/stores/events'
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
    expect(wrapper.find('[data-testid="save-certificate-image"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('Inferior Timing')
    expect(wrapper.text()).toContain('Compare in Race Flow')
    expect(wrapper.find('[data-testid="inferior-timing-link"]').exists()).toBe(true)
    expect(wrapper.find('.event-logo-image').exists()).toBe(true)
    expect(participantsApi.get).toHaveBeenCalledWith('p1')
  })
})
