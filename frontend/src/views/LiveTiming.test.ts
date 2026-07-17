import { describe, it, expect, vi, beforeEach, type Mock } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import LiveTiming from './LiveTiming.vue'
import { setupPinia, createTestRouter } from '@/test/helpers'
import { useRacesStore } from '@/stores/races'
import { checkpointsApi, rfidApi, timingApi } from '@/services/api'
import { participantsApi } from '@/services/api'

vi.mock('@/services/offlineQueue', () => ({
  enqueue: vi.fn().mockResolvedValue('synced'),
  getLocalPendingCount: vi.fn().mockResolvedValue(0),
  syncAll: vi.fn().mockResolvedValue({ synced: 0, failed: 0 }),
  onOnline: vi.fn(() => () => {}),
  initOfflineQueue: vi.fn(),
}))

vi.mock('@/stores/races', async () => {
  const actual = await vi.importActual<typeof import('@/stores/races')>('@/stores/races')
  return { ...actual, useRacesStore: vi.fn() }
})

vi.mock('@/services/api', async () => {
  const actual = await vi.importActual<typeof import('@/services/api')>('@/services/api')
  return {
    ...actual,
    checkpointsApi: { listByRace: vi.fn() },
    timingApi: { getLive: vi.fn() },
    rfidApi: {
      scan: vi.fn(),
      manualEntry: vi.fn(),
      getSyncStatus: vi.fn(),
      getBridgeStatus: vi.fn(),
      getLocalBridgeStatus: vi.fn(),
      syncPending: vi.fn(),
    },
    participantsApi: { list: vi.fn() },
  }
})

describe('LiveTiming.vue', () => {
  let racesStore: {
    currentRace: Record<string, unknown> | null
    loading: boolean
    error: string | null
    fetchRace: Mock
  }

  const checkpoints = [
    {
      id: 'cp-1',
      race_id: 'race-1',
      name: 'Start',
      checkpoint_type: 'start' as const,
      is_active: true,
    },
    {
      id: 'cp-2',
      race_id: 'race-1',
      name: 'Finish',
      checkpoint_type: 'finish' as const,
      is_active: true,
    },
  ]

  beforeEach(() => {
    setupPinia()
    racesStore = {
      currentRace: null,
      loading: false,
      error: null,
      fetchRace: vi.fn(),
    }
    ;(useRacesStore as unknown as Mock).mockReturnValue(racesStore)
    vi.clearAllMocks()
    ;(checkpointsApi.listByRace as Mock).mockResolvedValue({
      data: { data: checkpoints, total: 2 },
    })
    ;(timingApi.getLive as Mock).mockResolvedValue({
      data: { race_id: 'race-1', records: [] },
    })
    ;(rfidApi.getSyncStatus as Mock).mockResolvedValue({
      data: { pending_count: 2, failed_count: 0, synced_count: 10 },
    })
    ;(rfidApi.getBridgeStatus as Mock).mockResolvedValue({
      data: { connected: true, pending_count: 0, syncing: false },
    })
    ;(rfidApi.getLocalBridgeStatus as Mock).mockResolvedValue(null)
  })

  it('loads race, checkpoints, and live records on mount', async () => {
    racesStore.currentRace = {
      id: 'race-1',
      name: '50K',
      race_type: 'time_based',
      status: 'active',
    }

    const router = createTestRouter()
    await router.push('/timing/live/race-1')
    await router.isReady()

    mount(LiveTiming, { global: { plugins: [router] } })
    await flushPromises()

    expect(racesStore.fetchRace).toHaveBeenCalledWith('race-1')
    expect(checkpointsApi.listByRace).toHaveBeenCalledWith('race-1', { limit: 100 })
    expect(timingApi.getLive).toHaveBeenCalledWith('race-1')
  })

  it('looks up participant by bib number', async () => {
    racesStore.currentRace = {
      id: 'race-1',
      name: '50K',
      race_type: 'time_based',
      status: 'active',
    }
    ;(participantsApi.list as Mock).mockResolvedValue({
      data: {
        data: [
          {
            id: 'p-1',
            race_id: 'race-1',
            bib_number: '42',
            first_name: 'Alex',
            last_name: 'Runner',
            status: 'started',
          },
        ],
        total: 1,
      },
    })

    const router = createTestRouter()
    await router.push('/timing/live/race-1')
    await router.isReady()

    const wrapper = mount(LiveTiming, { global: { plugins: [router] } })
    await flushPromises()

    const bibInput = wrapper.find('[data-testid="bib-lookup"]')
    await bibInput.setValue('42')
    await wrapper.find('[data-testid="bib-lookup-btn"]').trigger('click')
    await flushPromises()

    expect(participantsApi.list).toHaveBeenCalledWith(
      expect.objectContaining({ race_id: 'race-1' }),
    )
    expect(wrapper.text()).toContain('Alex')
    expect(wrapper.text()).toContain('42')
  })

  it('looks up participant by RFID scan', async () => {
    racesStore.currentRace = {
      id: 'race-1',
      name: '50K',
      race_type: 'time_based',
      status: 'active',
    }
    ;(rfidApi.scan as Mock).mockResolvedValue({
      data: {
        id: 'p-2',
        race_id: 'race-1',
        bib_number: '7',
        first_name: 'Sam',
        last_name: 'Tag',
        rfid_tag_uid: 'TAG-001',
        status: 'started',
      },
    })

    const router = createTestRouter()
    await router.push('/timing/live/race-1')
    await router.isReady()

    const wrapper = mount(LiveTiming, { global: { plugins: [router] } })
    await flushPromises()

    await wrapper.find('[data-testid="rfid-lookup"]').setValue('TAG-001')
    await wrapper.find('[data-testid="rfid-lookup-btn"]').trigger('click')
    await flushPromises()

    expect(rfidApi.scan).toHaveBeenCalledWith('TAG-001')
    expect(wrapper.text()).toContain('Sam')
  })

  it('shows sync status counts', async () => {
    racesStore.currentRace = {
      id: 'race-1',
      name: '50K',
      race_type: 'time_based',
      status: 'active',
    }

    const router = createTestRouter()
    await router.push('/timing/live/race-1')
    await router.isReady()

    const wrapper = mount(LiveTiming, { global: { plugins: [router] } })
    await flushPromises()

    expect(wrapper.text()).toContain('Pending')
    expect(wrapper.text()).toContain('2')
  })
})
