import { describe, it, expect, vi, beforeEach, afterEach, type Mock } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { nextTick } from 'vue'
import AddTapDialog from './AddTapDialog.vue'
import { eventParticipantsApi, eventTapsApi } from '@/services/api'

vi.mock('@/services/api', async () => {
  const actual = await vi.importActual<typeof import('@/services/api')>('@/services/api')
  return {
    ...actual,
    eventParticipantsApi: { list: vi.fn() },
    eventTapsApi: { create: vi.fn() },
  }
})

const sampleParticipants = [
  {
    id: 'p1',
    race_id: 'race-1',
    bib_number: '42',
    first_name: 'Alex',
    last_name: 'Runner',
    status: 'started' as const,
    race: {
      id: 'race-1',
      event_id: 'e1',
      name: '12 Hour',
      race_type: 'lap_based' as const,
      status: 'active' as const,
    },
  },
]

describe('AddTapDialog.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.useFakeTimers()
    ;(eventParticipantsApi.list as Mock).mockResolvedValue({
      data: { data: sampleParticipants, total: 1 },
    })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('debounces search and lists results with race context', async () => {
    const wrapper = mount(AddTapDialog, { props: { eventId: 'e1' } })

    await wrapper.find('[data-testid="tap-participant-search"]').setValue('42')
    await nextTick()
    expect(eventParticipantsApi.list).not.toHaveBeenCalled()

    await vi.advanceTimersByTimeAsync(250)
    await flushPromises()

    expect(eventParticipantsApi.list).toHaveBeenCalledWith('e1', { q: '42', limit: 20 })
    const options = wrapper.findAll('[data-testid="tap-participant-option"]')
    expect(options).toHaveLength(1)
    expect(options[0].text()).toContain('#42 Alex Runner (12 Hour)')
  })

  it('submits with karaoke_bonus true when the toggle is on', async () => {
    ;(eventTapsApi.create as Mock).mockResolvedValue({ data: {} })
    const wrapper = mount(AddTapDialog, { props: { eventId: 'e1' } })

    await wrapper.find('[data-testid="tap-participant-search"]').setValue('42')
    await vi.advanceTimersByTimeAsync(250)
    await flushPromises()
    await wrapper.find('[data-testid="tap-participant-option"]').trigger('click')

    await wrapper.find('[data-testid="karaoke-toggle"]').setValue(true)
    await wrapper.find('[data-testid="add-tap-submit"]').trigger('click')
    await flushPromises()

    expect(eventTapsApi.create).toHaveBeenCalledWith('e1', {
      participant_id: 'p1',
      karaoke_bonus: true,
    })
    expect(wrapper.emitted('refresh')).toBeTruthy()
    expect(wrapper.emitted('close')).toBeTruthy()
  })

  it('submits with karaoke_bonus false by default', async () => {
    ;(eventTapsApi.create as Mock).mockResolvedValue({ data: {} })
    const wrapper = mount(AddTapDialog, { props: { eventId: 'e1' } })

    await wrapper.find('[data-testid="tap-participant-search"]').setValue('42')
    await vi.advanceTimersByTimeAsync(250)
    await flushPromises()
    await wrapper.find('[data-testid="tap-participant-option"]').trigger('click')
    await wrapper.find('[data-testid="add-tap-submit"]').trigger('click')
    await flushPromises()

    expect(eventTapsApi.create).toHaveBeenCalledWith('e1', {
      participant_id: 'p1',
      karaoke_bonus: false,
    })
  })

  it('requires a selected racer before submit', async () => {
    const wrapper = mount(AddTapDialog, { props: { eventId: 'e1' } })

    await wrapper.find('[data-testid="add-tap-submit"]').trigger('click')
    await flushPromises()

    expect(eventTapsApi.create).not.toHaveBeenCalled()
    expect(wrapper.find('[data-testid="add-tap-error"]').exists()).toBe(true)
  })

  it('emits close on cancel', async () => {
    const wrapper = mount(AddTapDialog, { props: { eventId: 'e1' } })

    await wrapper.find('[data-testid="add-tap-cancel"]').trigger('click')

    expect(wrapper.emitted('close')).toBeTruthy()
  })
})
