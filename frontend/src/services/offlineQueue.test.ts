import 'fake-indexeddb/auto'
import { beforeEach, describe, expect, it, vi, type Mock } from 'vitest'
import type { ManualTimingEntryPayload } from '@/types/models'
import { rfidApi } from './api'
import { deleteDatabase } from './timingStorage'
import {
  enqueue,
  getLocalPendingCount,
  initOfflineQueue,
  isOnline,
  syncAll,
} from './offlineQueue'

vi.mock('./api', () => ({
  rfidApi: {
    manualEntry: vi.fn(),
  },
}))

const samplePayload = (): ManualTimingEntryPayload => ({
  race_id: 'race-1',
  checkpoint_id: 'cp-1',
  bib_number: '42',
  timestamp: '2024-06-01T10:00:00.000Z',
})

describe('offlineQueue', () => {
  beforeEach(async () => {
    vi.clearAllMocks()
    await deleteDatabase()
    Object.defineProperty(navigator, 'onLine', {
      configurable: true,
      value: true,
    })
  })

  it('reports online status from navigator.onLine', () => {
    Object.defineProperty(navigator, 'onLine', { configurable: true, value: true })
    expect(isOnline()).toBe(true)

    Object.defineProperty(navigator, 'onLine', { configurable: true, value: false })
    expect(isOnline()).toBe(false)
  })

  it('submits directly to the API when online and request succeeds', async () => {
    ;(rfidApi.manualEntry as Mock).mockResolvedValue({ data: { id: 'rec-1' } })

    const result = await enqueue(samplePayload())

    expect(result).toBe('synced')
    expect(rfidApi.manualEntry).toHaveBeenCalledWith(samplePayload())
    expect(await getLocalPendingCount()).toBe(0)
  })

  it('queues locally when offline', async () => {
    Object.defineProperty(navigator, 'onLine', { configurable: true, value: false })

    const result = await enqueue(samplePayload())

    expect(result).toBe('queued')
    expect(rfidApi.manualEntry).not.toHaveBeenCalled()
    expect(await getLocalPendingCount()).toBe(1)
  })

  it('queues locally when online but API request fails', async () => {
    ;(rfidApi.manualEntry as Mock).mockRejectedValue(new Error('Server unavailable'))

    const result = await enqueue(samplePayload())

    expect(result).toBe('queued')
    expect(await getLocalPendingCount()).toBe(1)
  })

  it('syncs all pending records and removes successful ones', async () => {
    Object.defineProperty(navigator, 'onLine', { configurable: true, value: false })
    await enqueue(samplePayload())
    await enqueue({ ...samplePayload(), bib_number: '7' })
    Object.defineProperty(navigator, 'onLine', { configurable: true, value: true })
    ;(rfidApi.manualEntry as Mock).mockResolvedValue({ data: { id: 'rec-1' } })

    const result = await syncAll()

    expect(result).toEqual({ synced: 2, failed: 0 })
    expect(rfidApi.manualEntry).toHaveBeenCalledTimes(2)
    expect(await getLocalPendingCount()).toBe(0)
  })

  it('marks failed sync attempts and keeps records in storage', async () => {
    Object.defineProperty(navigator, 'onLine', { configurable: true, value: false })
    await enqueue(samplePayload())
    Object.defineProperty(navigator, 'onLine', { configurable: true, value: true })
    ;(rfidApi.manualEntry as Mock).mockRejectedValue(new Error('Sync failed'))

    const result = await syncAll()

    expect(result).toEqual({ synced: 0, failed: 1 })
    expect(await getLocalPendingCount()).toBe(1)
  })

  it('auto-syncs pending records when connection is restored', async () => {
    Object.defineProperty(navigator, 'onLine', { configurable: true, value: false })
    await enqueue(samplePayload())
    ;(rfidApi.manualEntry as Mock).mockResolvedValue({ data: { id: 'rec-1' } })

    initOfflineQueue()
    Object.defineProperty(navigator, 'onLine', { configurable: true, value: true })
    window.dispatchEvent(new Event('online'))

    await vi.waitFor(async () => {
      expect(await getLocalPendingCount()).toBe(0)
    })
  })
})
