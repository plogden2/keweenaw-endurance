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
    id: 'bib-10',
    bib_number: '10',
    tag_count: 0,
    tag_uids: [],
  },
  {
    id: 'bib-100',
    bib_number: '100',
    tag_count: 3,
    tag_uids: [],
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

  it('sorts bib rows by bib number ascending numerically', async () => {
    const wrapper = await mountEventBibs()

    const nums = wrapper
      .findAll('[data-testid="event-bibs-table"] tbody tr')
      .filter((row) => row.attributes('data-testid')?.startsWith('bib-row-'))
      .map((row) => row.find('.bib-num').text())

    expect(nums).toEqual(['1', '2', '10', '100'])
  })

  it('program tag calls writeTag with bib_id and logical_uuid', async () => {
    authenticate()
    const wrapper = await mountEventBibs()

    const programBtns = wrapper.findAll('[data-testid="bib-program-tag"]')
    expect(programBtns.length).toBeGreaterThan(0)
    // Row order is bib asc: 1, 2, 10, 100 — program bib 2
    await programBtns[1].trigger('click')
    await flushPromises()

    expect(rfidApi.writeTag).toHaveBeenCalledWith({
      bib_id: 'bib-2',
      logical_uuid: 'bib-2',
    })
  })

  it('shows write success message after programming a tag', async () => {
    authenticate()
    const wrapper = await mountEventBibs()

    await wrapper.findAll('[data-testid="bib-program-tag"]')[1].trigger('click')
    await flushPromises()

    const success = wrapper.find('[data-testid="bib-program-success"]')
    expect(success.exists()).toBe(true)
    expect(success.text()).toMatch(/wrote tag for bib 2/i)
  })

  it('shows write fail message when programming fails', async () => {
    authenticate()
    ;(rfidApi.writeTag as Mock).mockRejectedValueOnce(new Error('Proxmark unavailable'))
    const wrapper = await mountEventBibs()

    await wrapper.findAll('[data-testid="bib-program-tag"]')[1].trigger('click')
    await flushPromises()

    const error = wrapper.find('[data-testid="bib-program-error"]')
    expect(error.exists()).toBe(true)
    expect(error.text()).toContain('Proxmark unavailable')
    expect(wrapper.find('[data-testid="bib-program-success"]').exists()).toBe(false)
  })

  it('shows axios 503 body and clears Writing state after write-tag fails', async () => {
    authenticate()
    const axiosErr = Object.assign(new Error('Request failed with status code 503'), {
      isAxiosError: true,
      response: {
        status: 503,
        data: { error: 'bridge unavailable' },
      },
    })
    ;(rfidApi.writeTag as Mock).mockRejectedValueOnce(axiosErr)
    const wrapper = await mountEventBibs()

    const btn = wrapper.findAll('[data-testid="bib-program-tag"]')[1]
    await btn.trigger('click')
    await flushPromises()

    const error = wrapper.find('[data-testid="bib-program-error"]')
    expect(error.exists()).toBe(true)
    expect(error.text()).toMatch(/bridge unavailable/i)
    expect(error.text()).toMatch(/503/)
    expect(btn.text()).toMatch(/Program/i)
    expect(btn.text()).not.toMatch(/Writing/i)
  })
})
