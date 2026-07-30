# Proxmark HF Antenna Gain — Design

**Date**: 2026-07-30  
**Status**: Approved (approach #1)  
**Related**: reader-gui (`2026-07-25-proxmark-reader-gui`), instant-tap (`2026-07-26-instant-tap-beep`)

## Problem

Finish-line operators need a way to increase Proxmark HF read sensitivity from the reader GUI. Proxmark has no true HF “gain” dial; ISO14443-A detection sensitivity is controlled by `hw sethfthresh -t <1–63>` (lower = more sensitive; firmware default 7).

## Decision

Expose an operator-facing **Gain** control (1–63, default **63** = max sensitivity) that maps to:

```
threshold = 64 - gain
```

Command applied to the device:

```
hw sethfthresh -t <threshold>
```

(`-i` / `-l` left at Proxmark defaults.)

## Behavior

| Case | Behavior |
|------|----------|
| Fresh install / missing config | `hf_gain = 63` |
| Bridge start (hardware reader) | Apply threshold once before polling |
| Session reconnect | Re-apply threshold after new session opens |
| One-shot Windows CLI mode | Apply once on first hardware use / when gain changes (device keeps runtime thresh until reset) |
| GUI change while bridge running | Apply immediately + persist config (same pattern as write-only) |
| GUI change while stopped | Persist only; apply on next start |
| Mock reader | No-op |

## Components

1. **`bridgeapp.Config`**: `HFGain int` JSON `hf_gain`; normalize clamp 1–63; default 63; env `PROXMARK3_HF_GAIN`.
2. **`CLIProxmarkReader`**: store gain; `SetHFGain(gain int)`; `HFThreshFromGain`; apply via `hw sethfthresh -t N` on ensure-session / first one-shot apply / SetHFGain.
3. **`bridgeapp.App`**: pass gain into reader; `SetHFGain` live update.
4. **`reader-gui`**: slider (or Fyne equivalent) labeled “HF gain” next to Proxmark controls; default max; on change save + live apply if running.

## Testing

- Unit: gain↔threshold mapping; config default/round-trip/env; reader issues correct `hw sethfthresh` command on construct path / SetHFGain; mock unchanged.
- Manual: GUI shows 63; Start bridge; lower gain; taps still work; restart restores saved gain.

## Out of scope

- LF antenna settings  
- Proxmark firmware changes  
- Changing sniff (`-i`) or Legic (`-l`) thresholds  
