import type { ManualTimingEntryPayload } from '@/types/models'
import {
  rfidApi,
  scansApi,
  syncApi,
  timingRecordsApi,
  type PostScanPayload,
  type ScanResult,
} from './api'
import * as timingStorage from './timingStorage'

export type EnqueueResult = 'synced' | 'queued'

export interface EnqueueScanResult {
  status: EnqueueResult
  scan?: ScanResult
}

export interface SyncResult {
  synced: number
  failed: number
}

const onlineListeners: Array<() => void> = []
const pendingListeners: Array<(count: number) => void> = []
let initialized = false

export function isOnline(): boolean {
  return typeof navigator !== 'undefined' ? navigator.onLine : true
}

async function notifyPending(): Promise<void> {
  const count = await getLocalPendingCount()
  pendingListeners.forEach((cb) => cb(count))
}

export function initOfflineQueue(): void {
  if (initialized || typeof window === 'undefined') {
    return
  }

  initialized = true
  window.addEventListener('online', () => {
    void syncAll().then(() => {
      onlineListeners.forEach((callback) => callback())
    })
  })
}

export function onOnline(callback: () => void): () => void {
  onlineListeners.push(callback)
  return () => {
    const index = onlineListeners.indexOf(callback)
    if (index >= 0) {
      onlineListeners.splice(index, 1)
    }
  }
}

export function onPendingChange(callback: (count: number) => void): () => void {
  pendingListeners.push(callback)
  return () => {
    const index = pendingListeners.indexOf(callback)
    if (index >= 0) {
      pendingListeners.splice(index, 1)
    }
  }
}

export async function enqueue(payload: ManualTimingEntryPayload): Promise<EnqueueResult> {
  if (isOnline()) {
    try {
      await rfidApi.manualEntry(payload)
      return 'synced'
    } catch {
      // Fall through to local queue when the API is unreachable.
    }
  }

  await timingStorage.addPendingRecord(payload)
  await notifyPending()
  return 'queued'
}

export async function enqueueScan(
  eventId: string,
  payload: PostScanPayload,
): Promise<EnqueueScanResult> {
  if (isOnline()) {
    try {
      const { data } = await scansApi.postScan(eventId, payload)
      return { status: 'synced', scan: data }
    } catch {
      // Fall through to WAQ when local API blips.
    }
  }

  await timingStorage.addPendingScan(eventId, payload)
  await notifyPending()
  const scan = await timingStorage.provisionalScanFromCache(payload.tag_uid)
  return { status: 'queued', scan: scan ?? undefined }
}

export async function enqueueKaraoke(timingRecordId: string): Promise<EnqueueResult> {
  if (isOnline()) {
    try {
      await timingRecordsApi.karaokeBonus(timingRecordId)
      return 'synced'
    } catch {
      // Fall through to WAQ.
    }
  }

  await timingStorage.addPendingKaraoke({
    timing_record_id: timingRecordId,
    source_lap_id: timingRecordId,
  })
  await notifyPending()
  return 'queued'
}

export async function syncAll(): Promise<SyncResult> {
  let synced = 0
  let failed = 0

  const pending = await timingStorage.getAllPendingRecords()
  for (const record of pending) {
    try {
      await rfidApi.manualEntry(record.payload)
      await timingStorage.removePendingRecord(record.id)
      synced++
    } catch (err) {
      failed++
      await timingStorage.updatePendingRecord(record.id, {
        sync_status: 'failed_sync',
        last_error: err instanceof Error ? err.message : 'Sync failed',
      })
    }
  }

  const scans = await timingStorage.getAllPendingScans()
  for (const record of scans) {
    try {
      await scansApi.postScan(record.event_id, record.payload)
      await timingStorage.removePendingScan(record.id)
      synced++
    } catch (err) {
      failed++
      await timingStorage.updatePendingScan(record.id, {
        sync_status: 'failed_sync',
        last_error: err instanceof Error ? err.message : 'Sync failed',
      })
    }
  }

  const karaoke = await timingStorage.getAllPendingKaraoke()
  for (const record of karaoke) {
    try {
      await timingRecordsApi.karaokeBonus(record.payload.timing_record_id)
      await timingStorage.removePendingKaraoke(record.id)
      synced++
    } catch (err) {
      failed++
      await timingStorage.updatePendingKaraoke(record.id, {
        sync_status: 'failed_sync',
        last_error: err instanceof Error ? err.message : 'Sync failed',
      })
    }
  }

  // Best-effort station → hosted push when browser is online again.
  if (isOnline()) {
    try {
      await syncApi.push()
      await syncApi.pull()
    } catch {
      // Hosted may be unset or unreachable; local WAQ still cleared above.
    }
  }

  await notifyPending()
  return { synced, failed }
}

export async function getLocalPendingCount(): Promise<number> {
  return timingStorage.getWAQPendingCount()
}
