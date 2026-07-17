import { describe, it, expect, vi, beforeEach, afterEach, type Mock } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { rfidApi } from '@/services/api'
import { syncAll } from '@/services/offlineQueue'

vi.mock('@/services/api', async () => {
  const actual = await vi.importActual<typeof import('@/services/api')>('@/services/api')
  return {
    ...actual,
    rfidApi: {
      getBridgeStatus: vi.fn(),
      getLocalBridgeStatus: vi.fn().mockResolvedValue(null),
      syncPending: vi.fn().mockResolvedValue({ data: { synced_count: 0 } }),
    },
  }
})

vi.mock('@/services/offlineQueue', () => ({
  syncAll: vi.fn().mockResolvedValue({ synced: 0, failed: 0 }),
}))

import { useBridgeSyncStatus } from './useBridgeSyncStatus'

function mountComposable() {
  let exposed: ReturnType<typeof useBridgeSyncStatus> | undefined
  const Host = defineComponent({
    setup() {
      exposed = useBridgeSyncStatus()
      return () => null
    },
  })
  const wrapper = mount(Host)
  return { wrapper, get exposed() { return exposed! } }
}

describe('useBridgeSyncStatus', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.useFakeTimers()
    Object.defineProperty(navigator, 'onLine', { value: true, configurable: true })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('auto-syncs when transitioning from offline to online', async () => {
    ;(rfidApi.getBridgeStatus as Mock)
      .mockResolvedValueOnce({
        data: { connected: false, pending_count: 3, syncing: false },
      })
      .mockResolvedValue({
        data: { connected: true, pending_count: 0, syncing: false },
      })
    ;(rfidApi.getLocalBridgeStatus as Mock).mockResolvedValue(null)

    const { exposed, wrapper } = mountComposable()
    await flushPromises()

    expect(exposed.chipState.value).toBe('offline')
    expect(rfidApi.getLocalBridgeStatus).toHaveBeenCalled()

    await exposed.refresh()
    await flushPromises()

    // Instant reconnect still shows Syncing briefly (Offline → Syncing → Online · Synced).
    expect(exposed.chipState.value).toBe('syncing')
    expect(syncAll).toHaveBeenCalled()
    expect(rfidApi.syncPending).toHaveBeenCalled()

    await vi.advanceTimersByTimeAsync(1500)
    await flushPromises()
    expect(exposed.chipState.value).toBe('online_synced')

    wrapper.unmount()
  })

  it('shows Offline when local bridge is offline but hosted still connected', async () => {
    ;(rfidApi.getBridgeStatus as Mock).mockResolvedValue({
      data: { connected: true, pending_count: 0, syncing: false },
    })
    ;(rfidApi.getLocalBridgeStatus as Mock).mockResolvedValue({
      connected: false,
      pending_count: 2,
      syncing: false,
      mode: 'offline',
    })

    const { exposed, wrapper } = mountComposable()
    await flushPromises()

    expect(exposed.chipState.value).toBe('offline')
    expect(rfidApi.getLocalBridgeStatus).toHaveBeenCalled()

    wrapper.unmount()
  })
})
