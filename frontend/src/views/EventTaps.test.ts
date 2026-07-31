import { describe, it, expect, vi, beforeEach, type Mock } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import EventTaps from './EventTaps.vue'
import { setupPinia, createTestRouter } from '@/test/helpers'
import { eventsApi, eventTapsApi, timingRecordsApi } from '@/services/api'
import { usePinAuthStore } from '@/stores/pinAuth'

vi.mock('@/services/api', async () => {
  const actual = await vi.importActual<typeof import('@/services/api')>('@/services/api')
  return {
    ...actual,
    eventsApi: {
      get: vi.fn(),
      list: vi.fn(),
      create: vi.fn(),
      update: vi.fn(),
      remove: vi.fn(),
    },
    eventTapsApi: { list: vi.fn(), create: vi.fn() },
    timingRecordsApi: { voidRecord: vi.fn(), restoreRecord: vi.fn(), karaokeBonus: vi.fn() },
  }
})

const race = {
  id: 'race-1',
  event_id: 'e1',
  name: '12 Hour',
  race_type: 'lap_based' as const,
  status: 'active' as const,
}

const sampleTaps = [
  {
    id: 't1',
    participant_id: 'p1',
    checkpoint_id: 'cp1',
    timestamp: '2026-08-01T12:00:00Z',
    local_timestamp: '2026-08-01T12:00:00Z',
    sync_status: 'synced' as const,
    record_type: 'rfid_lap' as const,
    voided_at: null,
    participant: {
      id: 'p1',
      race_id: 'race-1',
      bib_number: '42',
      first_name: 'Alex',
      last_name: 'Runner',
      status: 'started' as const,
      race,
    },
  },
  {
    id: 't2',
    participant_id: 'p2',
    checkpoint_id: 'cp1',
    timestamp: '2026-08-01T11:00:00Z',
    local_timestamp: '2026-08-01T11:00:00Z',
    sync_status: 'synced' as const,
    record_type: 'karaoke_bonus' as const,
    voided_at: '2026-08-01T11:05:00Z',
    participant: {
      id: 'p2',
      race_id: 'race-1',
      bib_number: '7',
      first_name: 'Sam',
      last_name: 'Voided',
      status: 'started' as const,
      race,
    },
  },
]

function authenticate() {
  const pin = usePinAuthStore()
  pin.token = 'test-token'
  pin.role = 'admin'
  pin.expiresAt = Math.floor(Date.now() / 1000) + 3600
}

describe('EventTaps.vue', () => {
  beforeEach(() => {
    setupPinia()
    vi.clearAllMocks()
    ;(eventsApi.get as Mock).mockResolvedValue({
      data: { id: 'e1', name: 'Bluffet', event_date: '2026-08-01', status: 'active' },
    })
    ;(eventTapsApi.list as Mock).mockResolvedValue({
      data: { data: sampleTaps, total: 2, page: 1, limit: 50 },
    })
  })

  async function mountEventTaps() {
    const router = createTestRouter([
      { path: '/events/:eventId/taps', name: 'event-taps', component: EventTaps },
    ])
    await router.push('/events/e1/taps')
    await router.isReady()
    const wrapper = mount(EventTaps, { global: { plugins: [router] } })
    await flushPromises()
    return wrapper
  }

  it('renders rows from the event taps API', async () => {
    const wrapper = await mountEventTaps()

    expect(eventTapsApi.list).toHaveBeenCalledWith('e1', { page: 1, limit: 50 })
    expect(wrapper.find('[data-testid="event-taps-table"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="tap-row-t1"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('Alex')
    expect(wrapper.text()).toContain('Lap')
    expect(wrapper.text()).toContain('Karaoke')
  })

  it('grays out voided rows and shows a badge', async () => {
    const wrapper = await mountEventTaps()
    const voidedRow = wrapper.find('[data-testid="tap-row-t2"]')
    expect(voidedRow.classes()).toContain('voided')
    expect(voidedRow.find('[data-testid="voided-badge"]').exists()).toBe(true)
  })

  it('hides add tap button and row actions without PIN', async () => {
    const wrapper = await mountEventTaps()
    expect(wrapper.find('[data-testid="add-tap-btn"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="void-tap-btn"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="restore-tap-btn"]').exists()).toBe(false)
  })

  it('shows add tap button and row actions when PIN authenticated', async () => {
    authenticate()
    const wrapper = await mountEventTaps()
    expect(wrapper.find('[data-testid="add-tap-btn"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="void-tap-btn"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="restore-tap-btn"]').exists()).toBe(true)
  })

  it('voids a tap after confirm and reloads the table', async () => {
    authenticate()
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    ;(timingRecordsApi.voidRecord as Mock).mockResolvedValue({ data: {} })
    const wrapper = await mountEventTaps()

    await wrapper.find('[data-testid="void-tap-btn"]').trigger('click')
    await flushPromises()

    expect(timingRecordsApi.voidRecord).toHaveBeenCalledWith('t1')
    expect(eventTapsApi.list).toHaveBeenCalledTimes(2)
  })

  it('restores a voided tap after confirm', async () => {
    authenticate()
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    ;(timingRecordsApi.restoreRecord as Mock).mockResolvedValue({ data: {} })
    const wrapper = await mountEventTaps()

    await wrapper.find('[data-testid="restore-tap-btn"]').trigger('click')
    await flushPromises()

    expect(timingRecordsApi.restoreRecord).toHaveBeenCalledWith('t2')
  })

  it('opens the add tap dialog', async () => {
    authenticate()
    const wrapper = await mountEventTaps()

    expect(wrapper.find('[data-testid="add-tap-dialog"]').exists()).toBe(false)
    await wrapper.find('[data-testid="add-tap-btn"]').trigger('click')
    await flushPromises()
    expect(wrapper.find('[data-testid="add-tap-dialog"]').exists()).toBe(true)
  })

  it('shows pagination controls when there are multiple pages', async () => {
    ;(eventTapsApi.list as Mock).mockResolvedValue({
      data: { data: sampleTaps, total: 120, page: 1, limit: 50 },
    })
    const wrapper = await mountEventTaps()

    expect(wrapper.find('[data-testid="taps-pagination"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="taps-prev-page"]').attributes('disabled')).toBeDefined()

    await wrapper.find('[data-testid="taps-next-page"]').trigger('click')
    await flushPromises()

    expect(eventTapsApi.list).toHaveBeenLastCalledWith('e1', { page: 2, limit: 50 })
  })
})
