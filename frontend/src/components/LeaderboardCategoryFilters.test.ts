import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import LeaderboardCategoryFilters from './LeaderboardCategoryFilters.vue'
import { DEFAULT_CATEGORY_FILTER } from '@/utils/leaderboardCategoryFilter'

describe('LeaderboardCategoryFilters', () => {
  it('renders class and gender rows when both facets available', () => {
    const wrapper = mount(LeaderboardCategoryFilters, {
      props: {
        modelValue: DEFAULT_CATEGORY_FILTER,
        showSkill: true,
        showGender: true,
      },
    })
    expect(wrapper.find('[data-testid="lb-filter-skill"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="lb-filter-gender"]').exists()).toBe(true)
  })

  it('hides class row when showSkill is false', () => {
    const wrapper = mount(LeaderboardCategoryFilters, {
      props: {
        modelValue: DEFAULT_CATEGORY_FILTER,
        showSkill: false,
        showGender: true,
      },
    })
    expect(wrapper.find('[data-testid="lb-filter-skill"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="lb-filter-gender"]').exists()).toBe(true)
  })

  it('emits update when a chip is pressed', async () => {
    const wrapper = mount(LeaderboardCategoryFilters, {
      props: {
        modelValue: DEFAULT_CATEGORY_FILTER,
        showSkill: true,
        showGender: true,
      },
    })
    await wrapper.get('[data-testid="lb-filter-skill-expert"]').trigger('click')
    expect(wrapper.emitted('update:modelValue')?.[0]?.[0]).toEqual({
      skill: 'expert',
      gender: 'all',
    })
  })

  it('marks pressed chip with aria-pressed', () => {
    const wrapper = mount(LeaderboardCategoryFilters, {
      props: {
        modelValue: { skill: 'expert', gender: 'women' },
        showSkill: true,
        showGender: true,
      },
    })
    expect(
      wrapper.get('[data-testid="lb-filter-skill-expert"]').attributes('aria-pressed'),
    ).toBe('true')
    expect(
      wrapper.get('[data-testid="lb-filter-gender-women"]').attributes('aria-pressed'),
    ).toBe('true')
  })
})
