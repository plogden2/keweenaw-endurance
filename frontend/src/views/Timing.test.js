import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import Timing from './Timing.vue'
import { setupPinia, createTestRouter } from '../test/helpers.js'
import { useEventsStore } from '../stores/events.js'

vi.mock('../stores/events.js', async () => {
  const actual = await vi.importActual('../stores/events.js')
  return {
    ...actual,
    useEventsStore: vi.fn(),
  }
})

describe('Timing.vue', () => {
  let eventsStore

  beforeEach(() => {
    setupPinia()
    eventsStore = {
      events: [],
      activeEvents: [],
      pastEvents: [],
      loading: false,
      error: null,
      fetchEvents: vi.fn(),
    }
    useEventsStore.mockReturnValue(eventsStore)
  })

  it('fetches events on mount', async () => {
    const router = createTestRouter()
    mount(Timing, {
      global: { plugins: [router] },
    })
    await flushPromises()
    expect(eventsStore.fetchEvents).toHaveBeenCalledWith({ limit: 100 })
  })

  it('renders active and past event tables', async () => {
    eventsStore.activeEvents = [
      { id: '1', name: 'Summer Run', event_date: '2024-06-15', status: 'active' },
    ]
    eventsStore.pastEvents = [
      { id: '2', name: 'Spring Run', event_date: '2024-05-01', status: 'completed' },
    ]

    const router = createTestRouter()
    const wrapper = mount(Timing, {
      global: { plugins: [router] },
    })

    expect(wrapper.text()).toContain('Active Events')
    expect(wrapper.text()).toContain('Past Events')
    expect(wrapper.text()).toContain('Summer Run')
    expect(wrapper.text()).toContain('Spring Run')
  })

  it('links events to event details route', async () => {
    eventsStore.activeEvents = [
      { id: 'evt-1', name: 'Trail Day', event_date: '2024-07-01', status: 'active' },
    ]

    const router = createTestRouter()
    const wrapper = mount(Timing, {
      global: { plugins: [router] },
    })

    const link = wrapper.find('a.table-row')
    expect(link.attributes('href')).toBe('/timing/evt-1')
  })
})
