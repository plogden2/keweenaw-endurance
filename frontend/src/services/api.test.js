import { describe, it, expect, vi, beforeEach } from 'vitest'
import { apiClient, eventsApi, racesApi, participantsApi } from './api.js'

vi.mock('axios', () => {
  const create = vi.fn(() => ({
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
    defaults: {
      headers: { 'Content-Type': 'application/json' },
    },
  }))
  return { default: { create } }
})

describe('api client', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('configures JSON content type on the shared client', () => {
    expect(apiClient.defaults.headers['Content-Type']).toBe(
      'application/json',
    )
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
    apiClient.get.mockResolvedValue({ data: { data: [], total: 0 } })
    await eventsApi.list({ page: 2, limit: 10 })
    expect(apiClient.get).toHaveBeenCalledWith('/api/events', {
      params: { page: 2, limit: 10 },
    })
  })

  it('gets a single event by id', async () => {
    apiClient.get.mockResolvedValue({ data: { id: 'evt-1' } })
    await eventsApi.get('evt-1')
    expect(apiClient.get).toHaveBeenCalledWith('/api/events/evt-1')
  })

  it('creates, updates, and deletes events', async () => {
    const payload = { name: 'Trail Run' }
    apiClient.post.mockResolvedValue({ data: payload })
    apiClient.put.mockResolvedValue({ data: payload })
    apiClient.delete.mockResolvedValue({ data: null })

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
    apiClient.get.mockResolvedValue({ data: { data: [] } })
    await racesApi.list({ event_id: 'evt-1' })
    expect(apiClient.get).toHaveBeenCalledWith('/api/races', {
      params: { event_id: 'evt-1' },
    })
  })

  it('gets, creates, updates, and deletes races', async () => {
    const payload = { name: '50K', event_id: 'evt-1' }
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
    apiClient.get.mockResolvedValue({ data: { data: [] } })
    await participantsApi.list({ race_id: 'race-1' })
    expect(apiClient.get).toHaveBeenCalledWith('/api/participants', {
      params: { race_id: 'race-1' },
    })
  })

  it('gets, creates, updates, and deletes participants', async () => {
    const payload = { bib_number: '42', race_id: 'race-1' }
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
