# Reader GUI Write-Only Mode — Design

**Date**: 2026-07-26  
**Status**: Implemented

## Behavior

- Toggle **Write-only mode** in reader-gui (`Config.WriteOnly` / `write_only`).
- When ON: Proxmark still polls; last tap shows name/bib/UUID; no bridge `read`, no offline enqueue, no beep; pending flush held; writes + manual entry still work.
- When OFF: normal recording resumes; any held pending laps flush if online.
- Live `App.SetWriteOnly` — no bridge restart required.
