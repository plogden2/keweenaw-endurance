# Bluffet 2026 Kawaii Theme Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the new 2026 Bluffet poster/logo and apply scoped kawaii “themed chrome” whenever the user is in Bluffet event context.

**Architecture:** Static poster + replaced logo assets; `bluffet.css` design tokens; `useBluffetTheme()` drives a `#app.theme-bluffet` class from event UUID/name or station `eventId`; Home uses `<picture>` for the poster hero. No API/schema changes.

**Tech Stack:** Vue 3, Pinia, Vue Router, Vitest, Playwright, CSS custom properties, `@fontsource/yuji-mai` + `@fontsource/ibm-plex-sans`

**Spec:** `docs/superpowers/specs/2026-07-12-bluffet-2026-kawaii-theme-design.md`

---

## File map

| File | Responsibility |
|---|---|
| `frontend/public/images/bluffet-2026-poster.avif` | Full poster (AVIF) |
| `frontend/public/images/bluffet-2026-poster.png` | Full poster PNG fallback |
| `frontend/public/images/bluffet-2026-logo.png` | Replace in place — 512×512 whole-cat crop |
| `frontend/src/themes/bluffetConstants.ts` | Seed UUID, event name, asset paths |
| `frontend/src/themes/bluffet.css` | Tokens, typography hooks, shared chrome classes, motion |
| `frontend/src/composables/useBluffetTheme.ts` | Activation logic |
| `frontend/src/composables/useBluffetTheme.spec.ts` | Unit tests for activation |
| `frontend/src/main.ts` | Import theme CSS + font packages |
| `frontend/src/App.vue` | Bind `theme-bluffet` on `#app` |
| `frontend/src/components/AppHeader.vue` | Show ~28px logo mark when themed |
| `frontend/src/views/Home.vue` | Poster `<picture>`, featured themed chrome |
| `frontend/src/views/EventDetails.vue` | Brush title + outlined panels under theme |
| `frontend/src/views/EventLive.vue` | Tabs/countdown/chips themed accents |
| `frontend/src/views/RaceDetails.vue` | Brush title + certificate frame accents |
| `frontend/src/views/Racers.vue` | Token accents when themed |
| `frontend/src/views/PinUnlock.vue` | Token accents when themed |
| `frontend/src/views/StationConfig.vue` | Token accents when themed |
| `frontend/src/components/ScanPopup.vue` | Stamp-in modal when themed |
| `frontend/src/components/ResultCertificate.vue` | Light paper/red frame when themed |
| `frontend/e2e/bluffet-theme.spec.ts` | Theme class + asset URL smoke |

Source art: `assets/allYouCanEastBluff_26_052126.avif`  
Bluffet seed id: `1441674d-a011-471a-a601-722b88b117f5` (same as `frontend/e2e/fixtures/rfid.ts`)

---

### Task 1: Assets — poster + logo crop

**Files:**
- Create: `frontend/public/images/bluffet-2026-poster.avif`
- Create: `frontend/public/images/bluffet-2026-poster.png`
- Modify: `frontend/public/images/bluffet-2026-logo.png` (replace)

- [ ] **Step 1: Generate poster PNG + scaled AVIF and logo crop**

From repo root (Pillow already available in prior sessions; install if needed: `pip install pillow pillow-avif-plugin` or use `pillow` with avif support):

```powershell
python -c @"
from PIL import Image
src = Image.open(r'assets/allYouCanEastBluff_26_052126.avif').convert('RGBA')
# Poster PNG ~1600 long edge
w, h = src.size
scale = 1600 / max(w, h)
poster = src.resize((int(w*scale), int(h*scale)), Image.Resampling.LANCZOS)
poster.convert('RGB').save(r'frontend/public/images/bluffet-2026-poster.png', quality=92)
poster.save(r'frontend/public/images/bluffet-2026-poster.avif', quality=60)
# Whole-cat crop from original proportions (validated in brainstorm)
ow, oh = src.size
left, top, right, bottom = int(ow*0.12), int(oh*0.18), int(ow*0.88), int(oh*0.72)
crop = src.crop((left, top, right, bottom))
cw, ch = crop.size
side = max(cw, ch)
sq = Image.new('RGBA', (side, side), (196, 165, 116, 255))
sq.paste(crop, ((side-cw)//2, (side-ch)//2), crop)
sq.resize((512, 512), Image.Resampling.LANCZOS).save(r'frontend/public/images/bluffet-2026-logo.png')
print('assets ok', poster.size)
"@
```

- [ ] **Step 2: Verify files exist and are non-trivial**

```powershell
Get-ChildItem frontend/public/images/bluffet-2026-* | Select-Object Name, Length
```

Expected: three files; logo ≈ tens of KB+; poster PNG larger than logo; AVIF present.

- [ ] **Step 3: Commit**

```bash
git add frontend/public/images/bluffet-2026-poster.avif frontend/public/images/bluffet-2026-poster.png frontend/public/images/bluffet-2026-logo.png
git commit -m "Add Bluffet 2026 poster assets and cat logo crop"
```

---

### Task 2: Constants + theme activation composable (TDD)

**Files:**
- Create: `frontend/src/themes/bluffetConstants.ts`
- Create: `frontend/src/composables/useBluffetTheme.ts`
- Create: `frontend/src/composables/useBluffetTheme.spec.ts`

- [ ] **Step 1: Write failing unit tests**

Create `frontend/src/composables/useBluffetTheme.spec.ts`:

```typescript
import { describe, it, expect, beforeEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { defineComponent, h, nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import { useEventsStore } from '@/stores/events'
import { useStationStore } from '@/stores/station'
import {
  BLUFFET_EVENT_ID,
  BLUFFET_EVENT_NAME,
  BLUFFET_LOGO_PATH,
  BLUFFET_POSTER_AVIF,
  BLUFFET_POSTER_PNG,
  BLUFFET_THEME_CLASS,
} from '@/themes/bluffetConstants'
import { useBluffetTheme } from './useBluffetTheme'

async function mountWithRoute(path: string) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: { template: '<div />' } },
      { path: '/timing/:eventId', component: { template: '<div />' } },
      { path: '/timing/:eventId/live', component: { template: '<div />' } },
    ],
  })
  await router.push(path)
  await router.isReady()

  let api!: ReturnType<typeof useBluffetTheme>
  const Comp = defineComponent({
    setup() {
      api = useBluffetTheme()
      return () => h('div')
    },
  })
  mount(Comp, { global: { plugins: [router] } })
  await nextTick()
  return api
}

describe('useBluffetTheme', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('exposes asset path constants via return value', async () => {
    const theme = await mountWithRoute('/')
    expect(theme.posterAvif.value).toBe(BLUFFET_POSTER_AVIF)
    expect(theme.posterPng.value).toBe(BLUFFET_POSTER_PNG)
    expect(theme.logoPath.value).toBe(BLUFFET_LOGO_PATH)
    expect(theme.themeClass.value).toBe(BLUFFET_THEME_CLASS)
  })

  it('activates when route eventId matches Bluffet UUID', async () => {
    const theme = await mountWithRoute(`/timing/${BLUFFET_EVENT_ID}`)
    expect(theme.active.value).toBe(true)
  })

  it('activates when currentEvent name matches', async () => {
    const events = useEventsStore()
    events.currentEvent = {
      id: 'other-id',
      name: BLUFFET_EVENT_NAME,
      event_date: '2026-08-01',
      status: 'upcoming',
    }
    const theme = await mountWithRoute('/')
    expect(theme.active.value).toBe(true)
  })

  it('activates when station eventId matches Bluffet UUID', async () => {
    const station = useStationStore()
    station.eventId = BLUFFET_EVENT_ID
    const theme = await mountWithRoute('/')
    expect(theme.active.value).toBe(true)
  })

  it('is inactive for unrelated event', async () => {
    const events = useEventsStore()
    events.currentEvent = {
      id: 'chtf',
      name: 'CHTF',
      event_date: '2025-01-01',
      status: 'upcoming',
    }
    const theme = await mountWithRoute('/timing/chtf')
    expect(theme.active.value).toBe(false)
  })
})
```

- [ ] **Step 2: Run tests — expect FAIL (module missing)**

```bash
cd frontend && npx vitest run src/composables/useBluffetTheme.spec.ts
```

Expected: FAIL resolving `@/themes/bluffetConstants` or `./useBluffetTheme`.

- [ ] **Step 3: Implement constants**

Create `frontend/src/themes/bluffetConstants.ts`:

```typescript
/** Seeded All You Can East Bluffet 2026 — keep in sync with e2e/fixtures/rfid.ts */
export const BLUFFET_EVENT_ID = '1441674d-a011-471a-a601-722b88b117f5'
export const BLUFFET_EVENT_NAME = 'All You Can East Bluffet'
export const BLUFFET_THEME_CLASS = 'theme-bluffet'
export const BLUFFET_LOGO_PATH = '/images/bluffet-2026-logo.png'
export const BLUFFET_POSTER_AVIF = '/images/bluffet-2026-poster.avif'
export const BLUFFET_POSTER_PNG = '/images/bluffet-2026-poster.png'
```

- [ ] **Step 4: Implement composable**

Create `frontend/src/composables/useBluffetTheme.ts`:

```typescript
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useEventsStore } from '@/stores/events'
import { useStationStore } from '@/stores/station'
import {
  BLUFFET_EVENT_ID,
  BLUFFET_EVENT_NAME,
  BLUFFET_LOGO_PATH,
  BLUFFET_POSTER_AVIF,
  BLUFFET_POSTER_PNG,
  BLUFFET_THEME_CLASS,
} from '@/themes/bluffetConstants'

function isBluffetId(id: string | null | undefined): boolean {
  return Boolean(id && id === BLUFFET_EVENT_ID)
}

function isBluffetName(name: string | null | undefined): boolean {
  return Boolean(name && name === BLUFFET_EVENT_NAME)
}

export function useBluffetTheme() {
  const route = useRoute()
  const events = useEventsStore()
  const station = useStationStore()

  const active = computed(() => {
    const routeEventId = typeof route.params.eventId === 'string' ? route.params.eventId : null
    if (isBluffetId(routeEventId)) return true
    if (isBluffetId(events.currentEvent?.id) || isBluffetName(events.currentEvent?.name)) return true
    if (isBluffetId(station.eventId)) return true
    return false
  })

  return {
    active,
    themeClass: computed(() => BLUFFET_THEME_CLASS),
    posterAvif: computed(() => BLUFFET_POSTER_AVIF),
    posterPng: computed(() => BLUFFET_POSTER_PNG),
    logoPath: computed(() => BLUFFET_LOGO_PATH),
  }
}
```

- [ ] **Step 5: Run tests — expect PASS**

```bash
cd frontend && npx vitest run src/composables/useBluffetTheme.spec.ts
```

Expected: all tests PASS.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/themes/bluffetConstants.ts frontend/src/composables/useBluffetTheme.ts frontend/src/composables/useBluffetTheme.spec.ts
git commit -m "Add Bluffet theme activation composable"
```

---

### Task 3: Theme CSS + fonts + App root class

**Files:**
- Create: `frontend/src/themes/bluffet.css`
- Modify: `frontend/package.json` (add font packages)
- Modify: `frontend/src/main.ts`
- Modify: `frontend/src/App.vue`
- Test: extend or add `frontend/src/App.test.ts` if present; otherwise assert via a small mount in a new `frontend/src/App.theme.spec.ts`

- [ ] **Step 1: Install fonts**

```bash
cd frontend && npm install @fontsource/yuji-mai @fontsource/ibm-plex-sans
```

- [ ] **Step 2: Write failing App theme-class test**

Create `frontend/src/App.theme.spec.ts`:

```typescript
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import App from './App.vue'
import { BLUFFET_EVENT_ID, BLUFFET_THEME_CLASS } from '@/themes/bluffetConstants'
import { useStationStore } from '@/stores/station'

vi.mock('@/composables/useReaderStation', () => ({
  useReaderStation: () => ({
    lastScan: { value: null },
    clearLastScan: vi.fn(),
    start: vi.fn(),
    stop: vi.fn(),
  }),
}))

describe('App Bluffet theme class', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('adds theme-bluffet on #app when station is Bluffet', async () => {
    const station = useStationStore()
    station.eventId = BLUFFET_EVENT_ID
    vi.spyOn(station, 'fetchCurrent').mockResolvedValue(undefined as never)

    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/', component: { template: '<div />' } }],
    })
    await router.push('/')
    await router.isReady()

    const wrapper = mount(App, {
      global: { plugins: [router, createPinia()] },
      attachTo: document.body,
    })
    await flushPromises()

    expect(wrapper.find('#app').classes()).toContain(BLUFFET_THEME_CLASS)
    wrapper.unmount()
  })
})
```

Note: if Pinia double-create causes issues, reuse `setActivePinia` instance via `global: { plugins: [router] }` only after `setActivePinia(createPinia())` once. Adjust to match existing App test patterns if any appear during implementation.

- [ ] **Step 3: Run test — expect FAIL**

```bash
cd frontend && npx vitest run src/App.theme.spec.ts
```

Expected: FAIL — `#app` lacks `theme-bluffet`.

- [ ] **Step 4: Add `bluffet.css`**

Create `frontend/src/themes/bluffet.css`:

```css
:root {
  --bluffet-paper: #c4a574;
  --bluffet-panel: #f7f3ea;
  --bluffet-red: #c8102e;
  --bluffet-teal: #0d9488;
  --bluffet-ink: #1a1a1a;
  --bluffet-outline: 2px solid var(--bluffet-ink);
  --bluffet-shadow: 4px 4px 0 var(--bluffet-ink);
}

#app.theme-bluffet {
  --accent: var(--bluffet-teal);
  background: var(--bluffet-panel);
  color: var(--bluffet-ink);
  font-family: 'IBM Plex Sans', 'Segoe UI', system-ui, sans-serif;
}

#app.theme-bluffet .bluffet-display,
#app.theme-bluffet h1.page-title,
#app.theme-bluffet .featured-event h2 {
  font-family: 'Yuji Mai', 'Segoe UI', system-ui, sans-serif;
  font-weight: 400;
  letter-spacing: 0.02em;
}

#app.theme-bluffet .bluffet-panel,
#app.theme-bluffet .race-link,
#app.theme-bluffet .panel {
  background: var(--bluffet-panel);
  border: var(--bluffet-outline);
  box-shadow: var(--bluffet-shadow);
  border-radius: 6px;
}

#app.theme-bluffet .bluffet-chip,
#app.theme-bluffet .legend-item,
#app.theme-bluffet .race-tabs button[aria-selected='true'] {
  background: var(--bluffet-red);
  color: #fff;
  border: var(--bluffet-outline);
}

#app.theme-bluffet .countdown {
  color: var(--bluffet-red);
}

#app.theme-bluffet .featured-timing-link,
#app.theme-bluffet .btn:not(.secondary) {
  background: var(--bluffet-red);
  border: var(--bluffet-outline);
  box-shadow: var(--bluffet-shadow);
}

#app.theme-bluffet .featured-link {
  color: var(--bluffet-teal);
  border-color: var(--bluffet-teal);
}

#app.theme-bluffet .scan-overlay .modal {
  border: 3px solid var(--bluffet-ink);
  box-shadow: var(--bluffet-shadow);
  animation: bluffet-stamp 200ms ease-out;
}

#app.theme-bluffet .bluffet-poster {
  animation: bluffet-poster-in 400ms ease-out;
}

#app.theme-bluffet .race-tabs button,
#app.theme-bluffet .bluffet-chip {
  animation: bluffet-stamp 220ms ease-out;
}

@keyframes bluffet-stamp {
  from { transform: scale(0.92); opacity: 0; }
  to { transform: scale(1); opacity: 1; }
}

@keyframes bluffet-poster-in {
  from { transform: scale(0.98); opacity: 0; }
  to { transform: scale(1); opacity: 1; }
}

@media (prefers-reduced-motion: reduce) {
  #app.theme-bluffet .bluffet-poster,
  #app.theme-bluffet .race-tabs button,
  #app.theme-bluffet .bluffet-chip,
  #app.theme-bluffet .scan-overlay .modal {
    animation: none;
  }
}
```

- [ ] **Step 5: Wire fonts + CSS in `main.ts`**

At top of `frontend/src/main.ts` (after existing imports is fine):

```typescript
import '@fontsource/ibm-plex-sans/400.css'
import '@fontsource/ibm-plex-sans/600.css'
import '@fontsource/ibm-plex-sans/700.css'
import '@fontsource/yuji-mai/400.css'
import '@/themes/bluffet.css'
```

Keep existing `./assets/main.css` import.

- [ ] **Step 6: Bind class on `App.vue`**

Change root element and script:

```vue
<template>
  <div id="app" :class="{ [themeClass]: bluffetActive }">
    <!-- existing children unchanged -->
  </div>
</template>

<script setup lang="ts">
// ...existing imports...
import { useBluffetTheme } from '@/composables/useBluffetTheme'

const { active: bluffetActive, themeClass } = useBluffetTheme()
// ...rest unchanged...
</script>
```

- [ ] **Step 7: Run App theme test — expect PASS**

```bash
cd frontend && npx vitest run src/App.theme.spec.ts src/composables/useBluffetTheme.spec.ts
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add frontend/package.json frontend/package-lock.json frontend/src/themes/bluffet.css frontend/src/main.ts frontend/src/App.vue frontend/src/App.theme.spec.ts
git commit -m "Wire Bluffet theme CSS and App root class"
```

---

### Task 4: Home featured poster + themed chrome

**Files:**
- Modify: `frontend/src/views/Home.vue`
- Modify: `frontend/src/views/Home.test.ts`

- [ ] **Step 1: Update Home test expectations**

In `shows featured Bluffet links`, replace logo assertions with poster picture checks:

```typescript
const picture = wrapper.find('[data-testid="bluffet-poster"]')
expect(picture.exists()).toBe(true)
expect(wrapper.find('source[type="image/avif"]').attributes('srcset')).toBe(
  '/images/bluffet-2026-poster.avif',
)
expect(wrapper.find('.featured-logo').attributes('src')).toBe(
  '/images/bluffet-2026-poster.png',
)
expect(wrapper.find('.featured-event').classes()).toContain('bluffet-theme')
```

- [ ] **Step 2: Run test — expect FAIL**

```bash
cd frontend && npx vitest run src/views/Home.test.ts
```

Expected: FAIL missing `bluffet-poster` / classes.

- [ ] **Step 3: Update `Home.vue` featured markup**

Replace the featured `<img>` with:

```vue
<section class="featured-event bluffet-theme" aria-labelledby="featured-event">
  <h2 id="featured-event">Featured Event</h2>
  <div class="featured-content">
    <picture data-testid="bluffet-poster" class="bluffet-poster">
      <source type="image/avif" :srcset="posterAvif" />
      <img
        :src="posterPng"
        alt="All You Can East Bluffet"
        class="featured-logo"
      />
    </picture>
    <!-- keep date, actions, description -->
  </div>
</section>
```

In script:

```typescript
import {
  BLUFFET_EVENT_NAME,
  BLUFFET_POSTER_AVIF,
  BLUFFET_POSTER_PNG,
} from '@/themes/bluffetConstants'

const posterAvif = BLUFFET_POSTER_AVIF
const posterPng = BLUFFET_POSTER_PNG
// keep existing BLUFFET_EVENT_NAME usage; import constant instead of local string duplicate
```

Remove hardcoded `/images/bluffet-2026-logo.png` for the featured hero.

Optionally tighten featured CSS under `.featured-event.bluffet-theme` (tan panel, outlined buttons) — theme CSS already covers much of this when `#app.theme-bluffet` is on; local `.bluffet-theme` class on the section ensures styling even if global class is off on `/`.

Add to `bluffet.css`:

```css
.featured-event.bluffet-theme {
  background: var(--bluffet-paper, #c4a574);
  border: 2px solid var(--bluffet-ink, #1a1a1a);
  box-shadow: 4px 4px 0 var(--bluffet-ink, #1a1a1a);
}
```

- [ ] **Step 4: Run Home tests — expect PASS**

```bash
cd frontend && npx vitest run src/views/Home.test.ts
```

- [ ] **Step 5: Commit**

```bash
git add frontend/src/views/Home.vue frontend/src/views/Home.test.ts frontend/src/themes/bluffet.css
git commit -m "Use Bluffet 2026 poster on home featured section"
```

---

### Task 5: AppHeader mark + event/race/live chrome classes

**Files:**
- Modify: `frontend/src/components/AppHeader.vue`
- Modify: `frontend/src/views/EventDetails.vue`
- Modify: `frontend/src/views/EventLive.vue`
- Modify: `frontend/src/views/RaceDetails.vue`
- Modify tests that assert logo `src` if any (`RaceDetails.test.ts`)

- [ ] **Step 1: AppHeader — show logo when theme active**

```vue
<template>
  <header class="header">
    <nav class="nav">
      <router-link to="/" class="logo">
        <img
          v-if="bluffetActive"
          :src="logoPath"
          alt=""
          class="bluffet-nav-mark"
          width="28"
          height="28"
          data-testid="bluffet-nav-mark"
        />
        <h1>Inferior Timing</h1>
      </router-link>
      <!-- nav-links unchanged -->
    </nav>
  </header>
</template>

<script setup lang="ts">
import { useBluffetTheme } from '@/composables/useBluffetTheme'
const { active: bluffetActive, logoPath } = useBluffetTheme()
</script>
```

CSS:

```css
.logo {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
.bluffet-nav-mark {
  border-radius: 50%;
  border: 2px solid #fff;
}
```

- [ ] **Step 2: Ensure EventDetails / EventLive / RaceDetails use theme-friendly hooks**

No structural rewrite required if Task 3 CSS targets existing classes (`.page-title`, `.race-link`, `.panel`, `.countdown`, `.race-tabs`, `.legend-item`). Verify selectors match actual markup in each file; adjust `bluffet.css` selectors if class names differ (e.g. EventLive may use `.btn` / different tab markup).

For EventLive specifically, confirm countdown selector is `.countdown` and tabs use `aria-selected` — already true in current `EventLive.vue`.

- [ ] **Step 3: Run related unit tests**

```bash
cd frontend && npx vitest run src/views/EventDetails.test.ts src/views/RaceDetails.test.ts src/views/EventLive.test.ts 2>$null; npx vitest run src/views/RaceDetails.test.ts src/views/EventLive.test.ts
```

Fix any logo path assertions that expected the old orange artwork dimensions only if they break; `logo_url` path string stays `/images/bluffet-2026-logo.png`.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/AppHeader.vue frontend/src/themes/bluffet.css frontend/src/views/EventDetails.vue frontend/src/views/EventLive.vue frontend/src/views/RaceDetails.vue
git commit -m "Apply Bluffet themed chrome to header and event pages"
```

---

### Task 6: Station surfaces + scan popup + certificate

**Files:**
- Modify: `frontend/src/views/Racers.vue`
- Modify: `frontend/src/views/PinUnlock.vue`
- Modify: `frontend/src/views/StationConfig.vue`
- Modify: `frontend/src/components/ScanPopup.vue`
- Modify: `frontend/src/components/ResultCertificate.vue`

- [ ] **Step 1: Rely on `#app.theme-bluffet` + existing `--accent`**

These views already define `--accent: #1a5276`. Under `#app.theme-bluffet`, Task 3 overrides `--accent` to teal. Spot-check that buttons using `var(--accent)` pick up the theme without per-file rewrites.

If a view hardcodes `#1a5276` instead of `var(--accent)`, change those spots to `var(--accent)` (minimal diff).

- [ ] **Step 2: Certificate frame**

In `ResultCertificate.vue` scoped styles, add:

```css
:global(#app.theme-bluffet) .certificate {
  background: #f7f3ea;
  border: 3px solid #c8102e;
  box-shadow: 4px 4px 0 #1a1a1a;
}
```

(Adjust `.certificate` to the actual root class name in that component.)

- [ ] **Step 3: Scan popup stamp**

Already covered by `#app.theme-bluffet .scan-overlay .modal` in `bluffet.css`. Confirm ScanPopup root uses `.scan-overlay` and `.modal` (it does). No code change unless class names differ.

- [ ] **Step 4: Run station-related unit tests**

```bash
cd frontend && npx vitest run src/views/Racers.test.ts src/views/PinUnlock.test.ts src/components/ScanPopup.test.ts src/components/ResultCertificate.test.ts
```

Expected: PASS (or skip missing test files; run only those that exist).

- [ ] **Step 5: Commit**

```bash
git add frontend/src/views/Racers.vue frontend/src/views/PinUnlock.vue frontend/src/views/StationConfig.vue frontend/src/components/ResultCertificate.vue frontend/src/themes/bluffet.css
git commit -m "Extend Bluffet theme to station UI and certificates"
```

---

### Task 7: E2E smoke — theme class + assets

**Files:**
- Create: `frontend/e2e/bluffet-theme.spec.ts`

- [ ] **Step 1: Write e2e spec**

```typescript
import { test, expect } from '@playwright/test'
import { BLUFFET } from './fixtures/rfid'

test.describe('Bluffet theme', () => {
  test('home featured poster assets resolve and section is themed', async ({ page, request }) => {
    const avif = await request.get('/images/bluffet-2026-poster.avif')
    const png = await request.get('/images/bluffet-2026-poster.png')
    const logo = await request.get('/images/bluffet-2026-logo.png')
    expect(avif.ok()).toBeTruthy()
    expect(png.ok()).toBeTruthy()
    expect(logo.ok()).toBeTruthy()

    await page.goto('/')
    await expect(page.getByTestId('bluffet-poster')).toBeVisible()
    await expect(page.locator('.featured-event.bluffet-theme')).toBeVisible()
  })

  test('Bluffet live view gets theme-bluffet on app root', async ({ page }) => {
    await page.goto(`/events/${BLUFFET.eventId}/live`)
    await expect(page.locator('#app')).toHaveClass(/theme-bluffet/)
  })

  test('non-Bluffet timing list does not force theme from station alone', async ({ page }) => {
    await page.goto('/timing')
    // Without station armed, theme should be off
    await expect(page.locator('#app')).not.toHaveClass(/theme-bluffet/)
  })
})
```

**Important:** Live route is `/events/:eventId/live` (see `frontend/src/router/index.ts` and `frontend/e2e/multi-station.spec.ts`).

- [ ] **Step 2: Run e2e (requires stack up per project quickstart)**

```bash
cd frontend && npm run test:e2e -- e2e/bluffet-theme.spec.ts
```

Expected: PASS when frontend+API+seed are running. If env not up, note in commit message and verify in CI.

- [ ] **Step 3: Commit**

```bash
git add frontend/e2e/bluffet-theme.spec.ts
git commit -m "Add Bluffet theme e2e smoke checks"
```

---

### Task 8: Final verification

- [ ] **Step 1: Typecheck + unit suite**

```bash
cd frontend && npm run typecheck && npx vitest run
```

Expected: typecheck clean; unit tests pass.

- [ ] **Step 2: Manual spot-check checklist**

1. `/` — poster visible, featured section tan/outlined  
2. `/timing/<bluffet-id>` — theme class on, brush title, outlined races  
3. `/events/<bluffet-id>/live` — red tabs, red countdown, themed chips  
4. Arm station to Bluffet — PIN/racers pick up teal accent  
5. Non-Bluffet event — no `theme-bluffet`  
6. OS reduced-motion — no stamp/poster animations  

- [ ] **Step 3: Commit any residual selector fixes**

```bash
git add -u frontend/src
git commit -m "Polish Bluffet theme selectors after verification"
```

(Skip empty commit if nothing changed.)

---

## Self-review (plan vs spec)

| Spec requirement | Task |
|---|---|
| Poster + PNG fallback | Task 1, 4 |
| Logo crop replace in place | Task 1 |
| Theme package CSS tokens | Task 3 |
| `useBluffetTheme` UUID/name/station | Task 2 |
| `#app.theme-bluffet` | Task 3 |
| Home / event / live / station / cert / header | Tasks 4–6 |
| Yuji Mai + IBM Plex Sans | Task 3 |
| Motion + reduced-motion | Task 3 CSS |
| Fallbacks | Task 1 assets + CSS font stacks |
| Unit/component/e2e tests | Tasks 2, 3, 4, 7 |
| No API changes | (none) |

No intentional placeholders remain. Live e2e path must be confirmed against router in Task 7 Step 1.
