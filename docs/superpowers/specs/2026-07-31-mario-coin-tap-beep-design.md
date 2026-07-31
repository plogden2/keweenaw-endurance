# Mario Coin Tap Beep — Design

**Date:** 2026-07-31  
**Status:** Implemented (2026-07-31)  
**Related:** instant-tap-beep (`2026-07-26-instant-tap-beep-design.md`)

## Goal

Replace the laptop RFID reader's synthesized tap tone with the Mario coin sound effect, with leading silence removed so the coin starts immediately on tap.

## Non-goals

- Changing the website scored-lap sound (`new-lap.mp3` / ScanPopup).
- Playing coin audio on non-Windows (keep terminal bell / existing `beep_other.go` behavior).
- Runtime-configurable sound files or a settings UI.

## Context

Today `PlayTapBeep` / `PrewarmTapBeep` in `backend/internal/rfid/beep_windows.go` synthesize a short 1200 Hz PCM WAV and play it via `winmm.PlaySound` (`SND_ASYNC|SND_MEMORY`). The RFID path must not block on audio.

Source asset: `frontend/src/assets/audio/Mario-coin-sound.mp3` (~1.33s, 44.1 kHz stereo MP3). Probe shows ~36 ms of leading silence before audible content.

Windows `PlaySound` with `SND_MEMORY` expects PCM WAV, not MP3.

## Approach

Trim leading silence from the MP3, convert once to mono 16-bit PCM WAV, embed the WAV in the reader binary with `go:embed`, and play that buffer instead of `synthBeepWAV`.

## Asset pipeline

1. Input: `frontend/src/assets/audio/Mario-coin-sound.mp3`
2. ffmpeg (or equivalent) once at build/prep time:
   - Remove leading silence (threshold ~2% FS / ~−34 dB; cut at first audible sample)
   - Trim trailing silence after the last audible sample (same threshold); keep a few ms of natural decay if present in the source
   - Convert to mono, 16-bit PCM, 22050 Hz WAV (matches current beep sample rate and keeps embed size small)
3. Output committed asset: `backend/internal/rfid/assets/tap-coin.wav`
4. Keep the MP3 in frontend assets as the source of truth for re-export; do not wire the MP3 into the Go reader.

## Runtime behavior

| Call site | Behavior |
|-----------|----------|
| `PlayTapBeep` | Async play of embedded coin WAV (goroutine, same as today) |
| `PrewarmTapBeep` | Load/embed buffer once; short silent PlaySound prime of winmm |
| Play failure | Fall back to `kernel32.Beep` (same as today) |
| Non-Windows | Unchanged (`\a` / no-op prewarm) |
| Write-only mode / beep disabled | Unchanged — no play when beep is off |
| `/beep` diagnostic + Test Proxmark | Uses `PlayTapBeep` → coin sound |

## Components

- `backend/internal/rfid/assets/tap-coin.wav` — trimmed PCM asset
- `backend/internal/rfid/beep_windows.go` — `go:embed` the WAV; drop synth as the primary path (synth may remain only if useful for the silent prime buffer, or use a tiny zero buffer)
- `Beeper` interface and call sites — no API change

## Testing

- Unit: ensure Windows build still compiles with embed; recording beeper tests unchanged (they inject `Beeper`, not WAV).
- Manual: wave a tag / hit Test Proxmark / `GET /beep` — coin starts with no perceptible lead-in silence; RFID path remains snappy (no sync wait on PlaySound).

## Docs

One-line update in `docs/production-reader.md`: laptop tap feedback is the coin sound (scored-lap Mario Kart on the site unchanged).
