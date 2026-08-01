# Rotator lap jump (live view)

## Goal

When a new lap is recorded on Event Live, if the fullscreen rotator is open and **not paused**, jump the rotator to that racer’s race so spectators see the update with the celebration.

## Behavior

1. **Trigger:** same path as lap celebration (`startCelebration`), including poll fallback. Duplicate-suppressed celebrations do not jump.
2. **Gate:** only when `rotatorOpen && rotatorPlaying` (enforced inside `jumpToRace`). When the rotator is playing, celebration eligibility expands to all rotator races (12h+6h) so an off-page lap can jump the display; when paused/closed, celebration stays limited to the currently visible race(s).
3. **Mode preference:** Team if the participant has a non-empty `team_id`, else Individual. Team preference may fall back to Individual if teams is disabled; Individual preference must **not** fall back to Team (leave the team page for solo racers). If the target page is unavailable, no-op.
4. **Dwell:** always reset the dwell timer after a successful jump (including when already on that page), so the page stays up a full dwell.
5. **Settings dialog open:** no jump (leave `pageIndex` alone); when settings close, normal scheduling resumes.
6. **90m / unknown races:** rotator only knows `12h`/`6h`; other race IDs are ignored.
7. **Team membership:** cache `participant_id → hasTeam` from the event/race participant roster (loaded with live data / refreshed opportunistically). `LapRecordedEvent` has no `team_id`.

## API

`useFullscreenRotator.jumpToRace(race: RotatorRaceKey, preferredMode: RotatorMode): boolean`

- No-op (return false) if closed, paused, or no active page matches race (with preferred then fallback mode).
- Sets `pageIndex` to the matching active page and calls `scheduleAdvance()` (resets dwell when playing and settings closed).
- Returns true when index was set (including already-current page).
