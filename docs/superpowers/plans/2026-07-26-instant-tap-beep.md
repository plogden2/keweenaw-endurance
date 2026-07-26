# Instant RFID Tap + Beep Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Proxmark taps near-instant via a persistent CLI session, beep locally on successful read, keep Mario Kart for scored laps.

**Architecture:** `CLIProxmarkReader` keeps one interactive `pm3` process; `Poll` runs `hf mfu rdbl -b 4`, parses 16 bytes, calls injectable `Beeper`. Session dies → lazy reconnect with backoff. Writes share the same session under the existing mutex.

**Tech Stack:** Go (`backend/internal/rfid`), Windows MessageBeep for default beeper, Testify.

**Spec:** [docs/superpowers/specs/2026-07-26-instant-tap-beep-design.md](../specs/2026-07-26-instant-tap-beep-design.md)

---

## File map

| File | Responsibility |
|------|----------------|
| `backend/internal/rfid/session.go` | `PM3Session` interface + process-backed interactive session |
| `backend/internal/rfid/session_test.go` | Fake session / reconnect tests |
| `backend/internal/rfid/beep.go` | `Beeper` + Windows/default beep |
| `backend/internal/rfid/beep_test.go` | Recording beeper |
| `backend/internal/rfid/cli_proxmark.go` | Persistent session Poll/Write; single rdbl; beep on success |
| `backend/internal/rfid/cli_proxmark_test.go` | Update command expectations + beep + 16-byte parse |
| `docs/production-reader.md` | Note instant tap + laptop beep (brief) |

---

### Task 1: Parse 16-byte single rdbl + beep interface (TDD)

**Files:** `cli_proxmark.go`, `cli_proxmark_test.go`, `beep.go`, `beep_test.go`

- [x] Failing tests: Poll issues `hf mfu rdbl -b 4`; parses `Data :` 16 bytes; parses four page rows; calls Beep once on success; no beep on empty
- [x] Implement parse helper + Beeper wiring (still via injected Runner until Task 2)
- [x] `go test ./internal/rfid/ -count=1` green for these cases

### Task 2: Session abstraction + fake reconnect

**Files:** `session.go`, `session_test.go`, wire into `CLIProxmarkReader`

- [x] Failing tests: Run waits for prompt; EOF clears session and next Poll recreates; backoff honored without holding mutex in test via fake clock/factory
- [x] Process session: start `-p PORT -f --incognito`, write cmd+`\n`, read until prompt regex
- [x] Wire Poll/Write through session when factory set; keep CLICommandRunner path for simple unit tests OR migrate all tests to session fake

### Task 3: Default Windows beeper + docs

**Files:** `beep_windows.go` / `beep_other.go`, `production-reader.md`

- [x] Default Beeper uses MessageBeep (Windows) / `\a` elsewhere
- [x] One-line ops note: taps should beep on the laptop immediately

### Task 4: Verify

- [x] `go test ./internal/rfid/ ./internal/bridgeapp/ -count=1`
- [x] Optional hardware smoke if COM free (second poll ~600µs)
