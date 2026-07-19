import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import AppHeader from './AppHeader.vue'
import { BLUFFET_EVENT_ID, BLUFFET_LOGO_PATH } from '@/themes/bluffetConstants'

const headerVue = readFileSync(join(process.cwd(), 'src/components/AppHeader.vue'), 'utf8')

async function mountHeader(path = '/') {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: { template: '<div />' } },
      { path: '/timing', component: { template: '<div />' } },
      { path: '/timing/:eventId', component: { template: '<div />' } },
      { path: '/station', component: { template: '<div />' } },
      { path: '/pin', component: { template: '<div />' } },
    ],
  })

  await router.push(path)
  await router.isReady()

  return mount(AppHeader, {
    global: {
      plugins: [router],
    },
  })
}

describe('AppHeader', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('shows the Bluffet nav mark when the Bluffet theme is active', async () => {
    const wrapper = await mountHeader(`/timing/${BLUFFET_EVENT_ID}`)
    const mark = wrapper.get('[data-testid="bluffet-nav-mark"]')

    expect(mark.attributes('src')).toBe(BLUFFET_LOGO_PATH)
    expect(mark.attributes('alt')).toBe('')
    expect(mark.attributes('width')).toBe('28')
    expect(mark.attributes('height')).toBe('28')
  })

  it('hides the Bluffet nav mark when the Bluffet theme is inactive', async () => {
    const wrapper = await mountHeader('/')

    expect(wrapper.find('[data-testid="bluffet-nav-mark"]').exists()).toBe(false)
  })

  it('uses evergreen ink token for header chrome', () => {
    expect(headerVue).toMatch(/background(-color)?:\s*var\(--ink\)/)
  })

  it('exposes Station and PIN nav links on every page', async () => {
    const wrapper = await mountHeader('/')
    expect(wrapper.get('[data-testid="nav-station"]').attributes('href')).toBe('/station')
    expect(wrapper.get('[data-testid="nav-pin"]').text()).toBe('PIN')
    expect(wrapper.get('[data-testid="nav-pin"]').attributes('href')).toBe('/pin')
  })
})
