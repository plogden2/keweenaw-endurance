import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import EventDetails from './EventDetails.vue'
import { setupPinia, createTestRouter } from '../test/helpers.js'
import { useEventsStore } from '../stores/events.js'
import { useRacesStore } from '../stores/races.js'

vi.mock('../stores/events.js', async () => {
  const actual = await vi.importActual('../stores/events.js')
  return { ...actual, useEventsStore: vi.fn() }
})

vi.mock('../stores/races.js', async () => {
  const actual = await vi.importActual('../stores/races.js')
  return { ...actual, useRacesStore: vi.fn() }
})

describe('EventDetails.vue', () => {
  let eventsStore
  let racesStore

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
    useEventsStore.mockReturnValue(eventsStore)
    useRacesStore.mockReturnValue(racesStore)
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
    racesStore.races = [
      { id: 'race-1', name: '50K', status: 'active' },
    ]

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
