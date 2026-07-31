# Multi Tag Type Implementation Plan

> **For agentic workers:** Implement TDD; run `go test ./internal/rfid/`; rebuild reader-gui via `scripts/build-reader-gui.ps1` and copy to `%LOCALAPPDATA%\KeweenawReader\reader-gui.exe`. Do **not** commit unless asked.

**Spec:** `docs/superpowers/specs/2026-07-30-multi-tag-type-classic-design.md`

## Files

| File | Change |
|------|--------|
| `backend/internal/rfid/tag_family.go` | `TagFamily` enum + `ClassifyISO14443A(stdout) (TagFamily, error)` from SAK/ATQA |
| `backend/internal/rfid/tag_family_test.go` | Classification fixtures |
| `backend/internal/rfid/classic.go` | Classic block-1 read/write cmd builders + parse `hf mf rdbl` 16-byte dump |
| `backend/internal/rfid/classic_test.go` | Parse/build tests |
| `backend/internal/rfid/cli_proxmark.go` | Detect → dispatch in WriteLogicalUUID, Poll, ArmScan paths; detect helper |
| `backend/internal/rfid/cli_proxmark_test.go` | Dispatch tests with injected runner |
| `backend/internal/rfid/pm3_continuous_arm.lua` / `arm_lua.go` | After wait-for-card, branch mfu vs mf read based on SAK in select output |
| `backend/internal/rfid/classify` errors | Unsupported type message |

## Tasks

### Task 1: Classification + Classic I/O helpers (TDD)

1. `TagFamilyUltralight`, `TagFamilyClassic1K`, `TagFamilyUnsupported`, `TagFamilyNone`
2. Parse SAK hex from `hf 14a reader` stdout (e.g. `SAK: 00` / `SAK: 08`)
3. Classic: `classicReadBlock1Cmd`, `classicWriteBlock1Cmd(hex16)`, `parseClassicBlockDump(stdout) ([]byte, error)`
4. Default key `FFFFFFFFFFFF`, block 1

### Task 2: Wire WriteLogicalUUID + Poll

1. Before write/read: `hf 14a reader` (or reuse detect), classify
2. Ultralight → existing mfu pages 4–7
3. Classic → mf block 1
4. Unsupported/none → short errors (`unsupported tag type (SAK …) — use NTAG/Ultralight or Classic 1K`)
5. Keep multi-tag classification + retry behavior

### Task 3: Continuous arm / Lua

1. Update Lua (or post-process in Go from tap transcript) so Classic taps also return 16-byte UUID dumps parseable by existing/extended parsers
2. Prefer: include SAK in tap transcript; Go `parsePollUUID` / new parser picks family and extracts UUID
3. If Lua changes are heavy, acceptable MVP: after `hf 14a reader -w`, run family-specific read in Go (non-Lua arm path already does command chaining — extend that; for Lua, emit select info + run rdbl/wrbl in script per SAK)

### Task 4: Verify + ship binary

```
go test ./internal/rfid/ -count=1
powershell -File scripts/build-reader-gui.ps1
Copy to %LOCALAPPDATA%\KeweenawReader\reader-gui.exe (stop running reader-gui first)
```
