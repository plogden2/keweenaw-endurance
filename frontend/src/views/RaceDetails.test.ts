import { describe, it, expect, vi, beforeEach, type Mock } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import RaceDetails from './RaceDetails.vue'
import { setupPinia, createTestRouter } from '@/test/helpers'
import { useRacesStore } from '@/stores/races'
import { timingApi } from '@/services/api'

vi.mock('@/stores/races', async () => {
  const actual = await vi.importActual<typeof import('@/stores/races')>('@/stores/races')
  return { ...actual, useRacesStore: vi.fn() }
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
  }
})

describe('RaceDetails.vue', () => {
  let racesStore: {
    currentRace: Record<string, unknown> | null
    loading: boolean
    error: string | null
    fetchRace: Mock
  }

  beforeEach(() => {
    setupPinia()
    racesStore = {
      currentRace: null,
      loading: false,
      error: null,
      fetchRace: vi.fn(),
    }
    ;(useRacesStore as unknown as Mock).mockReturnValue(racesStore)
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
            total_time_seconds: 3661,
            status: 'finished',
          },
        ],
      },
    })
    ;(timingApi.getLive as Mock).mockResolvedValue({
      data: { race_id: 'race-1', records: [] },
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
            total_time_seconds: 3661,
            status: 'finished',
          },
        ],
      },
    })
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

    expect(wrapper.text()).toContain('Leaderboard')
    expect(wrapper.text()).toContain('Race Flow')
    expect(wrapper.text()).toContain('Statistics')
    expect(wrapper.text()).toContain('Alex')
    expect(wrapper.text()).toContain('7')
  })
})
