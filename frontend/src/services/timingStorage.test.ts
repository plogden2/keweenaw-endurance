import 'fake-indexeddb/auto'
import { beforeEach, describe, expect, it } from 'vitest'
import type { ManualTimingEntryPayload } from '@/types/models'
import {
  DB_NAME,
  addPendingKaraoke,
  addPendingRecord,
  addPendingScan,
  deleteDatabase,
  getAllPendingKaraoke,
  getAllPendingRecords,
  getAllPendingScans,
  getDisplayCache,
  getPendingCount,
  getWAQPendingCount,
  removePendingKaraoke,
  removePendingRecord,
  removePendingScan,
  setDisplayCache,
  updatePendingRecord,
} from './timingStorage'

const samplePayload = (): ManualTimingEntryPayload => ({
  race_id: 'race-1',
  checkpoint_id: 'cp-1',
  bib_number: '42',
  timestamp: '2024-06-01T10:00:00.000Z',
})

describe('timingStorage (WAQ / UI cache only)', () => {
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

  it('queues pending RFID scans separately from manual timing records', async () => {
    const scan = await addPendingScan('evt-1', {
      tag_uid: 'DEMO-TAG-0001',
      device_id: 'laptop-finish-1',
      local_timestamp: '2026-08-01T12:00:00.000Z',
    })

    expect(scan.id).toBeTruthy()
    expect(scan.event_id).toBe('evt-1')
    expect(scan.payload.tag_uid).toBe('DEMO-TAG-0001')
    expect(scan.sync_status).toBe('pending_sync')
    expect(await getAllPendingScans()).toHaveLength(1)
    expect(await getAllPendingRecords()).toHaveLength(0)
  })

  it('queues pending karaoke bonuses for replay', async () => {
    const bonus = await addPendingKaraoke({
      timing_record_id: 'lap-1',
      source_lap_id: 'lap-1',
    })

    expect(bonus.payload.timing_record_id).toBe('lap-1')
    expect(await getAllPendingKaraoke()).toHaveLength(1)
    await removePendingKaraoke(bonus.id)
    expect(await getAllPendingKaraoke()).toHaveLength(0)
  })

  it('stores minimal display cache for event/race labels (not system of record)', async () => {
    await setDisplayCache({
      event_id: 'evt-1',
      event_name: 'All You Can East Bluffet',
      races: [{ id: 'r-12', name: '12 Hour' }],
      tags: {
        'DEMO-TAG-0001': {
          participant_name: 'Alex Rivera',
          bib_number: '12',
          race_name: '12 Hour',
        },
      },
    })

    const cache = await getDisplayCache()
    expect(cache?.event_name).toBe('All You Can East Bluffet')
    expect(cache?.races[0].name).toBe('12 Hour')
    expect(cache?.tags['DEMO-TAG-0001'].participant_name).toBe('Alex Rivera')
  })

  it('counts all WAQ pending items (manual + scans + karaoke)', async () => {
    await addPendingRecord(samplePayload())
    await addPendingScan('evt-1', {
      tag_uid: 'T1',
      device_id: 'd1',
      local_timestamp: '2026-08-01T12:00:00.000Z',
    })
    await addPendingKaraoke({ timing_record_id: 'lap-1' })
    expect(await getWAQPendingCount()).toBe(3)
    await removePendingScan((await getAllPendingScans())[0].id)
    expect(await getWAQPendingCount()).toBe(2)
  })
})
