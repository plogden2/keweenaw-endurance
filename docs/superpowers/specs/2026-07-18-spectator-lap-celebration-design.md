# Spectator lap celebration (live view)

**Date:** 2026-07-18  
**Status:** Approved for planning  
**Surface:** Public event live view (`/events/:eventId/live` → `EventLive.vue`)

## Goal

When a new lap is recorded for a racer on the race currently visible to a spectator, show a kawaii +1 name celebration, and—only if the spectator is idle—highlight their race-flow line and briefly focus their leaderboard row.

## Decisions (locked)

| Topic | Choice |
|-------|--------|
| Delivery | Public WebSocket `GET /api/events/{eventId}/live/stream` (not poll-diff) |
| Visible race | Selected tab **or** currently shown fullscreen-rotator panel; overlap tab matches either visible race |
| Animation style | Sparkle name card (name-first, tilted +1 sticker, Bluffet red/cream/ink) |
| Card placement | Top-right overlay; no dimming scrim; `pointer-events: none` |
| Busy / idle | Busy if race-flow legend has non-empty search, narrowed filters (not all statuses/genders/age groups selected), or legend list scrolled from top; **or** page/leaderboard scrolled from top; **or** any click/touch/keydown in the last **3 seconds** |
| When busy | Still show sparkle card; **skip** race-flow highlight and leaderboard scroll |
| Rapid laps | **Latest wins** — replace in-flight celebration and restart timers |
| Karaoke bonus | Publish when a karaoke bonus increases scored lap count (same `lap_recorded` event) |
| Board data | Keep existing 2s HTTP live poll for leaderboard numbers; WS is for celebration timing only |

## Architecture

```
Finish scan / karaoke bonus (result bumps laps)
        ↓
  LiveStreamHub.Publish(eventId, lap_recorded)
        ↓
  WS clients on /api/events/{eventId}/live/stream
        ↓
  EventLive: race visible? → celebration state (latest-wins)
        ├─ always: LapCelebrationOverlay (top-right, ~2.5s)
        └─ if idle: highlightParticipantId + scroll leaderboard 3s → reset top + clear highlight
                    + RaceFlowChart.loadRecords() so the line includes the new lap
```

### WebSocket contract

**Endpoint:** `GET /api/events/{eventId}/live/stream` (public, no PIN)

**Server → client message:**

```json
{
  "type": "lap_recorded",
  "event_id": "<uuid>",
  "race_id": "<uuid>",
  "participant_id": "<uuid>",
  "participant_name": "Alex Rivera",
  "bib_number": "42",
  "lap_count": 7,
  "recorded_at": "2026-07-18T16:00:00-04:00"
}
```

Document in `specs/002-rfid-race-scanner/contracts/api-rfid-scanner.md`.

### Publish points (backend)

1. After finish-mode scan scoring with `result: "lap"`.
2. After karaoke bonus creation that increases the participant’s scored lap total for that race.

Do **not** publish for cooldown, unknown_tag, test_read, checkpoint-only, or non-scoring outcomes.

### Hub behavior

- Fan-out keyed by `event_id`.
- Drop messages for slow subscribers (non-blocking send); celebration is best-effort.
- Reconnect is client responsibility (backoff).

## Frontend design

### Composables

**`useEventLiveStream(eventId)`**
- Opens WS to the live stream URL derived from API base.
- Exposes latest `lap_recorded` (or a callback/event).
- Reconnects with exponential backoff while the live view is mounted.
- Tears down on unmount / eventId change.

**`useSpectatorIdle(options)`**
- Tracks:
  - last pointer/keyboard interaction timestamp (busy for 3s after)
  - whether page or leaderboard scroll container is away from top
  - race-flow legend busy signals from `RaceFlowChart` (`isLegendBusy`: search non-empty, **or** any of status/gender/age-group filters is a proper subset of available options, **or** legend list `scrollTop > 0`)
- Exposes `isBusy: ComputedRef<boolean>` (and optionally `isIdle`).

`RaceFlowChart` exposes `isLegendBusy` (computed or via template ref) so the parent can include legend search/filter/scroll in idle detection without scraping the DOM.

### Components

**`LapCelebrationOverlay.vue`**
- Props: `name`, `visible` (driven by celebration state).
- Top-right fixed/absolute within the live view; sparkle accents; Bluffet tokens.
- Auto-hides after ~2.5s via parent state (parent owns latest-wins timing).

**`EventLive.vue` wiring**
- Subscribe to stream while live data is shown.
- `visibleRaceIds`: from active tab (single race, or both on overlap) or rotator’s current panel race id.
- On matching `lap_recorded`:
  1. Set celebration payload (replace if one is active).
  2. Show overlay.
  3. If `!isBusy`: set `highlightParticipantId`, scroll matching `leaderboard-row` into view, hold **3 seconds**, then scroll leaderboard container back to top and clear highlight.
  4. Call `loadRecords` on the relevant `RaceFlowChart` ref(s) so the highlighted series includes the new lap.
- Leaderboard rows need a stable selector (e.g. `data-participant-id`) for scroll targeting.

### Timing

| Effect | Duration |
|--------|----------|
| Sparkle card visible | ~2.5s (may be cut short by latest-wins replace) |
| Highlight + leaderboard focus | 3s, then clear / scroll to top |
| Interaction “busy” window | 3s after last click/touch/keydown |

## Edge cases

| Case | Behavior |
|------|----------|
| WS disconnected | No celebration; HTTP poll still updates boards; no error toast spam |
| Lap for non-visible race | Ignore |
| Overlap tab | Celebrate if `race_id` is either shown race; highlight on the chart that owns that participant when identifiable; otherwise card-only |
| Participant not on current leaderboard DOM | Show card; skip scroll |
| Tab / rotator changes mid-celebration | Clear highlight/scroll focus if race no longer visible; overlay may finish its fade |
| User becomes busy during a focus window | Do not start new scroll/highlight; if already scrolling, cancel remaining auto-scroll reset only if user takes over scroll (prefer: cancel pending auto reset when user scrolls) |

## Testing

1. **Backend:** hub publish + WS client receives `lap_recorded` for lap and karaoke-bonus publish paths; non-lap results do not publish.
2. **Unit:** idle busy rules (legend / scroll / recent interaction); latest-wins replaces celebration state.
3. **Component:** overlay shows name and +1; EventLive applies highlight + scroll only when idle.
4. **Integration / light e2e:** inject or process a lap → spectator live view shows card (and focus when idle).

## Out of scope

- Changing reader `ScanPopup` / reader RFID stream behavior beyond publishing into the new hub.
- Replacing the 2s HTTP live poll with WS for full board payloads.
- Sound effects on the spectator board.
- Celebrations on admin `RaceDetails` or operator `LiveTiming` views.

## Implementation notes

- Prefer a dedicated `LiveStreamHub` (or similarly named service) rather than overloading the RFID tag-read subscriber bus, so public spectators do not receive raw `tag_read` / station traffic.
- Mirror existing Gin WebSocket upgrade patterns from `handlers/rfid.go` (origin checks consistent with public live pages).
