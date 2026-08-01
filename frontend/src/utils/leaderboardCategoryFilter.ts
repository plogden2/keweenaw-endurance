export type SkillFilter = 'all' | 'expert' | 'intermediate'
export type GenderFilter = 'all' | 'men' | 'women'

export interface LeaderboardCategoryFilter {
  skill: SkillFilter
  gender: GenderFilter
}

export interface CategoryFacets {
  skill?: 'expert' | 'intermediate'
  gender?: 'men' | 'women'
}

export const DEFAULT_CATEGORY_FILTER: LeaderboardCategoryFilter = {
  skill: 'all',
  gender: 'all',
}

export function parseCategoryKey(key: string | undefined | null): CategoryFacets {
  const k = String(key || '')
    .trim()
    .toLowerCase()
  if (!k) return {}

  let skill: CategoryFacets['skill']
  let rest = k
  if (k.startsWith('expert_')) {
    skill = 'expert'
    rest = k.slice('expert_'.length)
  } else if (k.startsWith('intermediate_')) {
    skill = 'intermediate'
    rest = k.slice('intermediate_'.length)
  } else if (k === 'expert') {
    return { skill: 'expert' }
  } else if (k === 'intermediate') {
    return { skill: 'intermediate' }
  }

  let gender: CategoryFacets['gender']
  if (rest === 'men' || rest === 'male') gender = 'men'
  else if (rest === 'women' || rest === 'female') gender = 'women'
  else if (k.endsWith('_men') || k.endsWith('_male')) gender = 'men'
  else if (k.endsWith('_women') || k.endsWith('_female')) gender = 'women'

  return { skill, gender }
}

export function matchesFilter(
  categoryKey: string | undefined | null,
  filter: LeaderboardCategoryFilter,
): boolean {
  if (filter.skill === 'all' && filter.gender === 'all') return true
  const facets = parseCategoryKey(categoryKey)
  if (filter.skill !== 'all' && facets.skill !== filter.skill) return false
  if (filter.gender !== 'all' && facets.gender !== filter.gender) return false
  return true
}

export function filterLeaderboard<T extends Record<string, unknown>>(
  entries: T[],
  filter: LeaderboardCategoryFilter,
  placeField: 'place' | 'position',
): T[] {
  const matched = entries.filter((e) =>
    matchesFilter(e.category_key as string | undefined, filter),
  )
  if (filter.skill === 'all' && filter.gender === 'all') {
    return matched.map((e) => ({ ...e }))
  }
  return matched.map((e, i) => ({ ...e, [placeField]: i + 1 }))
}

export function availableFacets(keys: Array<string | undefined | null>): {
  hasSkill: boolean
  hasGender: boolean
} {
  let hasSkill = false
  let hasGender = false
  for (const key of keys) {
    const f = parseCategoryKey(key)
    if (f.skill) hasSkill = true
    if (f.gender) hasGender = true
  }
  return { hasSkill, hasGender }
}

export function isFilterActive(filter: LeaderboardCategoryFilter): boolean {
  return filter.skill !== 'all' || filter.gender !== 'all'
}
