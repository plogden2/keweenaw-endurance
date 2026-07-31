import { describe, it, expect, vi, beforeEach, type Mock } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import EventTaps from './EventTaps.vue'
import { setupPinia, createTestRouter } from '@/test/helpers'
import { eventsApi, eventParticipantsApi, eventTapsApi, timingRecordsApi } from '@/services/api'
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
    eventParticipantsApi: { list: vi.fn() },
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

const sampleParticipant = {
  id: 'p1',
  race_id: 'race-1',
  bib_number: '42',
  first_name: 'Alex',
  last_name: 'Runner',
  status: 'started' as const,
  race,
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
    ;(eventParticipantsApi.list as Mock).mockResolvedValue({
      data: { data: [sampleParticipant], total: 1 },
    })
    ;(eventTapsApi.create as Mock).mockResolvedValue({ data: {} })
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

  async function submitBib(wrapper: Awaited<ReturnType<typeof mountEventTaps>>, bib: string) {
    const input = wrapper.find('[data-testid="inline-bib-input"]')
    await input.setValue(bib)
    await input.trigger('keydown.enter')
    await flushPromises()
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

  it('hides inline bib input and row actions without PIN', async () => {
    const wrapper = await mountEventTaps()
    expect(wrapper.find('[data-testid="inline-bib-input"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="void-tap-btn"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="restore-tap-btn"]').exists()).toBe(false)
  })

  it('shows inline bib input and row actions when PIN authenticated', async () => {
    authenticate()
    const wrapper = await mountEventTaps()
    expect(wrapper.find('[data-testid="inline-bib-input"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="void-tap-btn"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="restore-tap-btn"]').exists()).toBe(true)
  })

  it('records a tap on Enter with exact bib match and karaoke_bonus false by default', async () => {
    authenticate()
    const wrapper = await mountEventTaps()

    expect(wrapper.find('[data-testid="inline-karaoke-toggle"]').exists()).toBe(true)
    await submitBib(wrapper, '42')

    expect(eventParticipantsApi.list).toHaveBeenCalledWith('e1', { q: '42', limit: 20 })
    expect(eventTapsApi.create).toHaveBeenCalledWith('e1', {
      participant_id: 'p1',
      karaoke_bonus: false,
    })
  })

  it('records karaoke_bonus true when the karaoke toggle is checked', async () => {
    authenticate()
    const wrapper = await mountEventTaps()

    await wrapper.find('[data-testid="inline-karaoke-toggle"]').setValue(true)
    await submitBib(wrapper, '42')

    expect(eventTapsApi.create).toHaveBeenCalledWith('e1', {
      participant_id: 'p1',
      karaoke_bonus: true,
    })
  })

  it('shows not-found error and does not create when zero exact matches', async () => {
    authenticate()
    ;(eventParticipantsApi.list as Mock).mockResolvedValue({
      data: { data: [], total: 0 },
    })
    const wrapper = await mountEventTaps()

    await submitBib(wrapper, '99')

    expect(eventTapsApi.create).not.toHaveBeenCalled()
    const error = wrapper.find('[data-testid="inline-bib-error"]')
    expect(error.exists()).toBe(true)
    expect(error.text().toLowerCase()).toMatch(/not found|no match/)
  })

  it('shows error and does not create when multiple exact matches', async () => {
    authenticate()
    ;(eventParticipantsApi.list as Mock).mockResolvedValue({
      data: {
        data: [
          sampleParticipant,
          { ...sampleParticipant, id: 'p1b', race_id: 'race-2', race: { ...race, id: 'race-2', name: '100 Mile' } },
        ],
        total: 2,
      },
    })
    const wrapper = await mountEventTaps()

    await submitBib(wrapper, '42')

    expect(eventTapsApi.create).not.toHaveBeenCalled()
    expect(wrapper.find('[data-testid="inline-bib-error"]').exists()).toBe(true)
  })

  it('clears the input and refreshes taps after a successful create', async () => {
    authenticate()
    const wrapper = await mountEventTaps()
    const listCallsBefore = (eventTapsApi.list as Mock).mock.calls.length

    await submitBib(wrapper, '42')

    const input = wrapper.find('[data-testid="inline-bib-input"]')
    expect((input.element as HTMLInputElement).value).toBe('')
    expect((eventTapsApi.list as Mock).mock.calls.length).toBeGreaterThan(listCallsBefore)
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
