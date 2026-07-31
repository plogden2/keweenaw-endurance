import { describe, it, expect, vi, beforeEach, type Mock } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import EventBibs from './EventBibs.vue'
import { setupPinia, createTestRouter } from '@/test/helpers'
import { eventsApi, eventBibsApi, rfidApi } from '@/services/api'
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
    eventBibsApi: {
      list: vi.fn(),
      bulkCreate: vi.fn(),
      listTags: vi.fn(),
      addTag: vi.fn(),
    },
    rfidApi: {
      writeTag: vi.fn(),
    },
  }
})

const sampleBibs = [
  {
    id: 'bib-1',
    bib_number: '1',
    tag_count: 1,
    tag_uids: ['uuid-1'],
    participant_id: 'p1',
    participant_name: 'Alex Runner',
  },
  {
    id: 'bib-2',
    bib_number: '2',
    tag_count: 0,
    tag_uids: [],
  },
]

function authenticate() {
  const pin = usePinAuthStore()
  pin.token = 'test-token'
  pin.role = 'admin'
  pin.expiresAt = Math.floor(Date.now() / 1000) + 3600
}

describe('EventBibs.vue', () => {
  beforeEach(() => {
    setupPinia()
    vi.clearAllMocks()
    ;(eventsApi.get as Mock).mockResolvedValue({
      data: { id: 'e1', name: 'Bluffet', event_date: '2026-08-01', status: 'active' },
    })
    ;(eventBibsApi.list as Mock).mockResolvedValue({
      data: { data: sampleBibs },
    })
    ;(eventBibsApi.bulkCreate as Mock).mockResolvedValue({ data: [] })
    ;(rfidApi.writeTag as Mock).mockResolvedValue({
      data: { bib_id: 'bib-2', tag_uid: 'bib-2', tag_uids: ['bib-2'] },
    })
  })

  async function mountEventBibs() {
    const router = createTestRouter([
      { path: '/pin', name: 'pin-unlock', component: { template: '<div />' } },
      { path: '/events/:eventId/live', name: 'event-live', component: { template: '<div />' } },
      { path: '/events/:eventId/bibs', name: 'event-bibs', component: EventBibs },
    ])
    await router.push('/events/e1/bibs')
    await router.isReady()
    const wrapper = mount(EventBibs, { global: { plugins: [router] } })
    await flushPromises()
    return wrapper
  }

  it('renders inventory from the event bibs API', async () => {
    const wrapper = await mountEventBibs()

    expect(eventBibsApi.list).toHaveBeenCalledWith('e1')
    expect(wrapper.find('[data-testid="event-bibs-page"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="event-bibs-table"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('1')
    expect(wrapper.text()).toContain('Alex Runner')
    expect(wrapper.text()).toContain('unassigned')
  })

  it('hides bulk create and program actions without PIN', async () => {
    const wrapper = await mountEventBibs()
    expect(wrapper.find('[data-testid="bibs-bulk-create"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="bib-program-tag"]').exists()).toBe(false)
  })

  it('bulk create calls API when PIN authenticated', async () => {
    authenticate()
    const wrapper = await mountEventBibs()

    const form = wrapper.find('[data-testid="bibs-bulk-create"]')
    expect(form.exists()).toBe(true)
    await form.find('[data-testid="bibs-bulk-from"]').setValue('1')
    await form.find('[data-testid="bibs-bulk-to"]').setValue('100')
    await form.trigger('submit')
    await flushPromises()

    expect(eventBibsApi.bulkCreate).toHaveBeenCalledWith('e1', 1, 100)
    expect(eventBibsApi.list).toHaveBeenCalledTimes(2)
  })

  it('program tag calls writeTag with bib_id and logical_uuid', async () => {
    authenticate()
    const wrapper = await mountEventBibs()

    const programBtns = wrapper.findAll('[data-testid="bib-program-tag"]')
    expect(programBtns.length).toBeGreaterThan(0)
    await programBtns[1].trigger('click')
    await flushPromises()

    expect(rfidApi.writeTag).toHaveBeenCalledWith({
      bib_id: 'bib-2',
      logical_uuid: 'bib-2',
    })
  })
})
