import 'fake-indexeddb/auto'
import { beforeEach, describe, expect, it } from 'vitest'
import type { ManualTimingEntryPayload } from '@/types/models'
import {
  DB_NAME,
  addPendingRecord,
  deleteDatabase,
  getAllPendingRecords,
  getPendingCount,
  removePendingRecord,
  updatePendingRecord,
} from './timingStorage'

const samplePayload = (): ManualTimingEntryPayload => ({
  race_id: 'race-1',
  checkpoint_id: 'cp-1',
  bib_number: '42',
  timestamp: '2024-06-01T10:00:00.000Z',
})

describe('timingStorage', () => {
  beforeEach(async () => {
    await deleteDatabase()
  })

  it('opens the IndexedDB database with pending-records store', async () => {
    await addPendingRecord(samplePayload())
    const dbs = await indexedDB.databases()
    expect(dbs.some((db) => db.name === DB_NAME)).toBe(true)
  })

  it('adds a pending timing record with generated id and pending_sync status', async () => {
    const record = await addPendingRecord(samplePayload())

    expect(record.id).toBeTruthy()
    expect(record.payload).toEqual(samplePayload())
    expect(record.sync_status).toBe('pending_sync')
    expect(record.created_at).toBeTruthy()
  })

  it('retrieves all pending records ordered by creation time', async () => {
    const first = await addPendingRecord(samplePayload())
    const second = await addPendingRecord({
      ...samplePayload(),
      bib_number: '99',
    })

    const records = await getAllPendingRecords()
    expect(records).toHaveLength(2)
    expect(records.map((r) => r.id)).toEqual([first.id, second.id])
  })

  it('returns pending count for unsynced records', async () => {
    expect(await getPendingCount()).toBe(0)
    await addPendingRecord(samplePayload())
    await addPendingRecord({ ...samplePayload(), bib_number: '7' })
    expect(await getPendingCount()).toBe(2)
  })

  it('updates sync status and error message on a record', async () => {
    const record = await addPendingRecord(samplePayload())
    await updatePendingRecord(record.id, {
      sync_status: 'failed_sync',
      last_error: 'Network error',
    })

    const records = await getAllPendingRecords()
    expect(records[0].sync_status).toBe('failed_sync')
    expect(records[0].last_error).toBe('Network error')
  })

  it('removes a record after successful sync', async () => {
    const record = await addPendingRecord(samplePayload())
    await removePendingRecord(record.id)

    expect(await getAllPendingRecords()).toHaveLength(0)
    expect(await getPendingCount()).toBe(0)
  })
})
