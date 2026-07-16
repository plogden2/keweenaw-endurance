# Default Superior Forest Theme Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the generic default frontend chrome with Superior Forest + copper tokens, Outfit typography, page wash, full hex→token sweep, legend/series retint, and certificate/PWA updates — without breaking Bluffet overrides.

**Architecture:** CSS custom properties on `:root` in `main.css` drive default chrome. Vue SFCs consume `var(--*)` instead of hardcoded blues/slates/reds. Bluffet continues to override via `#app.theme-bluffet`. Category legend/series colors move to a fixed brand-family palette.

**Tech Stack:** Vue 3, Vite, Pinia, Vitest, `@fontsource/outfit`, existing Bluffet theme package

**Spec:** `docs/superpowers/specs/2026-07-16-default-superior-forest-theme-design.md`

---

## File map

| File | Responsibility |
|------|----------------|
| `frontend/src/assets/main.css` | Token block, body wash, shared buttons/forms/cards/focus/spinner |
| `frontend/src/main.ts` | Load Outfit (keep Bluffet fonts) |
| `frontend/package.json` | Add `@fontsource/outfit` |
| `frontend/vite.config.ts` | PWA `theme_color` → `#1a3f3d` |
| `frontend/src/App.vue` | Shell/footer colors via tokens if hardcoded |
| `frontend/src/components/AppHeader.vue` | Evergreen header |
| Shared components | ScanPopup, SyncStatus, RaceCard, ManualTimingForm → tokens |
| Views | Home, EventDetails, EventsTable, Timing, LiveTiming, RaceDetails, Racers, StationConfig, PinUnlock, CsvRecovery, EventLive → tokens |
| `frontend/src/components/RaceFlowChart.vue` | Chrome + now-line `--signal`; series colors if hardcoded blue |
| `frontend/src/components/ResultCertificate.vue` | Default + export chrome → tokens |
| `frontend/src/themes/defaultLegend.ts` (create) | Shared category color map for frontend display retint |
| Tests | Update default-chrome / legend fixtures; leave Bluffet CSS assertions intact |

### Category legend / series palette (locked)

| Key / role | Color | Token family |
|------------|-------|--------------|
| `advanced_men` | `#1a3f3d` | ink |
| `advanced_women` | `#2f6b5a` | accent-link |
| `beginner_men` | `#9b654e` | copper |
| `beginner_women` | `#a1b383` | sage |
| fallback unknown | `#6b7a76` | muted |

If live API returns other keys, map unknown keys through a stable hash into `{ink, accent-link, copper, sage, muted}` — prefer exact key map first.

---

### Task 1: Tokens, Outfit, main.css shell, PWA

**Files:**
- Modify: `frontend/package.json`
- Modify: `frontend/src/main.ts`
- Modify: `frontend/src/assets/main.css`
- Modify: `frontend/vite.config.ts`
- Create: `frontend/src/assets/main.theme.spec.ts` (or `frontend/src/assets/themeTokens.test.ts`)

- [ ] **Step 1: Write failing token/CSS contract test**

Create `frontend/src/assets/themeTokens.test.ts`:

```ts
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const mainCss = readFileSync(join(process.cwd(), 'src/assets/main.css'), 'utf8')

describe('default theme tokens', () => {
  it('defines Superior Forest token block on :root', () => {
    expect(mainCss).toMatch(/:root\s*\{[^}]*--ink:\s*#1a3f3d;/s)
    expect(mainCss).toMatch(/:root\s*\{[^}]*--accent:\s*var\(--ink\);|:root\s*\{[^}]*--accent:\s*#1a3f3d;/s)
    expect(mainCss).toMatch(/--accent-link:\s*#2f6b5a;/)
    expect(mainCss).toMatch(/--sage:\s*#a1b383;/)
    expect(mainCss).toMatch(/--copper:\s*#9b654e;/)
    expect(mainCss).toMatch(/--mist:\s*#eff1f0;/)
    expect(mainCss).toMatch(/--signal:\s*#c45c38;/)
    expect(mainCss).toMatch(/--success:\s*#27ae60;/)
    expect(mainCss).toMatch(/--muted:\s*#6b7a76;/)
    expect(mainCss).toMatch(/--border:\s*#d4dad7;/)
    expect(mainCss).toMatch(/--surface:\s*#ffffff;/)
    expect(mainCss).toMatch(/--ink-deep:\s*#203429;/)
  })

  it('uses Outfit on body and landscape wash on page background', () => {
    expect(mainCss).toMatch(/font-family:[^;]*Outfit/)
    expect(mainCss).toMatch(/linear-gradient\([^)]*#e8efe6[^)]*#eff1f0[^)]*#e8d5c8/)
  })

  it('wires shared chrome to tokens', () => {
    expect(mainCss).toMatch(/\.btn-primary\s*\{[^}]*background[^;]*var\(--accent\)/s)
    expect(mainCss).toMatch(/:focus-visible\s*\{[^}]*outline:[^;]*var\(--sage\)/s)
    expect(mainCss).toMatch(/\.status-active\s*\{[^}]*color:\s*var\(--success\)/s)
    expect(mainCss).toMatch(/\.status-cancelled\s*\{[^}]*color:\s*var\(--signal\)/s)
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run src/assets/themeTokens.test.ts`
Expected: FAIL (tokens / Outfit / wash missing)

- [ ] **Step 3: Install Outfit and implement main.css + main.ts + PWA**

```bash
cd frontend && npm install @fontsource/outfit@5
```

In `main.ts`, add (keep existing Bluffet fonts):

```ts
import '@fontsource/outfit/400.css'
import '@fontsource/outfit/600.css'
import '@fontsource/outfit/700.css'
```

Replace the top of `main.css` with:

```css
:root {
  --ink: #1a3f3d;
  --ink-deep: #203429;
  --accent: var(--ink);
  --accent-link: #2f6b5a;
  --sage: #a1b383;
  --copper: #9b654e;
  --mist: #eff1f0;
  --signal: #c45c38;
  --success: #27ae60;
  --muted: #6b7a76;
  --border: #d4dad7;
  --surface: #ffffff;
}

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html {
  font-size: 16px;
  line-height: 1.6;
}

body {
  font-family: 'Outfit', system-ui, sans-serif;
  font-weight: 400;
  background-color: var(--mist);
  background-image: linear-gradient(165deg, #e8efe6 0%, #eff1f0 45%, #e8d5c8 100%);
  background-attachment: fixed;
  color: var(--ink);
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}
```

Update remaining shared rules in the same file:
- `:focus` → `:focus-visible { outline: 2px solid var(--sage); outline-offset: 2px; }`
- `.btn` use `font-weight: 600`
- `.btn-primary` → `background-color: var(--accent); color: var(--surface);` hover `var(--ink-deep)`
- `.btn-secondary` → `background-color: var(--muted); color: var(--surface);`
- `.form-input` borders → `var(--border)`; focus border `var(--sage)`
- `.card` → `background: var(--surface);` keep radius/shadow reasonable
- `.loading` spinner top border → `var(--accent)`
- `.status-active` → `var(--success)`; `.status-cancelled` → `var(--signal)`; `.status-completed` → `var(--muted)`

In `vite.config.ts` set `theme_color: '#1a3f3d'`.

- [ ] **Step 4: Run token test to verify it passes**

Run: `cd frontend && npx vitest run src/assets/themeTokens.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/package.json frontend/package-lock.json frontend/src/main.ts frontend/src/assets/main.css frontend/src/assets/themeTokens.test.ts frontend/vite.config.ts
git commit -m "feat(theme): add Superior Forest tokens, Outfit, and PWA color"
```

---

### Task 2: App shell + AppHeader

**Files:**
- Modify: `frontend/src/components/AppHeader.vue`
- Modify: `frontend/src/App.vue` (footer/shell if hardcoded)
- Modify: `frontend/src/components/AppHeader.test.ts` if color assertions exist

- [ ] **Step 1: Write/update failing header expectation**

If `AppHeader.test.ts` does not assert colors, add a lightweight CSS-contract style check or assert computed class styles use CSS variables by reading the SFC style block in the test file (same pattern as Home.test reading bluffet.css):

```ts
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

const headerVue = readFileSync(join(process.cwd(), 'src/components/AppHeader.vue'), 'utf8')

it('uses evergreen ink token for header chrome', () => {
  expect(headerVue).toMatch(/background(-color)?:\s*var\(--ink\)/)
})
```

- [ ] **Step 2: Run test — expect FAIL**

Run: `cd frontend && npx vitest run src/components/AppHeader.test.ts`

- [ ] **Step 3: Implement**

In `AppHeader.vue` scoped CSS, replace `#2c3e50` with `var(--ink)` (and text/link colors to `var(--surface)` / `var(--sage)` as appropriate). Map any `font-weight: 500` → `600`.

In `App.vue`, replace any hardcoded footer/shell hex with tokens (`var(--ink)`, `var(--muted)`, `var(--border)`).

- [ ] **Step 4: Run tests — expect PASS**

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/AppHeader.vue frontend/src/components/AppHeader.test.ts frontend/src/App.vue
git commit -m "feat(theme): restyle app header and shell with Forest tokens"
```

---

### Task 3: Shared components color sweep

**Files:**
- Modify: `frontend/src/components/ScanPopup.vue`
- Modify: `frontend/src/components/SyncStatus.vue`
- Modify: `frontend/src/components/RaceCard.vue`
- Modify: `frontend/src/components/ManualTimingForm.vue`
- Modify related `*.test.ts` only if they assert old hex

- [ ] **Step 1: Write a sweep contract test**

Create `frontend/src/components/sharedChrome.theme.spec.ts`:

```ts
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

const files = [
  'src/components/ScanPopup.vue',
  'src/components/SyncStatus.vue',
  'src/components/RaceCard.vue',
  'src/components/ManualTimingForm.vue',
]

describe('shared component chrome tokens', () => {
  for (const file of files) {
    it(`${file} does not hardcode legacy blue chrome`, () => {
      const src = readFileSync(join(process.cwd(), file), 'utf8')
      // Allow strings in comments/tests only — style blocks must not use these
      const style = src.split('<style')[1] ?? ''
      expect(style).not.toMatch(/#3498db|#2980b9|#1a5276|#2c3e50/i)
    })
  }
})
```

- [ ] **Step 2: Run — expect FAIL**

- [ ] **Step 3: Map hex → tokens per spec**

| Old | New |
|-----|-----|
| `#3498db` / `#2980b9` | `var(--accent)` or `var(--accent-link)` by role |
| `#1a5276` / `var(--accent, #1a5276)` | `var(--accent-link)` or `var(--accent)` for primary buttons |
| `#2c3e50` | `var(--ink)` |
| `#27ae60` | `var(--success)` |
| `#e74c3c` / danger reds | `var(--signal)` |
| `#95a5a6` / `#7f8c8d` | `var(--muted)` |
| `#ddd` | `var(--border)` |
| `font-weight: 500` | `600` |

Do not change Bluffet-only rules. Keep behavior identical.

- [ ] **Step 4: Run shared + related component tests — expect PASS**

Run: `cd frontend && npx vitest run src/components/sharedChrome.theme.spec.ts src/components/ScanPopup.test.ts src/components/ManualTimingForm.test.ts`

- [ ] **Step 5: Commit**

```bash
git commit -m "feat(theme): token-sweep shared timing chrome components"
```

---

### Task 4: Views color sweep (non-live)

**Files:**
- Modify: `Home.vue`, `EventDetails.vue`, `EventsTable.vue`, `Timing.vue`, `LiveTiming.vue`, `RaceDetails.vue`, `Racers.vue`, `StationConfig.vue`, `PinUnlock.vue`, `CsvRecovery.vue`
- Update tests only where they assert old default hex

- [ ] **Step 1: Failing sweep contract**

Create `frontend/src/views/defaultChrome.theme.spec.ts` listing those view files; assert their `<style>` blocks do not contain `#3498db|#2980b9|#1a5276|#2c3e50|#e74c3c|#c0392b` (signal must be `var(--signal)`; Home featured CTAs that were `#e74c3c` become `var(--signal)`).

- [ ] **Step 2: Run — FAIL**

- [ ] **Step 3: Replace hex with tokens** using the same map as Task 3. Local `--accent: #1a5276` → `--accent: var(--accent-link)` or remove local override and use root. Featured/copper highlights may use `var(--copper)` where a warm non-error accent is intended (e.g. secondary featured chip); unified former reds use `--signal`.

- [ ] **Step 4: Run view unit tests that still apply**

Run: `cd frontend && npx vitest run src/views/Home.test.ts src/views/defaultChrome.theme.spec.ts`

- [ ] **Step 5: Commit**

```bash
git commit -m "feat(theme): token-sweep default event and timing views"
```

---

### Task 5: EventLive legend retint + live chrome

**Files:**
- Create: `frontend/src/themes/defaultLegend.ts`
- Create: `frontend/src/themes/defaultLegend.test.ts`
- Modify: `frontend/src/views/EventLive.vue`
- Modify: `frontend/src/views/EventLive.test.ts`

- [ ] **Step 1: Failing legend helper tests**

```ts
// frontend/src/themes/defaultLegend.test.ts
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
```

- [ ] **Step 2: Run — FAIL**

- [ ] **Step 3: Implement helper + wire EventLive**

```ts
// frontend/src/themes/defaultLegend.ts
export const DEFAULT_CATEGORY_COLORS: Record<string, string> = {
  advanced_men: '#1a3f3d',
  advanced_women: '#2f6b5a',
  beginner_men: '#9b654e',
  beginner_women: '#a1b383',
}

const FALLBACK_PALETTE = ['#1a3f3d', '#2f6b5a', '#9b654e', '#a1b383', '#6b7a76']

export function resolveCategoryColor(key: string, _apiColor?: string): string {
  if (DEFAULT_CATEGORY_COLORS[key]) return DEFAULT_CATEGORY_COLORS[key]
  let hash = 0
  for (let i = 0; i < key.length; i++) hash = (hash + key.charCodeAt(i) * (i + 1)) % 2147483647
  return FALLBACK_PALETTE[Math.abs(hash) % FALLBACK_PALETTE.length]
}
```

In `EventLive.vue`, use `resolveCategoryColor(key, apiColor)` wherever legend color is applied; replace local `--accent: #1a5276` with tokens; unify reds to `var(--signal)`.

Update `EventLive.test.ts` fixture `category_legend` colors to the new brand values (or assert rendered style uses resolved colors).

- [ ] **Step 4: Run EventLive + legend tests — PASS**

- [ ] **Step 5: Commit**

```bash
git commit -m "feat(theme): retint live category legend into Forest family"
```

---

### Task 6: RaceFlowChart signal + ResultCertificate

**Files:**
- Modify: `frontend/src/components/RaceFlowChart.vue`
- Modify: `frontend/src/components/ResultCertificate.vue`
- Modify tests if they assert `#e74c3c` / `#2c3e50` / `#152536`

- [ ] **Step 1: Failing style contracts**

Assert RaceFlowChart style/script no longer uses `#e74c3c` for now-line (use `var(--signal)` or `#c45c38` constant from token). Assert ResultCertificate does not use `#2c3e50` / `#152536` in on-screen/export chrome (map to ink / ink-deep). Keep `@media print` high-contrast neutrals if present.

- [ ] **Step 2: Run — FAIL**

- [ ] **Step 3: Implement**

- Chart current-time / danger strokes → `--signal` / `#c45c38`
- Participant series colors: if derived from blues, route through `resolveCategoryColor` or brand palette — do not leave `#1a5276`
- Certificate toolbar/buttons/export background → `var(--ink)` / `var(--ink-deep)` / computed hex `#1a3f3d` / `#203429` where canvas export needs concrete colors
- Do not alter Bluffet certificate overrides in `bluffet.css`

- [ ] **Step 4: Run chart + certificate tests — PASS**

Run: `cd frontend && npx vitest run src/components/RaceFlowChart.test.ts src/components/ResultCertificate.test.ts`

- [ ] **Step 5: Commit**

```bash
git commit -m "feat(theme): align charts and certificates with Forest tokens"
```

---

### Task 7: Bluffet guardrails + verification

**Files:** none new required; run existing suites

- [ ] **Step 1: Run Bluffet theme unit/e2e-guard unit tests**

Run:

```bash
cd frontend && npx vitest run src/views/Home.test.ts src/composables/useBluffetTheme.spec.ts src/App.theme.spec.ts src/assets/themeTokens.test.ts src/themes/defaultLegend.test.ts
```

Expected: PASS. Home.test Bluffet CSS expectations unchanged.

- [ ] **Step 2: Broader frontend unit suite**

Run: `cd frontend && npx vitest run`
Expected: PASS (fix any default-hex assertion failures by updating to tokens/brand colors — do not weaken Bluffet assertions)

- [ ] **Step 3: Optional smoke note in commit message if no browser e2e in this task**

If `e2e/bluffet-theme.spec.ts` can run quickly in this environment, run it; otherwise document that Bluffet e2e remains CI guardrail.

- [ ] **Step 4: Commit any test fixes**

```bash
git commit -m "test(theme): align default assertions; keep Bluffet guards green"
```

(or skip if clean)

---

## Spec coverage checklist

| Spec requirement | Task |
|------------------|------|
| Token block + Outfit + wash | 1 |
| Shared chrome buttons/focus/cards | 1 |
| PWA theme_color | 1 |
| AppHeader / shell | 2 |
| Component hex sweep | 3 |
| Views hex sweep | 4 |
| Legend/series retint | 5–6 |
| Certificates/export | 6 |
| Bluffet isolation + tests | 7 |
| `--success` distinct / `--signal` unify | 1, 3–6 |
| No VK assets | all (do not add) |
