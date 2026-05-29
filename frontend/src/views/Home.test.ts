import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import { createRouter, createWebHistory } from 'vue-router'
import Home from './Home.vue'

function createHomeRouter() {
  return createRouter({
    history: createWebHistory(),
    routes: [
      { path: '/', name: 'home', component: Home },
      { path: '/timing', name: 'timing', component: { template: '<div />' } },
    ],
  })
}

describe('Home.vue', () => {
  it('renders hero and link to timing section', async () => {
    const router = createHomeRouter()
    await router.push('/')
    await router.isReady()

    const wrapper = mount(Home, { global: { plugins: [router] } })

    expect(wrapper.text()).toContain('Keweenaw Endurance Race Timing')
    const cta = wrapper.find('[data-testid="timing-cta"]')
    expect(cta.attributes('href')).toBe('/timing')
  })

  it('shows featured Bluffet external link', () => {
    const router = createHomeRouter()
    const wrapper = mount(Home, { global: { plugins: [router] } })

    const link = wrapper.find('[data-testid="bluffet-link"]')
    expect(link.attributes('href')).toBe('https://www.copperharbortrails.org/bluffet')
    expect(link.attributes('target')).toBe('_blank')
    expect(link.text()).toContain('All You Can East Bluffet')
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
