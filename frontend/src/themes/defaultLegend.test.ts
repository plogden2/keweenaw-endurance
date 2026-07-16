import { describe, expect, it } from 'vitest'
import { resolveCategoryColor, DEFAULT_CATEGORY_COLORS } from './defaultLegend'

describe('defaultLegend', () => {
  it('maps known keys to brand-family colors', () => {
    expect(DEFAULT_CATEGORY_COLORS.advanced_men).toBe('#1a3f3d')
    expect(DEFAULT_CATEGORY_COLORS.advanced_women).toBe('#2f6b5a')
    expect(DEFAULT_CATEGORY_COLORS.beginner_men).toBe('#9b654e')
    expect(DEFAULT_CATEGORY_COLORS.beginner_women).toBe('#a1b383')
  })

  it('overrides API blue chrome with brand colors for known keys', () => {
    expect(resolveCategoryColor('advanced_men', '#1a5276')).toBe('#1a3f3d')
  })

  it('falls back to muted for unknown keys without blue', () => {
    const color = resolveCategoryColor('other_cat', '#1a5276')
    expect(color).not.toMatch(/#1a5276/i)
    expect(['#1a3f3d', '#2f6b5a', '#9b654e', '#a1b383', '#6b7a76']).toContain(color)
  })
})
