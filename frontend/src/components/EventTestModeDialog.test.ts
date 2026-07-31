import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { mount, flushPromises, type VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import EventTestModeDialog from './EventTestModeDialog.vue'
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
    ])
  })

  afterEach(() => {
    wrapper?.unmount()
    wrapper = null
  })

  it('renders banner, leaderboard, and records bib taps', async () => {
    wrapper = mount(EventTestModeDialog, mountOpts)
    await flushPromises()

    expect(wrapper.get('[data-testid="test-mode-banner"]').text()).toMatch(/This station only/)
    expect(wrapper.get('[data-testid="test-mode-leaderboard"]').text()).toMatch(/No test taps/)

    await wrapper.get('[data-testid="test-mode-bib-input"]').setValue('101')
    await wrapper.get('form.bib-form').trigger('submit')
    await flushPromises()

    expect(wrapper.get('[data-testid="test-mode-feedback"]').text()).toMatch(/Alex Rivera/)
    expect(wrapper.get('[data-testid="test-mode-leaderboard-laps"]').text()).toBe('1')
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
})
