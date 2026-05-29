import type { ManualTimingEntryPayload } from '@/types/models'

export const DB_NAME = 'keweenaw-timing'
export const STORE_NAME = 'pending-records'
export const DB_VERSION = 1

export interface PendingTimingRecord {
  id: string
  payload: ManualTimingEntryPayload
  created_at: string
  sync_status: 'pending_sync' | 'failed_sync'
  last_error?: string
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
