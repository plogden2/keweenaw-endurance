import { describe, it, expect, beforeEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { defineComponent, h, nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import { useEventsStore } from '@/stores/events'
import { useStationStore } from '@/stores/station'
import type { Event } from '@/types/models'
import {
  BLUFFET_EVENT_ID,
  BLUFFET_EVENT_NAME,
  BLUFFET_LOGO_PATH,
  BLUFFET_POSTER_AVIF,
  BLUFFET_POSTER_PNG,
  BLUFFET_THEME_CLASS,
} from '@/themes/bluffetConstants'
import { useBluffetTheme } from './useBluffetTheme'

async function mountWithRoute(path: string) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: { template: '<div />' } },
      { path: '/timing/:eventId', component: { template: '<div />' } },
      { path: '/timing/:eventId/live', component: { template: '<div />' } },
      { path: '/events/:eventId/live', component: { template: '<div />' } },
    ],
  })
  await router.push(path)
  await router.isReady()

  let api!: ReturnType<typeof useBluffetTheme>
  const Comp = defineComponent({
    setup() {
      api = useBluffetTheme()
      return () => h('div')
    },
  })
  mount(Comp, { global: { plugins: [router] } })
  await nextTick()
  return api
}

describe('useBluffetTheme', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('exposes asset path constants via return value', async () => {
    const theme = await mountWithRoute('/')
    expect(theme.posterAvif.value).toBe(BLUFFET_POSTER_AVIF)
    expect(theme.posterPng.value).toBe(BLUFFET_POSTER_PNG)
    expect(theme.logoPath.value).toBe(BLUFFET_LOGO_PATH)
    expect(theme.themeClass.value).toBe(BLUFFET_THEME_CLASS)
  })

  it('activates when route eventId matches Bluffet UUID', async () => {
    const theme = await mountWithRoute(`/timing/${BLUFFET_EVENT_ID}`)
    expect(theme.active.value).toBe(true)
  })

  it('activates when currentEvent name matches', async () => {
    const events = useEventsStore()
    events.currentEvent = {
      id: 'other-id',
      name: BLUFFET_EVENT_NAME,
      event_date: '2026-08-01',
      status: 'upcoming',
    } satisfies Event
    const theme = await mountWithRoute('/')
    expect(theme.active.value).toBe(true)
  })

  it('activates when station eventId matches Bluffet UUID', async () => {
    const station = useStationStore()
    station.eventId = BLUFFET_EVENT_ID
    const theme = await mountWithRoute('/')
    expect(theme.active.value).toBe(true)
  })

  it('is inactive for unrelated event', async () => {
    const events = useEventsStore()
    events.currentEvent = {
      id: 'chtf',
      name: 'CHTF',
      event_date: '2025-01-01',
      status: 'upcoming',
    } satisfies Event
    const theme = await mountWithRoute('/timing/chtf')
    expect(theme.active.value).toBe(false)
  })
})
