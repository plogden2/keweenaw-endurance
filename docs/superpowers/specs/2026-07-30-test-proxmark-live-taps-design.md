# Test Proxmark — Live Taps + Beep

**Date**: 2026-07-30  
**Status**: Approved (approach #1)  
**Related**: instant-tap-beep (`2026-07-26-instant-tap-beep-design.md`)

## Problem

**Test Proxmark** runs a one-shot `Poll()` and shows a dialog. It does not use continuous arm, does not update **Last tap**, and does not beep. Operators cannot verify antenna/audio before starting the bridge.

## Goal

Clicking **Test Proxmark** opens a live listen session: continuous Proxmark arm, beep on each decoded UUID, update Last tap / dialog text, **do not record or sync**. Stop via dialog dismiss.

## Behavior

| Topic | Decision |
|-------|----------|
| Trigger | **Test Proxmark** button |
| UI | Custom dialog: “Listening… wave a tag”; shows tap count + UUID; **Done** cancels |
| RF path | Same continuous `ArmScan` as finish bridge |
| Beep | One laptop beep per accepted UUID (reader internal beep off to avoid double-beep on Lua arm) |
| Debounce | 1s same-UUID (match bridge online cooldown) |
| Record/sync | Never |
| Bridge running | Error: stop bridge first (COM exclusive) |
| Mock mode | Error: disable mock to test hardware |
| Hardware off | Same clear error as today |

## Non-goals

- Changing bridge scoring / write-only / station arm
- Browser Mario Kart during test
- Auto-start bridge after test
