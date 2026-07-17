import { computed, onMounted, onUnmounted, ref } from 'vue'
import { rfidApi, updateLastBridgeSnapshot } from '@/services/api'
import type { BridgeStatusResponse } from '@/types/models'
import {
  deriveSyncChipState,
  syncChipLabel,
  type BridgeStatusSnapshot,
  type LocalBridgeStatusResponse,
  type SyncChipState,
} from '@/services/bridgeSyncStatus'
import { syncAll as syncOfflineQueue } from '@/services/offlineQueue'

const DEFAULT_DEVICE_ID = 'laptop-finish-1'

/** Minimum time to show Syncing after Offline so Offline → Syncing → Online · Synced is visible. */
export const MIN_SYNCING_VISIBLE_MS = 1500

export async function fetchBridgeSnapshot(
  deviceId = DEFAULT_DEVICE_ID,
): Promise<BridgeStatusSnapshot> {
  const navigatorOnline = typeof navigator !== 'undefined' ? navigator.onLine : true
  let hosted: BridgeStatusResponse | null = null
  let local: LocalBridgeStatusResponse | null = null

  if (navigatorOnline) {
    try {
      const { data } = await rfidApi.getBridgeStatus(deviceId)
      hosted = data
    } catch {
      hosted = null
    }
  }

  // Always probe loopback — during partition chaos hosted may still report connected
  // while the local bridge has already switched to offline mode.
  local = await rfidApi.getLocalBridgeStatus()

  const snapshot: BridgeStatusSnapshot = { navigatorOnline, hosted, local }
  updateLastBridgeSnapshot(snapshot)
  return snapshot
}

export function useBridgeSyncStatus(deviceId = DEFAULT_DEVICE_ID) {
  const chipState = ref<SyncChipState>('offline')
  const chipLabel = computed(() => syncChipLabel(chipState.value))
  const loading = ref(false)
  let pollTimer: number | undefined
  let holdTimer: number | undefined
  let previousState: SyncChipState | null = null
  let syncingHoldUntil = 0

  async function refresh(): Promise<void> {
    loading.value = true
    try {
      const snapshot = await fetchBridgeSnapshot(deviceId)
      let next = deriveSyncChipState(snapshot)

      // Instant flush after outage: still paint Syncing briefly for the race-day chip sequence.
      if (previousState === 'offline' && next === 'online_synced') {
        next = 'syncing'
        syncingHoldUntil = Date.now() + MIN_SYNCING_VISIBLE_MS
        if (holdTimer) window.clearTimeout(holdTimer)
        holdTimer = window.setTimeout(() => {
          void refresh()
        }, MIN_SYNCING_VISIBLE_MS)
      } else if (
        chipState.value === 'syncing' &&
        next === 'online_synced' &&
        Date.now() < syncingHoldUntil
      ) {
        next = 'syncing'
      } else if (next === 'online_synced') {
        syncingHoldUntil = 0
      }

      if (previousState === 'offline' && next !== 'offline' && navigator.onLine) {
        await autoSync()
      }
      previousState = next
      chipState.value = next
    } finally {
      loading.value = false
    }
  }

  async function autoSync(): Promise<void> {
    try {
      await syncOfflineQueue()
      if (navigator.onLine) {
        await rfidApi.syncPending()
      }
    } catch {
      // bridge auto-flush may handle this; manual button remains fallback
    }
  }

  onMounted(() => {
    void refresh()
    pollTimer = window.setInterval(() => {
      void refresh()
    }, 2000)

    window.addEventListener('online', onOnline)
    window.addEventListener('offline', onOffline)
  })

  function onOnline(): void {
    void refresh()
  }

  function onOffline(): void {
    chipState.value = 'offline'
    previousState = 'offline'
    syncingHoldUntil = 0
    if (holdTimer) window.clearTimeout(holdTimer)
    updateLastBridgeSnapshot({
      navigatorOnline: false,
      hosted: null,
      local: null,
    })
  }

  onUnmounted(() => {
    if (pollTimer) window.clearInterval(pollTimer)
    if (holdTimer) window.clearTimeout(holdTimer)
    window.removeEventListener('online', onOnline)
    window.removeEventListener('offline', onOffline)
  })

  return { chipState, chipLabel, loading, refresh, autoSync }
}
