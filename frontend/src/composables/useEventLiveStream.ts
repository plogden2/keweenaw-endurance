import {
  onMounted,
  onUnmounted,
  ref,
  unref,
  watch,
  type ComputedRef,
  type Ref,
} from 'vue'
import { eventLiveStreamUrl, type LapRecordedEvent } from '@/services/api'

export interface UseEventLiveStream {
  lastLap: Ref<LapRecordedEvent | null>
  connected: Ref<boolean>
  start: () => void
  stop: () => void
}

export function useEventLiveStream(
  eventId: Ref<string> | ComputedRef<string>,
): UseEventLiveStream {
  const lastLap = ref<LapRecordedEvent | null>(null)
  const connected = ref(false)
  let socket: WebSocket | null = null
  let intentionallyClosed = false
  let reconnectTimer: ReturnType<typeof setTimeout> | undefined

  function onMessage(ev: MessageEvent) {
    try {
      const raw = typeof ev.data === 'string' ? JSON.parse(ev.data) : ev.data
      if (raw?.type === 'lap_recorded') {
        lastLap.value = raw as LapRecordedEvent
      }
    } catch {
      // ignore malformed frames
    }
  }

  function start() {
    const id = unref(eventId)
    if (!id) return
    if (socket && (socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING)) {
      return
    }
    intentionallyClosed = false
    const url = eventLiveStreamUrl(id)
    socket = new WebSocket(url)
    socket.onopen = () => {
      connected.value = true
    }
    socket.onmessage = onMessage
    socket.onclose = () => {
      connected.value = false
      socket = null
      if (!intentionallyClosed) {
        reconnectTimer = window.setTimeout(() => {
          reconnectTimer = undefined
          if (!intentionallyClosed) start()
        }, 2000)
      }
    }
  }

  function stop() {
    intentionallyClosed = true
    if (reconnectTimer) {
      window.clearTimeout(reconnectTimer)
      reconnectTimer = undefined
    }
    if (socket) {
      socket.close()
      socket = null
    }
    connected.value = false
  }

  function restart() {
    stop()
    intentionallyClosed = false
    start()
  }

  onMounted(start)
  onUnmounted(stop)
  watch(eventId, restart)

  return { lastLap, connected, start, stop }
}
