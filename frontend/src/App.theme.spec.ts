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

async function mountApp() {
  const pinia = createPinia()
  setActivePinia(pinia)

  const router = createRouter({
    history: createMemoryHistory(),
    routes: [{ path: '/', component: { template: '<div />' } }],
  })
  await router.push('/')
  await router.isReady()

  const station = useStationStore()
  station.eventId = BLUFFET_EVENT_ID
  vi.spyOn(station, 'fetchCurrent').mockResolvedValue({
    event_id: BLUFFET_EVENT_ID,
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
})
