import { describe, it, expect, vi, beforeEach, afterEach, type Mock } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import 'fake-indexeddb/auto'
import { scansApi, rfidStreamUrl } from '@/services/api'
import { usePinAuthStore } from '@/stores/pinAuth'
import { useStationStore } from '@/stores/station'
import { deleteDatabase } from '@/services/timingStorage'
import { __resetReaderStationForTests } from './useReaderStation'

vi.mock('@/services/api', async () => {
  const actual = await vi.importActual<typeof import('@/services/api')>('@/services/api')
  return {
    ...actual,
    scansApi: {
      postScan: vi.fn(),
    },
    rfidStreamUrl: vi.fn(() => 'ws://localhost:8080/api/rfid/stream'),
    syncApi: {
      push: vi.fn().mockResolvedValue({ data: { pushed: 0 } }),
      pull: vi.fn().mockResolvedValue({ data: { imported: 0 } }),
    },
    timingRecordsApi: {
      karaokeBonus: vi.fn(),
    },
  }
})

type MessageHandler = (ev: MessageEvent) => void

class MockWebSocket {
  static instances: MockWebSocket[] = []
  static OPEN = 1
  static CLOSED = 3
  static CONNECTING = 0

  url: string
  readyState = MockWebSocket.OPEN
  onopen: ((ev: Event) => void) | null = null
  onmessage: MessageHandler | null = null
  onerror: ((ev: Event) => void) | null = null
  onclose: ((ev: CloseEvent) => void) | null = null
  close = vi.fn(() => {
    this.readyState = MockWebSocket.CLOSED
  })
  send = vi.fn()

  constructor(url: string) {
    this.url = url
    MockWebSocket.instances.push(this)
    queueMicrotask(() => this.onopen?.(new Event('open')))
  }

  emit(data: unknown) {
    this.onmessage?.(
      new MessageEvent('message', {
        data: typeof data === 'string' ? data : JSON.stringify(data),
      }),
    )
  }
}

describe('useReaderStation', () => {
  function unlockReaderPin() {
    const pin = usePinAuthStore()
    pin.token = 'test-token'
    pin.role = 'organizer'
    pin.expiresAt = Math.floor(Date.now() / 1000) + 3600
  }

  beforeEach(async () => {
    setActivePinia(createPinia())
    MockWebSocket.instances = []
    vi.stubGlobal('WebSocket', MockWebSocket as unknown as typeof WebSocket)
    vi.clearAllMocks()
    __resetReaderStationForTests()
    await deleteDatabase()
    Object.defineProperty(navigator, 'onLine', { configurable: true, value: true })
    ;(rfidStreamUrl as Mock).mockReturnValue('ws://localhost:8080/api/rfid/stream')
  })

  afterEach(() => {
    __resetReaderStationForTests()
    vi.unstubAllGlobals()
  })

  it('builds stream URL via rfidStreamUrl and opens a WebSocket on start', async () => {
    const { useReaderStation } = await import('./useReaderStation')
    const { start, stop } = useReaderStation()

    start()
    expect(rfidStreamUrl).toHaveBeenCalled()
    expect(MockWebSocket.instances).toHaveLength(1)
    expect(MockWebSocket.instances[0].url).toBe('ws://localhost:8080/api/rfid/stream')

    stop()
    expect(MockWebSocket.instances[0].close).toHaveBeenCalled()
  })

  it('on tag_read posts a scan for the current station event and exposes lastScan', async () => {
    unlockReaderPin()
    const station = useStationStore()
    station.eventId = 'evt-1'
    station.deviceId = 'laptop-finish-1'
    station.name = 'Finish A'
    station.mode = 'finish'

    const lapResult = {
      result: 'lap' as const,
      participant_name: 'Alex Rivera',
      race_name: '12 Hour',
      placement: 3,
      lap_count: 14,
      timing_record_id: 'tr-1',
      karaoke_available: true,
      bib_number: '12',
      category_label: 'Advanced Men',
    }
    ;(scansApi.postScan as Mock).mockResolvedValue({ data: lapResult })

    const { useReaderStation } = await import('./useReaderStation')
    const { start, stop, lastScan } = useReaderStation()
    start()

    MockWebSocket.instances[0].emit({
      type: 'tag_read',
      tag_uid: 'DEMO-TAG-0001',
      read_at: '2026-08-01T12:00:01-04:00',
      device_id: 'laptop-finish-1',
    })

    await vi.waitFor(() => {
      expect(lastScan.value?.result).toBe('lap')
    })

    expect(scansApi.postScan).toHaveBeenCalledWith(
      'evt-1',
      expect.objectContaining({
        tag_uid: 'DEMO-TAG-0001',
        device_id: 'laptop-finish-1',
      }),
    )
    expect(lastScan.value?.participant_name).toBe('Alex Rivera')
    expect(lastScan.value?.lap_count).toBe(14)

    stop()
  })

  it('applies scan_result from bridge without posting a second scan', async () => {
    unlockReaderPin()
    const station = useStationStore()
    station.eventId = 'evt-1'
    station.deviceId = 'laptop-finish-1'

    const { useReaderStation } = await import('./useReaderStation')
    const { start, stop, lastScan } = useReaderStation()
    start()

    MockWebSocket.instances[0].emit({
      type: 'scan_result',
      tag_uid: '9fe78eeb-a21c-594a-acc2-7e1efe378201',
      read_at: '2026-08-01T12:00:01-04:00',
      scan: {
        result: 'lap',
        participant_name: 'Alex Rivera',
        lap_count: 1,
        bib_number: '1',
        timing_record_id: 'tr-bridge-1',
        karaoke_available: true,
      },
    })

    await vi.waitFor(() => {
      expect(lastScan.value?.result).toBe('lap')
    })
    expect(scansApi.postScan).not.toHaveBeenCalled()
    expect(lastScan.value?.lap_count).toBe(1)
    expect(lastScan.value?.timing_record_id).toBe('tr-bridge-1')

    stop()
  })

  it('skips posting when browser is not PIN-unlocked (spectator)', async () => {
    const station = useStationStore()
    station.eventId = 'evt-1'
    station.deviceId = 'laptop-finish-1'

    const { useReaderStation } = await import('./useReaderStation')
    const { start, stop, lastScan } = useReaderStation()
    start()

    MockWebSocket.instances[0].emit({
      type: 'tag_read',
      tag_uid: 'DEMO-TAG-0001',
      read_at: '2026-08-01T12:00:01-04:00',
    })

    await nextTick()
    expect(scansApi.postScan).not.toHaveBeenCalled()
    expect(lastScan.value).toBeNull()
    stop()
  })

  it('skips posting when station has no event_id', async () => {
    unlockReaderPin()
    const station = useStationStore()
    station.eventId = null

    const { useReaderStation } = await import('./useReaderStation')
    const { start, stop } = useReaderStation()
    start()

    MockWebSocket.instances[0].emit({
      type: 'tag_read',
      tag_uid: 'DEMO-TAG-0001',
      read_at: '2026-08-01T12:00:01-04:00',
    })

    await nextTick()
    expect(scansApi.postScan).not.toHaveBeenCalled()
    stop()
  })

  it('exposes cooldown and test_read scan results', async () => {
    unlockReaderPin()
    const station = useStationStore()
    station.eventId = 'evt-1'
    station.deviceId = 'laptop-finish-1'

    ;(scansApi.postScan as Mock).mockResolvedValueOnce({
      data: {
        result: 'cooldown',
        participant_name: 'Jordan Lee',
        retry_after_seconds: 42,
        lap_count: 12,
      },
    })

    const { useReaderStation } = await import('./useReaderStation')
    const { start, stop, lastScan } = useReaderStation()
    start()

    MockWebSocket.instances[0].emit({
      type: 'tag_read',
      tag_uid: 'DEMO-TAG-0001',
      read_at: '2026-08-01T12:00:02-04:00',
    })

    await vi.waitFor(() => expect(lastScan.value?.result).toBe('cooldown'))
    expect(lastScan.value?.retry_after_seconds).toBe(42)

    ;(scansApi.postScan as Mock).mockResolvedValueOnce({
      data: {
        result: 'test_read',
        participant_name: 'Sam Ortiz',
      },
    })

    MockWebSocket.instances[0].emit({
      type: 'tag_read',
      tag_uid: 'DEMO-TAG-0003',
      read_at: '2026-08-01T12:00:03-04:00',
    })

    await vi.waitFor(() => expect(lastScan.value?.result).toBe('test_read'))
    stop()
  })

  it('queues scans offline and still exposes provisional lastScan', async () => {
    unlockReaderPin()
    const station = useStationStore()
    station.eventId = 'evt-1'
    station.deviceId = 'laptop-finish-1'
    ;(scansApi.postScan as Mock).mockRejectedValue(new Error('network down'))

    const { useReaderStation } = await import('./useReaderStation')
    const { start, stop, lastScan, clearLastScan, error } = useReaderStation()
    start()

    MockWebSocket.instances[0].emit('not-json{')
    MockWebSocket.instances[0].emit('null')
    MockWebSocket.instances[0].onmessage?.({ data: { type: 'ping' } } as MessageEvent)
    MockWebSocket.instances[0].emit({ type: 'other' })
    MockWebSocket.instances[0].emit({
      type: 'tag_read',
      tag_uid: 'DEMO-TAG-0001',
    })

    await vi.waitFor(() => expect(lastScan.value?.result).toBe('lap'))
    expect(error.value).toBeNull()
    clearLastScan()
    expect(lastScan.value).toBeNull()
    stop()
  })

  it('does not open a second socket while already connected', async () => {
    const { useReaderStation } = await import('./useReaderStation')
    const { start, stop } = useReaderStation()
    start()
    start()
    expect(MockWebSocket.instances).toHaveLength(1)
    stop()
  })

  it('sets stream error on websocket error and reconnects after unexpected close', async () => {
    vi.useFakeTimers()
    const { useReaderStation } = await import('./useReaderStation')
    const { start, stop, error, connected } = useReaderStation()
    start()
    const first = MockWebSocket.instances[0]
    first.onerror?.(new Event('error'))
    expect(error.value).toBe('RFID stream error')

    first.onclose?.(new CloseEvent('close'))
    expect(connected.value).toBe(false)
    expect(MockWebSocket.instances).toHaveLength(1)

    await vi.advanceTimersByTimeAsync(2000)
    expect(MockWebSocket.instances).toHaveLength(2)

    stop()
    vi.useRealTimers()
  })

  it('uses unknown-device and queues when scan API rejects non-Error', async () => {
    unlockReaderPin()
    const station = useStationStore()
    station.eventId = 'evt-1'
    station.deviceId = ''
    ;(scansApi.postScan as Mock).mockRejectedValue('boom')

    const { useReaderStation } = await import('./useReaderStation')
    const { start, stop, lastScan } = useReaderStation()
    start()

    MockWebSocket.instances[0].emit({
      type: 'tag_read',
      tag_uid: 'DEMO-TAG-0001',
    })

    await vi.waitFor(() => expect(lastScan.value?.result).toBe('lap'))
    expect(scansApi.postScan).toHaveBeenCalledWith(
      'evt-1',
      expect.objectContaining({ device_id: 'unknown-device' }),
    )
    stop()
  })

  it('skips tag_read frames without tag_uid and treats CONNECTING as already started', async () => {
    const { useReaderStation } = await import('./useReaderStation')
    const { start, stop } = useReaderStation()
    start()
    MockWebSocket.instances[0].emit({ type: 'tag_read' })
    expect(scansApi.postScan).not.toHaveBeenCalled()

    MockWebSocket.instances[0].readyState = MockWebSocket.CONNECTING
    start()
    expect(MockWebSocket.instances).toHaveLength(1)
    stop()
  })

  it('cancels reconnect when stopped during backoff', async () => {
    vi.useFakeTimers()
    const { useReaderStation } = await import('./useReaderStation')
    const { start, stop } = useReaderStation()
    start()
    MockWebSocket.instances[0].onclose?.(new CloseEvent('close'))
    stop()
    await vi.advanceTimersByTimeAsync(2000)
    expect(MockWebSocket.instances).toHaveLength(1)
    vi.useRealTimers()
  })
})

describe('rfidStreamUrl helper', () => {
  it('converts http(s) VITE_API_URL to ws(s) stream path', async () => {
    const { rfidStreamUrl: realUrl } = await vi.importActual<typeof import('@/services/api')>(
      '@/services/api',
    )
    expect(realUrl('http://localhost:8080')).toBe('ws://localhost:8080/api/rfid/stream')
    expect(realUrl('https://api.example.com')).toBe('wss://api.example.com/api/rfid/stream')
    expect(realUrl('')).toMatch(/\/api\/rfid\/stream$/)
  })
})
