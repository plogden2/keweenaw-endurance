# Fullscreen Rotator Live QR Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Optionally show a QR code in the fullscreen rotator top-right that links to the live event page, default off, configurable in rotator settings, with idle-hiding controls when enabled.

**Architecture:** Extend `RotatorSettings` with `showQrCode`. Add a tiny URL helper + `LiveEventQr` presentational component using the `qrcode` package. Wire checkbox + QR mount + 3s idle control hide into `EventLive.vue`.

**Tech Stack:** Vue 3, Vitest, Vue Test Utils, `qrcode` npm package, existing sessionStorage rotator settings.

**Spec:** `docs/superpowers/specs/2026-08-01-rotator-live-qr-design.md`

---

## File structure

| File | Responsibility |
|------|----------------|
| `frontend/src/utils/liveEventUrl.ts` | Build `/events/:id/live` absolute URL |
| `frontend/src/utils/liveEventUrl.spec.ts` | Unit tests for URL helper |
| `frontend/src/components/LiveEventQr.vue` | Render QR canvas + fixed caption |
| `frontend/src/components/LiveEventQr.test.ts` | Component tests |
| `frontend/src/composables/useFullscreenRotator.ts` | `showQrCode` in settings + `setShowQrCode` |
| `frontend/src/composables/useFullscreenRotator.spec.ts` | Settings default/persist tests |
| `frontend/src/views/EventLive.vue` | Checkbox, QR mount, idle controls |
| `frontend/src/views/EventLive.test.ts` | Integration tests |
| `frontend/package.json` | Add `qrcode` (+ `@types/qrcode` if needed) |

---

### Task 1: URL helper

**Files:**
- Create: `frontend/src/utils/liveEventUrl.ts`
- Create: `frontend/src/utils/liveEventUrl.spec.ts`

- [ ] **Step 1: Write the failing test**

```ts
import { describe, it, expect } from 'vitest'
import { liveEventUrl } from './liveEventUrl'

describe('liveEventUrl', () => {
  it('builds the live event path from origin and event id', () => {
    expect(liveEventUrl('https://keweenawendurance.com', 'evt-1')).toBe(
      'https://keweenawendurance.com/events/evt-1/live',
    )
  })

  it('strips a trailing slash on origin', () => {
    expect(liveEventUrl('http://localhost:5173/', 'abc')).toBe(
      'http://localhost:5173/events/abc/live',
    )
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `npx vitest run src/utils/liveEventUrl.spec.ts`  
Working directory: `frontend`  
Expected: FAIL (module not found)

- [ ] **Step 3: Write minimal implementation**

```ts
/** Absolute URL for the public event live page. */
export function liveEventUrl(origin: string, eventId: string): string {
  const base = origin.replace(/\/+$/, '')
  const id = String(eventId || '').trim()
  return `${base}/events/${id}/live`
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `npx vitest run src/utils/liveEventUrl.spec.ts`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/utils/liveEventUrl.ts frontend/src/utils/liveEventUrl.spec.ts
git commit -m "feat(live): add liveEventUrl helper for QR links"
```

---

### Task 2: Rotator settings `showQrCode`

**Files:**
- Modify: `frontend/src/composables/useFullscreenRotator.ts`
- Modify: `frontend/src/composables/useFullscreenRotator.spec.ts`

- [ ] **Step 1: Write failing tests** (add to existing describe)

```ts
it('defaults showQrCode to false', () => {
  const settings = loadRotatorSettings()
  expect(settings.showQrCode).toBe(false)
})

it('persists showQrCode when toggled', async () => {
  const { open, setShowQrCode } = setup()
  open.value = true
  await nextTick()
  setShowQrCode(true)
  const stored = JSON.parse(sessionStorage.getItem(FS_ROTATOR_SETTINGS_KEY)!)
  expect(stored.showQrCode).toBe(true)
  expect(loadRotatorSettings().showQrCode).toBe(true)
})

it('normalizes missing showQrCode to false', () => {
  sessionStorage.setItem(
    FS_ROTATOR_SETTINGS_KEY,
    JSON.stringify({ dwellMs: 5000, pages: [] }),
  )
  expect(loadRotatorSettings().showQrCode).toBe(false)
})
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `npx vitest run src/composables/useFullscreenRotator.spec.ts -t "showQrCode"`  
Expected: FAIL (`showQrCode` / `setShowQrCode` missing)

- [ ] **Step 3: Implement**

In `RotatorSettings` add `showQrCode: boolean`.

In `cloneDefaultSettings` set `showQrCode: false`.

In `normalizeSettings` after `dwellMs`:

```ts
const showQrCode = obj.showQrCode === true
```

Include `showQrCode` in every returned settings object from `normalizeSettings`.

Add setter:

```ts
function setShowQrCode(enabled: boolean) {
  settings.value = { ...settings.value, showQrCode: Boolean(enabled) }
  saveRotatorSettings(settings.value)
}
```

Export `setShowQrCode` from the composable return object.

- [ ] **Step 4: Run tests to verify they pass**

Run: `npx vitest run src/composables/useFullscreenRotator.spec.ts`  
Expected: all PASS (including existing tests)

- [ ] **Step 5: Commit**

```bash
git add frontend/src/composables/useFullscreenRotator.ts frontend/src/composables/useFullscreenRotator.spec.ts
git commit -m "feat(live): add showQrCode to rotator settings"
```

---

### Task 3: Install `qrcode` and build `LiveEventQr`

**Files:**
- Modify: `frontend/package.json` / `package-lock.json`
- Create: `frontend/src/components/LiveEventQr.vue`
- Create: `frontend/src/components/LiveEventQr.test.ts`

- [ ] **Step 1: Install dependency**

```bash
npm install qrcode
npm install -D @types/qrcode
```

Working directory: `frontend`

- [ ] **Step 2: Write failing component test**

```ts
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import LiveEventQr from './LiveEventQr.vue'

vi.mock('qrcode', () => ({
  default: {
    toCanvas: vi.fn(async (_canvas: HTMLCanvasElement, _text: string) => undefined),
  },
}))

import QRCode from 'qrcode'

describe('LiveEventQr', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders caption and generates QR for the given url', async () => {
    const wrapper = mount(LiveEventQr, {
      props: { url: 'https://keweenawendurance.com/events/evt-1/live' },
    })
    await flushPromises()
    expect(wrapper.text()).toContain('view results at keweenawendurance.com')
    expect(wrapper.find('[data-testid="rotator-live-qr"]').exists()).toBe(true)
    expect(QRCode.toCanvas).toHaveBeenCalled()
    const [, text] = (QRCode.toCanvas as ReturnType<typeof vi.fn>).mock.calls[0]!
    expect(text).toBe('https://keweenawendurance.com/events/evt-1/live')
  })
})
```

- [ ] **Step 3: Run test to verify it fails**

Run: `npx vitest run src/components/LiveEventQr.test.ts`  
Expected: FAIL (component missing)

- [ ] **Step 4: Implement `LiveEventQr.vue`**

```vue
<template>
  <div class="live-event-qr" data-testid="rotator-live-qr">
    <canvas ref="canvasRef" class="live-event-qr__canvas" width="128" height="128" />
    <p class="live-event-qr__caption">view results at keweenawendurance.com</p>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import QRCode from 'qrcode'

const props = defineProps<{ url: string }>()
const canvasRef = ref<HTMLCanvasElement | null>(null)

async function renderQr() {
  const canvas = canvasRef.value
  if (!canvas || !props.url) return
  try {
    await QRCode.toCanvas(canvas, props.url, {
      width: 128,
      margin: 2,
      errorCorrectionLevel: 'M',
      color: { dark: '#1a1a1a', light: '#ffffff' },
    })
  } catch {
    // Keep caption; board must not break if QR fails.
  }
}

onMounted(() => {
  void renderQr()
})
watch(
  () => props.url,
  () => {
    void renderQr()
  },
)
</script>

<style scoped>
.live-event-qr {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.35rem;
  padding: 0.5rem;
  background: #fff;
  border-radius: 6px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.12);
  max-width: 10rem;
}
.live-event-qr__canvas {
  display: block;
  width: 128px;
  height: 128px;
}
.live-event-qr__caption {
  margin: 0;
  font-size: 0.7rem;
  line-height: 1.25;
  text-align: center;
  color: #1a1a1a;
}
</style>
```

- [ ] **Step 5: Run tests**

Run: `npx vitest run src/components/LiveEventQr.test.ts`  
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add frontend/package.json frontend/package-lock.json frontend/src/components/LiveEventQr.vue frontend/src/components/LiveEventQr.test.ts
git commit -m "feat(live): add LiveEventQr component with qrcode"
```

---

### Task 4: Wire settings checkbox + QR into EventLive

**Files:**
- Modify: `frontend/src/views/EventLive.vue`
- Modify: `frontend/src/views/EventLive.test.ts`

- [ ] **Step 1: Write failing integration tests** (in EventLive.test.ts, new describe `rotator live QR`)

Use existing `mountLive` helpers. Stub `LiveEventQr` is NOT required if `qrcode` is mocked globally in the describe, or mock the component:

```ts
vi.mock('@/components/LiveEventQr.vue', () => ({
  default: {
    name: 'LiveEventQr',
    props: ['url'],
    template:
      '<div data-testid="rotator-live-qr">view results at keweenawendurance.com {{ url }}</div>',
  },
}))
```

Prefer mocking only inside the describe via dynamic approach, OR import real component with qrcode mocked at file top of EventLive.test — simplest: add to the new describe by mounting and checking absence by default without mock (qrcode may need mock at top of EventLive.test if component is used).

Safer pattern: mock `qrcode` at the top of EventLive.test.ts once:

```ts
vi.mock('qrcode', () => ({
  default: { toCanvas: vi.fn(async () => undefined) },
}))
```

Tests:

```ts
describe('rotator live QR', () => {
  beforeEach(() => {
    sessionStorage.clear()
  })

  it('does not show QR by default', async () => {
    const wrapper = await mountLive()
    await wrapper.find('[data-testid="fullscreen-rotator-toggle"]').trigger('click')
    await nextTick()
    expect(wrapper.find('[data-testid="rotator-live-qr"]').exists()).toBe(false)
  })

  it('shows QR with live url when enabled in settings', async () => {
    const wrapper = await mountLive()
    await wrapper.find('[data-testid="fullscreen-rotator-toggle"]').trigger('click')
    await nextTick()
    await wrapper.find('[data-testid="rotator-settings-open"]').trigger('click')
    await nextTick()
    const checkbox = wrapper.find('[data-testid="rotator-show-qr"]')
    expect(checkbox.exists()).toBe(true)
    await checkbox.setValue(true)
    await nextTick()
    await wrapper.find('[data-testid="rotator-settings-done"]').trigger('click')
    await nextTick()
    const qr = wrapper.find('[data-testid="rotator-live-qr"]')
    expect(qr.exists()).toBe(true)
    expect(qr.text()).toContain('view results at keweenawendurance.com')
  })
})
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `npx vitest run src/views/EventLive.test.ts -t "rotator live QR"`  
Expected: FAIL (checkbox / QR missing)

- [ ] **Step 3: Wire EventLive**

1. Import `LiveEventQr` and `liveEventUrl`.
2. Destructure `setShowQrCode` from `useFullscreenRotator`.
3. Computed:

```ts
const showRotatorQr = computed(() => Boolean(rotatorSettings.value.showQrCode))
const rotatorLiveUrl = computed(() =>
  liveEventUrl(
    typeof window !== 'undefined' ? window.location.origin : '',
    eventId.value,
  ),
)
```

4. In settings dialog, after dwell field:

```vue
<label class="rotator-settings-field">
  <input
    data-testid="rotator-show-qr"
    type="checkbox"
    :checked="rotatorSettings.showQrCode"
    @change="setShowQrCode(($event.target as HTMLInputElement).checked)"
  />
  Show QR code for live page
</label>
```

5. Inside `.fs-root`, after `.fs-controls` (or wrapping both in a corner stack):

```vue
<div class="fs-corner" data-testid="rotator-corner">
  <div
    class="fs-controls"
    data-testid="rotator-controls"
    :class="{ 'fs-controls--idle': rotatorControlsIdle }"
  >
    <!-- existing buttons unchanged -->
  </div>
  <LiveEventQr v-if="showRotatorQr" :url="rotatorLiveUrl" />
</div>
```

Move existing `.fs-controls` into `.fs-corner`. For this task, `rotatorControlsIdle` can be a `ref(false)` always false — idle logic is Task 5. Or omit the class until Task 5.

CSS for `.fs-corner`:

```css
.fs-corner {
  position: absolute;
  top: 1rem;
  right: 1rem;
  z-index: 2;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 0.5rem;
}
.fs-controls {
  position: static; /* was absolute top/right — now parent positions */
  display: flex;
  gap: 0.5rem;
}
```

- [ ] **Step 4: Run tests**

Run: `npx vitest run src/views/EventLive.test.ts -t "rotator live QR"`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/views/EventLive.vue frontend/src/views/EventLive.test.ts
git commit -m "feat(live): wire optional rotator QR and settings checkbox"
```

---

### Task 5: Idle-hide controls when QR is on

**Files:**
- Modify: `frontend/src/views/EventLive.vue`
- Modify: `frontend/src/views/EventLive.test.ts`

- [ ] **Step 1: Write failing tests**

```ts
it('hides controls after idle when QR is enabled', async () => {
  vi.useFakeTimers()
  sessionStorage.setItem(
    'event-live-fs-rotator-settings',
    JSON.stringify({
      dwellMs: 5000,
      showQrCode: true,
      pages: [
        { race: '12h', mode: 'individuals', enabled: true },
        { race: '12h', mode: 'teams', enabled: true },
        { race: '6h', mode: 'individuals', enabled: true },
        { race: '6h', mode: 'teams', enabled: true },
      ],
    }),
  )
  const wrapper = await mountLive()
  await wrapper.find('[data-testid="fullscreen-rotator-toggle"]').trigger('click')
  await nextTick()
  expect(wrapper.find('[data-testid="rotator-live-qr"]').exists()).toBe(true)
  const controls = wrapper.find('[data-testid="rotator-controls"]')
  expect(controls.classes()).not.toContain('fs-controls--idle')
  await vi.advanceTimersByTimeAsync(3000)
  await nextTick()
  expect(controls.classes()).toContain('fs-controls--idle')
  vi.useRealTimers()
})

it('does not idle-hide controls when QR is off', async () => {
  vi.useFakeTimers()
  const wrapper = await mountLive()
  await wrapper.find('[data-testid="fullscreen-rotator-toggle"]').trigger('click')
  await nextTick()
  await vi.advanceTimersByTimeAsync(5000)
  await nextTick()
  expect(wrapper.find('[data-testid="rotator-controls"]').classes()).not.toContain(
    'fs-controls--idle',
  )
  vi.useRealTimers()
})
```

Note: other EventLive tests already use fake timers in nested describes — ensure this describe restores real timers in `afterEach`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `npx vitest run src/views/EventLive.test.ts -t "hides controls after idle|does not idle-hide"`  
Expected: FAIL

- [ ] **Step 3: Implement idle logic in EventLive**

```ts
const ROTATOR_CONTROLS_IDLE_MS = 3000
const rotatorControlsIdle = ref(false)
let rotatorIdleTimer: number | undefined

function clearRotatorIdleTimer() {
  if (rotatorIdleTimer !== undefined) {
    window.clearTimeout(rotatorIdleTimer)
    rotatorIdleTimer = undefined
  }
}

function bumpRotatorControlsActivity() {
  rotatorControlsIdle.value = false
  clearRotatorIdleTimer()
  if (!rotatorOpen.value || !showRotatorQr.value || rotatorSettingsOpen.value) return
  rotatorIdleTimer = window.setTimeout(() => {
    rotatorControlsIdle.value = true
    rotatorIdleTimer = undefined
  }, ROTATOR_CONTROLS_IDLE_MS)
}

watch(
  () => [rotatorOpen.value, showRotatorQr.value, rotatorSettingsOpen.value] as const,
  ([open, showQr, settingsOpen]) => {
    clearRotatorIdleTimer()
    rotatorControlsIdle.value = false
    if (open && showQr && !settingsOpen) bumpRotatorControlsActivity()
  },
)
```

On `.fs-root` / `.fs-corner`:

```vue
<div
  v-if="rotatorOpen"
  class="fs-root"
  ...
  @pointerdown="bumpRotatorControlsActivity"
  @pointermove="bumpRotatorControlsActivity"
  @keydown="bumpRotatorControlsActivity"
>
```

Also `@pointerenter` on `.fs-corner` calling `bumpRotatorControlsActivity`.

CSS:

```css
.fs-controls--idle {
  opacity: 0;
  pointer-events: none;
}
.fs-controls {
  transition: opacity 0.2s ease;
}
```

Clear idle timer in `onUnmounted`.

- [ ] **Step 4: Run tests**

Run: `npx vitest run src/views/EventLive.test.ts -t "rotator live QR"`  
Also: `npx vitest run src/views/EventLive.test.ts`  
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/views/EventLive.vue frontend/src/views/EventLive.test.ts
git commit -m "feat(live): idle-hide rotator controls when QR is shown"
```

---

## Spec coverage checklist

| Spec requirement | Task |
|------------------|------|
| `showQrCode` default false + normalize | Task 2 |
| Settings checkbox | Task 4 |
| URL = current origin live page | Tasks 1, 4 |
| Caption text | Task 3 |
| Top-right QR | Task 4 |
| Idle-hide controls ~3s when QR on | Task 5 |
| Controls always visible when QR off | Task 5 |
| `qrcode` package | Task 3 |
| Tests listed in spec | Tasks 1–5 |
