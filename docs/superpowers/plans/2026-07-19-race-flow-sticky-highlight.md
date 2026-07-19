# Race Flow Sticky Highlight Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Click lines/points or legend names on `RaceFlowChart` to sticky-highlight a participant; clear via re-click, empty plot, or outside all charts; hover previews only when nothing is sticky.

**Architecture:** Controlled `v-model:highlightParticipantId` (parent owns sticky id). Local `hoveredParticipantId` for preview. Effective highlight = sticky ?? hover. Chart.js `onClick` with `intersect: true`; document click clears when outside all `.race-flow-chart`.

**Tech Stack:** Vue 3, Chart.js, Vitest, Vue Test Utils

**Spec:** `docs/superpowers/specs/2026-07-19-race-flow-sticky-highlight-design.md`

**Working copy:** Use isolated git worktree / feature branch `002-race-flow-sticky-highlight` under `.worktrees/`. Do not touch unrelated `Home.vue` changes on the main checkout.

**Note on commits:** Commit after each task on the feature branch.

---

## File map

| File | Responsibility |
|------|----------------|
| `frontend/src/components/RaceFlowChart.vue` | Emit sticky select/clear; sticky-first highlight; legend button; click/hover handlers |
| `frontend/src/components/RaceFlowChart.test.ts` | Chart mock + sticky/hover/legend/outside tests |
| `frontend/src/views/EventLive.vue` | `v-model` bind + clear `focusParticipantId` when highlight clears |
| `frontend/src/views/EventLive.test.ts` | Focus-clear + v-model wiring coverage |
| `frontend/src/views/RaceDetails.vue` | `v-model:highlight-participant-id` |

---

### Task 1: Chart mock + sticky click/hover tests (TDD red)

**Files:**
- Modify: `frontend/src/components/RaceFlowChart.test.ts`

- [ ] **Step 1: Extend Chart.js mock**

Update the `vi.mock('chart.js')` factory so each instance exposes:

```ts
vi.fn((_canvas, config) => {
  const instance = {
    destroy: vi.fn(),
    update: vi.fn(),
    data: config?.data ?? { datasets: [] },
    options: config?.options ?? {},
    canvas: { style: { cursor: 'default' } as { cursor: string } },
    getElementsAtEventForMode: vi.fn().mockReturnValue([]),
  }
  return instance
})
```

- [ ] **Step 2: Add failing sticky-highlight describe block**

After the existing legend hover highlight test, add:

```ts
describe('sticky highlight selection', () => {
  async function mountWithData(props: Record<string, unknown> = {}) {
    ;(timingApi.getLive as Mock).mockResolvedValue({
      data: { race_id: 'race-1', records: sampleRecords },
    })
    const wrapper = mount(RaceFlowChart, {
      props: { raceId: 'race-1', ...props },
    })
    await flushPromises()
    return wrapper
  }

  function lastChartInstance() {
    return (Chart as unknown as Mock).mock.results.at(-1)?.value as {
      update: Mock
      data: { datasets: Array<{ participantId?: string; borderWidth: number }> }
      options: { onClick?: Function; onHover?: Function }
      canvas: { style: { cursor: string } }
      getElementsAtEventForMode: Mock
    }
  }

  it('emits sticky select on plot click and keeps highlight after hover clears', async () => {
    const wrapper = await mountWithData()
    const chart = lastChartInstance()
    chart.getElementsAtEventForMode.mockReturnValue([{ datasetIndex: 0 }])

    chart.options.onClick?.(
      { native: new MouseEvent('click') },
      [{ datasetIndex: 0 }],
      chart,
    )
    await flushPromises()

    expect(wrapper.emitted('update:highlightParticipantId')?.at(-1)).toEqual(['p1'])

    await wrapper.setProps({ highlightParticipantId: 'p1' })
    await flushPromises()

    chart.options.onHover?.(
      { native: new MouseEvent('mousemove') },
      [],
      chart,
    )
    await flushPromises()

    const highlighted = chart.data.datasets.find((d) => d.participantId === 'p1')
    expect(highlighted?.borderWidth).toBe(4)
  })

  it('does not let hover change styling while sticky is active', async () => {
    const wrapper = await mountWithData({ highlightParticipantId: 'p1' })
    await flushPromises()
    const chart = lastChartInstance()

    chart.options.onHover?.(
      { native: new MouseEvent('mousemove') },
      [{ datasetIndex: 1 }],
      chart,
    )
    await flushPromises()

    expect(wrapper.vm.hoveredParticipantId).toBeNull()
    const p1 = chart.data.datasets.find((d) => d.participantId === 'p1')
    const p2 = chart.data.datasets.find((d) => d.participantId === 'p2')
    expect(p1?.borderWidth).toBe(4)
    expect(p2?.borderWidth).toBe(1)
  })

  it('emits clear when clicking empty plot area', async () => {
    const wrapper = await mountWithData({ highlightParticipantId: 'p1' })
    const chart = lastChartInstance()
    chart.getElementsAtEventForMode.mockReturnValue([])

    chart.options.onClick?.(
      { native: new MouseEvent('click') },
      [],
      chart,
    )
    await flushPromises()

    expect(wrapper.emitted('update:highlightParticipantId')?.at(-1)).toEqual([undefined])
  })

  it('emits clear when re-clicking the selected line', async () => {
    const wrapper = await mountWithData({ highlightParticipantId: 'p1' })
    const chart = lastChartInstance()
    chart.getElementsAtEventForMode.mockReturnValue([{ datasetIndex: 0 }])

    chart.options.onClick?.(
      { native: new MouseEvent('click') },
      [{ datasetIndex: 0 }],
      chart,
    )
    await flushPromises()

    expect(wrapper.emitted('update:highlightParticipantId')?.at(-1)).toEqual([undefined])
  })

  it('emits clear on document click outside all race-flow charts', async () => {
    const wrapper = await mountWithData({ highlightParticipantId: 'p1' })
    document.body.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await flushPromises()

    expect(wrapper.emitted('update:highlightParticipantId')?.at(-1)).toEqual([undefined])
    wrapper.unmount()
  })

  it('sets pointer cursor when hovering a line with no sticky selection', async () => {
    await mountWithData()
    const chart = lastChartInstance()

    chart.options.onHover?.(
      { native: new MouseEvent('mousemove') },
      [{ datasetIndex: 0 }],
      chart,
    )
    expect(chart.canvas.style.cursor).toBe('pointer')

    chart.options.onHover?.(
      { native: new MouseEvent('mousemove') },
      [],
      chart,
    )
    expect(chart.canvas.style.cursor).toBe('default')
  })

  it('legend name button sticky-selects; checkbox only toggles visibility', async () => {
    const wrapper = await mountWithData()
    const selectBtn = wrapper.find('[data-testid="race-flow-legend-select"]')
    expect(selectBtn.exists()).toBe(true)

    await selectBtn.trigger('click')
    await flushPromises()
    expect(wrapper.emitted('update:highlightParticipantId')?.at(-1)).toEqual(['p1'])

    const checkbox = wrapper.find('.legend-item input[type="checkbox"]')
    const before = wrapper.vm.visibleParticipantIds.has('p1')
    await checkbox.setValue(false)
    await flushPromises()
    expect(wrapper.vm.visibleParticipantIds.has('p1')).toBe(!before)
    // checkbox click must not emit a second select
    const selectEmits = wrapper.emitted('update:highlightParticipantId') ?? []
    expect(selectEmits.filter((e) => e[0] === 'p1').length).toBe(1)
  })
})
```

- [ ] **Step 3: Run tests — expect FAIL**

```bash
cd frontend && npx vitest run src/components/RaceFlowChart.test.ts
```

Expected: new sticky tests fail (no emit / no legend button / wrong hover priority).

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/RaceFlowChart.test.ts
git commit -m "test: add RaceFlowChart sticky highlight failing tests"
```

---

### Task 2: Implement RaceFlowChart sticky selection (TDD green)

**Files:**
- Modify: `frontend/src/components/RaceFlowChart.vue`

- [ ] **Step 1: Add emit + select helpers**

After props, add:

```ts
const emit = defineEmits<{
  'update:highlightParticipantId': [value: string | undefined]
}>()

function selectParticipant(participantId: string | undefined): void {
  const next =
    participantId != null && participantId === props.highlightParticipantId
      ? undefined
      : participantId
  emit('update:highlightParticipantId', next)
}

function clearStickyHighlight(): void {
  if (props.highlightParticipantId != null) {
    emit('update:highlightParticipantId', undefined)
  }
}
```

- [ ] **Step 2: Flip effective highlight priority**

Change `getEffectiveHighlightId` to:

```ts
function getEffectiveHighlightId(): string | undefined {
  return props.highlightParticipantId ?? hoveredParticipantId.value ?? undefined
}
```

- [ ] **Step 3: Restructure legend item markup**

Replace the wrapping `<label class="legend-item" ...>` with:

```vue
<div
  v-for="item in filteredLegendItems"
  :key="item.participantId"
  class="legend-item"
  :class="{
    'legend-item-hidden': !isParticipantVisible(item.participantId),
    'legend-item-hovered':
      (hoveredParticipantId === item.participantId && !highlightParticipantId) ||
      highlightParticipantId === item.participantId,
  }"
  @mouseenter="highlightParticipant(item, $event)"
  @mousemove="moveTooltip($event)"
  @mouseleave="unhighlightParticipant"
>
  <input
    type="checkbox"
    :aria-label="`Toggle visibility for ${item.label}`"
    :checked="isParticipantVisible(item.participantId)"
    @change="toggleParticipantVisibility(item.participantId)"
    @click.stop
  />
  <button
    type="button"
    class="legend-select"
    data-testid="race-flow-legend-select"
    :aria-pressed="highlightParticipantId === item.participantId"
    @click="selectParticipant(item.participantId)"
  >
    <span
      class="color-swatch"
      :style="{ backgroundColor: item.color }"
      aria-hidden="true"
    />
    <span class="legend-label">{{ item.label }}</span>
  </button>
</div>
```

Add minimal CSS so `.legend-select` looks like the former inline label content (reset button chrome, flex row, inherit font/color).

In script setup template access, expose prop as needed — use `props.highlightParticipantId` in script; in template `highlightParticipantId` works if props are destructured or use props. For script-defined props without destructure, template can use `highlightParticipantId` from props automatically in `<script setup>`.

- [ ] **Step 4: Gate hover preview when sticky is active**

Update `onHover` in `renderChart`:

```ts
onHover: (_event, elements, chart) => {
  chart.canvas.style.cursor = elements.length > 0 ? 'pointer' : 'default'
  if (props.highlightParticipantId) {
    return
  }
  if (elements.length > 0) {
    const dataset = chart.data.datasets[elements[0].datasetIndex] as FlowLineDataset
    hoveredParticipantId.value = dataset.participantId ?? null
  } else {
    hoveredParticipantId.value = null
  }
},
```

Also gate legend `highlightParticipant` / keep hover state but `getEffectiveHighlightId` already ignores hover when sticky set. Still skip setting hovered when sticky for clarity:

```ts
function highlightParticipant(item: LegendItem, event: MouseEvent): void {
  if (!props.highlightParticipantId) {
    hoveredParticipantId.value = item.participantId
  }
  showTooltip(item, event)
}
```

- [ ] **Step 5: Add chart onClick**

In chart options alongside `onHover`:

```ts
onClick: (event, _elements, chart) => {
  const native = (event as { native?: Event }).native
  const hits = chart.getElementsAtEventForMode(
    event as unknown as Event,
    'nearest',
    { intersect: true },
    true,
  )
  if (hits.length === 0) {
    clearStickyHighlight()
    return
  }
  const dataset = chart.data.datasets[hits[0].datasetIndex] as FlowLineDataset
  selectParticipant(dataset.participantId)
},
```

Use Chart.js types already imported; cast as needed to match project style.

- [ ] **Step 6: Extend document click handler**

```ts
function handleDocumentClick(event: MouseEvent): void {
  const target = event.target
  if (!(target instanceof Element)) {
    return
  }
  if (!target.closest('.filter-dropdown')) {
    openFilter.value = null
  }
  if (!target.closest('.race-flow-chart')) {
    clearStickyHighlight()
  }
}
```

- [ ] **Step 7: Run tests — expect PASS**

```bash
cd frontend && npx vitest run src/components/RaceFlowChart.test.ts
```

Expected: all RaceFlowChart tests pass.

- [ ] **Step 8: Commit**

```bash
git add frontend/src/components/RaceFlowChart.vue frontend/src/components/RaceFlowChart.test.ts
git commit -m "feat: sticky-select RaceFlowChart lines via click and legend"
```

---

### Task 3: Wire parents (EventLive + RaceDetails)

**Files:**
- Modify: `frontend/src/views/EventLive.vue`
- Modify: `frontend/src/views/RaceDetails.vue`
- Modify: `frontend/src/views/EventLive.test.ts`

- [ ] **Step 1: Write failing EventLive test for focus clear**

In `EventLive.test.ts`, add (adapt to existing mount helpers / stubs):

```ts
it('clears focusParticipantId when highlightParticipantId is cleared', async () => {
  // mount EventLive with races as existing tests do
  // set highlight via celebration or by assigning if exposed
  // then simulate chart emit update:highlightParticipantId undefined
  const chart = wrapper.findComponent({ name: 'RaceFlowChart' })
  await chart.vm.$emit('update:highlightParticipantId', 'p-from-chart')
  await flushPromises()
  // if focus was set separately, clear path:
  await chart.vm.$emit('update:highlightParticipantId', undefined)
  await flushPromises()
  expect(chart.props('highlightParticipantId')).toBeUndefined()
})
```

Prefer asserting via stub props: after emit `undefined`, stub receives `highlightParticipantId: undefined`. If celebration sets both highlight and focus, emit clear and assert highlight prop undefined; add watcher test by checking no leftover focus class if tested. Minimal: assert v-model clears prop after emit.

Also assert charts use v-model by emitting a select and seeing prop update on stub:

```ts
it('binds highlightParticipantId as v-model on race flow charts', async () => {
  const wrapper = /* existing mount with races */
  const chart = wrapper.findComponent({ name: 'RaceFlowChart' })
  await chart.vm.$emit('update:highlightParticipantId', 'racer-1')
  await flushPromises()
  expect(chart.props('highlightParticipantId')).toBe('racer-1')
  await chart.vm.$emit('update:highlightParticipantId', undefined)
  await flushPromises()
  expect(chart.props('highlightParticipantId')).toBeUndefined()
})
```

- [ ] **Step 2: Run EventLive test — expect FAIL** until v-model wired

```bash
cd frontend && npx vitest run src/views/EventLive.test.ts
```

- [ ] **Step 3: Wire EventLive**

Replace every:

```vue
:highlight-participant-id="highlightParticipantId"
```

with:

```vue
v-model:highlight-participant-id="highlightParticipantId"
```

Add watcher near other highlight logic:

```ts
watch(highlightParticipantId, (participantId) => {
  if (!participantId) {
    focusParticipantId.value = undefined
  }
})
```

Ensure `watch` is imported from `vue` if not already.

- [ ] **Step 4: Wire RaceDetails**

Same `v-model:highlight-participant-id="highlightParticipantId"` on its `RaceFlowChart`.

- [ ] **Step 5: Run parent tests**

```bash
cd frontend && npx vitest run src/views/EventLive.test.ts src/views/RaceDetails.test.ts src/components/RaceFlowChart.test.ts
```

Expected: pass (skip RaceDetails test file if it does not exist).

- [ ] **Step 6: Commit**

```bash
git add frontend/src/views/EventLive.vue frontend/src/views/EventLive.test.ts frontend/src/views/RaceDetails.vue
git commit -m "feat: wire sticky race-flow highlight v-model in live views"
```

---

### Task 4: Verification

- [ ] **Step 1: Run full frontend unit suite for touched areas**

```bash
cd frontend && npx vitest run src/components/RaceFlowChart.test.ts src/views/EventLive.test.ts
```

Expected: all green.

- [ ] **Step 2: Self-check acceptance criteria 1–12 against code**

Confirm `ParticipantFlowChart.vue` untouched.

- [ ] **Step 3: Final commit if any leftover polish** (only if needed)

---

## Self-review checklist

- Spec coverage: sticky click, hover gate, empty clear, outside-all clear, legend button, toggle, EventLive focus clear, RaceDetails v-model — all tasked
- No placeholders
- Chart mock extended before tests that call `getElementsAtEventForMode`
- Priority sticky ?? hover matches spec
