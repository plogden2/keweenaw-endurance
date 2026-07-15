import { describe, it, expect, vi, beforeEach, type Mock } from 'vitest'
import {
  apiClient,
  eventsApi,
  racesApi,
  participantsApi,
  raceParticipantsApi,
  rfidApi,
  scansApi,
  eventsLiveApi,
  rfidStreamUrl,
  setAuthToken,
} from './api'

vi.mock('axios', () => {
  const create = vi.fn(() => ({
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
    defaults: {
      headers: {
        'Content-Type': 'application/json',
        common: {} as Record<string, string>,
      },
    },
  }))
  return { default: { create } }
})

describe('api client', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('configures JSON content type on the shared client', () => {
    expect(apiClient.defaults.headers['Content-Type']).toBe('application/json')
  })

  it('exposes shared apiClient instance', () => {
    expect(apiClient).toBeDefined()
    expect(apiClient.get).toBeDefined()
  })
})

describe('eventsApi', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('lists events with pagination params', async () => {
    ;(apiClient.get as Mock).mockResolvedValue({ data: { data: [], total: 0 } })
    await eventsApi.list({ page: 2, limit: 10 })
    expect(apiClient.get).toHaveBeenCalledWith('/api/events', {
      params: { page: 2, limit: 10 },
    })
  })

  it('gets a single event by id', async () => {
    ;(apiClient.get as Mock).mockResolvedValue({ data: { id: 'evt-1' } })
    await eventsApi.get('evt-1')
    expect(apiClient.get).toHaveBeenCalledWith('/api/events/evt-1')
  })

  it('creates, updates, and deletes events', async () => {
    const payload = { name: 'Trail Run', event_date: '2024-06-01', status: 'upcoming' as const }
    ;(apiClient.post as Mock).mockResolvedValue({ data: payload })
    ;(apiClient.put as Mock).mockResolvedValue({ data: payload })
    ;(apiClient.delete as Mock).mockResolvedValue({ data: null })

    await eventsApi.create(payload)
    await eventsApi.update('evt-1', payload)
    await eventsApi.remove('evt-1')

    expect(apiClient.post).toHaveBeenCalledWith('/api/events', payload)
    expect(apiClient.put).toHaveBeenCalledWith('/api/events/evt-1', payload)
    expect(apiClient.delete).toHaveBeenCalledWith('/api/events/evt-1')
  })
})

describe('racesApi', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('lists races with optional event_id filter', async () => {
    ;(apiClient.get as Mock).mockResolvedValue({ data: { data: [] } })
    await racesApi.list({ event_id: 'evt-1' })
    expect(apiClient.get).toHaveBeenCalledWith('/api/races', {
      params: { event_id: 'evt-1' },
    })
  })

  it('gets, creates, updates, and deletes races', async () => {
    const payload = {
      name: '50K',
      event_id: 'evt-1',
      race_type: 'time_based' as const,
      status: 'scheduled' as const,
    }
    await racesApi.get('race-1')
    await racesApi.create(payload)
    await racesApi.update('race-1', payload)
    await racesApi.remove('race-1')

    expect(apiClient.get).toHaveBeenCalledWith('/api/races/race-1')
    expect(apiClient.post).toHaveBeenCalledWith('/api/races', payload)
    expect(apiClient.put).toHaveBeenCalledWith('/api/races/race-1', payload)
    expect(apiClient.delete).toHaveBeenCalledWith('/api/races/race-1')
  })
})

describe('participantsApi', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('lists participants with optional race_id filter', async () => {
    ;(apiClient.get as Mock).mockResolvedValue({ data: { data: [] } })
    await participantsApi.list({ race_id: 'race-1' })
    expect(apiClient.get).toHaveBeenCalledWith('/api/participants', {
      params: { race_id: 'race-1' },
    })
  })

  it('gets, creates, updates, and deletes participants', async () => {
    const payload = {
      bib_number: '42',
      race_id: 'race-1',
      first_name: 'Alex',
      last_name: 'Runner',
      status: 'registered' as const,
    }
    await participantsApi.get('p-1')
    await participantsApi.create(payload)
    await participantsApi.update('p-1', payload)
    await participantsApi.remove('p-1')

    expect(apiClient.get).toHaveBeenCalledWith('/api/participants/p-1')
    expect(apiClient.post).toHaveBeenCalledWith('/api/participants', payload)
    expect(apiClient.put).toHaveBeenCalledWith('/api/participants/p-1', payload)
    expect(apiClient.delete).toHaveBeenCalledWith('/api/participants/p-1')
  })
})

describe('raceParticipantsApi', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('lists and creates race participants and manages tags', async () => {
    ;(apiClient.get as Mock).mockResolvedValue({ data: { data: [] } })
    ;(apiClient.post as Mock).mockResolvedValue({ data: {} })

    await raceParticipantsApi.list('race-1', { q: 'alex', limit: 200 })
    await raceParticipantsApi.create('race-1', {
      first_name: 'E2E',
      last_name: 'Racer',
      category_id: 'cat-1',
    })
    await raceParticipantsApi.listTags('race-1', 'p-1')
    await raceParticipantsApi.addTag('race-1', 'p-1', 'TAG-A')
    await raceParticipantsApi.listCategories('race-1')

    expect(apiClient.get).toHaveBeenCalledWith('/api/races/race-1/participants', {
      params: { q: 'alex', limit: 200 },
    })
    expect(apiClient.post).toHaveBeenCalledWith('/api/races/race-1/participants', {
      first_name: 'E2E',
      last_name: 'Racer',
      category_id: 'cat-1',
    })
    expect(apiClient.get).toHaveBeenCalledWith('/api/races/race-1/participants/p-1/tags')
    expect(apiClient.post).toHaveBeenCalledWith('/api/races/race-1/participants/p-1/tags', {
      tag_uid: 'TAG-A',
    })
    expect(apiClient.get).toHaveBeenCalledWith('/api/races/race-1/categories', {
      params: { limit: 100 },
    })
  })
})

describe('RFID scanner APIs', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('posts event scans', async () => {
    ;(apiClient.post as Mock).mockResolvedValue({ data: { result: 'lap' } })
    await scansApi.postScan('evt-1', {
      tag_uid: 'TAG-1',
      device_id: 'laptop-1',
      local_timestamp: '2026-08-01T12:00:00Z',
    })
    expect(apiClient.post).toHaveBeenCalledWith('/api/events/evt-1/scans', {
      tag_uid: 'TAG-1',
      device_id: 'laptop-1',
      local_timestamp: '2026-08-01T12:00:00Z',
    })
  })

  it('writes tag with participant_id only', async () => {
    ;(apiClient.post as Mock).mockResolvedValue({ data: { id: 'p-1' } })
    await rfidApi.writeTag({ participant_id: 'p-1' })
    expect(apiClient.post).toHaveBeenCalledWith('/api/rfid/write-tag', {
      participant_id: 'p-1',
    })
  })

  it('fetches event live payload', async () => {
    ;(apiClient.get as Mock).mockResolvedValue({ data: { races: [] } })
    await eventsLiveApi.getLive('evt-1', { category_id: 'cat-1' })
    expect(apiClient.get).toHaveBeenCalledWith('/api/events/evt-1/live', {
      params: { category_id: 'cat-1' },
    })
  })

  it('builds websocket stream URLs and sets auth header', () => {
    expect(rfidStreamUrl('http://localhost:8080')).toBe('ws://localhost:8080/api/rfid/stream')
    expect(rfidStreamUrl('https://api.example.com')).toBe('wss://api.example.com/api/rfid/stream')
    setAuthToken('tok')
    expect(apiClient.defaults.headers.common.Authorization).toBe('Bearer tok')
    setAuthToken(null)
    expect(apiClient.defaults.headers.common.Authorization).toBeUndefined()
  })
})
