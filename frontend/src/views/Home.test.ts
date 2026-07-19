import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { beforeEach, describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createRouter, createWebHistory } from 'vue-router'
import { createPinia, setActivePinia } from 'pinia'
import Home from './Home.vue'
import { useEventsStore } from '@/stores/events'
import { racesApi } from '@/services/api'

vi.mock('@/services/api', async () => {
  const actual = await vi.importActual<typeof import('@/services/api')>('@/services/api')
  return {
    ...actual,
    racesApi: {
      ...actual.racesApi,
      list: vi.fn(),
    },
  }
})

const bluffetCss = readFileSync(
  join(process.cwd(), 'src/themes/bluffet.css'),
  'utf8',
)

function createHomeRouter() {
  return createRouter({
    history: createWebHistory(),
    routes: [
      { path: '/', name: 'home', component: Home },
      { path: '/timing', name: 'timing', component: { template: '<div />' } },
      { path: '/timing/:eventId', name: 'event-timing', component: { template: '<div />' } },
      {
        path: '/events/:eventId/live',
        name: 'event-live',
        component: { template: '<div />' },
      },
    ],
  })
}

describe('Home.vue', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    const eventsStore = useEventsStore()
    vi.spyOn(eventsStore, 'fetchEvents').mockResolvedValue()
    vi.mocked(racesApi.list).mockReset()
    vi.mocked(racesApi.list).mockResolvedValue({ data: { data: [] } } as never)
  })

  it('shows featured event above the syndicate hero (upcoming races commented out)', async () => {
    const router = createHomeRouter()
    await router.push('/')
    await router.isReady()

    const wrapper = mount(Home, { global: { plugins: [router] } })
    const html = wrapper.html()

    expect(wrapper.find('#featured-event').exists()).toBe(true)
    expect(wrapper.find('[data-testid="timing-cta"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('Keweenaw Endurance Syndicate Race Timing')
    expect(wrapper.text()).not.toContain('Upcoming Races')
    expect(wrapper.findAll('[data-testid="race-card"]')).toHaveLength(0)
    expect(html.indexOf('featured-event')).toBeLessThan(html.indexOf('timing-cta'))
  })

  it('shows featured Bluffet links with Live race flow before races finish', async () => {
    const eventsStore = useEventsStore()
    eventsStore.events = [
      {
        id: 'a1b2c3',
        name: 'All You Can East Bluffet',
        event_date: '2026-08-01',
        status: 'upcoming',
      },
    ]
    vi.mocked(racesApi.list).mockResolvedValue({
      data: { data: [{ id: 'r1', status: 'scheduled' }] },
    } as never)

    const router = createHomeRouter()
    await router.push('/')
    await router.isReady()
    const wrapper = mount(Home, { global: { plugins: [router] } })
    await flushPromises()

    const registerLink = wrapper.find('[data-testid="bluffet-link"]')
    expect(registerLink.attributes('href')).toBe('https://www.copperharbortrails.org/bluffet')
    expect(registerLink.attributes('target')).toBe('_blank')

    const timingLink = wrapper.find('[data-testid="bluffet-timing-link"]')
    expect(timingLink.attributes('href')).toBe('/events/a1b2c3/live')
    expect(timingLink.text()).toBe('Live race flow')
    const poster = wrapper.find('[data-testid="bluffet-poster"]')
    expect(poster.exists()).toBe(true)
    expect(poster.find('source[type="image/avif"]').attributes('srcset')).toBe(
      '/images/bluffet-2026-poster.avif',
    )
    expect(wrapper.find('.featured-logo').attributes('src')).toBe(
      '/images/bluffet-2026-poster.png',
    )
    expect(wrapper.find('.featured-logo').attributes('alt')).toBe('All You Can East Bluffet')
    expect(wrapper.find('.featured-event').classes()).toContain('bluffet-theme')
    expect(wrapper.text()).toContain('August 1, 2026')
  })

  it('labels the featured timing link Results after races are finished', async () => {
    const eventsStore = useEventsStore()
    eventsStore.events = [
      {
        id: 'a1b2c3',
        name: 'All You Can East Bluffet',
        event_date: '2026-08-01',
        status: 'upcoming',
      },
    ]
    vi.mocked(racesApi.list).mockResolvedValue({
      data: {
        data: [
          { id: 'r1', status: 'finished' },
          { id: 'r2', status: 'finished' },
        ],
      },
    } as never)

    const router = createHomeRouter()
    await router.push('/')
    await router.isReady()
    const wrapper = mount(Home, { global: { plugins: [router] } })
    await flushPromises()

    const timingLink = wrapper.find('[data-testid="bluffet-timing-link"]')
    expect(timingLink.text()).toBe('Results')
    expect(timingLink.attributes('href')).toBe('/timing')
  })

  it('keeps Live race flow while any race is still active', async () => {
    const eventsStore = useEventsStore()
    eventsStore.events = [
      {
        id: 'a1b2c3',
        name: 'All You Can East Bluffet',
        event_date: '2026-08-01',
        status: 'active',
      },
    ]
    vi.mocked(racesApi.list).mockResolvedValue({
      data: {
        data: [
          { id: 'r1', status: 'finished' },
          { id: 'r2', status: 'active' },
        ],
      },
    } as never)

    const router = createHomeRouter()
    await router.push('/')
    await router.isReady()
    const wrapper = mount(Home, { global: { plugins: [router] } })
    await flushPromises()

    expect(wrapper.find('[data-testid="bluffet-timing-link"]').text()).toBe('Live race flow')
  })

  it('keeps the local Bluffet featured register link readable on tan paper', () => {
    expect(bluffetCss).toMatch(/--bluffet-teal:\s*#0f766e;/)
    expect(bluffetCss).toMatch(/#app\.theme-bluffet\s*\{[^}]*--accent:\s*var\(--bluffet-red\);/s)
    expect(bluffetCss).toMatch(
      /\.featured-event\.bluffet-theme\s+\.featured-link\s*\{[^}]*background:\s*var\(--bluffet-red[^;]*;[^}]*color:\s*#fff;/s,
    )
    expect(bluffetCss).toMatch(
      /\.featured-event\.bluffet-theme\s+\.featured-link\s*\{[^}]*border:\s*var\(--bluffet-outline[^;]*;/s,
    )
  })

  it('uses IBM Plex for Bluffet titles, not Yuji Mai', () => {
    expect(bluffetCss).not.toMatch(/Yuji Mai/)
    expect(bluffetCss).toMatch(
      /#app\.theme-bluffet\s*\{[^}]*font-family:\s*'IBM Plex Sans'/s,
    )
  })

  it('does not paint legend-item solid Bluffet red', () => {
    expect(bluffetCss).not.toMatch(
      /#app\.theme-bluffet\s+\.legend-item\s*\{[^}]*background:\s*var\(--bluffet-red\)/s,
    )
  })
})
