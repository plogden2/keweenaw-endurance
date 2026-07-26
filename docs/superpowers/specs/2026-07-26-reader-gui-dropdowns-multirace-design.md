# Reader GUI Dropdowns + Multi-Race + Tap Names — Design

**Date**: 2026-07-26  
**Status**: Implemented (2026-07-26) — approach #1; manual entry option C  
**Related**: instant-tap/beep (`2026-07-26-instant-tap-beep-design.md`), reader-gui (`2026-07-25-proxmark-reader-gui`)

## Goals

1. Event, race, and checkpoint are **dropdown selects** of available options (not UUID text fields).
2. Each tap shows **racer name alongside UUID** (and bib when known).
3. One finish reader can serve **all races in an event** at once.
4. Manual bib entry: **auto-resolve across event races**, with optional race override (option C).

## Non-goals

- Changing website Station arming UX (operators still arm finish station to the event once).
- Checkpoint-mode multi-race at one physical mat (finish-mode only for All-races).
- Redesigning Fyne visual theme.

## Behavior

### Config selects

| Control | Options source | Notes |
|---------|----------------|-------|
| Event | `GET /api/events` | Label = event name; value = event UUID |
| Race | `GET /api/races?event_id=` + **"All races (event finish)"** | All-races clears stored single race/checkpoint for RFID |
| Checkpoint | `GET /api/races/:id/checkpoints` | Only when a single race is selected; prefer finish checkpoint; hide/disable in All-races mode |

Offline / before first successful API: fall back to Bluffet seed options already in `bridgeapp`.

Persist selected IDs in `reader-gui-config.json` as today (`EventID`, `RaceID`, `CheckpointID`). All-races mode stores empty `RaceID`/`CheckpointID` (or a sentinel cleared on load — prefer empty).

### RFID multi-race

Server finish stations are already event-scoped and resolve the participant’s race from the tag. GUI All-races mode must not require a race UUID for the bridge to poll/send reads. Document: Station page on the website must be armed to the event in finish mode.

### Manual entry (C)

1. If race override selected → resolve bib in that race only (current behavior).
2. Else → search event-wide roster;  
   - 1 match → record lap for that race/UUID  
   - 0 → error “unknown bib”  
   - 2+ → error “ambiguous bib — select a race”

### Tap display

Status line / dedicated last-tap label:

```
{First} {Last} · bib {n} · {uuid}
```

Plus race name when known. Prefer hosted `scan_result` when online; else roster lookup by UUID.

## Components

| Area | Change |
|------|--------|
| `bridge/roster.go` | Event-wide entries: bib, name, race_id, race_name, logical_uuid |
| `bridgeapp` | Refresh roster for all races in event; ManualEntry auto-resolve; Status last-tap fields (name, bib, uuid, race) |
| `bridgeapp` catalog | Fetch events/races/checkpoints for GUI (reuse autofill HTTP) |
| `cmd/reader-gui/ui.go` | Selects instead of UUID entries; last-tap label; race override for manual |
| Tests | Roster ambiguity; All-races config; tap status fields |

## Success criteria

- Operator never pastes UUIDs for event/race/checkpoint when online.
- Wave tap shows name + UUID in the GUI within a second of scoring (online).
- With All races selected and station armed to event, tags from 12h/6h/kids all score without changing GUI race.
- Manual bib with duplicate numbers across races prompts for race override.
