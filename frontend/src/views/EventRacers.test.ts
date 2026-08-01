import { describe, it, expect, vi, beforeEach, afterEach, type Mock } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { nextTick } from 'vue'
import EventRacers from '@/views/EventRacers.vue'
import { setupPinia, createTestRouter } from '@/test/helpers'
import {
  eventParticipantsApi,
  eventsApi,
  raceParticipantsApi,
  racesApi,
  rfidApi,
} from '@/services/api'
import { usePinAuthStore } from '@/stores/pinAuth'

vi.mock('@/services/api', async () => {
  const actual = await vi.importActual<typeof import('@/services/api')>('@/services/api')
  return {
    ...actual,
    eventsApi: {
      get: vi.fn(),
    },
    racesApi: {
      list: vi.fn(),
    },
    eventParticipantsApi: {
      list: vi.fn(),
    },
    raceParticipantsApi: {
      create: vi.fn(),
      update: vi.fn(),
      remove: vi.fn(),
      listCategories: vi.fn(),
      listTags: vi.fn(),
    },
    rfidApi: {
      writeTag: vi.fn(),
    },
  }
})

const sampleRacers = [
  {
    id: 'p1',
    race_id: 'race-12h',
    bib_number: '12',
    first_name: 'Alex',
    last_name: 'Rivera',
    category_id: 'c1',
    tag_uids: ['TAG-A'],
    status: 'registered' as const,
    category: { id: 'c1', race_id: 'race-12h', name: 'Expert Men', category_type: 'custom' },
    race: {
      id: 'race-12h',
      event_id: 'e1',
      name: '12 Hour',
      race_type: 'time_based' as const,
      status: 'scheduled' as const,
    },
  },
  {
    id: 'p2',
    race_id: 'race-6h',
    bib_number: '18',
    first_name: 'Jordan',
    last_name: 'Lee',
    category_id: 'c2',
    tag_uids: [],
    status: 'registered' as const,
    category: {
      id: 'c2',
      race_id: 'race-6h',
      name: 'Open',
      category_type: 'custom',
    },
    race: {
      id: 'race-6h',
      event_id: 'e1',
      name: '6 Hour',
      race_type: 'time_based' as const,
      status: 'scheduled' as const,
    },
  },
  {
    id: 'p3',
    race_id: 'race-12h',
    bib_number: '',
    first_name: 'Sam',
    last_name: 'Ortiz',
    category_id: 'c1',
    tag_uids: [],
    status: 'registered' as const,
    category: { id: 'c1', race_id: 'race-12h', name: 'Expert Men', category_type: 'custom' },
    race: {
      id: 'race-12h',
      event_id: 'e1',
      name: '12 Hour',
      race_type: 'time_based' as const,
      status: 'scheduled' as const,
    },
  },
]

describe('EventRacers.vue', () => {
  beforeEach(() => {
    setupPinia()
    vi.clearAllMocks()
    vi.useFakeTimers()
    const pin = usePinAuthStore()
    pin.token = 'test-token'
    pin.role = 'admin'
    pin.expiresAt = Math.floor(Date.now() / 1000) + 3600
    ;(eventsApi.get as Mock).mockResolvedValue({
      data: { id: 'e1', name: 'All You Can East Bluffet', event_date: '2026-08-01', status: 'active' },
    })
    ;(racesApi.list as Mock).mockResolvedValue({
      data: {
        data: [
          {
            id: 'race-12h',
            event_id: 'e1',
            name: '12 Hour',
            race_type: 'time_based',
            status: 'scheduled',
          },
          {
            id: 'race-6h',
            event_id: 'e1',
            name: '6 Hour',
            race_type: 'time_based',
            status: 'scheduled',
          },
        ],
        total: 2,
      },
    })
    ;(eventParticipantsApi.list as Mock).mockResolvedValue({
      data: { data: structuredClone(sampleRacers), total: 3 },
    })
    ;(raceParticipantsApi.listCategories as Mock).mockResolvedValue({
      data: {
        data: [{ id: 'c1', race_id: 'race-12h', name: 'Expert Men', category_type: 'custom' }],
      },
    })
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  async function mountPage() {
    const router = createTestRouter([
      { path: '/pin', name: 'pin-unlock', component: { template: '<div />' } },
      {
        path: '/events/:eventId/racers',
        name: 'event-racers',
        component: EventRacers,
      },
      {
        path: '/events/:eventId/live',
        name: 'event-live',
        component: { template: '<div />' },
      },
      {
        path: '/races/:raceId/racers',
        name: 'race-racers',
        component: { template: '<div />' },
      },
    ])
    await router.push('/events/e1/racers')
    await router.isReady()

    const wrapper = mount(EventRacers, {
      global: { plugins: [router] },
    })
    await flushPromises()
    return wrapper
  }

  it('loads all event racers with race labels and search', async () => {
    const wrapper = await mountPage()
    expect(eventParticipantsApi.list).toHaveBeenCalledWith('e1', { limit: 500 })
    expect(wrapper.find('[data-testid="racers-page"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="event-racers-title"]').text()).toMatch(/Bluffet/)
    expect(wrapper.findAll('[data-testid="racer-row"]')).toHaveLength(3)
    expect(wrapper.text()).toContain('12 Hour')
    expect(wrapper.text()).toContain('6 Hour')

    await wrapper.find('[data-testid="racers-search"]').setValue('jordan')
    await vi.advanceTimersByTimeAsync(200)
    await nextTick()
    expect(wrapper.findAll('[data-testid="racer-row"]')).toHaveLength(1)
    expect(wrapper.text()).toContain('Jordan Lee')
  })

  it('filters by race and assigns bibs for unassigned racers', async () => {
    ;(raceParticipantsApi.update as Mock).mockResolvedValue({
      data: { ...sampleRacers[2], bib_number: '77', tag_uids: [] },
    })
    const wrapper = await mountPage()

    await wrapper.find('[data-testid="race-filter"]').setValue('race-12h')
    await nextTick()
    expect(wrapper.findAll('[data-testid="racer-row"]')).toHaveLength(2)

    const input = wrapper.find('[data-testid="bib-assign-input"]')
    expect(input.exists()).toBe(true)
    await input.setValue('77')
    await input.trigger('keydown.enter')
    await flushPromises()
    expect(raceParticipantsApi.update).toHaveBeenCalledWith('p3', { bib_number: '77' })
  })

  it('adds a racer after selecting a race', async () => {
    ;(raceParticipantsApi.create as Mock).mockResolvedValue({
      data: {
        id: 'p-new',
        race_id: 'race-6h',
        bib_number: '99',
        first_name: 'New',
        last_name: 'Racer',
        status: 'registered',
        tag_uids: [],
        race: sampleRacers[1].race,
      },
    })
    const wrapper = await mountPage()
    await wrapper.find('[data-testid="add-racer"]').trigger('click')
    await nextTick()

    await wrapper.find('[data-testid="add-race-select"]').setValue('race-6h')
    await flushPromises()
    expect(raceParticipantsApi.listCategories).toHaveBeenCalledWith('race-6h')

    await wrapper.find('[data-testid="racer-first-name"]').setValue('New')
    await wrapper.find('[data-testid="racer-last-name"]').setValue('Racer')
    await wrapper.find('[data-testid="racer-category"]').setValue('c1')
    await wrapper.find('[data-testid="racer-bib"]').setValue('99')
    await wrapper.get('[data-testid="add-racer-form"] form').trigger('submit')
    await flushPromises()

    expect(wrapper.find('.error').exists() ? wrapper.find('.error').text() : '').toBe('')
    expect(raceParticipantsApi.create).toHaveBeenCalledWith(
      'race-6h',
      expect.objectContaining({
        first_name: 'New',
        last_name: 'Racer',
        bib_number: '99',
      }),
    )
  })
})
