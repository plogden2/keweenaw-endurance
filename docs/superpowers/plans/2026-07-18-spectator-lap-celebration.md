# Spectator Lap Celebration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Spectators on the public live view see a top-right kawaii +1 sparkle card when a lap is recorded on the race they are looking at, and—only when idle—highlight that racer on the race-flow chart and focus their leaderboard row for 3 seconds.

**Architecture:** A dedicated public WebSocket (`GET /api/events/{eventId}/live/stream`) fans out `lap_recorded` events from finish scans and karaoke bonuses via a new `LiveStreamHub`. `EventLive.vue` consumes the stream, always shows `LapCelebrationOverlay`, and applies race-flow highlight + leaderboard scroll only when `useSpectatorIdle` reports not busy. HTTP 2s polling remains the source of board numbers.

**Tech Stack:** Go (Gin + gorilla/websocket), Vue 3 + Vitest, existing Chart.js `RaceFlowChart`

**Spec:** `docs/superpowers/specs/2026-07-18-spectator-lap-celebration-design.md`

**Working copy:** Implement **directly on `main`**. Do **not** create a feature branch, worktree, or PR branch for this work.

**Note on commits:** Do not commit unless the user explicitly asks. Skip commit steps during execution unless requested.

---

## File map

| File | Responsibility |
|------|----------------|
| `backend/internal/services/live_stream_hub.go` | Per-event fan-out hub for `lap_recorded` |
| `backend/internal/services/live_stream_hub_test.go` | Hub subscribe/publish/drop tests |
| `backend/internal/services/services.go` | Wire `LiveStream` onto `Services` |
| `backend/internal/handlers/live_stream.go` | WS upgrade handler |
| `backend/internal/handlers/live_stream_test.go` | WS receives published lap |
| `backend/internal/handlers/rfid.go` | Publish after `ProcessEventScan` lap |
| `backend/internal/handlers/bridge.go` | Publish after bridge finish lap |
| `backend/internal/handlers/timing.go` | Publish after karaoke bonus |
| `backend/cmd/server/main.go` | Register `GET /events/:id/live/stream` |
| `backend/internal/handlers/handlers_test.go` | Register route in test router |
| `specs/002-rfid-race-scanner/contracts/api-rfid-scanner.md` | Document WS contract |
| `frontend/src/services/api.ts` | `LapRecordedEvent`, `eventLiveStreamUrl` |
| `frontend/src/composables/useEventLiveStream.ts` | WS client + reconnect |
| `frontend/src/composables/useEventLiveStream.spec.ts` | Stream composable tests |
| `frontend/src/composables/useSpectatorIdle.ts` | Busy/idle tracking |
| `frontend/src/composables/useSpectatorIdle.spec.ts` | Idle rule tests |
| `frontend/src/components/LapCelebrationOverlay.vue` | Top-right sparkle +1 card |
| `frontend/src/components/LapCelebrationOverlay.test.ts` | Overlay render tests |
| `frontend/src/components/RaceFlowChart.vue` | Expose `isLegendBusy` |
| `frontend/src/components/RaceFlowChart.test.ts` | Legend-busy tests |
| `frontend/src/views/EventLive.vue` | Wire stream, overlay, highlight, scroll |
| `frontend/src/views/EventLive.test.ts` | Celebration + idle gating tests |

---

### Task 1: LiveStreamHub (backend)

**Files:**
- Create: `backend/internal/services/live_stream_hub.go`
- Create: `backend/internal/services/live_stream_hub_test.go`
- Modify: `backend/internal/services/services.go`

- [ ] **Step 1: Write failing hub tests**

```go
// backend/internal/services/live_stream_hub_test.go
package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestLiveStreamHub_PublishDeliversToSubscriber(t *testing.T) {
	hub := NewLiveStreamHub()
	eventID := uuid.New()
	ch := hub.Subscribe(eventID, 4)

	ev := LapRecordedEvent{
		Type:            "lap_recorded",
		EventID:         eventID.String(),
		RaceID:          uuid.New().String(),
		ParticipantID:   uuid.New().String(),
		ParticipantName: "Alex Rivera",
		BibNumber:       "42",
		LapCount:        7,
		RecordedAt:      time.Now().UTC(),
	}
	hub.Publish(eventID, ev)

	select {
	case got := <-ch:
		require.Equal(t, "lap_recorded", got.Type)
		require.Equal(t, "Alex Rivera", got.ParticipantName)
		require.Equal(t, 7, got.LapCount)
	case <-time.After(time.Second):
		t.Fatal("expected event")
	}
}

func TestLiveStreamHub_DoesNotDeliverToOtherEvent(t *testing.T) {
	hub := NewLiveStreamHub()
	a, b := uuid.New(), uuid.New()
	ch := hub.Subscribe(a, 4)
	hub.Publish(b, LapRecordedEvent{Type: "lap_recorded", EventID: b.String()})

	select {
	case <-ch:
		t.Fatal("should not receive other event")
	case <-time.After(50 * time.Millisecond):
	}
}
```

- [ ] **Step 2: Run tests — expect FAIL (types missing)**

Run: `cd backend && go test ./internal/services/ -run LiveStreamHub -count=1`

Expected: compile failure / undefined `NewLiveStreamHub`

- [ ] **Step 3: Implement hub + wire into Services**

```go
// backend/internal/services/live_stream_hub.go
package services

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// LapRecordedEvent is the public live-stream payload for spectator celebrations.
type LapRecordedEvent struct {
	Type            string    `json:"type"`
	EventID         string    `json:"event_id"`
	RaceID          string    `json:"race_id"`
	ParticipantID   string    `json:"participant_id"`
	ParticipantName string    `json:"participant_name"`
	BibNumber       string    `json:"bib_number,omitempty"`
	LapCount        int       `json:"lap_count"`
	RecordedAt      time.Time `json:"recorded_at"`
}

// LiveStreamHub fans out lap_recorded events to per-event WebSocket subscribers.
type LiveStreamHub struct {
	mu   sync.Mutex
	subs map[uuid.UUID][]chan LapRecordedEvent
}

func NewLiveStreamHub() *LiveStreamHub {
	return &LiveStreamHub{subs: make(map[uuid.UUID][]chan LapRecordedEvent)}
}

func (h *LiveStreamHub) Subscribe(eventID uuid.UUID, buffer int) <-chan LapRecordedEvent {
	if buffer < 1 {
		buffer = 8
	}
	ch := make(chan LapRecordedEvent, buffer)
	h.mu.Lock()
	h.subs[eventID] = append(h.subs[eventID], ch)
	h.mu.Unlock()
	return ch
}

func (h *LiveStreamHub) Unsubscribe(eventID uuid.UUID, ch <-chan LapRecordedEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	list := h.subs[eventID]
	out := list[:0]
	for _, c := range list {
		if c != ch {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		delete(h.subs, eventID)
	} else {
		h.subs[eventID] = out
	}
}

func (h *LiveStreamHub) Publish(eventID uuid.UUID, ev LapRecordedEvent) {
	if ev.Type == "" {
		ev.Type = "lap_recorded"
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, ch := range h.subs[eventID] {
		select {
		case ch <- ev:
		default:
			// drop if slow — celebration is best-effort
		}
	}
}
```

In `services.go`, add field `LiveStream *LiveStreamHub` and set `LiveStream: NewLiveStreamHub()` in `NewServicesWithReader`.

- [ ] **Step 4: Run tests — expect PASS**

Run: `cd backend && go test ./internal/services/ -run LiveStreamHub -count=1`

---

### Task 2: Publish on lap + karaoke; WS handler + route

**Files:**
- Create: `backend/internal/handlers/live_stream.go`
- Create: `backend/internal/handlers/live_stream_test.go`
- Modify: `backend/internal/handlers/rfid.go` (`ProcessEventScan`)
- Modify: `backend/internal/handlers/bridge.go` (after successful finish lap)
- Modify: `backend/internal/handlers/timing.go` (`CreateKaraokeBonus`)
- Modify: `backend/cmd/server/main.go`
- Modify: `backend/internal/handlers/handlers_test.go` (register route)
- Modify: `specs/002-rfid-race-scanner/contracts/api-rfid-scanner.md`

- [ ] **Step 1: Add publish helper on Handlers**

In a small helper (can live in `live_stream.go`):

```go
func (h *Handlers) publishLapRecorded(eventID uuid.UUID, result *scan.ScanResult) {
	if h.services.LiveStream == nil || result == nil || result.Result != "lap" {
		return
	}
	if result.RaceID == nil || result.Participant == nil {
		return
	}
	h.services.LiveStream.Publish(eventID, services.LapRecordedEvent{
		Type:            "lap_recorded",
		EventID:         eventID.String(),
		RaceID:          result.RaceID.UUID().String(),
		ParticipantID:   result.Participant.ID.UUID().String(),
		ParticipantName: result.ParticipantName,
		BibNumber:       result.BibNumber,
		LapCount:        result.LapCount,
		RecordedAt:      time.Now().UTC(),
	})
}
```

Call `h.publishLapRecorded(eventID, result)` at end of `ProcessEventScan` (before JSON response, when result is lap) and in `bridge.go` after `ProcessScan` when `result.Result == "lap"` (alongside existing `PublishScanResult`).

For karaoke in `CreateKaraokeBonus`: after success, load participant + race from the bonus/`source` timing record (participant has `RaceID`), then:

```go
h.services.LiveStream.Publish(eventID, services.LapRecordedEvent{
	Type:            "lap_recorded",
	EventID:         eventID.String(),
	RaceID:          participant.RaceID.UUID().String(),
	ParticipantID:   participant.ID.UUID().String(),
	ParticipantName: strings.TrimSpace(participant.FirstName + " " + participant.LastName),
	BibNumber:       participant.BibNumber,
	LapCount:        result.LapCount,
	RecordedAt:      time.Now().UTC(),
})
```

Resolve `eventID` from `participant.Race.EventID` (preload Race if needed). Prefer extending `AddKaraokeBonus` return or a thin handler-side DB lookup — keep publish in the handler layer so ScanService stays free of hub coupling.

- [ ] **Step 2: WS handler**

```go
// backend/internal/handlers/live_stream.go
var liveStreamUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (h *Handlers) StreamEventLive(c *gin.Context) {
	eventID, err := h.resolveEventID(c.Param("id"))
	if err != nil {
		respondServiceError(c, err)
		return
	}
	conn, err := liveStreamUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	sub := h.services.LiveStream.Subscribe(eventID, 32)
	defer h.services.LiveStream.Unsubscribe(eventID, sub)

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case ev, ok := <-sub:
			if !ok {
				return
			}
			if err := conn.WriteJSON(ev); err != nil {
				return
			}
		}
	}
}
```

Mirror `CheckOrigin` style from `rfidUpgrader` in `rfid.go` (same policy).

- [ ] **Step 3: Register route**

In `main.go` under events group:

```go
events.GET("/:id/live/stream", handlers.StreamEventLive)
```

Also register in `handlers_test.go` test router next to `GetEventLive`.

- [ ] **Step 4: WS integration test**

Pattern after `TestRFIDStream_WebSocketReceivesInject`: dial `ws…/api/events/{id}/live/stream`, call `hub.Publish`, assert JSON `type=lap_recorded`.

- [ ] **Step 5: Document contract**

Add to `api-rfid-scanner.md` under Live / leaderboard:

```markdown
### GET `/api/events/{eventId}/live/stream` (WebSocket, public)

Server → client on scored lap / karaoke bonus lap bump:
```json
{
  "type": "lap_recorded",
  "event_id": "<uuid>",
  "race_id": "<uuid>",
  "participant_id": "<uuid>",
  "participant_name": "Alex Rivera",
  "bib_number": "42",
  "lap_count": 7,
  "recorded_at": "2026-07-18T16:00:00Z"
}
```
Not sent for cooldown, unknown_tag, test_read, or checkpoint-only results.
```

- [ ] **Step 6: Run backend tests**

Run: `cd backend && go test ./internal/handlers/ ./internal/services/ -count=1`

Expected: PASS

---

### Task 3: Frontend stream URL + `useEventLiveStream`

**Files:**
- Modify: `frontend/src/services/api.ts`
- Create: `frontend/src/composables/useEventLiveStream.ts`
- Create: `frontend/src/composables/useEventLiveStream.spec.ts`

- [ ] **Step 1: Types + URL helper**

```ts
export interface LapRecordedEvent {
  type: 'lap_recorded'
  event_id: string
  race_id: string
  participant_id: string
  participant_name: string
  bib_number?: string
  lap_count: number
  recorded_at: string
}

export function eventLiveStreamUrl(eventId: string, apiBase: string = baseURL): string {
  const path = `/api/events/${eventId}/live/stream`
  const trimmed = (apiBase || '').replace(/\/$/, '')
  if (!trimmed) {
    const proto = typeof window !== 'undefined' && window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = typeof window !== 'undefined' ? window.location.host : 'localhost:8080'
    return `${proto}//${host}${path}`
  }
  if (trimmed.startsWith('https://')) return `wss://${trimmed.slice(8)}${path}`
  if (trimmed.startsWith('http://')) return `ws://${trimmed.slice(7)}${path}`
  if (trimmed.startsWith('wss://') || trimmed.startsWith('ws://')) return `${trimmed}${path}`
  return `ws://${trimmed}${path}`
}
```

(Follow the same branching style as `rfidStreamUrl`.)

- [ ] **Step 2: Failing composable tests**

Use the same `MockWebSocket` pattern as `useReaderStation.spec.ts`:

- opens URL from `eventLiveStreamUrl(eventId)`
- parses `lap_recorded` into `lastLap` ref
- ignores malformed JSON
- closes on `stop()`
- reconnects after close when not intentionally stopped (optional: assert one reconnect timer)

- [ ] **Step 3: Implement composable**

```ts
// sketch — mirror useReaderStation reconnect style
export function useEventLiveStream(eventId: Ref<string> | ComputedRef<string>) {
  const lastLap = ref<LapRecordedEvent | null>(null)
  const connected = ref(false)
  let socket: WebSocket | null = null
  let intentionallyClosed = false
  let backoffMs = 500

  function start() { /* open WS, onmessage set lastLap when type===lap_recorded */ }
  function stop() { intentionallyClosed = true; socket?.close() }

  onMounted(start)
  onUnmounted(stop)
  watch(eventId, () => { stop(); intentionallyClosed = false; start() })

  return { lastLap, connected, start, stop }
}
```

Do **not** use a process-wide singleton (unlike reader station) — each live page instance owns its socket.

- [ ] **Step 4: Run tests**

Run: `cd frontend && npx vitest run src/composables/useEventLiveStream.spec.ts`

Expected: PASS

---

### Task 4: `useSpectatorIdle`

**Files:**
- Create: `frontend/src/composables/useSpectatorIdle.ts`
- Create: `frontend/src/composables/useSpectatorIdle.spec.ts`

- [ ] **Step 1: Failing tests**

```ts
it('is busy for 3s after pointerdown', async () => {
  vi.useFakeTimers()
  const { isBusy, noteInteraction } = useSpectatorIdle({
    legendBusy: ref(false),
    pageScrolledFromTop: ref(false),
  })
  expect(isBusy.value).toBe(false)
  noteInteraction()
  expect(isBusy.value).toBe(true)
  vi.advanceTimersByTime(2999)
  expect(isBusy.value).toBe(true)
  vi.advanceTimersByTime(2)
  expect(isBusy.value).toBe(false)
})

it('is busy when legendBusy or pageScrolledFromTop', () => {
  const legendBusy = ref(true)
  const pageScrolledFromTop = ref(false)
  const { isBusy } = useSpectatorIdle({ legendBusy, pageScrolledFromTop })
  expect(isBusy.value).toBe(true)
  legendBusy.value = false
  expect(isBusy.value).toBe(false)
  pageScrolledFromTop.value = true
  expect(isBusy.value).toBe(true)
})
```

- [ ] **Step 2: Implement**

```ts
export function useSpectatorIdle(opts: {
  legendBusy: Ref<boolean>
  pageScrolledFromTop: Ref<boolean>
  interactionWindowMs?: number
}) {
  const windowMs = opts.interactionWindowMs ?? 3000
  const lastInteractionAt = ref(0)
  const now = ref(Date.now())
  let tick: number | undefined

  function noteInteraction() {
    lastInteractionAt.value = Date.now()
    now.value = Date.now()
  }

  const isBusy = computed(() => {
    if (opts.legendBusy.value) return true
    if (opts.pageScrolledFromTop.value) return true
    return now.value - lastInteractionAt.value < windowMs
  })

  onMounted(() => {
    const bump = () => noteInteraction()
    window.addEventListener('pointerdown', bump, { passive: true })
    window.addEventListener('keydown', bump)
    tick = window.setInterval(() => { now.value = Date.now() }, 250)
    onUnmounted(() => {
      window.removeEventListener('pointerdown', bump)
      window.removeEventListener('keydown', bump)
      if (tick) clearInterval(tick)
    })
  })

  return { isBusy, noteInteraction }
}
```

- [ ] **Step 3: Run tests**

Run: `cd frontend && npx vitest run src/composables/useSpectatorIdle.spec.ts`

---

### Task 5: `RaceFlowChart` exposes `isLegendBusy`

**Files:**
- Modify: `frontend/src/components/RaceFlowChart.vue`
- Modify: `frontend/src/components/RaceFlowChart.test.ts`

- [ ] **Step 1: Add failing test**

Mount chart with fixtures; set search input value → assert exposed `isLegendBusy` is true. Clear search, deselect one status filter → true. Restore all filters, scroll `.legend-items` → true.

- [ ] **Step 2: Implement**

Add `legendItemsEl` ref on `.legend-items`, track `legendScrollTop` via `@scroll`.

```ts
const isLegendBusy = computed(() => {
  if (searchQuery.value.trim() !== '') return true
  if (legendScrollTop.value > 0) return true
  if (
    availableStatuses.value.length > 0 &&
    selectedStatuses.value.length < availableStatuses.value.length
  ) {
    return true
  }
  if (
    availableGenders.value.length > 0 &&
    selectedGenders.value.length < availableGenders.value.length
  ) {
    return true
  }
  if (
    availableAgeGroups.value.length > 0 &&
    selectedAgeGroups.value.length < availableAgeGroups.value.length
  ) {
    return true
  }
  return false
})
```

Add `isLegendBusy` to `defineExpose({ … })`.

- [ ] **Step 3: Run tests**

Run: `cd frontend && npx vitest run src/components/RaceFlowChart.test.ts`

---

### Task 6: `LapCelebrationOverlay`

**Files:**
- Create: `frontend/src/components/LapCelebrationOverlay.vue`
- Create: `frontend/src/components/LapCelebrationOverlay.test.ts`

- [ ] **Step 1: Failing test**

Mount with `name="Alex Rivera"` and `visible=true` → find `data-testid="lap-celebration"` containing name and `+1`. `visible=false` → not in DOM (or `aria-hidden` / `v-if`).

- [ ] **Step 2: Implement UI**

Top-right absolute overlay inside a `position: relative` live root; Bluffet tokens (`var(--bluffet-red)` etc. when theme active, with ink/cream fallbacks); slight rotate; sparkle characters; `pointer-events: none`; enter animation via CSS (`@keyframes` scale/pop ~300ms).

```vue
<div
  v-if="visible"
  class="lap-celebration"
  data-testid="lap-celebration"
  aria-live="polite"
>
  <div class="sparkles" aria-hidden="true">★ ♥ ✦</div>
  <p class="eyebrow">New lap</p>
  <p class="name">{{ name }}</p>
  <span class="plus-one">+1</span>
</div>
```

- [ ] **Step 3: Run tests**

Run: `cd frontend && npx vitest run src/components/LapCelebrationOverlay.test.ts`

---

### Task 7: Wire celebration into `EventLive`

**Files:**
- Modify: `frontend/src/views/EventLive.vue`
- Modify: `frontend/src/views/EventLive.test.ts`

- [ ] **Step 1: DOM hooks**

On every leaderboard `<tr>` (all tabs + rotator):

```html
<tr
  …
  data-testid="leaderboard-row"
  :data-participant-id="e.participant_id"
  :class="{ 'leaderboard-row--focus': focusParticipantId === e.participant_id }"
>
```

Wrap the live content root with `class="event-live"` already present — ensure `position: relative` for overlay.

Add `<LapCelebrationOverlay :visible="celebrationVisible" :name="celebrationName" />`.

Pass `:highlight-participant-id="highlightParticipantId"` into each visible `RaceFlowChart`. Keep template refs for charts that need `loadRecords` (at least the active tab’s chart + rotator chart).

- [ ] **Step 2: Visible race + stream watch**

```ts
const visibleRaceIds = computed(() => {
  if (rotatorOpen.value) {
    return race12.value?.id ? [race12.value.id] : []
  }
  if (activeTab.value === 'overlap') {
    return [race12.value?.id, race6.value?.id].filter(Boolean) as string[]
  }
  if (activeTab.value === '12h') return race12.value?.id ? [race12.value.id] : []
  if (activeTab.value === '6h') return race6.value?.id ? [race6.value.id] : []
  return race90.value?.id ? [race90.value.id] : []
})
```

(Today’s rotator UI only shows the 12h race — match that. If rotator later cycles races, update this computed.)

```ts
const { lastLap } = useEventLiveStream(eventId)
const highlightParticipantId = ref<string | undefined>()
const focusParticipantId = ref<string | undefined>()
const celebrationVisible = ref(false)
const celebrationName = ref('')
let celebrationTimer: number | undefined
let focusTimer: number | undefined

const legendBusy = ref(false)
// poll/chart ref: when active chart ref updates, legendBusy = chartRef.isLegendBusy
const pageScrolledFromTop = ref(false)
// on scroll of window + leaderboard containers: pageScrolledFromTop = scrollTop > 0

const { isBusy } = useSpectatorIdle({ legendBusy, pageScrolledFromTop })

watch(lastLap, (ev) => {
  if (!ev || ev.type !== 'lap_recorded') return
  if (!visibleRaceIds.value.includes(ev.race_id)) return
  startCelebration(ev)
})

function startCelebration(ev: LapRecordedEvent) {
  // latest-wins: clear prior timers
  if (celebrationTimer) clearTimeout(celebrationTimer)
  if (focusTimer) clearTimeout(focusTimer)

  celebrationName.value = ev.participant_name
  celebrationVisible.value = true
  celebrationTimer = window.setTimeout(() => {
    celebrationVisible.value = false
  }, 2500)

  void refreshFlowForRace(ev.race_id)

  if (isBusy.value) {
    highlightParticipantId.value = undefined
    focusParticipantId.value = undefined
    return
  }

  highlightParticipantId.value = ev.participant_id
  focusParticipantId.value = ev.participant_id
  nextTick(() => {
    const row = document.querySelector(
      `[data-testid="leaderboard-row"][data-participant-id="${ev.participant_id}"]`,
    )
    row?.scrollIntoView({ block: 'center', behavior: 'smooth' })
  })

  focusTimer = window.setTimeout(() => {
    highlightParticipantId.value = undefined
    focusParticipantId.value = undefined
    window.scrollTo({ top: 0, behavior: 'smooth' })
    // also reset any overflow:auto leaderboard panel scrollTop = 0
  }, 3000)
}
```

Cancel pending focus reset if the user scrolls during the focus window (`pageScrolledFromTop` becomes true → clear `focusTimer` and stop auto scroll-to-top).

Sync `legendBusy` from the active chart ref via `watchEffect` reading `chartRef.value?.isLegendBusy`.

- [ ] **Step 3: Unit tests in EventLive.test.ts**

Mock `useEventLiveStream` to expose a controllable `lastLap` ref.

1. Emit lap for visible race → overlay appears with name.
2. With `isBusy` forced true (mock idle) → overlay still appears; no `highlight-participant-id` passed / no focus class.
3. Emit second lap quickly → overlay name updates (latest-wins).
4. Lap for other race id → no overlay.

- [ ] **Step 4: Run frontend tests**

Run:

```bash
cd frontend && npx vitest run src/views/EventLive.test.ts src/components/LapCelebrationOverlay.test.ts src/composables/useEventLiveStream.spec.ts src/composables/useSpectatorIdle.spec.ts src/components/RaceFlowChart.test.ts
```

Expected: PASS

---

### Task 8: Smoke verification

- [ ] **Step 1: Backend package tests**

Run: `cd backend && go test ./internal/handlers/ ./internal/services/ -count=1`

- [ ] **Step 2: Frontend targeted vitest** (command in Task 7 Step 4)

- [ ] **Step 3: Manual / optional e2e note**

If hardware dress rehearsal is available: open spectator live view, ensure idle, inject a finish lap for the visible race → expect top-right sparkle card, line highlight, leaderboard focus ~3s then reset. With search in legend → card only.

No new Playwright spec required unless time allows; prefer Vitest coverage above.

---

## Spec coverage checklist

| Spec requirement | Task |
|------------------|------|
| Public WS `/live/stream` + `lap_recorded` | 1–2 |
| Publish on finish lap + karaoke | 2 |
| Contract docs | 2 |
| Top-right sparkle card always | 6–7 |
| Visible race = tab / rotator / overlap | 7 |
| Idle gating (legend/search/filter/scroll + page scroll + 3s interaction) | 4–5, 7 |
| Highlight + leaderboard 3s then reset | 7 |
| Latest-wins | 7 |
| Refresh chart records on celebration | 7 |
| No branch/worktree; work on main | Header constraint |
| Skip commits unless asked | Header constraint |

## Self-review notes

- Karaoke publish needs participant/race/event loaded in handler — explicit in Task 2.
- Rotator today only renders 12h; `visibleRaceIds` matches current UI (documented).
- `Unsubscribe` required so hubs do not leak channels after WS disconnect.
- Filter “narrowed” = selected length < available length (after `mergeFilterSelections` defaults to all).
