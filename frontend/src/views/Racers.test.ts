import { describe, it, expect, vi, beforeEach, afterEach, type Mock } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { nextTick } from 'vue'
import Racers from '@/views/Racers.vue'
import { setupPinia, createTestRouter } from '@/test/helpers'
import { raceParticipantsApi, racesApi, raceTeamsApi, rfidApi } from '@/services/api'
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
    raceTeamsApi: {
      list: vi.fn(),
      create: vi.fn(),
      remove: vi.fn(),
      setMembers: vi.fn(),
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
    category: { id: 'c1', race_id: 'race-1', name: 'Expert Men', category_type: 'custom' },
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
          { id: 'c1', race_id: 'race-1', name: 'Expert Men', category_type: 'custom' },
          { id: 'c2', race_id: 'race-1', name: 'Intermediate Women', category_type: 'custom' },
        ],
      },
    })
    ;(raceParticipantsApi.list as Mock).mockResolvedValue({
      data: { data: structuredClone(sampleRacers), total: 2 },
    })
    ;(raceTeamsApi.list as Mock).mockResolvedValue({
      data: { data: [{ id: 'team-a', race_id: 'race-1', name: 'East Bluff A' }], total: 1 },
    })
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
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

  it('renders teams section and assigns team from racer dropdown', async () => {
    ;(raceParticipantsApi.update as Mock).mockResolvedValue({
      data: { ...sampleRacers[0], team_id: 'team-a' },
    })
    const wrapper = await mountRacers()

    expect(wrapper.find('[data-testid="teams-section"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="teams-list"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('East Bluff A')

    const select = wrapper.find('[data-testid="racer-team-select"]')
    await select.setValue('team-a')
    await flushPromises()

    expect(raceParticipantsApi.update).toHaveBeenCalledWith('p1', { team_id: 'team-a' })
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
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    ;(raceParticipantsApi.update as Mock).mockResolvedValue({
      data: { ...sampleRacers[0], bib_number: '9999', tag_uids: ['TAG-A'] },
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

  it('warns after bib save when returned tag_uids are empty', async () => {
    ;(raceParticipantsApi.update as Mock).mockResolvedValue({
      data: { ...sampleRacers[1], bib_number: '42', tag_uids: [] },
    })
    const wrapper = await mountRacers()
    const rows = wrapper.findAll('[data-testid="racer-row"]')
    await rows[1].find('[data-testid="bib-edit"]').trigger('click')
    await nextTick()

    await wrapper.find('[data-testid="bib-edit-input"]').setValue('42')
    await nextTick()
    await wrapper.find('[data-testid="bib-save"]').trigger('click')
    await flushPromises()

    expect(raceParticipantsApi.update).toHaveBeenCalledWith('p2', { bib_number: '42' })
    const warn = wrapper.find('[data-testid="bib-no-tags-warn"]')
    expect(warn.exists()).toBe(true)
    expect(warn.text().length).toBeGreaterThan(0)
  })

  it('confirms before changing bib when racer has tags; cancel leaves old bib', async () => {
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(false)
    ;(raceParticipantsApi.update as Mock).mockResolvedValue({
      data: { ...sampleRacers[0], bib_number: '9999' },
    })
    const wrapper = await mountRacers()
    const row = wrapper.find('[data-testid="racer-row"]')
    await row.find('[data-testid="bib-edit"]').trigger('click')
    await nextTick()

    await wrapper.find('[data-testid="bib-edit-input"]').setValue('9999')
    await nextTick()
    await wrapper.find('[data-testid="bib-save"]').trigger('click')
    await flushPromises()

    expect(confirmSpy).toHaveBeenCalled()
    expect(raceParticipantsApi.update).not.toHaveBeenCalled()
    // Exit edit without saving — display still shows original bib
    if (wrapper.find('[data-testid="bib-edit-input"]').exists()) {
      await wrapper.find('[data-testid="bib-edit-input"]').trigger('keydown.escape')
      await nextTick()
    }
    expect(row.find('[data-testid="bib-edit"]').text()).toContain('12')
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

    expect(rfidApi.writeTag).toHaveBeenCalledWith({
      participant_id: 'p2',
      race_id: 'race-1',
      logical_uuid: undefined,
    })
    expect(raceParticipantsApi.listTags).toHaveBeenCalledWith('race-1', 'p2')
    expect(wrapper.find('[data-testid="program-tag-list"]').text()).toContain(logicalUuid)
  })
})
