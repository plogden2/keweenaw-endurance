# Instant RFID Tap + Beep — Design

**Date**: 2026-07-26  
**Status**: Implemented (2026-07-26) — approach #2 + advisor amendments  
**Feature branch context**: `002-rfid-race-scanner` / reader-gui + device-bridge

## Problem

Finish-line taps require holding a tag on the Proxmark antenna for seconds. Root cause: `CLIProxmarkReader.Poll()` spawns a new Proxmark CLI process per poll and runs four `hf mfu rdbl` commands. Windows process + COM connect dominates latency. There is no immediate local “got it” sound; Mario Kart only plays in the browser after a scored lap over the network.

## Goals

1. Recognize a brief wave/tap near-instantly (well under 1s after session warm-up).
2. Play an immediate local beep when a tag UUID is successfully read.
3. Keep existing Mario Kart celebration when a lap is scored in the SPA.
4. Preserve write-tag reliability and exclusive COM access.

## Non-goals

- Changing server 1-minute cooldown or offline enqueue rules.
- Proxmark firmware changes.
- Replacing Mario Kart with the local beep.

## Decisions (user + advisor)

| Topic | Decision |
|-------|----------|
| Architecture | Persistent interactive Proxmark CLI session (stdin/stdout), not spawn-per-poll |
| Read command | Single `hf mfu rdbl -b 4` (Type-2 READ returns 16 bytes = pages 4–7) |
| Parse | Prefer 16-byte `Data : …` line; fallback to four labelled 4-byte page rows; require exactly 16 bytes |
| Proxmark beep | **Not available** on deployed RRG client (`hw help` has no beep). Use **laptop MessageBeep** on successful Poll instead |
| Browser sound | Unchanged Mario Kart on `result === 'lap'` |
| Online same-UUID debounce | Keep **2s** (faster polls would otherwise spam while tag rests on mat) |
| Offline debounce | Keep **60s** |
| Session start | `proxmark3 -p <port> -f --incognito` (no `-c`); wait for `(?:\[.*\]\s*)?pm3 -->` |
| Crash/reconnect | On EOF/exit/timeout: close, clear session, lazy reconnect next op with backoff 1s→15s cap; never sleep while holding mutex |
| Poll vs write | Keep `Poll` `TryLock`; writes take exclusive lock so programming is never delayed by a poll |
| Testing | Inject `PM3Session` / factory fakes; do not require real process in unit tests |

## Audio model (option C, amended)

```
Tap → Poll succeeds → laptop beep (immediate, offline-capable)
                   → bridge send / score
                   → browser Mario Kart (celebration, online UI)
```

Laptop beep implementation: short system beep (`MessageBeep` on Windows via syscall, or `\a` / small embedded WAV). Injectable `Beeper` interface for tests (no-op / record calls).

## Component shape

```go
type PM3Session interface {
    Run(ctx context.Context, command string) (stdout string, err error)
    Close() error
}

type PM3SessionFactory func(ctx context.Context) (PM3Session, error)

type Beeper interface {
    Beep()
}
```

`CLIProxmarkReader` owns:

- session factory + current session
- reconnect backoff state
- mutex (unchanged semantics)
- optional `Beeper` (default OS beeper when nil)

`Poll()`:

1. TryLock; if busy return `""`
2. Ensure session (reconnect with backoff if needed)
3. `Run(ctx, "hf mfu rdbl -b 4")` with ~2s command deadline
4. Parse 16 bytes → decode logical UUID
5. On success: `Beeper.Beep()` then return UUID
6. On session death: Close, clear, return error/empty for this tick

`WriteLogicalUUID()`:

1. Lock (blocking)
2. Ensure session
3. Run four `hf mfu wrbl` in one `Run` (or chained `; ` in one line) as today
4. Verify via single `rdbl -b 4` when needed

## Success criteria

- Unit tests cover: single-command poll string, 16-byte parse paths, beep on success only, session restart after EOF, TryLock skip during write.
- With hardware: wave-through tap registers without multi-second hold; beep heard at the laptop immediately; Mario Kart still plays on scored lap in Chrome.

## Advisor note

Upstream RRG Proxmark3 on this laptop has no `hw beep`. If a future firmware/client adds a verified device beep command, wire it behind the same `Beeper` (or a composite beeper) without changing Poll call sites.
