# Event Test Mode Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** PIN-gated Event Test Mode dialog that intercepts RFID/manual taps into an ephemeral combined race-flow + leaderboard without polluting production scores.

**Architecture:** Pinia `eventTestMode` store holds open state, roster, and in-memory taps. `useReaderStation` short-circuits before `enqueueScan` when active. Dialog reuses `RaceFlowChart` with external timing records (lap-based). Close clears store.

**Tech Stack:** Vue 3, Pinia, Vitest, Playwright, existing `raceFlowData` / `RaceFlowChart`

---

### Task 1: Store + flow utils (TDD)

- [x] Write failing tests for `eventTestMode` store (open/close, recordTap, leaderboard sort, clear, tag/bib resolve)
- [x] Write failing tests for `eventTestModeFlow` (synthetic TimingRecords + leaderboard rows)
- [x] Implement store + utils until green

### Task 2: Reader intercept (TDD)

- [x] Extend `useReaderStation.spec.ts`: when test mode open for event, tag_read does not call `enqueueScan` and records into store
- [x] Implement intercept in `useReaderStation.ts`
- [x] Suppress `ScanPopup` in `App.vue` when test mode open

### Task 3: Dialog + EventLive entry

- [x] Add optional external records/participants props to `RaceFlowChart`
- [x] Build `EventTestModeDialog.vue` (banner, feedback, bib form, chart, leaderboard, close confirm)
- [x] Wire Test mode button + dialog in `EventLive.vue`
- [x] Block `EventTaps` submit when test mode open for event
- [x] Component tests for dialog open/close discard

### Task 4: E2E + verify

- [x] Playwright: open test mode → inject/manual → assert no production lap pollution → close → empty
- [x] Run unit tests for touched packages; fix failures
