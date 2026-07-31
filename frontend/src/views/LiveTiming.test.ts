import { describe, it, expect, vi, beforeEach, type Mock } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import LiveTimingRedirect from './LiveTimingRedirect.vue'
import { createTestRouter } from '@/test/helpers'
import { racesApi } from '@/services/api'

vi.mock('@/services/api', async () => {
  const actual = await vi.importActual<typeof import('@/services/api')>('@/services/api')
  return {
    ...actual,
    racesApi: { get: vi.fn() },
  }
})

describe('LiveTimingRedirect.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    ;(racesApi.get as Mock).mockResolvedValue({
      data: { id: 'race-1', event_id: 'event-1', name: '12 Hour' },
    })
  })

  it('redirects legacy race-scoped manual entry to event taps', async () => {
    const router = createTestRouter([
      {
        path: '/timing/live/:raceId',
        name: 'live-timing',
        component: LiveTimingRedirect,
      },
      {
        path: '/events/:eventId/taps',
        name: 'event-taps',
        component: { template: '<div>Event taps</div>' },
      },
    ])
    await router.push('/timing/live/race-1')
    await router.isReady()

    mount(LiveTimingRedirect, { global: { plugins: [router] } })
    await flushPromises()

    expect(racesApi.get).toHaveBeenCalledWith('race-1')
    expect(router.currentRoute.value.fullPath).toBe('/events/event-1/taps')
  })
})
