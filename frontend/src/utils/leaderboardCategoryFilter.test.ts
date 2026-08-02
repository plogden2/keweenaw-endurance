import { describe, it, expect } from 'vitest'
import {
  parseCategoryKey,
  matchesFilter,
  filterLeaderboard,
  availableFacets,
  DEFAULT_CATEGORY_FILTER,
  type LeaderboardCategoryFilter,
} from './leaderboardCategoryFilter'

describe('parseCategoryKey', () => {
  it('parses expert/intermediate × men/women', () => {
    expect(parseCategoryKey('expert_men')).toEqual({ skill: 'expert', gender: 'men' })
    expect(parseCategoryKey('intermediate_women')).toEqual({
      skill: 'intermediate',
      gender: 'women',
    })
  })

  it('parses kids gender-only keys', () => {
    expect(parseCategoryKey('men')).toEqual({ skill: undefined, gender: 'men' })
    expect(parseCategoryKey('women')).toEqual({ skill: undefined, gender: 'women' })
  })

  it('returns empty facets for unknown keys', () => {
    expect(parseCategoryKey('open')).toEqual({ skill: undefined, gender: undefined })
    expect(parseCategoryKey('')).toEqual({ skill: undefined, gender: undefined })
  })
})

describe('matchesFilter', () => {
  const all: LeaderboardCategoryFilter = { skill: 'all', gender: 'all' }

  it('All/All matches everything', () => {
    expect(matchesFilter('expert_men', all)).toBe(true)
    expect(matchesFilter('men', all)).toBe(true)
    expect(matchesFilter('open', all)).toBe(true)
  })

  it('ANDs skill and gender', () => {
    const f: LeaderboardCategoryFilter = { skill: 'expert', gender: 'women' }
    expect(matchesFilter('expert_women', f)).toBe(true)
    expect(matchesFilter('expert_men', f)).toBe(false)
    expect(matchesFilter('intermediate_women', f)).toBe(false)
  })

  it('gender-only filter matches any skill', () => {
    const f: LeaderboardCategoryFilter = { skill: 'all', gender: 'women' }
    expect(matchesFilter('expert_women', f)).toBe(true)
    expect(matchesFilter('intermediate_women', f)).toBe(true)
    expect(matchesFilter('women', f)).toBe(true)
    expect(matchesFilter('expert_men', f)).toBe(false)
  })

  it('skill filter excludes keys without that skill', () => {
    const f: LeaderboardCategoryFilter = { skill: 'expert', gender: 'all' }
    expect(matchesFilter('expert_men', f)).toBe(true)
    expect(matchesFilter('men', f)).toBe(false)
  })
})

describe('filterLeaderboard', () => {
  const rows = [
    { place: 1, category_key: 'expert_men', name: 'A' },
    { place: 2, category_key: 'expert_women', name: 'B' },
    { place: 3, category_key: 'intermediate_women', name: 'C' },
  ]

  it('passthrough All/All keeps places', () => {
    const out = filterLeaderboard(rows, DEFAULT_CATEGORY_FILTER, 'place')
    expect(out.map((r) => r.place)).toEqual([1, 2, 3])
  })

  it('renumbers places within filtered set', () => {
    const out = filterLeaderboard(
      rows,
      { skill: 'all', gender: 'women' },
      'place',
    )
    expect(out.map((r) => ({ name: r.name, place: r.place }))).toEqual([
      { name: 'B', place: 1 },
      { name: 'C', place: 2 },
    ])
  })

  it('supports position field name for Race Details', () => {
    const withPos = rows.map((r) => ({
      position: r.place,
      category_key: r.category_key,
      name: r.name,
    }))
    const out = filterLeaderboard(
      withPos,
      { skill: 'expert', gender: 'all' },
      'position',
    )
    expect(out.map((r) => ({ name: r.name, position: r.position }))).toEqual([
      { name: 'A', position: 1 },
      { name: 'B', position: 2 },
    ])
  })
})

describe('availableFacets', () => {
  it('detects skill and gender from keys', () => {
    const f = availableFacets(['expert_men', 'intermediate_women', 'men'])
    expect(f.hasSkill).toBe(true)
    expect(f.hasGender).toBe(true)
  })

  it('kids-only has gender but not skill', () => {
    const f = availableFacets(['men', 'women'])
    expect(f.hasSkill).toBe(false)
    expect(f.hasGender).toBe(true)
  })
})
