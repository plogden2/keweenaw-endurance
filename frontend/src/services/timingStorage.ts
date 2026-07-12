import type { ManualTimingEntryPayload } from '@/types/models'
import type { PostScanPayload, ScanResult } from './api'

export const DB_NAME = 'keweenaw-timing'
export const STORE_NAME = 'pending-records'
export const SCAN_STORE = 'pending-scans'
export const KARAOKE_STORE = 'pending-karaoke'
export const DISPLAY_CACHE_STORE = 'display-cache'
export const DISPLAY_CACHE_KEY = 'current'
/** Bump when adding WAQ stores (scans/karaoke/display-cache). */
export const DB_VERSION = 2

export interface PendingTimingRecord {
  id: string
  payload: ManualTimingEntryPayload
  created_at: string
  sync_status: 'pending_sync' | 'failed_sync'
  last_error?: string
}

export interface PendingScanRecord {
  id: string
  event_id: string
  payload: PostScanPayload
  created_at: string
  sync_status: 'pending_sync' | 'failed_sync'
  last_error?: string
}

export interface PendingKaraokePayload {
  timing_record_id: string
  source_lap_id?: string
}

export interface PendingKaraokeRecord {
  id: string
  payload: PendingKaraokePayload
  created_at: string
  sync_status: 'pending_sync' | 'failed_sync'
  last_error?: string
}

export interface TagDisplayInfo {
  participant_name: string
  bib_number?: string
  race_name?: string
  category_label?: string
}

/** Minimal UI cache — not the offline system of record (local Postgres is). */
export interface DisplayCache {
  event_id: string
  event_name: string
  races: Array<{ id: string; name: string }>
  tags: Record<string, TagDisplayInfo>
  updated_at?: string
}

function openDatabase(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, DB_VERSION)

    request.onerror = () => {
      reject(request.error ?? new Error('Failed to open IndexedDB'))
    }

    request.onupgradeneeded = (event) => {
      const db = (event.target as IDBOpenDBRequest).result
      if (!db.objectStoreNames.contains(STORE_NAME)) {
        const store = db.createObjectStore(STORE_NAME, { keyPath: 'id' })
        store.createIndex('sync_status', 'sync_status', { unique: false })
        store.createIndex('created_at', 'created_at', { unique: false })
      }
      if (!db.objectStoreNames.contains(SCAN_STORE)) {
        const store = db.createObjectStore(SCAN_STORE, { keyPath: 'id' })
        store.createIndex('created_at', 'created_at', { unique: false })
      }
      if (!db.objectStoreNames.contains(KARAOKE_STORE)) {
        const store = db.createObjectStore(KARAOKE_STORE, { keyPath: 'id' })
        store.createIndex('created_at', 'created_at', { unique: false })
      }
      if (!db.objectStoreNames.contains(DISPLAY_CACHE_STORE)) {
        db.createObjectStore(DISPLAY_CACHE_STORE, { keyPath: 'id' })
      }
    }

    request.onsuccess = () => {
      resolve(request.result)
    }
  })
}

function promisifyRequest<T>(request: IDBRequest<T>): Promise<T> {
  return new Promise((resolve, reject) => {
    request.onsuccess = () => resolve(request.result)
    request.onerror = () => {
      reject(request.error ?? new Error('IndexedDB request failed'))
    }
  })
}

function waitForTransaction(tx: IDBTransaction): Promise<void> {
  return new Promise((resolve, reject) => {
    tx.oncomplete = () => resolve()
    tx.onerror = () => reject(tx.error ?? new Error('IndexedDB transaction failed'))
  })
}

export async function addPendingRecord(
  payload: ManualTimingEntryPayload,
): Promise<PendingTimingRecord> {
  if (typeof indexedDB === 'undefined') {
    throw new Error('IndexedDB is not available')
  }
  const record: PendingTimingRecord = {
    id: crypto.randomUUID(),
    payload,
    created_at: new Date().toISOString(),
    sync_status: 'pending_sync',
  }

  const db = await openDatabase()
  try {
    const tx = db.transaction(STORE_NAME, 'readwrite')
    await promisifyRequest(tx.objectStore(STORE_NAME).add(record))
    await waitForTransaction(tx)
    return record
  } finally {
    db.close()
  }
}

export async function getAllPendingRecords(): Promise<PendingTimingRecord[]> {
  const db = await openDatabase()
  try {
    const tx = db.transaction(STORE_NAME, 'readonly')
    const records = await promisifyRequest(
      tx.objectStore(STORE_NAME).index('created_at').getAll(),
    )
    await waitForTransaction(tx)
    return records
  } finally {
    db.close()
  }
}

export async function getPendingCount(): Promise<number> {
  if (typeof indexedDB === 'undefined') {
    return 0
  }

  const records = await getAllPendingRecords()
  return records.length
}

export async function updatePendingRecord(
  id: string,
  updates: Pick<PendingTimingRecord, 'sync_status' | 'last_error'>,
): Promise<void> {
  const db = await openDatabase()
  try {
    const tx = db.transaction(STORE_NAME, 'readwrite')
    const store = tx.objectStore(STORE_NAME)
    const existing = await promisifyRequest(store.get(id))
    if (!existing) {
      throw new Error(`Pending record ${id} not found`)
    }
    await promisifyRequest(
      store.put({
        ...existing,
        ...updates,
      }),
    )
    await waitForTransaction(tx)
  } finally {
    db.close()
  }
}

export async function removePendingRecord(id: string): Promise<void> {
  const db = await openDatabase()
  try {
    const tx = db.transaction(STORE_NAME, 'readwrite')
    await promisifyRequest(tx.objectStore(STORE_NAME).delete(id))
    await waitForTransaction(tx)
  } finally {
    db.close()
  }
}

export async function addPendingScan(
  eventId: string,
  payload: PostScanPayload,
): Promise<PendingScanRecord> {
  if (typeof indexedDB === 'undefined') {
    throw new Error('IndexedDB is not available')
  }
  const record: PendingScanRecord = {
    id: crypto.randomUUID(),
    event_id: eventId,
    payload,
    created_at: new Date().toISOString(),
    sync_status: 'pending_sync',
  }
  const db = await openDatabase()
  try {
    const tx = db.transaction(SCAN_STORE, 'readwrite')
    await promisifyRequest(tx.objectStore(SCAN_STORE).add(record))
    await waitForTransaction(tx)
    return record
  } finally {
    db.close()
  }
}

export async function getAllPendingScans(): Promise<PendingScanRecord[]> {
  const db = await openDatabase()
  try {
    const tx = db.transaction(SCAN_STORE, 'readonly')
    const records = await promisifyRequest(
      tx.objectStore(SCAN_STORE).index('created_at').getAll(),
    )
    await waitForTransaction(tx)
    return records
  } finally {
    db.close()
  }
}

export async function removePendingScan(id: string): Promise<void> {
  const db = await openDatabase()
  try {
    const tx = db.transaction(SCAN_STORE, 'readwrite')
    await promisifyRequest(tx.objectStore(SCAN_STORE).delete(id))
    await waitForTransaction(tx)
  } finally {
    db.close()
  }
}

export async function updatePendingScan(
  id: string,
  updates: Pick<PendingScanRecord, 'sync_status' | 'last_error'>,
): Promise<void> {
  const db = await openDatabase()
  try {
    const tx = db.transaction(SCAN_STORE, 'readwrite')
    const store = tx.objectStore(SCAN_STORE)
    const existing = await promisifyRequest(store.get(id))
    if (!existing) {
      throw new Error(`Pending scan ${id} not found`)
    }
    await promisifyRequest(store.put({ ...existing, ...updates }))
    await waitForTransaction(tx)
  } finally {
    db.close()
  }
}

export async function addPendingKaraoke(
  payload: PendingKaraokePayload,
): Promise<PendingKaraokeRecord> {
  if (typeof indexedDB === 'undefined') {
    throw new Error('IndexedDB is not available')
  }
  const record: PendingKaraokeRecord = {
    id: crypto.randomUUID(),
    payload,
    created_at: new Date().toISOString(),
    sync_status: 'pending_sync',
  }
  const db = await openDatabase()
  try {
    const tx = db.transaction(KARAOKE_STORE, 'readwrite')
    await promisifyRequest(tx.objectStore(KARAOKE_STORE).add(record))
    await waitForTransaction(tx)
    return record
  } finally {
    db.close()
  }
}

export async function getAllPendingKaraoke(): Promise<PendingKaraokeRecord[]> {
  const db = await openDatabase()
  try {
    const tx = db.transaction(KARAOKE_STORE, 'readonly')
    const records = await promisifyRequest(
      tx.objectStore(KARAOKE_STORE).index('created_at').getAll(),
    )
    await waitForTransaction(tx)
    return records
  } finally {
    db.close()
  }
}

export async function removePendingKaraoke(id: string): Promise<void> {
  const db = await openDatabase()
  try {
    const tx = db.transaction(KARAOKE_STORE, 'readwrite')
    await promisifyRequest(tx.objectStore(KARAOKE_STORE).delete(id))
    await waitForTransaction(tx)
  } finally {
    db.close()
  }
}

export async function updatePendingKaraoke(
  id: string,
  updates: Pick<PendingKaraokeRecord, 'sync_status' | 'last_error'>,
): Promise<void> {
  const db = await openDatabase()
  try {
    const tx = db.transaction(KARAOKE_STORE, 'readwrite')
    const store = tx.objectStore(KARAOKE_STORE)
    const existing = await promisifyRequest(store.get(id))
    if (!existing) {
      throw new Error(`Pending karaoke ${id} not found`)
    }
    await promisifyRequest(store.put({ ...existing, ...updates }))
    await waitForTransaction(tx)
  } finally {
    db.close()
  }
}

export async function setDisplayCache(cache: DisplayCache): Promise<void> {
  if (typeof indexedDB === 'undefined') {
    return
  }
  const db = await openDatabase()
  try {
    const tx = db.transaction(DISPLAY_CACHE_STORE, 'readwrite')
    await promisifyRequest(
      tx.objectStore(DISPLAY_CACHE_STORE).put({
        id: DISPLAY_CACHE_KEY,
        ...cache,
        updated_at: new Date().toISOString(),
      }),
    )
    await waitForTransaction(tx)
  } finally {
    db.close()
  }
}

export async function getDisplayCache(): Promise<DisplayCache | null> {
  if (typeof indexedDB === 'undefined') {
    return null
  }
  const db = await openDatabase()
  try {
    const tx = db.transaction(DISPLAY_CACHE_STORE, 'readonly')
    const row = await promisifyRequest(
      tx.objectStore(DISPLAY_CACHE_STORE).get(DISPLAY_CACHE_KEY),
    )
    await waitForTransaction(tx)
    if (!row) return null
    const { id: _id, ...rest } = row as DisplayCache & { id: string }
    return rest as DisplayCache
  } finally {
    db.close()
  }
}

/** Build a provisional lap ScanResult from the display cache (WAQ / UI only). */
export async function provisionalScanFromCache(
  tagUid: string,
): Promise<ScanResult | null> {
  const cache = await getDisplayCache()
  const tag = cache?.tags?.[tagUid]
  if (!tag) {
    return {
      result: 'lap',
      participant_name: 'Racer',
      message: 'Queued offline — will sync when connected',
    }
  }
  return {
    result: 'lap',
    participant_name: tag.participant_name,
    bib_number: tag.bib_number,
    race_name: tag.race_name,
    category_label: tag.category_label,
    message: 'Queued offline — will sync when connected',
  }
}

export async function getWAQPendingCount(): Promise<number> {
  if (typeof indexedDB === 'undefined') {
    return 0
  }
  const [manual, scans, karaoke] = await Promise.all([
    getAllPendingRecords(),
    getAllPendingScans(),
    getAllPendingKaraoke(),
  ])
  return manual.length + scans.length + karaoke.length
}

export async function deleteDatabase(): Promise<void> {
  if (typeof indexedDB === 'undefined') {
    return
  }

  await new Promise<void>((resolve, reject) => {
    const request = indexedDB.deleteDatabase(DB_NAME)
    request.onsuccess = () => resolve()
    request.onerror = () => {
      reject(request.error ?? new Error('Failed to delete IndexedDB'))
    }
    request.onblocked = () => resolve()
  })
}
