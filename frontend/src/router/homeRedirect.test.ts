import { beforeEach, describe, expect, it, vi } from 'vitest'

const BLUFFET_EVENT_ID = 'a1b2c3'

const { eventsApiList, eventsLiveApiGetLive, racesApiList, stationFetchCurrent } = vi.hoisted(
  () => ({
    eventsApiList: vi.fn(),
    eventsLiveApiGetLive: vi.fn(),
    racesApiList: vi.fn(),
    stationFetchCurrent: vi.fn(),
  }),
)

vi.mock('@/services/api', async () => {
  const actual = await vi.importActual<typeof import('@/services/api')>('@/services/api')
  return {
    ...actual,
    eventsApi: {
      ...actual.eventsApi,
      list: eventsApiList,
    },
    eventsLiveApi: {
      getLive: eventsLiveApiGetLive,
    },
    racesApi: {
      ...actual.racesApi,
      list: racesApiList,
    },
  }
})

vi.mock('@/stores/station', () => ({
  useStationStore: () => ({
    eventId: null,
    fetchCurrent: stationFetchCurrent,
  }),
}))

/** Payloads the removed home redirect guard consulted — would force event-live if reintroduced. */
function mockActiveBluffetRaceDiscovery() {
  eventsApiList.mockResolvedValue({
    data: {
      data: [
        {
          id: BLUFFET_EVENT_ID,
          name: 'All You Can East Bluffet',
          event_date: '2026-08-01',
          status: 'active',
        },
      ],
    },
  })
  eventsLiveApiGetLive.mockResolvedValue({
    data: {
      races: [{ id: 'race-1', status: 'active' }],
    },
  })
  racesApiList.mockResolvedValue({
    data: {
      data: [{ id: 'race-1', status: 'active' }],
    },
  })
  stationFetchCurrent.mockResolvedValue(undefined)
}

describe('router home redirect (production)', () => {
  beforeEach(() => {
    vi.resetModules()
    eventsApiList.mockReset()
    eventsLiveApiGetLive.mockReset()
    racesApiList.mockReset()
    stationFetchCurrent.mockReset()
    mockActiveBluffetRaceDiscovery()
  })

  it('does not redirect / to event-live when active races exist', async () => {
    const { default: router } = await import('@/router')

    await router.push('/')
    await router.isReady()

    expect(router.currentRoute.value.name).toBe('home')
    expect(router.currentRoute.value.path).toBe('/')
  })
})
