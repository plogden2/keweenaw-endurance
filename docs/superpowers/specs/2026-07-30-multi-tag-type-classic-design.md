# Multi Tag Type — NTAG/Ultralight + MIFARE Classic 1K

**Date**: 2026-07-30  
**Status**: Approved  
**Related**: RFID scanner (`002-rfid-race-scanner`), instant-tap-beep

## Problem

Finish reader/writer only uses `hf mfu` (NTAG / MIFARE Ultralight pages 4–7). Presenting a MIFARE Classic (or other) chip produces BCC/collision noise and opaque website 500s.

## Goal

Auto-detect ISO14443-A family and read/write the **same logical UUID** on:

1. **NTAG / MIFARE Ultralight** — existing path (`hf mfu` pages 4–7)
2. **MIFARE Classic 1K** — `hf mf` **block 1**, key `FFFFFFFFFFFF`

Unsupported types return a short operator message (no BCC dump as the primary error).

## Detection

Parse `hf 14a reader` (or arm select transcript) for SAK/ATQA:

| Classification | Heuristic (typical) |
|----------------|---------------------|
| Ultralight/NTAG | SAK `00` / Ultralight family (existing mfu path) |
| Classic 1K | SAK `08` (and not Ultralight) |
| Unsupported | Present but neither → clear error |

Prefer explicit SAK-based classification over try/fallback. Optional fallback: if classified Ultralight read fails empty and Classic block 1 has a UUID, accept Classic (defensive only).

## Storage

| Family | Proxmark commands | Location |
|--------|-------------------|----------|
| Ultralight/NTAG | `hf mfu rdbl/wrbl -b 4..7` | Pages 4–7 (16 bytes) |
| Classic 1K | `hf mf rdbl/wrbl --blk 1 -k FFFFFFFFFFFF` | Block 1 (16 bytes) |

Logical UUID encoding unchanged (`EncodeLogicalUUID` / `DecodeLogicalUUID`).

## Call sites

- **WriteLogicalUUID**: detect → dispatch write; multi-tag / unsupported → short error
- **Poll / ArmScan / Lua arm**: after card select, detect family → read with matching commands; emit same logical UUID
- **classifyProxmarkWriteError**: keep multi-tag messages; add unsupported-type message

## Non-goals

- DESFire, Classic 4K, custom sector keys
- Rewriting silicon UID / block 0
- Changing hosted DB identity model

## Tests

- Unit: SAK/ATQA → family classification
- Unit: Classic read/write command construction + parse of `hf mf rdbl` block dump
- Unit: Write/Poll dispatch uses mfu vs mf based on detect stub
- Hardware (optional): Classic 1K + NTAG round-trip when `RFID_HARDWARE=true`
