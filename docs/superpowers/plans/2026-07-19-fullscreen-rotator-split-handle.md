# Fullscreen Rotator Split Handle — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a drag handle between plot and leaderboard in Fullscreen rotate so operators can adjust relative pane widths.

**Architecture:** Keep `.fs-grid` as a two-column CSS grid. Insert a thin splitter button between the flow and leaderboard panels. Pointer drag updates a clamped percentage stored in a Vue ref and mirrored to `--fs-flow-width` plus `sessionStorage`.

**Tech Stack:** Vue 3, Vitest + Vue Test Utils, existing EventLive.vue scoped CSS / Bluffet tokens.

---

### Task 1: Failing tests for split handle markup + drag behavior

**Files:**
- Modify: `frontend/src/views/EventLive.test.ts`

- [ ] **Step 1: Write failing tests**

Add tests that:

1. After opening fullscreen rotator, `[data-testid="rotator-split-handle"]` exists between flow and leaderboard.
2. `.fs-grid` style/template uses `--fs-flow-width` (source contract or rendered style).
3. Simulating pointerdown/move/up on the handle (or calling an exposed helper if cleaner) updates the grid column ratio / `--fs-flow-width` and clamps to 25–75.
4. Source contract: `sessionStorage` key like `event-live-fs-flow-width` is read/written.

Keep tests focused; mock `sessionStorage` if needed.

- [ ] **Step 2: Run tests — expect FAIL**

```bash
cd frontend && npm test -- --run src/views/EventLive.test.ts -t "split|rotator-split|fs-flow"
```

- [ ] **Step 3: Commit** (only if user/repo policy allows mid-plan commits; otherwise leave uncommitted until feature done)

---

### Task 2: Implement split handle in EventLive fullscreen rotator

**Files:**
- Modify: `frontend/src/views/EventLive.vue`
- Optionally: `frontend/src/themes/bluffet.css` (handle hover/outline if needed)

- [ ] **Step 1: Template**

Inside `.fs-grid`, between rotator-flow and rotator-leaderboard:

```html
<button
  type="button"
  class="fs-split"
  data-testid="rotator-split-handle"
  aria-label="Resize race flow and leaderboard"
  aria-orientation="vertical"
  @pointerdown="onFsSplitPointerDown"
/>
```

Bind grid style: `:style="{ '--fs-flow-width': fsFlowWidthPercent + '%' }"`.

- [ ] **Step 2: Script**

- `fsFlowWidthPercent` ref, default `52`
- On mount / when rotator opens: restore from `sessionStorage` (`event-live-fs-flow-width`), clamp 25–75
- Pointer handlers: set pointer capture, track clientX vs grid rect, set percent, persist on pointerup
- Cleanup listeners on unmount / pointerup

- [ ] **Step 3: CSS**

```css
.fs-grid {
  display: grid;
  grid-template-columns: minmax(0, var(--fs-flow-width, 52%)) 10px minmax(0, 1fr);
  /* was 1.1fr 1fr — replace with percent + handle + remainder */
}
.fs-split {
  cursor: col-resize;
  width: 10px;
  padding: 0;
  border: none;
  background: transparent;
  /* visible center line via ::before */
}
```

Respect Bluffet ink line; do not reintroduce dark-green fullscreen chrome.

- [ ] **Step 4: Run tests — expect PASS**

```bash
cd frontend && npm test -- --run src/views/EventLive.test.ts
```

- [ ] **Step 5: Commit if requested**

---

### Task 3: Smoke + redeploy note

- [ ] Confirm handle only appears under `[data-testid="fullscreen-rotator"]`
- [ ] If local Docker prod frontend is in use, rebuild frontend image so localhost picks up the change (hard refresh)
