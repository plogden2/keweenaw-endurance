import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { mount, flushPromises, type VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import EventTestModeDialog from './EventTestModeDialog.vue'
import { rfidApi } from '@/services/api'
import { useEventTestModeStore } from '@/stores/eventTestMode'

vi.mock('chart.js', () => ({
  Chart: Object.assign(
    vi.fn(() => ({
      destroy: vi.fn(),
      update: vi.fn(),
      resize: vi.fn(),
      data: { datasets: [] },
      options: {},
      canvas: { style: { cursor: 'default' } },
      getElementsAtEventForMode: vi.fn().mockReturnValue([]),
      getDatasetMeta: vi.fn().mockReturnValue({ data: [] }),
    })),
    { register: vi.fn() },
  ),
  LineController: vi.fn(),
  LineElement: vi.fn(),
  PointElement: vi.fn(),
  LinearScale: vi.fn(),
  CategoryScale: vi.fn(),
  Title: vi.fn(),
  Tooltip: vi.fn(),
  Legend: vi.fn(),
}))

const playMock = vi.fn().mockResolvedValue(undefined)

vi.stubGlobal(
  'Audio',
  vi.fn(function AudioMock(this: { play: typeof playMock; src: string }) {
    this.play = playMock
    this.src = ''
    return this
  }),
)

vi.mock('@/services/api', async () => {
  const actual = await vi.importActual<typeof import('@/services/api')>('@/services/api')
  return {
    ...actual,
    rfidApi: {
      ...actual.rfidApi,
      getLocalBridgeStatusForTestMode: vi.fn().mockResolvedValue(null),
      playLocalBridgeBeep: vi.fn().mockResolvedValue(false),
    },
  }
})

const mountOpts = {
  global: {
    stubs: {
      // Render dialog content inline so VTU can query without Teleport quirks.
      Teleport: { template: '<div><slot /></div>' },
      RaceFlowChart: {
        template: '<div data-testid="race-flow-chart-stub" />',
      },
    },
  },
}

describe('EventTestModeDialog', () => {
  let wrapper: VueWrapper | null = null

  beforeEach(() => {
    setActivePinia(createPinia())
    playMock.mockClear()
    vi.mocked(rfidApi.getLocalBridgeStatusForTestMode).mockResolvedValue(null)
    vi.mocked(rfidApi.playLocalBridgeBeep).mockResolvedValue(false)
    const store = useEventTestModeStore()
    store.open('ev1', [
      {
        id: 'p1',
        race_id: 'r1',
        bib_number: '101',
        first_name: 'Alex',
        last_name: 'Rivera',
        status: 'registered',
        tag_uids: ['TAG-1'],
        race: {
          id: 'r1',
          event_id: 'ev1',
          name: '12 Hour',
          race_type: 'time_based',
          status: 'active',
          duration_minutes: 720,
        },
      },
      {
        id: 'p-ben',
        race_id: 'r1',
        bib_number: '3',
        first_name: 'Benjamin',
        last_name: 'Ciavola',
        status: 'registered',
        tag_uids: ['7db35ca0-fdfc-44b5-a220-6d322d867f6f'],
      },
    ])
  })

  afterEach(() => {
    wrapper?.unmount()
    wrapper = null
    vi.useRealTimers()
  })

  it('renders banner, leaderboard, and records bib taps', async () => {
    const focusSpy = vi.spyOn(HTMLElement.prototype, 'focus')
    wrapper = mount(EventTestModeDialog, mountOpts)
    await flushPromises()

    expect(wrapper.get('[data-testid="test-mode-banner"]').text()).toMatch(/This station only/)
    expect(wrapper.get('[data-testid="test-mode-leaderboard"]').text()).toMatch(/No test taps/)

    await wrapper.get('[data-testid="test-mode-bib-input"]').setValue('101')
    focusSpy.mockClear()
    await wrapper.get('form.bib-form').trigger('submit')
    await flushPromises()

    expect(wrapper.get('[data-testid="test-mode-feedback"]').text()).toMatch(/Alex Rivera/)
    expect(wrapper.get('[data-testid="test-mode-leaderboard-laps"]').text()).toBe('1')
    expect(playMock).toHaveBeenCalled()
    expect(focusSpy).toHaveBeenCalled()
    focusSpy.mockRestore()
  })

  it('uses local bridge beep when bridge is online', async () => {
    vi.mocked(rfidApi.getLocalBridgeStatusForTestMode).mockResolvedValue({
      connected: true,
      pending_count: 0,
      syncing: false,
      mode: 'online_synced',
    })
    vi.mocked(rfidApi.playLocalBridgeBeep).mockResolvedValue(true)

    wrapper = mount(EventTestModeDialog, mountOpts)
    await flushPromises()

    await wrapper.get('[data-testid="test-mode-bib-input"]').setValue('101')
    await wrapper.get('form.bib-form').trigger('submit')
    await flushPromises()

    expect(rfidApi.playLocalBridgeBeep).toHaveBeenCalled()
    expect(playMock).not.toHaveBeenCalled()
  })

  it('emits close immediately when empty; confirms discard when taps exist', async () => {
    wrapper = mount(EventTestModeDialog, mountOpts)
    await flushPromises()

    await wrapper.get('[data-testid="test-mode-close"]').trigger('click')
    expect(wrapper.emitted('close')).toHaveLength(1)
    wrapper.unmount()

    const store = useEventTestModeStore()
    store.open('ev1', store.roster)
    store.recordBibTap('101')

    wrapper = mount(EventTestModeDialog, mountOpts)
    await flushPromises()
    await wrapper.get('[data-testid="test-mode-close"]').trigger('click')
    expect(wrapper.emitted('close')).toBeUndefined()
    expect(wrapper.find('[data-testid="test-mode-discard-confirm"]').exists()).toBe(true)
    await wrapper.get('[data-testid="test-mode-discard-confirm-btn"]').trigger('click')
    expect(wrapper.emitted('close')).toHaveLength(1)
  })

  it('polls local bridge status and records new RFID taps after baseline', async () => {
    vi.useFakeTimers()
    const statusSpy = vi.mocked(rfidApi.getLocalBridgeStatusForTestMode)
    statusSpy.mockResolvedValue({
      connected: true,
      pending_count: 0,
      syncing: false,
      mode: 'online_synced',
      last_tap_uuid: '7db35ca0-fdfc-44b5-a220-6d322d867f6f',
      last_tap_at: '2026-07-31T21:25:00.000Z',
      last_tap_bib: '3',
      last_tap_race_id: 'r1',
    })

    wrapper = mount(EventTestModeDialog, mountOpts)
    await flushPromises()

    const store = useEventTestModeStore()
    expect(store.taps).toHaveLength(0)

    statusSpy.mockResolvedValue({
      connected: true,
      pending_count: 0,
      syncing: false,
      mode: 'online_synced',
      last_tap_uuid: '7db35ca0-fdfc-44b5-a220-6d322d867f6f',
      last_tap_at: '2026-07-31T21:26:10.000Z',
      last_tap_bib: '3',
      last_tap_race_id: 'r1',
    })
    await vi.advanceTimersByTimeAsync(500)
    await flushPromises()

    expect(store.taps).toHaveLength(1)
    expect(store.lastFeedback?.participant_name).toMatch(/Benjamin/)
    expect(wrapper.get('[data-testid="test-mode-feedback"]').text()).toMatch(/Benjamin/)
  })
})
