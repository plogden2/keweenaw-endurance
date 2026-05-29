import type { ManualTimingEntryPayload } from '@/types/models'
import { rfidApi } from './api'
import * as timingStorage from './timingStorage'

export type EnqueueResult = 'synced' | 'queued'

export interface SyncResult {
  synced: number
  failed: number
}

const onlineListeners: Array<() => void> = []
let initialized = false

export function isOnline(): boolean {
  return typeof navigator !== 'undefined' ? navigator.onLine : true
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
  return 'queued'
}

export async function syncAll(): Promise<SyncResult> {
  const pending = await timingStorage.getAllPendingRecords()
  let synced = 0
  let failed = 0

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

  return { synced, failed }
}

export async function getLocalPendingCount(): Promise<number> {
  return timingStorage.getPendingCount()
}
