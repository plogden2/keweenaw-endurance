import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import LapCelebrationOverlay from './LapCelebrationOverlay.vue'

const overlaySrc = readFileSync(
  join(process.cwd(), 'src/components/LapCelebrationOverlay.vue'),
  'utf8',
)

describe('LapCelebrationOverlay.vue', () => {
  it('renders celebration with name and +1 when visible', () => {
    const wrapper = mount(LapCelebrationOverlay, {
      props: { name: 'Alex Rivera', visible: true },
    })

    const celebration = wrapper.find('[data-testid="lap-celebration"]')
    expect(celebration.exists()).toBe(true)
    expect(celebration.text()).toContain('Alex Rivera')
    expect(celebration.text()).toContain('+1')
  })

  it('does not render when visible is false', () => {
    const wrapper = mount(LapCelebrationOverlay, {
      props: { name: 'Alex Rivera', visible: false },
    })

    expect(wrapper.find('[data-testid="lap-celebration"]').exists()).toBe(false)
  })

  it('pins to the viewport above the fullscreen rotator and sync chip', () => {
    const style = overlaySrc.split('<style scoped>')[1]?.split('</style>')[0] ?? ''
    const block = style.match(/\.lap-celebration\s*\{[^}]+\}/s)?.[0] ?? ''
    expect(block).toMatch(/position:\s*fixed/)
    // fs-root is 1000; sync-bar--overlay is 1100
    expect(block).toMatch(/z-index:\s*12\d{2}/)
  })
})
