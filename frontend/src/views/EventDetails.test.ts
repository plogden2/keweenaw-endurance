import { describe, it, expect, vi, beforeEach, type Mock } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import EventDetails from './EventDetails.vue'
import { setupPinia, createTestRouter } from '@/test/helpers'
import { useEventsStore } from '@/stores/events'
import { useRacesStore } from '@/stores/races'

vi.mock('@/stores/events', async () => {
  const actual = await vi.importActual<typeof import('@/stores/events')>('@/stores/events')
  return { ...actual, useEventsStore: vi.fn() }
})

vi.mock('@/stores/races', async () => {
  const actual = await vi.importActual<typeof import('@/stores/races')>('@/stores/races')
  return { ...actual, useRacesStore: vi.fn() }
})

describe('EventDetails.vue', () => {
  let eventsStore: {
    currentEvent: Record<string, unknown> | null
    loading: boolean
    error: string | null
    fetchEvent: Mock
  }
  let racesStore: {
    races: Array<{ id: string; name: string; status: string }>
    loading: boolean
    fetchRaces: Mock
  }

  beforeEach(() => {
    setupPinia()
    eventsStore = {
      currentEvent: null,
      loading: false,
      error: null,
      fetchEvent: vi.fn(),
    }
    racesStore = {
      races: [],
      loading: false,
      fetchRaces: vi.fn(),
    }
    ;(useEventsStore as unknown as Mock).mockReturnValue(eventsStore)
    ;(useRacesStore as unknown as Mock).mockReturnValue(racesStore)
  })

  it('loads event and races for route param', async () => {
    const router = createTestRouter()
    await router.push('/timing/evt-1')
    await router.isReady()

    mount(EventDetails, {
      global: { plugins: [router] },
    })
    await flushPromises()

    expect(eventsStore.fetchEvent).toHaveBeenCalledWith('evt-1')
    expect(racesStore.fetchRaces).toHaveBeenCalledWith({ event_id: 'evt-1' })
  })

  it('displays event name and race links', async () => {
    eventsStore.currentEvent = {
      id: 'evt-1',
      name: 'Copper Harbor Classic',
      event_date: '2024-08-01',
      location: 'Copper Harbor',
    }
    racesStore.races = [{ id: 'race-1', name: '50K', status: 'active' }]

    const router = createTestRouter()
    await router.push('/timing/evt-1')
    await router.isReady()

    const wrapper = mount(EventDetails, {
      global: { plugins: [router] },
    })

    expect(wrapper.text()).toContain('Copper Harbor Classic')
    expect(wrapper.text()).toContain('50K')
    const link = wrapper.find('a.race-link')
    expect(link.attributes('href')).toBe('/timing/evt-1/race/race-1')
  })
})
