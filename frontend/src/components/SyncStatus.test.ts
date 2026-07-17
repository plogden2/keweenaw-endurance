import { describe, it, expect, vi, beforeEach, type Mock } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import SyncStatus from '@/components/SyncStatus.vue'
import { rfidApi } from '@/services/api'

vi.mock('@/services/offlineQueue', () => ({
  getLocalPendingCount: vi.fn().mockResolvedValue(0),
  onOnline: vi.fn(() => () => {}),
  syncAll: vi.fn().mockResolvedValue({ synced: 0, failed: 0 }),
}))

vi.mock('@/services/api', async () => {
  const actual = await vi.importActual<typeof import('@/services/api')>('@/services/api')
  return {
    ...actual,
    rfidApi: {
      getSyncStatus: vi.fn().mockResolvedValue({
        data: { pending_count: 0, failed_count: 0, synced_count: 0 },
      }),
      getBridgeStatus: vi.fn().mockResolvedValue({
        data: { connected: true, pending_count: 0, syncing: false },
      }),
      getLocalBridgeStatus: vi.fn().mockResolvedValue(null),
      syncPending: vi.fn().mockResolvedValue({ data: { synced_count: 0 } }),
    },
  }
})

describe('SyncStatus.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    Object.defineProperty(navigator, 'onLine', { value: true, configurable: true })
  })

  it('shows Online · Synced chip when bridge is connected', async () => {
    ;(rfidApi.getBridgeStatus as Mock).mockResolvedValue({
      data: { connected: true, pending_count: 0, syncing: false },
    })

    const wrapper = mount(SyncStatus)
    await flushPromises()

    expect(wrapper.find('[data-testid="sync-chip-online"]').text()).toBe('Online · Synced')
  })

  it('shows Offline chip when navigator is offline', async () => {
    Object.defineProperty(navigator, 'onLine', { value: false, configurable: true })

    const wrapper = mount(SyncStatus)
    await flushPromises()

    expect(wrapper.find('[data-testid="sync-chip-offline"]').text()).toBe('Offline')
  })

  it('shows Syncing chip when bridge reports syncing', async () => {
    ;(rfidApi.getBridgeStatus as Mock).mockResolvedValue({
      data: { connected: true, pending_count: 2, syncing: true },
    })

    const wrapper = mount(SyncStatus)
    await flushPromises()

    expect(wrapper.find('[data-testid="sync-chip-syncing"]').text()).toBe('Syncing')
  })
})
