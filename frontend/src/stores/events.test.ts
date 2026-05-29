import { describe, it, expect, vi, beforeEach, type Mock } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useEventsStore } from './events'
import { useRacesStore } from './races'
import { useParticipantsStore } from './participants'
import { eventsApi, racesApi, participantsApi } from '@/services/api'

vi.mock('@/services/api', () => ({
  eventsApi: {
    list: vi.fn(),
    get: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    remove: vi.fn(),
  },
  racesApi: {
    list: vi.fn(),
    get: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    remove: vi.fn(),
  },
  participantsApi: {
    list: vi.fn(),
    get: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    remove: vi.fn(),
  },
}))

describe('useEventsStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('fetches and caches paginated events', async () => {
    const events = [
      { id: '1', name: 'Active Race', status: 'active' as const, event_date: '2024-06-01' },
      { id: '2', name: 'Done Race', status: 'completed' as const, event_date: '2024-05-01' },
    ]
    ;(eventsApi.list as Mock).mockResolvedValue({
      data: { data: events, total: 2, page: 1, limit: 20 },
    })

    const store = useEventsStore()
    await store.fetchEvents()

    expect(store.events).toEqual(events)
    expect(store.total).toBe(2)
    expect(store.loading).toBe(false)
    expect(store.error).toBeNull()
  })

  it('exposes active and past events from cached list', async () => {
    ;(eventsApi.list as Mock).mockResolvedValue({
      data: {
        data: [
          { id: '1', status: 'active', name: 'A', event_date: '2024-06-01' },
          { id: '2', status: 'completed', name: 'B', event_date: '2024-05-01' },
          { id: '3', status: 'upcoming', name: 'C', event_date: '2024-07-01' },
        ],
        total: 3,
      },
    })

    const store = useEventsStore()
    await store.fetchEvents()

    expect(store.activeEvents).toHaveLength(1)
    expect(store.pastEvents).toHaveLength(1)
  })

  it('fetches and caches a single event', async () => {
    const event = { id: 'evt-1', name: 'Copper Harbor', status: 'active' as const, event_date: '2024-08-01' }
    ;(eventsApi.get as Mock).mockResolvedValue({ data: event })

    const store = useEventsStore()
    await store.fetchEvent('evt-1')

    expect(store.currentEvent).toEqual(event)
  })

  it('records API errors', async () => {
    ;(eventsApi.list as Mock).mockRejectedValue(new Error('network error'))

    const store = useEventsStore()
    await store.fetchEvents()

    expect(store.error).toBe('network error')
    expect(store.loading).toBe(false)
  })
})

describe('useRacesStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('fetches and caches races for an event', async () => {
    const races = [{
      id: 'r1',
      name: '50K',
      event_id: 'evt-1',
      race_type: 'time_based' as const,
      status: 'scheduled' as const,
    }]
    ;(racesApi.list as Mock).mockResolvedValue({
      data: { data: races, total: 1, page: 1, limit: 20 },
    })

    const store = useRacesStore()
    await store.fetchRaces({ event_id: 'evt-1' })

    expect(store.races).toEqual(races)
    expect(store.total).toBe(1)
    expect(racesApi.list).toHaveBeenCalledWith({ event_id: 'evt-1' })
  })

  it('fetches and caches a single race', async () => {
    const race = {
      id: 'r1',
      name: 'Marathon',
      event_id: 'evt-1',
      race_type: 'time_based' as const,
      status: 'active' as const,
    }
    ;(racesApi.get as Mock).mockResolvedValue({ data: race })

    const store = useRacesStore()
    await store.fetchRace('r1')

    expect(store.currentRace).toEqual(race)
  })
})

describe('useParticipantsStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('fetches and caches participants for a race', async () => {
    const participants = [
      {
        id: 'p1',
        bib_number: '1',
        race_id: 'r1',
        first_name: 'A',
        last_name: 'B',
        status: 'registered' as const,
      },
      {
        id: 'p2',
        bib_number: '2',
        race_id: 'r1',
        first_name: 'C',
        last_name: 'D',
        status: 'registered' as const,
      },
    ]
    ;(participantsApi.list as Mock).mockResolvedValue({
      data: { data: participants, total: 2 },
    })

    const store = useParticipantsStore()
    await store.fetchParticipants({ race_id: 'r1' })

    expect(store.participants).toEqual(participants)
    expect(store.total).toBe(2)
    expect(participantsApi.list).toHaveBeenCalledWith({ race_id: 'r1' })
  })

  it('fetches and caches a single participant', async () => {
    const participant = {
      id: 'p1',
      bib_number: '42',
      race_id: 'r1',
      first_name: 'Alex',
      last_name: 'Runner',
      status: 'registered' as const,
    }
    ;(participantsApi.get as Mock).mockResolvedValue({ data: participant })

    const store = useParticipantsStore()
    await store.fetchParticipant('p1')

    expect(store.currentParticipant).toEqual(participant)
  })
})
