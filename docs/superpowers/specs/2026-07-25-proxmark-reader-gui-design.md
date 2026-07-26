# Proxmark Reader GUI — Design

**Date:** 2026-07-25  
**Status:** Approved via advisor + user “do not stop until complete”

## Goal

Ship a single Windows `.exe` operator panel that runs device-bridge logic in-process so race-day staff can start/stop the Proxmark bridge, see status, and record a missed lap by bib when hardware fails — without PowerShell env-var ceremony.

## Decisions

| Topic | Choice |
|-------|--------|
| Purpose | Operator control panel (not website replacement) |
| Toolkit | Fyne (pure Go UI → single portable `reader-gui.exe`) |
| Bridge | In-process reuse of bridge/rfid packages; keep headless `device-bridge` |
| Manual entry | Online: `POST /api/rfid/manual-entry` with PIN JWT; offline: bib→UUID roster cache → `LocalStore.EnqueueLap` |
| Style | Utilitarian race-ops panel; light Superior Forest accent |

## Architecture

```
reader-gui.exe
├── Fyne UI (config, start/stop, status, manual entry)
└── internal/bridgeapp (extracted from cmd/device-bridge)
    ├── internal/bridge (auth, local CSV, sync, WS helpers)
    └── internal/rfid (CLIProxmarkReader / MockReader)
```

- Local loopback `127.0.0.1:8091` (`/status`, `/write-tag`) unchanged for website tag programming.
- Config persisted to `{BRIDGE_DATA_DIR}/reader-gui-config.json`; env vars override on load.
- COM ports enumerated on Windows; editable always.

## UI (v1)

1. **Config:** HOSTED_API_URL, BRIDGE_TOKEN, ORGANIZER_PIN, DEVICE_ID, EVENT_ID, RACE_ID, CHECKPOINT_ID, PROXMARK3_PORT, PROXMARK3_CLI, BRIDGE_DATA_DIR, hardware/mock toggles.
2. **Controls:** Save config · Start · Stop · Test Proxmark.
3. **Status:** mode, connected, pending count, last sync, last read UUID/time, last error.
4. **Manual entry:** bib (required), timestamp (default now), race/checkpoint from config; submit online or queue offline.

## Out of scope (v1)

Tag programming UI, karaoke, leaderboards, station arming, add-racer, CSV recovery, macOS/Linux packaging.

## Success criteria

Operator double-clicks `reader-gui.exe`, configures once, starts bridge, sees `online_synced`, and can record a missed finish by bib online or offline (queues + flushes later).
