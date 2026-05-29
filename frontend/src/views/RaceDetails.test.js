import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import RaceDetails from './RaceDetails.vue'
import { setupPinia, createTestRouter } from '../test/helpers.js'
import { useRacesStore } from '../stores/races.js'
import { timingApi } from '../services/api.js'

vi.mock('../stores/races.js', async () => {
  const actual = await vi.importActual('../stores/races.js')
  return { ...actual, useRacesStore: vi.fn() }
})

vi.mock('../services/api.js', async () => {
  const actual = await vi.importActual('../services/api.js')
  return {
    ...actual,
    timingApi: {
      getLeaderboard: vi.fn(),
      getResults: vi.fn(),
      getLive: vi.fn(),
    },
  }
})

describe('RaceDetails.vue', () => {
  let racesStore

  beforeEach(() => {
    setupPinia()
    racesStore = {
      currentRace: null,
      loading: false,
      error: null,
      fetchRace: vi.fn(),
    }
    useRacesStore.mockReturnValue(racesStore)
    vi.clearAllMocks()
  })

  it('loads race and leaderboard on mount', async () => {
    racesStore.currentRace = {
      id: 'race-1',
      name: 'Marathon',
      race_type: 'time_based',
      status: 'active',
    }
    timingApi.getLeaderboard.mockResolvedValue({
      data: {
        data: [
          {
            position: 1,
            participant_id: 'p1',
            bib_number: '7',
            first_name: 'Alex',
            last_name: 'Runner',
            total_time_seconds: 3661,
            status: 'finished',
          },
        ],
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
    timingApi.getLeaderboard.mockResolvedValue({
      data: {
        data: [
          {
            position: 1,
            participant_id: 'p1',
            bib_number: '7',
            first_name: 'Alex',
            last_name: 'Runner',
            total_time_seconds: 3661,
            status: 'finished',
          },
        ],
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
    expect(wrapper.text()).toContain('Alex')
    expect(wrapper.text()).toContain('7')
  })
})
