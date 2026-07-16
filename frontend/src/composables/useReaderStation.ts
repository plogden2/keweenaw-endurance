import { ref, type Ref } from 'vue'
import { type ScanResult } from '@/services/api'
import { enqueueScan } from '@/services/offlineQueue'
import { setDisplayCache, getDisplayCache } from '@/services/timingStorage'
import { usePinAuthStore } from '@/stores/pinAuth'
import { useStationStore } from '@/stores/station'
import { rfidStreamUrl } from '@/services/api'

export interface UseReaderStation {
  lastScan: Ref<ScanResult | null>
  connected: Ref<boolean>
  error: Ref<string | null>
  start: () => void
  stop: () => void
  clearLastScan: () => void
}

let shared: UseReaderStation | null = null

function createReaderStation(): UseReaderStation {
  const lastScan = ref<ScanResult | null>(null)
  const connected = ref(false)
  const error = ref<string | null>(null)
  let socket: WebSocket | null = null
  let intentionallyClosed = false

  async function rememberScan(tagUid: string, data: ScanResult) {
    if (data.result !== 'lap') return
    const station = useStationStore()
    const existing = (await getDisplayCache()) ?? {
      event_id: station.eventId || '',
      event_name: '',
      races: [],
      tags: {},
    }
    existing.tags = {
      ...existing.tags,
      [tagUid]: {
        participant_name: data.participant_name || 'Racer',
        bib_number: data.bib_number,
        race_name: data.race_name,
        category_label: data.category_label,
      },
    }
    if (station.eventId) {
      existing.event_id = station.eventId
    }
    await setDisplayCache(existing)
  }

  async function handleTagRead(tagUid: string, readAt?: string) {
    const pinAuth = usePinAuthStore()
    if (!pinAuth.isAuthenticated) return

    const station = useStationStore()
    if (!station.eventId) {
      try {
        await station.fetchCurrent()
      } catch {
        // offline or unconfigured
      }
    }
    if (!station.eventId) return

    const deviceId = station.deviceId || 'unknown-device'
    const payload = {
      tag_uid: tagUid,
      device_id: deviceId,
      local_timestamp: readAt || new Date().toISOString(),
    }

    try {
      const result = await enqueueScan(station.eventId, payload)
      if (result.scan) {
        lastScan.value = result.scan
        if (result.status === 'synced') {
          await rememberScan(tagUid, result.scan)
        }
      } else if (result.status === 'queued') {
        lastScan.value = {
          result: 'lap',
          participant_name: 'Racer',
          message: 'Queued offline — will sync when connected',
        }
      }
      error.value = null
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Scan failed'
    }
  }

  function onMessage(ev: MessageEvent) {
    try {
      const raw = typeof ev.data === 'string' ? JSON.parse(ev.data) : ev.data
      if (raw?.type === 'tag_read' && raw.tag_uid) {
        void handleTagRead(String(raw.tag_uid), raw.read_at)
      }
    } catch {
      // ignore malformed frames
    }
  }

  function onDomTagRead(ev: Event) {
    const detail = (ev as CustomEvent<{ tag_uid?: string; read_at?: string }>).detail
    if (detail?.tag_uid) {
      void handleTagRead(String(detail.tag_uid), detail.read_at)
    }
  }

  function start() {
    if (socket && (socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING)) {
      return
    }
    intentionallyClosed = false
    error.value = null
    if (typeof window !== 'undefined') {
      window.addEventListener('rfid-tag-read', onDomTagRead)
    }
    const url = rfidStreamUrl()
    socket = new WebSocket(url)
    socket.onopen = () => {
      connected.value = true
    }
    socket.onmessage = onMessage
    socket.onerror = () => {
      error.value = 'RFID stream error'
    }
    socket.onclose = () => {
      connected.value = false
      socket = null
      if (!intentionallyClosed) {
        window.setTimeout(() => {
          if (!intentionallyClosed) start()
        }, 2000)
      }
    }
  }

  function stop() {
    intentionallyClosed = true
    if (typeof window !== 'undefined') {
      window.removeEventListener('rfid-tag-read', onDomTagRead)
    }
    if (socket) {
      socket.close()
      socket = null
    }
    connected.value = false
  }

  function clearLastScan() {
    lastScan.value = null
  }

  return { lastScan, connected, error, start, stop, clearLastScan }
}

/** App-shell singleton so tag reads continue across routes. */
export function useReaderStation(): UseReaderStation {
  if (!shared) {
    shared = createReaderStation()
  }
  return shared
}

/** Test helper to reset singleton between specs. */
export function __resetReaderStationForTests() {
  shared?.stop()
  shared = null
}
