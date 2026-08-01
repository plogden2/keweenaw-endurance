# Fullscreen rotator live-page QR code

**Date:** 2026-08-01  
**Status:** Approved for planning  
**Surface:** Event live fullscreen rotator (`EventLive.vue`)

## Goal

Optionally show a QR code in the top-right of the fullscreen rotator so spectators can open the public live event page on their phones. Default off; toggle in rotator settings.

## Decisions (locked)

| Topic | Choice |
|-------|--------|
| Placement | Top-right corner; same region as Play / Settings / Exit |
| Controls vs QR | When QR is on, controls auto-hide after ~3s idle; QR stays. Pointer/keyboard (or hover the corner) reveals controls again |
| When QR off | Current behavior — controls always visible, no QR |
| Encoded URL | Current host: `${window.location.origin}/events/${eventId}/live` |
| Caption | Fixed: `view results at keweenawendurance.com` |
| Settings storage | Extend existing `RotatorSettings` in sessionStorage (`event-live-fs-rotator-settings`) |
| Default | `showQrCode: false` |
| QR library | `qrcode` npm package |

## Settings

```ts
interface RotatorSettings {
  dwellMs: number
  pages: RotatorPageConfig[]
  showQrCode: boolean // NEW, default false
}
```

- `normalizeSettings` / defaults treat missing or non-boolean `showQrCode` as `false`.
- Settings dialog checkbox: **Show QR code for live page** (near dwell seconds).
- Persist via existing `saveRotatorSettings` on change / Done.

## Layout

Top-right stack inside `.fs-root`:

1. `.fs-controls` — Play / Settings / Exit (existing)
2. QR block (when `showQrCode`) — canvas/img ~128px with light padding; caption under it

When controls are visible they sit above the QR. When idle-hidden, only the QR + caption remain in that corner.

Settings dialog open → keep controls visible (do not idle-hide while the dialog is open).

## Idle controls

- Idle window: **3000ms** without pointer or keyboard activity while fullscreen is open.
- Reset idle timer on pointerdown / pointermove / keydown inside `.fs-root`.
- Also show controls when pointer enters the top-right corner hit area (controls + QR region).
- When idle: hide controls (`opacity: 0`, `pointer-events: none`) so the QR is unobstructed; do not remove from a11y tree permanently — prefer visually hidden but keep focusable path via settings cog when revealed.
- When `showQrCode` is false: skip idle-hide entirely.

## Components

**`LiveEventQr.vue`** (presentational)

- Props: `url: string`
- Renders QR for `url` + caption `view results at keweenawendurance.com`
- `data-testid="rotator-live-qr"`

**URL helper** (e.g. `frontend/src/utils/liveEventUrl.ts`)

- `liveEventUrl(origin: string, eventId: string): string`
- Returns `{origin}/events/{eventId}/live` (trim trailing slash on origin)

**`useFullscreenRotator`**

- Expose `showQrCode` from settings + `setShowQrCode(enabled: boolean)` (or mutate via applySettings / dedicated setter that saves).

**`EventLive.vue`**

- Wire checkbox, idle visibility, mount `LiveEventQr` when `rotatorOpen && showQrCode`.

## Out of scope

- Configurable caption or custom URL base
- QR on non-fullscreen live tabs
- Changing celebration overlay placement
- Deep-link to a specific race tab

## Testing

1. Settings default `showQrCode === false`; normalize missing → false.
2. Toggle on → persists in sessionStorage; QR mounts with caption.
3. QR encodes `liveEventUrl(window.location.origin, eventId)`.
4. With QR on: controls become hidden after idle timeout; reappear on pointer activity.
5. With QR off: controls remain visible (no idle-hide).

## Implementation notes

- Prefer generating QR to a canvas or data URL in `onMounted` / `watch(url)`.
- Keep QR generation failure quiet (omit image, keep caption) — celebration board must not break.
- Do not block rotator open on QR render.
