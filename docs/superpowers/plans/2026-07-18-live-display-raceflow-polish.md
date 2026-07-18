# Live Display + Race-Flow Polish — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix Bluffet live display: keep home reachable, apply full event chrome, cap race-flow x-axis to race duration, fix legend contrast, drop Yuji Mai, and enlarge type for laptop + TV.

**Architecture:** Small frontend-first changes with one backend field addition (`duration_minutes` on event-live races). RaceFlowChart clamps wall-clock “now” and Chart.js x-max to that duration. EventLive sets `events.currentEvent` from the live payload so `theme-bluffet` activates by name when the seeded UUID differs. Bluffet CSS loses solid-red legend items and Yuji Mai; Event Live gains baseline + fullscreen type scales.

**Tech Stack:** Vue 3, Vitest, Chart.js, Pinia, Go/Gin live API

**Spec:** `docs/superpowers/specs/2026-07-18-live-display-raceflow-polish-design.md`

**Note on commits:** Do not commit unless the user explicitly asks. Skip commit steps during execution unless requested.

---

## File map

| File | Responsibility |
|---|---|
| `frontend/src/router/index.ts` | Remove `/` → live auto-redirect |
| `frontend/src/views/Home.vue` | Keep hero “View Live Timing” as top CTA (no redirect dependency) |
| `frontend/src/views/Home.test.ts` | Assert home stays on `/` / CTA prominence if needed |
| `frontend/src/composables/useBluffetTheme.ts` | Ensure name/id activation still works with live |
| `frontend/src/views/EventLive.vue` | Set `currentEvent` from live; pass `duration_minutes`; TV type CSS |
| `frontend/src/themes/bluffet.css` | Full live chrome; remove Yuji Mai + solid-red `.legend-item` |
| `frontend/src/main.ts` | Drop `@fontsource/yuji-mai` import |
| `frontend/package.json` | Remove `@fontsource/yuji-mai` dependency |
| `backend/internal/services/results_service.go` | Add `duration_minutes` to `LiveRaceView` |
| `backend/internal/services/results_service_test.go` | Assert duration in live payload |
| `frontend/src/services/api.ts` | `EventLiveRace.duration_minutes` |
| `frontend/src/utils/raceFlowData.ts` | `clampElapsedToDuration` + `resolveRaceFlowAxisMaxMinutes` |
| `frontend/src/components/RaceFlowChart.vue` | Prop + clamp axis/now/extrapolation; chart fonts |
| `frontend/src/components/ParticipantFlowChart.vue` | Same duration clamp for shared live path |
| `frontend/src/components/RaceFlowChart.test.ts` | Failing→passing duration-cap tests |
| `frontend/src/views/EventLive.test.ts` | Theme store + duration prop wiring |
| `frontend/src/views/Home.test.ts` / bluffet theme tests | Yuji Mai / legend CSS assertions |

---

### Task 1: Remove home auto-redirect

**Files:**
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/views/Home.test.ts` (add redirect-regression test if useful)

- [ ] **Step 1: Write a failing router behavior test**

Add to `frontend/src/views/Home.test.ts` (or a new `frontend/src/router/homeRedirect.test.ts` if cleaner):

```ts
it('does not force-navigate away from home when races are active', async () => {
  // Home mount must remain on name "home" — router guard must not rewrite /
  const router = createHomeRouter()
  await router.push('/')
  await router.isReady()
  expect(router.currentRoute.value.name).toBe('home')
})
```

Also assert the hero CTA exists and is the primary live entry:

```ts
expect(wrapper.find('[data-testid="timing-cta"]').exists()).toBe(true)
```

- [ ] **Step 2: Run test (baseline)**

Run: `cd frontend; npx vitest run src/views/Home.test.ts`

Expected: PASS for existing tests; new assertion should pass once redirect is removed (if it currently never redirected in unit tests because APIs are mocked, still proceed — the production bug is the guard itself).

- [ ] **Step 3: Delete the home redirect guard**

In `frontend/src/router/index.ts`, remove the entire `router.beforeEach` block (lines ~74–132) and unused imports (`eventsApi`, `eventsLiveApi`, `racesApi`, `useStationStore`) if they become unused.

Replace with nothing — keep routes only:

```ts
const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
})

export default router
```

- [ ] **Step 4: Confirm Home CTA still points at live when Bluffet is known**

`Home.vue` already does:

```ts
const liveTimingTarget = computed(() =>
  bluffetEventId.value ? `/events/${bluffetEventId.value}/live` : '/timing',
)
```

No layout change required — hero CTA stays topmost.

- [ ] **Step 5: Re-run Home tests**

Run: `cd frontend; npx vitest run src/views/Home.test.ts`

Expected: PASS

---

### Task 2: Activate Bluffet theme on Event Live

**Files:**
- Modify: `frontend/src/views/EventLive.vue`
- Modify: `frontend/src/views/EventLive.test.ts`
- Modify: `frontend/src/composables/useBluffetTheme.spec.ts` (optional reinforcement)

- [ ] **Step 1: Write failing test — live load sets currentEvent name**

In `EventLive.test.ts`, after mounting and flushing live load:

```ts
it('sets events store currentEvent from live payload for theme activation', async () => {
  // ...existing mount with livePayload where event.name is Bluffet...
  const events = useEventsStore()
  expect(events.currentEvent?.id).toBe(livePayload.event.id)
  expect(events.currentEvent?.name).toBe(livePayload.event.name)
})
```

Ensure `livePayload.event.name` is `'All You Can East Bluffet'` in the fixture (or set it in this test).

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend; npx vitest run src/views/EventLive.test.ts`

Expected: FAIL — `currentEvent` is null

- [ ] **Step 3: Set currentEvent when live loads**

In `EventLive.vue` `loadLive` (and the poll path that assigns `live.value = data`):

```ts
import { useEventsStore } from '@/stores/events'

const eventsStore = useEventsStore()

// inside successful getLive:
live.value = data
eventsStore.currentEvent = {
  id: data.event.id,
  name: data.event.name,
  event_date: eventsStore.currentEvent?.event_date ?? '',
  status: eventsStore.currentEvent?.status ?? 'active',
}
```

(`Event` type may require more fields — satisfy the type with minimal placeholders matching other tests, or call `eventsStore.fetchEvent(eventId)` if preferred and cheaper than inventing fields.)

Preferred if `fetchEvent` is reliable:

```ts
void eventsStore.fetchEvent(eventId.value)
```

But that is an extra network call. Prefer assigning from live payload with a type-safe partial cast only if the store type allows; otherwise extend store with `setCurrentEventSummary({ id, name })`.

Minimal store addition (cleaner):

```ts
// events.ts actions
setCurrentEventSummary(summary: { id: string; name: string }) {
  this.currentEvent = {
    ...(this.currentEvent ?? {
      event_date: '',
      status: 'active' as const,
    }),
    id: summary.id,
    name: summary.name,
  }
}
```

Call `eventsStore.setCurrentEventSummary(data.event)` after each successful live fetch/poll.

- [ ] **Step 4: Run tests**

Run: `cd frontend; npx vitest run src/views/EventLive.test.ts src/composables/useBluffetTheme.spec.ts src/App.theme.spec.ts`

Expected: PASS

---

### Task 3: Drop Yuji Mai + fix legend red + full live Bluffet chrome

**Files:**
- Modify: `frontend/src/themes/bluffet.css`
- Modify: `frontend/src/main.ts`
- Modify: `frontend/package.json` (+ lockfile via npm uninstall)
- Modify: `frontend/src/views/Home.test.ts` (CSS contract assertions)

- [ ] **Step 1: Write failing CSS contract tests**

In `Home.test.ts` (or a dedicated `bluffet.css` test), replace/extend assertions:

```ts
it('uses IBM Plex for Bluffet titles, not Yuji Mai', () => {
  expect(bluffetCss).not.toMatch(/Yuji Mai/)
  expect(bluffetCss).toMatch(
    /#app\.theme-bluffet\s*\{[^}]*font-family:\s*'IBM Plex Sans'/s,
  )
})

it('does not paint legend-item solid Bluffet red', () => {
  expect(bluffetCss).not.toMatch(
    /#app\.theme-bluffet\s+\.legend-item\s*\{[^}]*background:\s*var\(--bluffet-red\)/s,
  )
})
```

- [ ] **Step 2: Run to verify fail**

Run: `cd frontend; npx vitest run src/views/Home.test.ts`

Expected: FAIL on new assertions

- [ ] **Step 3: Update `bluffet.css`**

1. Remove the Yuji Mai block entirely (`#app.theme-bluffet .bluffet-display, h1.page-title, .featured-event h2 { font-family: 'Yuji Mai' ... }`). Titles inherit IBM Plex from `#app.theme-bluffet`.

2. Change the chip/tab selected rule so `.legend-item` is **not** included:

```css
#app.theme-bluffet .bluffet-chip,
#app.theme-bluffet .race-tabs button[aria-selected='true'] {
  background: var(--bluffet-red);
  color: #fff;
  border: var(--bluffet-outline);
}
```

3. Add readable legend selected/hover under Bluffet:

```css
#app.theme-bluffet .legend-item {
  background: var(--bluffet-panel);
  color: var(--bluffet-ink);
  border: 1px solid color-mix(in srgb, var(--bluffet-ink) 20%, transparent);
}

#app.theme-bluffet .legend-item-hovered,
#app.theme-bluffet .filter-dropdown-option.selected {
  background: color-mix(in srgb, var(--bluffet-red) 12%, var(--bluffet-panel));
  color: var(--bluffet-ink);
}
```

4. Strengthen live chrome (panels, tabs, page background already partially set). Ensure Event Live panels get outline/shadow via existing `.panel` / `.race-tabs` rules. Add if missing:

```css
#app.theme-bluffet .event-live {
  /* inherits #app background/color; ensure contrast */
}

#app.theme-bluffet .event-live .panel,
#app.theme-bluffet .event-live .chart-wrap,
#app.theme-bluffet .event-live .legend {
  background: var(--bluffet-panel);
  border: var(--bluffet-outline);
  box-shadow: var(--bluffet-shadow);
  border-radius: 6px;
}
```

(Adjust selectors to match actual EventLive class names — `.panel`, `.chart-wrap`, `.legend` exist in `EventLive.vue`.)

- [ ] **Step 4: Remove font package**

In `frontend/src/main.ts`, delete:

```ts
import '@fontsource/yuji-mai/400.css'
```

Run: `cd frontend; npm uninstall @fontsource/yuji-mai`

- [ ] **Step 5: Re-run CSS / Home tests**

Run: `cd frontend; npx vitest run src/views/Home.test.ts`

Expected: PASS

---

### Task 4: Expose `duration_minutes` on event-live API

**Files:**
- Modify: `backend/internal/services/results_service.go`
- Modify: `backend/internal/services/results_service_test.go`
- Modify: `frontend/src/services/api.ts`

- [ ] **Step 1: Write failing Go test**

In `results_service_test.go` near `GetEventLive` coverage, assert:

```go
require.Equal(t, 720, live.Races[0].DurationMinutes) // for the 12 Hour fixture race
```

(Use the existing Bluffet/lap fixture that already sets `DurationMinutes: 720`.)

- [ ] **Step 2: Run to verify fail**

Run: `cd backend; go test ./internal/services/ -run GetEventLive -count=1`

Expected: FAIL — field missing or zero

- [ ] **Step 3: Add field to `LiveRaceView` and populate**

```go
type LiveRaceView struct {
	ID                 uuidutil.PublicUUID `json:"id"`
	Name               string              `json:"name"`
	RaceType           string              `json:"race_type"`
	Status             string              `json:"status"`
	StartTime          time.Time           `json:"start_time"`
	DurationMinutes    int                 `json:"duration_minutes"`
	CountdownSeconds   int                 `json:"countdown_seconds"`
	LeaderboardOverall []LiveOverallEntry  `json:"leaderboard_overall"`
	FlowSeries         []interface{}       `json:"flow_series"`
}
```

When building views:

```go
views = append(views, LiveRaceView{
	// ...existing fields...
	DurationMinutes: race.DurationMinutes,
})
```

- [ ] **Step 4: Update frontend type**

```ts
export interface EventLiveRace {
  id: string
  name: string
  race_type: RaceType
  status: string
  start_time: string
  duration_minutes?: number
  countdown_seconds: number
  leaderboard_overall: LiveLeaderboardEntry[]
  flow_series: unknown[]
}
```

- [ ] **Step 5: Run Go + any API type tests**

Run: `cd backend; go test ./internal/services/ -run GetEventLive -count=1`

Expected: PASS

---

### Task 5: Duration helpers in `raceFlowData` (TDD)

**Files:**
- Modify: `frontend/src/utils/raceFlowData.ts`
- Modify: `frontend/src/components/RaceFlowChart.test.ts` (utils describe block already imports from raceFlowData)

- [ ] **Step 1: Write failing unit tests**

```ts
import {
  clampElapsedToDuration,
  resolveRaceFlowAxisMaxMinutes,
} from '@/utils/raceFlowData'

it('clamps wall-clock elapsed to duration', () => {
  expect(clampElapsedToDuration(2559, 720)).toBe(720)
  expect(clampElapsedToDuration(100, 720)).toBe(100)
  expect(clampElapsedToDuration(100, undefined)).toBe(100)
})

it('uses duration as axis max when present', () => {
  expect(resolveRaceFlowAxisMaxMinutes(720, 12, 2559)).toBe(720)
  expect(resolveRaceFlowAxisMaxMinutes(360, 5, 40)).toBe(360)
})

it('falls back to recorded/live max when duration missing', () => {
  expect(resolveRaceFlowAxisMaxMinutes(undefined, 45, 50)).toBe(50)
  expect(resolveRaceFlowAxisMaxMinutes(0, 45, null)).toBe(45)
})
```

- [ ] **Step 2: Run to verify fail**

Run: `cd frontend; npx vitest run src/components/RaceFlowChart.test.ts -t "clamps wall-clock"`

Expected: FAIL — exports missing

- [ ] **Step 3: Implement helpers**

```ts
export function clampElapsedToDuration(
  elapsedMinutes: number,
  durationMinutes?: number | null,
): number {
  if (durationMinutes != null && durationMinutes > 0) {
    return Math.min(elapsedMinutes, durationMinutes)
  }
  return elapsedMinutes
}

export function resolveRaceFlowAxisMaxMinutes(
  durationMinutes: number | null | undefined,
  recordedMaxMinutes: number,
  currentElapsedMinutes: number | null,
): number {
  if (durationMinutes != null && durationMinutes > 0) {
    return durationMinutes
  }
  return Math.max(recordedMaxMinutes, currentElapsedMinutes ?? 0)
}
```

- [ ] **Step 4: Run tests**

Run: `cd frontend; npx vitest run src/components/RaceFlowChart.test.ts -t "clamps wall-clock|uses duration|falls back"`

Expected: PASS

---

### Task 6: Wire duration into RaceFlowChart (+ ParticipantFlowChart)

**Files:**
- Modify: `frontend/src/components/RaceFlowChart.vue`
- Modify: `frontend/src/components/ParticipantFlowChart.vue`
- Modify: `frontend/src/views/EventLive.vue`
- Modify: `frontend/src/views/RaceDetails.vue` (if it mounts RaceFlowChart — pass `race.duration_minutes` when available)
- Modify: `frontend/src/components/RaceFlowChart.test.ts`
- Modify: `frontend/src/views/EventLive.test.ts`

- [ ] **Step 1: Write failing chart test — axis capped**

```ts
it('caps x-axis and extrapolations to duration_minutes for active races', async () => {
  vi.useFakeTimers()
  vi.setSystemTime(new Date('2024-06-03T12:00:00.000Z')) // ~2 days after start

  ;(timingApi.getLive as Mock).mockResolvedValue({
    data: { race_id: 'race-1', records: sampleRecords },
  })

  mount(RaceFlowChart, {
    props: {
      raceId: 'race-1',
      raceStatus: 'active',
      raceStartTime: '2024-06-01T10:30:00.000Z',
      raceType: 'lap_based',
      durationMinutes: 720,
    },
  })
  await flushPromises()

  const chartConfig = (Chart as unknown as Mock).mock.calls.at(-1)?.[1] as {
    options: { scales: { x: { max?: number } }; plugins: { currentTimeLine: { xMinutes: number | null } } }
    data: { datasets: Array<{ data: Array<{ x: number }> }> }
  }

  expect(chartConfig.options.scales.x.max).toBe(720)
  expect(chartConfig.options.plugins.currentTimeLine.xMinutes).toBeLessThanOrEqual(720)

  const xs = chartConfig.data.datasets.flatMap((d) => d.data.map((p) => p.x))
  expect(Math.max(...xs)).toBeLessThanOrEqual(720)

  vi.useRealTimers()
})
```

(Adapt field names to match how datasets encode points — existing tests already inspect extrapolation / plugins.)

- [ ] **Step 2: Run to verify fail**

Run: `cd frontend; npx vitest run src/components/RaceFlowChart.test.ts -t "caps x-axis"`

Expected: FAIL — `max` ≫ 720 or undefined

- [ ] **Step 3: Add prop and apply helpers in RaceFlowChart**

```ts
const props = defineProps<{
  raceId: string
  raceStatus?: RaceStatus
  raceStartTime?: string
  raceType?: RaceType
  durationMinutes?: number
  highlightParticipantId?: string
}>()
```

Update `currentElapsedMinutes` computed:

```ts
const elapsed = clampElapsedToDuration(
  getCurrentElapsedMinutes(raceStartMs.value, nowMs.value),
  props.durationMinutes,
)
```

In `renderChart` / dataset builders, when calling `buildExtrapolationPoint`, pass the clamped elapsed. Set:

```ts
const recordedMax = /* existing reduce over visible flows without unbounded extrapolation */
const axisMax = resolveRaceFlowAxisMaxMinutes(
  props.durationMinutes,
  recordedMax,
  currentElapsedMinutes.value,
)

// scales.x.max = axisMax when duration present OR when showCurrentTime
max: props.durationMinutes || showCurrentTime ? axisMax : undefined,
```

Do **not** multiply by `1.05` when duration is set (that reintroduces empty runway past finish). Optional 1.02 padding only when duration is absent.

- [ ] **Step 4: Pass duration from EventLive**

Every `<RaceFlowChart>`:

```vue
:duration-minutes="race12.duration_minutes"
```

Overlap tab: use `Math.max(race12?.duration_minutes ?? 0, race6?.duration_minutes ?? 0) || undefined` for each chart, or pass each race’s own duration to its own chart (preferred — each chart keeps its race window).

- [ ] **Step 5: Mirror clamp in ParticipantFlowChart**

Same `durationMinutes` prop + `clampElapsedToDuration` on its live elapsed path.

- [ ] **Step 6: Run chart + EventLive tests**

Run: `cd frontend; npx vitest run src/components/RaceFlowChart.test.ts src/views/EventLive.test.ts`

Expected: PASS

---

### Task 7: TV / display type scales

**Files:**
- Modify: `frontend/src/views/EventLive.vue` (scoped CSS)
- Modify: `frontend/src/components/RaceFlowChart.vue` (Chart.js font sizes; optional CSS vars)
- Modify: `frontend/src/views/EventLive.test.ts` or a small CSS contract test

- [ ] **Step 1: Add CSS contract assertions**

```ts
it('defines baseline and fullscreen display type scales', () => {
  const src = readFileSync(join(process.cwd(), 'src/views/EventLive.vue'), 'utf8')
  expect(src).toMatch(/--live-display-scale:\s*1/)
  expect(src).toMatch(/fullscreen-rotator[\s\S]*--live-display-scale:\s*1\.35|--live-display-scale:\s*1\.25/)
})
```

- [ ] **Step 2: Implement baseline + fullscreen scales in EventLive**

```css
.event-live {
  --live-display-scale: 1;
  --live-title-size: calc(1.75rem * var(--live-display-scale));
  --live-tab-size: calc(1rem * var(--live-display-scale));
  --live-body-size: calc(1rem * var(--live-display-scale));
}

.page-title {
  font-size: var(--live-title-size);
}

.race-tabs button {
  font-size: var(--live-tab-size);
}

.legend,
.panel,
.meta-bar {
  font-size: var(--live-body-size);
}

/* Baseline bump for room readability even before fullscreen */
@media (min-width: 900px) {
  .event-live {
    --live-display-scale: 1.15;
  }
}

[data-testid='fullscreen-rotator'] {
  --live-display-scale: 1.35;
  font-size: calc(1rem * var(--live-display-scale));
}
```

Tune numbers so laptop isn’t absurd; fullscreen is clearly larger.

- [ ] **Step 3: Chart.js fonts follow scale**

In RaceFlowChart, read CSS variable from canvas parent or use props `displayScale?: number`. Simplest approach without prop plumbing:

```ts
function chartFontSize(base: number): number {
  if (typeof window === 'undefined') return base
  const raw = getComputedStyle(document.documentElement)
    .getPropertyValue('--live-display-scale')
    .trim()
  const scale = Number.parseFloat(raw || '1') || 1
  return Math.round(base * scale)
}
```

Apply to `options.scales.x.title.font.size`, `ticks.font.size`, `plugins.title.font.size` (e.g. base 12/14).

Better: set `--live-display-scale` on `.event-live` and on fullscreen rotator; RaceFlowChart reads from `canvasRef.value?.closest('.event-live, [data-testid=fullscreen-rotator]')`.

- [ ] **Step 4: Run EventLive + RaceFlowChart tests**

Run: `cd frontend; npx vitest run src/views/EventLive.test.ts src/components/RaceFlowChart.test.ts`

Expected: PASS

---

### Task 8: Verification sweep

- [ ] **Step 1: Frontend unit suite (touched areas)**

Run:

```bash
cd frontend
npx vitest run src/views/Home.test.ts src/views/EventLive.test.ts src/components/RaceFlowChart.test.ts src/composables/useBluffetTheme.spec.ts src/App.theme.spec.ts src/views/Home.test.ts
```

Expected: all PASS

- [ ] **Step 2: Backend live duration**

Run: `cd backend; go test ./internal/services/ -run GetEventLive -count=1`

Expected: PASS

- [ ] **Step 3: Manual checklist (against running stack)**

1. With an active race, open `/` — stays on home; hero **View Live Timing** is top CTA.
2. Open live — cream/paper Bluffet chrome, red accents, IBM Plex only.
3. Legend participant rows readable (not solid red).
4. 6h chart x-max ≈ 360; 12h ≈ 720; early data not crushed left; no dashed runway to 2500+.
5. Fullscreen rotate text/axes clearly larger than baseline.

---

## Spec coverage self-check

| Spec requirement | Task |
|---|---|
| §1 No home redirect; live CTA on home | Task 1 |
| §2 Full Bluffet chrome on live | Tasks 2–3 |
| §2 Drop Yuji Mai / IBM Plex | Task 3 |
| §2 Legend not solid red | Task 3 |
| §3 Duration-capped axis + clamp now/extrapolation | Tasks 4–6 |
| §3 Overlap uses each race duration / longest where shared | Task 6 |
| §4 Baseline + fullscreen type | Task 7 |
| Verification | Task 8 |

## Placeholder scan

No TBD/TODO placeholders. Commit steps omitted per repo preference unless user requests commits.
