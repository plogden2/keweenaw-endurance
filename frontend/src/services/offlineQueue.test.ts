import 'fake-indexeddb/auto'
import { beforeEach, describe, expect, it, vi, type Mock } from 'vitest'
import type { ManualTimingEntryPayload } from '@/types/models'
import { rfidApi, scansApi, timingRecordsApi } from './api'
import { deleteDatabase, getAllPendingScans, getDisplayCache, setDisplayCache } from './timingStorage'
import {
  enqueue,
  enqueueKaraoke,
  enqueueScan,
  getLocalPendingCount,
  initOfflineQueue,
  isOnline,
  syncAll,
} from './offlineQueue'

vi.mock('./api', () => ({
  rfidApi: {
    manualEntry: vi.fn(),
  },
  scansApi: {
    postScan: vi.fn(),
  },
  timingRecordsApi: {
    karaokeBonus: vi.fn(),
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

  it('enqueues RFID scans when API unreachable and replays on sync', async () => {
    ;(scansApi.postScan as Mock).mockRejectedValue(new Error('unreachable'))

    const result = await enqueueScan('evt-1', {
      tag_uid: 'DEMO-TAG-0001',
      device_id: 'laptop-finish-1',
      local_timestamp: '2026-08-01T12:00:00.000Z',
    })

    expect(result.status).toBe('queued')
    expect(await getAllPendingScans()).toHaveLength(1)

    ;(scansApi.postScan as Mock).mockResolvedValue({
      data: { result: 'lap', participant_name: 'Alex', lap_count: 1 },
    })
    const sync = await syncAll()
    expect(sync.synced).toBeGreaterThanOrEqual(1)
    expect(await getAllPendingScans()).toHaveLength(0)
  })

  it('posts scans immediately when online and API succeeds', async () => {
    ;(scansApi.postScan as Mock).mockResolvedValue({
      data: { result: 'lap', participant_name: 'Alex', lap_count: 2 },
    })

    const result = await enqueueScan('evt-1', {
      tag_uid: 'DEMO-TAG-0001',
      device_id: 'd1',
      local_timestamp: '2026-08-01T12:00:00.000Z',
    })

    expect(result.status).toBe('synced')
    expect(result.scan).toMatchObject({ result: 'lap', lap_count: 2 })
    expect(await getAllPendingScans()).toHaveLength(0)
  })

  it('enqueues karaoke bonuses when API unreachable and replays on sync', async () => {
    ;(timingRecordsApi.karaokeBonus as Mock).mockRejectedValue(new Error('down'))

    const queued = await enqueueKaraoke('lap-99')
    expect(queued).toBe('queued')

    ;(timingRecordsApi.karaokeBonus as Mock).mockResolvedValue({ data: { id: 'bonus-1' } })
    const sync = await syncAll()
    expect(sync.synced).toBeGreaterThanOrEqual(1)
    expect(timingRecordsApi.karaokeBonus).toHaveBeenCalledWith('lap-99')
  })

  it('builds provisional lap feedback from display cache when scan is queued', async () => {
    await setDisplayCache({
      event_id: 'evt-1',
      event_name: 'Bluffet',
      races: [{ id: 'r1', name: '12 Hour' }],
      tags: {
        'DEMO-TAG-0001': {
          participant_name: 'Alex Rivera',
          bib_number: '12',
          race_name: '12 Hour',
        },
      },
    })
    Object.defineProperty(navigator, 'onLine', { configurable: true, value: false })

    const result = await enqueueScan('evt-1', {
      tag_uid: 'DEMO-TAG-0001',
      device_id: 'd1',
      local_timestamp: '2026-08-01T12:00:00.000Z',
    })

    expect(result.status).toBe('queued')
    expect(result.scan).toMatchObject({
      result: 'lap',
      participant_name: 'Alex Rivera',
      bib_number: '12',
      race_name: '12 Hour',
    })
    expect(await getDisplayCache()).not.toBeNull()
  })
})
