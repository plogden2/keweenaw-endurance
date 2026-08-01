# Leaderboard Class / Gender Filters Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add two-axis Class (Expert/Intermediate) and Gender (Men/Women) chip filters on Event Live individuals and Race Details leaderboards, with category place renumbering; leave fullscreen rotator unfiltered.

**Architecture:** Client-side filter util parses `category_key` into skill/gender facets, matches AND filters, and renumbers places. A presentational chip component drives selection. EventLive and RaceDetails apply the util to already-loaded boards.

**Tech Stack:** Vue 3, TypeScript, Vitest, Vue Test Utils

**Spec:** `docs/superpowers/specs/2026-08-01-leaderboard-category-filters-design.md`

---

## File structure

| File | Responsibility |
|------|----------------|
| `frontend/src/utils/leaderboardCategoryFilter.ts` | Parse keys, match filter, filter + renumber |
| `frontend/src/utils/leaderboardCategoryFilter.test.ts` | Unit tests |
| `frontend/src/components/LeaderboardCategoryFilters.vue` | Two chip rows UI |
| `frontend/src/components/LeaderboardCategoryFilters.test.ts` | Component tests |
| `frontend/src/types/models.ts` | Add `category_key?: string` on `LeaderboardEntry` |
| `frontend/src/views/EventLive.vue` | Filter state + apply to individuals tables (not rotator) |
| `frontend/src/views/EventLive.test.ts` | Integration tests |
| `frontend/src/views/RaceDetails.vue` | Same filters on race leaderboard |
| `frontend/src/views/RaceDetails.test.ts` | Integration tests |

---

### Task 1: Filter util + LeaderboardEntry type

**Files:**
- Create: `frontend/src/utils/leaderboardCategoryFilter.ts`
- Create: `frontend/src/utils/leaderboardCategoryFilter.test.ts`
- Modify: `frontend/src/types/models.ts` (add `category_key?: string` to `LeaderboardEntry`)

- [ ] **Step 1: Write the failing tests**

Create `frontend/src/utils/leaderboardCategoryFilter.test.ts`:

```ts
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `npx vitest run src/utils/leaderboardCategoryFilter.test.ts`  
Working directory: `frontend`  
Expected: FAIL (module not found)

- [ ] **Step 3: Implement util + type**

`frontend/src/utils/leaderboardCategoryFilter.ts`:

```ts
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
```

In `frontend/src/types/models.ts`, add to `LeaderboardEntry`:

```ts
  category_key?: string
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `npx vitest run src/utils/leaderboardCategoryFilter.test.ts`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/utils/leaderboardCategoryFilter.ts frontend/src/utils/leaderboardCategoryFilter.test.ts frontend/src/types/models.ts
git commit -m "feat(live): add leaderboard class/gender filter util"
```

---

### Task 2: LeaderboardCategoryFilters component

**Files:**
- Create: `frontend/src/components/LeaderboardCategoryFilters.vue`
- Create: `frontend/src/components/LeaderboardCategoryFilters.test.ts`

- [ ] **Step 1: Write failing component tests**

```ts
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
```

- [ ] **Step 2: Run tests — expect FAIL**

Run: `npx vitest run src/components/LeaderboardCategoryFilters.test.ts`  
Working directory: `frontend`

- [ ] **Step 3: Implement component**

Create `LeaderboardCategoryFilters.vue`:

- Props: `modelValue: LeaderboardCategoryFilter`, `showSkill: boolean`, `showGender: boolean`
- Emit: `update:modelValue`
- Root: `data-testid="leaderboard-category-filters"`
- Skill row `data-testid="lb-filter-skill"` with buttons: All / Expert / Intermediate  
  testids: `lb-filter-skill-all`, `lb-filter-skill-expert`, `lb-filter-skill-intermediate`
- Gender row `data-testid="lb-filter-gender"` with buttons: All / Men / Women  
  testids: `lb-filter-gender-all`, `lb-filter-gender-men`, `lb-filter-gender-women`
- Use `aria-pressed` like EventLive `.mode-toggle`
- Style chips similarly (compact row, pressed = filled). Scoped CSS OK; can mirror mode-toggle look with local classes `.lb-filter-row` / `.lb-filter-row button`.
- Labels: “Class” and “Gender” (visually small)

- [ ] **Step 4: Run tests — expect PASS**

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/LeaderboardCategoryFilters.vue frontend/src/components/LeaderboardCategoryFilters.test.ts
git commit -m "feat(live): add LeaderboardCategoryFilters chip component"
```

---

### Task 3: Wire EventLive (individuals only, not rotator)

**Files:**
- Modify: `frontend/src/views/EventLive.vue`
- Modify: `frontend/src/views/EventLive.test.ts`

- [ ] **Step 1: Write failing integration tests** (add to EventLive.test.ts)

Cover:

1. When individuals mode and live payload has mixed categories, filters render (`leaderboard-category-filters`).
2. Selecting Expert + Women shows only matching rows with renumbered places (use existing live mock / extend fixture with multiple `category_key`s on 12h board).
3. Filters are **not** present inside `[data-testid="fullscreen-rotator"]` when rotator is open (open via existing toggle test pattern).
4. Teams mode: filters hidden (or not affecting teams table — prefer `v-if` hide).

Use whatever mock shape EventLive.test.ts already uses for `eventsLiveApi.getLive`; extend `leaderboard_overall` with at least 3 category keys.

- [ ] **Step 2: Run focused EventLive tests — expect FAIL on new cases**

Run: `npx vitest run src/views/EventLive.test.ts`  
Working directory: `frontend`

- [ ] **Step 3: Wire EventLive.vue**

1. Import `LeaderboardCategoryFilters`, filter util helpers, types.
2. `const categoryFilter = ref({ ...DEFAULT_CATEGORY_FILTER })` — shared across race tabs.
3. Computed `activeRaceKeys` from the **active tab** race’s `leaderboard_overall` (+ optionally `category_legend` keys) → `availableFacets`.
4. Computed filtered boards:
   - `filtered12`, `filtered6`, `filtered90` via `filterLeaderboard(..., categoryFilter, 'place')`
5. In template, above each individuals table (or once above the active panel — prefer **one** filter control near the mode toggle, above race panels, so it applies to whichever tab is active):
   - Show when `leaderboardMode === 'individuals' && !rotatorOpen`
   - `:show-skill="facets.hasSkill"` `:show-gender="facets.hasGender"`
   - `v-model="categoryFilter"`
6. Change individuals `v-for` sources from `race12?.leaderboard_overall` to `filtered12` (etc.).
7. Empty filtered: if source had rows but filtered empty, show row “No racers match”; if source empty keep “No results yet”.
8. **Do not** change rotator table — keep `rotatorRace?.leaderboard_overall`.
9. Optional: update leaderboard H2 subtitle when filter active (e.g. keep “Combined overall” when All/All; otherwise “Filtered” is fine — or leave title unchanged).

- [ ] **Step 4: Run EventLive tests — PASS**

- [ ] **Step 5: Commit**

```bash
git add frontend/src/views/EventLive.vue frontend/src/views/EventLive.test.ts
git commit -m "feat(live): wire class/gender filters on Event Live individuals"
```

---

### Task 4: Wire RaceDetails

**Files:**
- Modify: `frontend/src/views/RaceDetails.vue`
- Modify: `frontend/src/views/RaceDetails.test.ts`

- [ ] **Step 1: Write failing tests**

Extend RaceDetails.test.ts:

1. Mock `getLeaderboard` returning entries with `category_key` + `position`.
2. Assert filters render on leaderboard tab.
3. Click Expert Women → only matching rows; positions 1…n.
4. Empty match shows “No racers match”.

- [ ] **Step 2: Run RaceDetails tests — expect FAIL**

Run: `npx vitest run src/views/RaceDetails.test.ts`  
Working directory: `frontend`

- [ ] **Step 3: Wire RaceDetails.vue**

1. Import filters + util.
2. `categoryFilter` ref + computed `filteredLeaderboard = filterLeaderboard(leaderboard.value, categoryFilter.value, 'position')`.
3. Facets from `leaderboard` keys.
4. Mount `LeaderboardCategoryFilters` above the table (when not showing participant detail).
5. `v-for="entry in filteredLeaderboard"`.
6. Empty states as in EventLive.

- [ ] **Step 4: Run RaceDetails tests — PASS**

Also run:  
`npx vitest run src/utils/leaderboardCategoryFilter.test.ts src/components/LeaderboardCategoryFilters.test.ts src/views/EventLive.test.ts src/views/RaceDetails.test.ts`

- [ ] **Step 5: Commit**

```bash
git add frontend/src/views/RaceDetails.vue frontend/src/views/RaceDetails.test.ts
git commit -m "feat(races): wire class/gender filters on Race Details leaderboard"
```

---

## Spec coverage checklist

| Spec requirement | Task |
|------------------|------|
| Two-axis chips Class + Gender | 2, 3, 4 |
| Single-select per axis | 2 |
| Renumber 1…n | 1, 3, 4 |
| Event Live individuals | 3 |
| Race Details | 4 |
| Exclude fullscreen rotator | 3 |
| Exclude teams | 3 |
| Kids gender-only (hide Class) | 1 `availableFacets` + 2/3 `showSkill` |
| Client-side only | all |
| Persist filter across Event Live tabs | 3 (shared ref) |
| `category_key` on LeaderboardEntry | 1 |
| Empty “No racers match” | 3, 4 |

## Self-review notes

- No TBD placeholders.
- `place` vs `position` handled via `placeField` argument.
- Rotator explicitly untouched in Task 3.
