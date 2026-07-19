import { describe, it, expect, vi, beforeEach, afterEach, type Mock } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { defineComponent, ref, type ComputedRef, type Ref } from 'vue'
import { eventLiveStreamUrl } from '@/services/api'

vi.mock('@/services/api', async () => {
  const actual = await vi.importActual<typeof import('@/services/api')>('@/services/api')
  return {
    ...actual,
    eventLiveStreamUrl: vi.fn(
      (eventId: string) => `ws://localhost:8080/api/events/${eventId}/live/stream`,
    ),
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

function mountComposable(eventId: Ref<string> | ComputedRef<string>) {
  let exposed: ReturnType<typeof import('./useEventLiveStream').useEventLiveStream> | undefined
  const Host = defineComponent({
    setup() {
      exposed = useEventLiveStream(eventId)
      return () => null
    },
  })
  const wrapper = mount(Host)
  return {
    wrapper,
    get exposed() {
      return exposed!
    },
  }
}

// eslint-disable-next-line import/first
import { useEventLiveStream } from './useEventLiveStream'

describe('useEventLiveStream', () => {
  beforeEach(() => {
    MockWebSocket.instances = []
    vi.stubGlobal('WebSocket', MockWebSocket as unknown as typeof WebSocket)
    vi.clearAllMocks()
    ;(eventLiveStreamUrl as Mock).mockImplementation(
      (eventId: string) => `ws://localhost:8080/api/events/${eventId}/live/stream`,
    )
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('opens URL from eventLiveStreamUrl(eventId) on start', async () => {
    const eventId = ref('evt-42')
    const { exposed, wrapper } = mountComposable(eventId)
    await flushPromises()

    expect(eventLiveStreamUrl).toHaveBeenCalledWith('evt-42')
    expect(MockWebSocket.instances).toHaveLength(1)
    expect(MockWebSocket.instances[0].url).toBe(
      'ws://localhost:8080/api/events/evt-42/live/stream',
    )

    exposed.stop()
    expect(MockWebSocket.instances[0].close).toHaveBeenCalled()
    wrapper.unmount()
  })

  it('auto-starts on mount and exposes connected after open', async () => {
    const eventId = ref('evt-1')
    const { exposed, wrapper } = mountComposable(eventId)
    await flushPromises()

    expect(eventLiveStreamUrl).toHaveBeenCalledWith('evt-1')
    expect(MockWebSocket.instances).toHaveLength(1)
    expect(exposed.connected.value).toBe(true)

    wrapper.unmount()
    expect(MockWebSocket.instances[0].close).toHaveBeenCalled()
  })

  it('parses lap_recorded into lastLap ref', async () => {
    const eventId = ref('evt-1')
    const { exposed, wrapper } = mountComposable(eventId)
    await flushPromises()

    const lap = {
      type: 'lap_recorded' as const,
      event_id: 'evt-1',
      race_id: 'race-1',
      participant_id: 'p-1',
      participant_name: 'Alex Rivera',
      bib_number: '12',
      lap_count: 14,
      recorded_at: '2026-08-01T12:00:01-04:00',
    }
    MockWebSocket.instances[0].emit(lap)

    expect(exposed.lastLap.value).toEqual(lap)
    wrapper.unmount()
  })

  it('ignores malformed JSON and non-lap_recorded types', async () => {
    const eventId = ref('evt-1')
    const { exposed, wrapper } = mountComposable(eventId)
    await flushPromises()

    MockWebSocket.instances[0].emit('not-json{')
    MockWebSocket.instances[0].emit({ type: 'ping' })
    MockWebSocket.instances[0].emit({ type: 'tag_read', tag_uid: 'x' })

    expect(exposed.lastLap.value).toBeNull()

    MockWebSocket.instances[0].emit({
      type: 'lap_recorded',
      event_id: 'evt-1',
      race_id: 'race-1',
      participant_id: 'p-2',
      participant_name: 'Jordan Lee',
      lap_count: 3,
      recorded_at: '2026-08-01T12:00:02-04:00',
    })
    expect(exposed.lastLap.value?.participant_name).toBe('Jordan Lee')

    wrapper.unmount()
  })

  it('closes on stop()', async () => {
    const eventId = ref('evt-1')
    const { exposed, wrapper } = mountComposable(eventId)
    await flushPromises()
    expect(exposed.connected.value).toBe(true)

    exposed.stop()
    expect(MockWebSocket.instances[0].close).toHaveBeenCalled()
    expect(exposed.connected.value).toBe(false)
    wrapper.unmount()
  })

  it('reconnects after unexpected close', async () => {
    vi.useFakeTimers()
    const eventId = ref('evt-1')
    const { exposed, wrapper } = mountComposable(eventId)
    await flushPromises()

    const first = MockWebSocket.instances[0]
    first.onclose?.(new CloseEvent('close'))
    expect(exposed.connected.value).toBe(false)
    expect(MockWebSocket.instances).toHaveLength(1)

    await vi.advanceTimersByTimeAsync(2000)
    expect(MockWebSocket.instances).toHaveLength(2)
    expect(eventLiveStreamUrl).toHaveBeenCalledTimes(2)

    exposed.stop()
    wrapper.unmount()
    vi.useRealTimers()
  })

  it('cancels reconnect when stopped during backoff', async () => {
    vi.useFakeTimers()
    const eventId = ref('evt-1')
    const { exposed, wrapper } = mountComposable(eventId)
    await flushPromises()

    MockWebSocket.instances[0].onclose?.(new CloseEvent('close'))
    exposed.stop()
    await vi.advanceTimersByTimeAsync(2000)
    expect(MockWebSocket.instances).toHaveLength(1)

    wrapper.unmount()
    vi.useRealTimers()
  })

  it('restarts stream when eventId changes', async () => {
    const eventId = ref('evt-1')
    const { wrapper } = mountComposable(eventId)
    await flushPromises()
    expect(MockWebSocket.instances).toHaveLength(1)
    expect(eventLiveStreamUrl).toHaveBeenLastCalledWith('evt-1')

    eventId.value = 'evt-2'
    await flushPromises()
    expect(MockWebSocket.instances[0].close).toHaveBeenCalled()
    expect(MockWebSocket.instances).toHaveLength(2)
    expect(eventLiveStreamUrl).toHaveBeenLastCalledWith('evt-2')

    wrapper.unmount()
  })
})

describe('eventLiveStreamUrl helper', () => {
  it('converts http(s) VITE_API_URL to ws(s) stream path', async () => {
    const { eventLiveStreamUrl: realUrl } = await vi.importActual<typeof import('@/services/api')>(
      '@/services/api',
    )
    expect(realUrl('evt-99', 'http://localhost:8080')).toBe(
      'ws://localhost:8080/api/events/evt-99/live/stream',
    )
    expect(realUrl('evt-99', 'https://api.example.com')).toBe(
      'wss://api.example.com/api/events/evt-99/live/stream',
    )
    expect(realUrl('evt-99', '')).toMatch(/\/api\/events\/evt-99\/live\/stream$/)
  })
})
