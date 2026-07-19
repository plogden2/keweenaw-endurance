import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import LapCelebrationOverlay from './LapCelebrationOverlay.vue'

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
})
