# Inline Bib Tap Entry — Design

**Date:** 2026-07-31  
**Status:** Approved (advisor)  
**Surface:** `/events/:eventId/taps` (`EventTaps.vue`)

## Goal

Replace the Add-tap dialog with a PIN-gated inline bib field. Typing a bib and pressing Enter records a finish lap for that racer, optimized for rapid finish-line entry.

## Decisions

| Topic | Choice |
|-------|--------|
| Karaoke | Inline checkbox next to bib input; checked → `karaoke_bonus: true` on create |
| Bib → participant | Frontend-only: `GET` participants `?q=<bib>`, require exactly one exact `bib_number` match, then existing `POST` with `participant_id` |
| API | No backend change; keep `participant_id` create payload |
| Missing / ambiguous | Inline error under input; keep focus and select contents |
| Success | Clear input, keep focus, refresh taps table; optional brief “Recorded #N Name” that clears on next key or ~2s |
| PIN | Unlocked → always-visible bib input (no Add button). Locked → hide entry UI |
| Dialog | Delete `AddTapDialog.vue` and its unit tests |

## UX

When PIN-unlocked, above the taps table:

```
[ Bib number ________ ]  [ ] Karaoke  ← Enter submits
Lap #12 Alex Rivera   (ephemeral success)
Bib not found         (error)
```

- Enter with empty input: no-op (or ignore).
- While a submit is in flight: disable input (or ignore duplicate Enter).
- Void / restore / pagination unchanged.

## Data flow

1. User presses Enter with bib text (trim whitespace).
2. `eventParticipantsApi.list(eventId, { q: bib, limit: 20 })`.
3. Filter to `p.bib_number === bib` (string equality after trim).
4. If count ≠ 1 → set inline error, select input text, stop.
5. Else `eventTapsApi.create(eventId, { participant_id, karaoke_bonus: false })`.
6. On success → clear input, focus, set ephemeral success, `loadTaps()` (page 1 or current page — prefer current page so operator still sees recent rows if newest-first already shows them; reload page 1 if list is newest-first).
7. On API error → inline error message from existing error helper.

## Out of scope

- Custom timestamps / checkpoint picker
- Server-side `bib_number` on POST
- Bib conflicts across races (operator asserts none for Bluffet)

## Tests

- Unit: EventTaps — PIN hide/show input; Enter resolves exact bib and creates tap; 0 / many matches show errors; success clears and refreshes.
- Delete AddTapDialog unit tests.
- E2E: PIN → type bib → Enter → row appears → void/restore still work.

## Docs

Update `docs/production-reader.md` Manual entry section if it still describes the dialog.
