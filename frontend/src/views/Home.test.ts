import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { beforeEach, describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createRouter, createWebHistory } from 'vue-router'
import { createPinia, setActivePinia } from 'pinia'
import Home from './Home.vue'
import { useEventsStore } from '@/stores/events'

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
  })

  it('renders hero and link to timing section', async () => {
    const router = createHomeRouter()
    await router.push('/')
    await router.isReady()

    const wrapper = mount(Home, { global: { plugins: [router] } })

    expect(wrapper.text()).toContain('Keweenaw Endurance Syndicate Race Timing')
    const cta = wrapper.find('[data-testid="timing-cta"]')
    // No Bluffet id yet → events list
    expect(cta.attributes('href')).toBe('/timing')
  })

  it('shows featured Bluffet links', async () => {
    const eventsStore = useEventsStore()
    eventsStore.events = [
      {
        id: 'a1b2c3',
        name: 'All You Can East Bluffet',
        event_date: '2026-08-01',
        status: 'upcoming',
      },
    ]

    const router = createHomeRouter()
    await router.push('/')
    await router.isReady()
    const wrapper = mount(Home, { global: { plugins: [router] } })
    await wrapper.vm.$nextTick()

    const registerLink = wrapper.find('[data-testid="bluffet-link"]')
    expect(registerLink.attributes('href')).toBe('https://www.copperharbortrails.org/bluffet')
    expect(registerLink.attributes('target')).toBe('_blank')

    const timingLink = wrapper.find('[data-testid="bluffet-timing-link"]')
    expect(timingLink.attributes('href')).toBe('/events/a1b2c3/live')
    expect(wrapper.find('[data-testid="timing-cta"]').attributes('href')).toBe(
      '/events/a1b2c3/live',
    )
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

  it('renders teaser race cards with external links only', () => {
    const router = createHomeRouter()
    const wrapper = mount(Home, { global: { plugins: [router] } })

    const cards = wrapper.findAll('[data-testid="race-card"]')
    expect(cards).toHaveLength(3)

    const externalLinks = wrapper.findAll('[data-testid="race-external-link"]')
    expect(externalLinks).toHaveLength(3)
    for (const link of externalLinks) {
      expect(link.attributes('href')).toMatch(/^https:\/\//)
      expect(link.attributes('rel')).toContain('nofollow')
      expect(link.attributes('target')).toBe('_blank')
    }

    expect(wrapper.text()).toContain('Summer Trail Challenge')
    expect(wrapper.text()).not.toContain('event_date')
  })
})
