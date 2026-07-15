import { describe, it, expect, vi, beforeEach, afterEach, type Mock } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { nextTick } from 'vue'
import Racers from '@/views/Racers.vue'
import { setupPinia, createTestRouter } from '@/test/helpers'
import { raceParticipantsApi, racesApi, rfidApi } from '@/services/api'
import { usePinAuthStore } from '@/stores/pinAuth'

vi.mock('@/services/api', async () => {
  const actual = await vi.importActual<typeof import('@/services/api')>('@/services/api')
  return {
    ...actual,
    racesApi: {
      get: vi.fn(),
    },
    raceParticipantsApi: {
      list: vi.fn(),
      create: vi.fn(),
      update: vi.fn(),
      listCategories: vi.fn(),
      listTags: vi.fn(),
      addTag: vi.fn(),
    },
    rfidApi: {
      writeTag: vi.fn(),
    },
  }
})

const sampleRacers = [
  {
    id: 'p1',
    race_id: 'race-1',
    bib_number: '12',
    first_name: 'Alex',
    last_name: 'Rivera',
    category_id: 'c1',
    tag_uids: ['TAG-A'],
    status: 'registered' as const,
    category: { id: 'c1', race_id: 'race-1', name: 'Advanced Men', category_type: 'custom' },
  },
  {
    id: 'p2',
    race_id: 'race-1',
    bib_number: '18',
    first_name: 'Jordan',
    last_name: 'Lee',
    category_id: 'c2',
    tag_uids: [],
    status: 'registered' as const,
    category: {
      id: 'c2',
      race_id: 'race-1',
      name: 'Intermediate Women',
      category_type: 'custom',
    },
  },
]

describe('Racers.vue', () => {
  beforeEach(() => {
    setupPinia()
    vi.clearAllMocks()
    vi.useFakeTimers()
    const pin = usePinAuthStore()
    pin.token = 'test-token'
    pin.role = 'admin'
    pin.expiresAt = Math.floor(Date.now() / 1000) + 3600
    ;(racesApi.get as Mock).mockResolvedValue({
      data: { id: 'race-1', name: '12 Hour', event_id: 'e1', race_type: 'time_based', status: 'scheduled' },
    })
    ;(raceParticipantsApi.listCategories as Mock).mockResolvedValue({
      data: {
        data: [
          { id: 'c1', race_id: 'race-1', name: 'Advanced Men', category_type: 'custom' },
          { id: 'c2', race_id: 'race-1', name: 'Intermediate Women', category_type: 'custom' },
        ],
      },
    })
    ;(raceParticipantsApi.list as Mock).mockResolvedValue({
      data: { data: sampleRacers, total: 2 },
    })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  async function mountRacers() {
    const router = createTestRouter([
      { path: '/pin', name: 'pin-unlock', component: { template: '<div />' } },
      {
        path: '/races/:raceId/racers',
        name: 'race-racers',
        component: Racers,
      },
    ])
    await router.push('/races/race-1/racers')
    await router.isReady()

    const wrapper = mount(Racers, {
      global: { plugins: [router] },
    })
    await flushPromises()
    return wrapper
  }

  it('renders search and racer rows', async () => {
    const wrapper = await mountRacers()
    expect(wrapper.find('[data-testid="racers-search"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="racers-list"]').exists()).toBe(true)
    expect(wrapper.findAll('[data-testid="racer-row"]')).toHaveLength(2)
  })

  it('debounces search filtering (~200ms)', async () => {
    const wrapper = await mountRacers()
    const search = wrapper.find('[data-testid="racers-search"]')
    await search.setValue('zzzz-no-match')
    await nextTick()
    // Before debounce fires, rows still visible
    expect(wrapper.findAll('[data-testid="racer-row"]')).toHaveLength(2)

    await vi.advanceTimersByTimeAsync(200)
    await nextTick()
    expect(wrapper.findAll('[data-testid="racer-row"]')).toHaveLength(0)

    await search.setValue('alex')
    await vi.advanceTimersByTimeAsync(200)
    await nextTick()
    expect(wrapper.findAll('[data-testid="racer-row"]')).toHaveLength(1)
    expect(wrapper.text()).toContain('Alex Rivera')
  })

  it('shows bib save only when dirty and persists on save', async () => {
    ;(raceParticipantsApi.update as Mock).mockResolvedValue({
      data: { ...sampleRacers[0], bib_number: '9999' },
    })
    const wrapper = await mountRacers()
    const row = wrapper.find('[data-testid="racer-row"]')
    await row.find('[data-testid="bib-edit"]').trigger('click')
    await nextTick()

    expect(wrapper.find('[data-testid="bib-edit-input"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="bib-save"]').exists()).toBe(false)

    await wrapper.find('[data-testid="bib-edit-input"]').setValue('9999')
    await nextTick()
    expect(wrapper.find('[data-testid="bib-save"]').exists()).toBe(true)

    await wrapper.find('[data-testid="bib-save"]').trigger('click')
    await flushPromises()
    expect(raceParticipantsApi.update).toHaveBeenCalledWith('p1', { bib_number: '9999' })
  })

  it('programs tag via writeTag without silicon UID input', async () => {
    const logicalUuid = 'a1b2c3d4-e5f6-7890-abcd-ef1234567890'
    ;(rfidApi.writeTag as Mock).mockResolvedValue({
      data: { ...sampleRacers[1], tag_uids: [logicalUuid] },
    })
    ;(raceParticipantsApi.listTags as Mock).mockResolvedValue({
      data: { data: [{ tag_uid: logicalUuid, participant_id: 'p2', active: true }] },
    })
    const wrapper = await mountRacers()
    const rows = wrapper.findAll('[data-testid="racer-row"]')
    await rows[1].find('[data-testid="program-tag"]').trigger('click')
    await nextTick()

    expect(wrapper.find('[data-testid="program-tag-uid"]').exists()).toBe(false)
    const writeBtn = wrapper.find('[data-testid="program-tag-write"]')
    expect(writeBtn.attributes('disabled')).toBeUndefined()

    await writeBtn.trigger('click')
    await flushPromises()

    expect(rfidApi.writeTag).toHaveBeenCalledWith({ participant_id: 'p2' })
    expect(raceParticipantsApi.listTags).toHaveBeenCalledWith('race-1', 'p2')
    expect(wrapper.find('[data-testid="program-tag-list"]').text()).toContain(logicalUuid)
  })
})
