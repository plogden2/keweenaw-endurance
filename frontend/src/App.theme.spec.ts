import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { ref } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import App from '@/App.vue'
import { useStationStore } from '@/stores/station'
import { BLUFFET_EVENT_ID, BLUFFET_THEME_CLASS } from '@/themes/bluffetConstants'

vi.mock('@/composables/useReaderStation', () => ({
  useReaderStation: () => ({
    lastScan: ref(null),
    clearLastScan: vi.fn(),
    start: vi.fn(),
    stop: vi.fn(),
  }),
}))

async function mountApp(eventId = BLUFFET_EVENT_ID) {
  const pinia = createPinia()
  setActivePinia(pinia)

  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: { template: '<div />' } },
      { path: '/pin', component: { template: '<div />' } },
      { path: '/station', component: { template: '<div />' } },
      { path: '/csv', component: { template: '<div />' } },
    ],
  })
  await router.push('/')
  await router.isReady()

  const station = useStationStore()
  station.eventId = eventId
  vi.spyOn(station, 'fetchCurrent').mockResolvedValue({
    event_id: eventId,
    mode: 'finish',
    device_id: 'finish-1',
    name: 'Finish 1',
    checkpoint_id: null,
  })

  return mount(App, {
    global: {
      plugins: [pinia, router],
      stubs: {
        AppHeader: true,
        ScanPopup: true,
        UnitToggle: true,
      },
    },
  })
}

describe('App theme class', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('adds the Bluffet theme class when the station is armed for Bluffet', async () => {
    const wrapper = await mountApp()

    expect(wrapper.get('#app').classes()).toContain(BLUFFET_THEME_CLASS)
  })

  it('does not add the Bluffet theme class for an unrelated station eventId', async () => {
    const wrapper = await mountApp('unrelated-event-id')

    expect(wrapper.get('#app').classes()).not.toContain(BLUFFET_THEME_CLASS)
  })

  it('exposes a PIN link in the footer', async () => {
    const wrapper = await mountApp()
    const pin = wrapper.get('[data-testid="footer-pin"]')
    expect(pin.text()).toBe('PIN')
    expect(pin.attributes('href')).toBe('/pin')
  })

  it('exposes Station and CSV recovery in the footer', async () => {
    const wrapper = await mountApp()
    expect(wrapper.get('[data-testid="footer-station"]').attributes('href')).toBe('/station')
    expect(wrapper.get('[data-testid="footer-csv"]').attributes('href')).toBe('/csv')
  })
})
