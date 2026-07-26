# Proxmark Reader GUI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship `reader-gui.exe` — Fyne Windows operator panel that runs device-bridge in-process with offline-capable bib manual entry.

**Architecture:** Extract `cmd/device-bridge` into `internal/bridgeapp` (start/stop/status/poll/WS/local HTTP). Headless `device-bridge` and new `cmd/reader-gui` both use it. Manual entry + config + COM helpers live under `internal/bridge` / `bridgeapp`.

**Tech Stack:** Go 1.22+, Fyne v2, existing `internal/bridge` + `internal/rfid`, Testify.

---

### Task 1: Config file load/save

**Files:**
- Create: `backend/internal/bridgeapp/config.go`
- Create: `backend/internal/bridgeapp/config_test.go`

- [x] **Step 1:** Failing tests for JSON round-trip + env override
- [x] **Step 2:** Implement `Config`, `LoadConfig(path)`, `SaveConfig(path)`, `ApplyEnv()`

### Task 2: Extract bridgeapp runtime from device-bridge

**Files:**
- Create: `backend/internal/bridgeapp/app.go` (moved logic)
- Create: `backend/internal/bridgeapp/app_test.go` (status snapshot / mock start smoke)
- Modify: `backend/cmd/device-bridge/main.go` → thin wrapper

- [x] **Step 1:** Move app into package with `New`, `Start(ctx)`, `Stop()`, `Status()`, `Running()`
- [x] **Step 2:** Keep device-bridge behavior identical; run existing bridge tests

### Task 3: Manual entry client + roster cache

**Files:**
- Create: `backend/internal/bridge/manual_entry.go`
- Create: `backend/internal/bridge/manual_entry_test.go`
- Create: `backend/internal/bridge/roster.go`
- Create: `backend/internal/bridge/roster_test.go`

- [x] **Step 1:** Tests for online POST + offline enqueue via bib→UUID
- [x] **Step 2:** Implement `PostManualEntry`, `RosterCache`, `EnsureBearer` (PIN exchange when needed)

### Task 4: COM port list + Proxmark test helper

**Files:**
- Create: `backend/internal/bridgeapp/serial_windows.go` / `serial_stub.go`
- Create: `backend/internal/bridgeapp/serial_test.go`

- [x] **Step 1:** `ListSerialPorts()` Windows implementation
- [x] **Step 2:** `TestProxmark(cfg)` wraps a short CLI probe

### Task 5: Fyne reader-gui

**Files:**
- Create: `backend/cmd/reader-gui/main.go`
- Create: `backend/cmd/reader-gui/ui.go`

- [x] **Step 1:** Wire config form, start/stop, status ticker, manual entry
- [x] **Step 2:** `go build -o reader-gui.exe ./cmd/reader-gui`

### Task 6: Docs + verify

**Files:**
- Modify: `docs/production-reader.md`
- Modify: `backend/cmd/device-bridge/README.md` (pointer to GUI)

- [x] **Step 1:** Document GUI as primary race-day path
- [x] **Step 2:** Run `go test ./internal/bridge/... ./internal/bridgeapp/...` and build both exes
