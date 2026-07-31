# Inline Bib Tap Entry Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace Event Taps Add-tap dialog with a PIN-gated inline bib field that records a lap on Enter.

**Architecture:** Frontend-only. Resolve bib via `eventParticipantsApi.list` exact `bib_number` match, then existing `eventTapsApi.create({ participant_id, karaoke_bonus: false })`. Delete `AddTapDialog`.

**Tech Stack:** Vue 3, Pinia (`pinAuth`), Vitest, Playwright, existing event taps/participants API clients.

**Spec:** `docs/superpowers/specs/2026-07-31-inline-bib-tap-entry-design.md`

---

## File map

| File | Action |
|------|--------|
| `frontend/src/views/EventTaps.vue` | Inline bib form; remove dialog |
| `frontend/src/views/EventTaps.test.ts` | Rewrite add-flow tests |
| `frontend/src/components/AddTapDialog.vue` | Delete |
| `frontend/src/components/AddTapDialog.test.ts` | Delete |
| `frontend/e2e/event-taps.spec.ts` | Inline Enter flow |
| `docs/production-reader.md` | Update Manual entry copy if needed |

---

### Task 1: Failing EventTaps unit tests for inline entry

**Files:**
- Modify: `frontend/src/views/EventTaps.test.ts`

- [ ] **Step 1:** Remove tests that open `AddTapDialog` / click `add-tap-btn` expecting a dialog.
- [ ] **Step 2:** Add tests (PIN unlocked):
  - `data-testid="inline-bib-input"` is visible; locked → not present
  - Typing bib + Enter calls participants list then `eventTapsApi.create` with matched `participant_id` and `karaoke_bonus: false`
  - Zero exact matches → shows `data-testid="inline-bib-error"` with not-found message; create not called
  - Multiple exact matches → error; create not called
  - Success → input cleared, `list` taps refreshed
- [ ] **Step 3:** Mock `eventParticipantsApi.list` and `eventTapsApi.create` as needed.
- [ ] **Step 4:** Run `npx vitest run src/views/EventTaps.test.ts` — expect failures until Task 2.
- [ ] **Step 5:** Commit: `test(event-taps): expect inline bib Enter flow`

---

### Task 2: Implement inline bib entry in EventTaps.vue

**Files:**
- Modify: `frontend/src/views/EventTaps.vue`
- Delete: `frontend/src/components/AddTapDialog.vue`

- [ ] **Step 1:** Remove `showAddDialog`, Add tap button, and `<AddTapDialog>` usage/import.
- [ ] **Step 2:** When `pinAuth.isAuthenticated`, render toolbar with:
  - `<input data-testid="inline-bib-input" …>` bound to bib string
  - `@keydown.enter.prevent="submitBib"`
  - Optional ephemeral success `data-testid="inline-bib-success"`
  - Error `data-testid="inline-bib-error"`
- [ ] **Step 3:** Implement `submitBib`:
  1. Trim bib; if empty return
  2. Guard re-entry while `submitting`
  3. `eventParticipantsApi.list(eventId, { q: bib, limit: 20 })`
  4. Exact filter `String(p.bib_number) === bib`
  5. 0 → error “Bib not found”; select input; return
  6. >1 → error “Multiple matches”; select input; return
  7. `eventTapsApi.create(eventId, { participant_id: match.id, karaoke_bonus: false })`
  8. Success → clear bib, focus input, set success text briefly, `loadTaps()`
  9. Catch → inline error via `getErrorMessage`
- [ ] **Step 4:** Run `npx vitest run src/views/EventTaps.test.ts` — all pass.
- [ ] **Step 5:** Commit: `feat(event-taps): inline bib Enter records lap`

---

### Task 3: Remove AddTapDialog tests and dead code

**Files:**
- Delete: `frontend/src/components/AddTapDialog.test.ts`
- Grep for remaining `AddTapDialog` imports/refs and remove.

- [ ] **Step 1:** Delete `AddTapDialog.test.ts` (and `.vue` if not deleted in Task 2).
- [ ] **Step 2:** `rg AddTapDialog frontend` — zero hits (except docs/history if any).
- [ ] **Step 3:** Commit: `chore(event-taps): remove AddTapDialog`

---

### Task 4: E2E + production-reader docs

**Files:**
- Modify: `frontend/e2e/event-taps.spec.ts`
- Modify: `docs/production-reader.md` (Manual entry section)

- [ ] **Step 1:** Rewrite e2e: PIN unlock → fill `inline-bib-input` with a known Bluffet bib → Enter → expect row in `event-taps-table` → void → restore. Drop dialog selectors.
- [ ] **Step 2:** Update production-reader Manual entry to describe bib + Enter (no dialog/karaoke on this page).
- [ ] **Step 3:** Commit: `test(e2e): inline bib taps; docs: manual entry`

---

### Task 5: Verification

- [ ] **Step 1:** `cd frontend && npx vitest run src/views/EventTaps.test.ts`
- [ ] **Step 2:** Confirm no `AddTapDialog` references in `frontend/src`.
- [ ] **Step 3:** Self-check against design doc checklist (PIN gate, no karaoke, exact match, errors, success clear+focus).
