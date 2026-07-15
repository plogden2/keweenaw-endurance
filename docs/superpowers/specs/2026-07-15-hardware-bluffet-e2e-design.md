# Hardware East Bluffet e2e dress rehearsal — design

**Date:** 2026-07-15  
**Status:** Approved — implementation plan at `docs/superpowers/plans/2026-07-15-hardware-bluffet-e2e.md`  
**Prerequisite:** Proxmark driver + logical tag UUID (`docs/superpowers/plans/2026-07-15-proxmark3-tag-uuid.md`)  
**Feature context:** Speckit `002-rfid-race-scanner` / All You Can East Bluffet  
**Approach:** Scripted Playwright harness + Cursor monitoring agent (Approach 1)

## Goal

Run a full wall-clock, Proxmark3-backed dress rehearsal of a compressed All You Can East Bluffet event. A Cursor agent starts the run and wakes every minute to monitor health and track issues. The suite proves race-day flows under realistic lap traffic, late registration, DNFs, reader browser crash, and a temporary client→API outage — then iterates (fix → commit → restart) until a clean run produces no bugs, limitations, or improvement ideas.

## Decisions

| Topic | Choice |
|---|---|
| Hardware | Real Proxmark3 only (no mock inject). Agent/harness fully controls write + read. |
| Tag identity | Logical RFID **UUID** written into tag user memory (not silicon UID). Per-racer UUID is permanent; dress rehearsal rewrites that UUID onto the single physical chip before each lap. |
| Spectators | Two Playwright contexts: desktop laptop + iPhone 13 device profile (emulation, not a physical phone). |
| Last-minute signups | **3** total: **2** immediately before start, **1** at **T+2:00**. Extra on top of 100 pre-registered. |
| No-shows | **9** of the 100 pre-registered never get a first lap. |
| Mid-race DNFs | ~**10** racers drop out (removed from lap rotation). |
| Lap model | Randomized “real-world” intervals: mean ~**1 min**, clamped **30s–3 min** per racer. |
| Race schedule | Bluffet-like: **30 min** + **15 min** share start at **T+0**; **5 min** race starts at **T+20**. |
| Suite clock | Always set shared 30/15 `start_time` to **now + 2 minutes** when the harness starts. |
| API outage | Simulate **clients unable to hit the API** for ~**5 minutes**. Reader keeps scoring on local Postgres; spectators go stale; on restore, sync catch-up is visible to spectators. |
| Reader crash | Close reader browser/context mid-race; reopen; confirm no scored data loss; catch up missed taps via **manual entry**. |
| Video | Playwright records **3×1080p** contexts; post-run **ffmpeg** composes one **1440p** side-by-side (Reader \| Laptop \| iPhone). |
| Control model | Harness owns timeline + hardware; agent **starts** the test and **monitors every 1 minute**. |
| Iterate policy | **Critical** → stop early, fix, commit, restart. **Minor** → finish run, address all issues/suggestions, commit, restart. Repeat until a run ends with an empty issues list. |

## Architecture

```text
┌─────────────────────────────────────────────────────────────┐
│ Cursor monitoring agent                                      │
│  - Entrypoint: “start the East Bluffet e2e test”             │
│  - Wake every 60s → read status.json + issues                │
│  - Critical stop / minor note / post-run fix+restart         │
└───────────────────────────┬─────────────────────────────────┘
                            │ launches / supervises
┌───────────────────────────▼─────────────────────────────────┐
│ Playwright hardware harness                                  │
│  contexts: reader | spectator-laptop | spectator-iphone13    │
│  drivers: Proxmark write→read, chaos, late signup, DNF       │
│  artifacts: videos, screenshots, status.json, issues.md      │
└───────┬─────────────────┬───────────────────┬───────────────┘
        │                 │                   │
   local stack      Proxmark3 USB        hosted/API path
   (Vue+Go+PG)      (single physical     (blocked 5 min
                    tag rewrite/read)     during outage)
```

### Components

1. **Compressed Bluffet seed** — Generator/variant of `database/seed/generate_bluffet_seed.py` (or a dedicated hardware-e2e seed) keeping AYCEB structure (categories, 100 racers, 3 races) with durations **30 / 15 / 5** minutes. Deterministic IDs preferred so fixtures stay stable; start times overwritten at harness boot to **now+2m** (kids/5‑min at **T+20**).

2. **Harness package** — `frontend/e2e/hardware-bluffet/` (config, timeline, Proxmark helper, spectator scripts, chaos, artifact writers). Separate from the existing mock-inject CI specs under `frontend/e2e/*.spec.ts`.

3. **Status channel** — `e2e-artifacts/bluffet-hardware/<run-id>/status.json` updated continuously (phase, wall clocks, laps scored, pending sync, last Proxmark op, active chaos flags). `issues.jsonl` + rendered `issues.md` for the agent and humans.

4. **Proxmark3 path** — Prerequisite plan delivers CLI hardware + logical UUID model. This harness: `WriteTag(participant_id)` (programs that racer’s permanent logical UUID onto the chip) → wait for authentic Poll/WS read → score. Silicon UID is never the lookup key.

5. **Video pipeline** — Playwright `video` at 1920×1080 per context → three files → ffmpeg compose to 1440p side-by-side with labels.

## Timeline

| When | What |
|---|---|
| **T−2:00** | Boot/verify stack; load seed; set start times; arm finish station; open 3 contexts; start recordings; pre-program/verify tag write path for roster. |
| **T−0:30 → T−0:05** | 2 last-minute signups; add to rotation; program tag when their first lap is due. |
| **T+0** | 30‑ and 15‑min races auto-start. Lap engine + reader page rotation + spectator churn begin. 9 no-shows never scored. |
| **T+2:00** | 1 last-minute signup; join rotation. |
| **During** | ~10 DNFs; reader crash + manual entry recovery; 5‑min client→API outage + sync catch-up; spectators search/track 5 friends each with multiple page/search changes. |
| **T+20** | 5‑min race starts. |
| **T+25 / T+30** | Races end as durations elapse; stop videos; ffmpeg compose; finalize issues + status. |

### Lap engine

- Maintain an active roster (100 − 9 no-shows + late adds − DNFs).
- For each active racer, sample next-lap delay from a distribution with mean ~60s, clamped to [30s, 180s].
- On due lap: `POST /api/rfid/write-tag` with `{ participant_id }` only (overwrites chip user memory with that racer’s permanent logical UUID) → await real Proxmark read of that UUID (timeout → issue) → assert UI/API lap increment.
- Serialize on the single physical chip (one write→read in flight). No DB reassignment of silicon UIDs.

### Reader UI schedule

- Bias toward carousel / fullscreen rotator.
- Periodically visit live, racers, and station routes so continuous background read across SPA routes is exercised.
- Start on countdown before T+0.

### Spectator behavior

- Each picks 5 “friends” from the seeded field.
- Throughout the race: change searches, open leaderboards / race tabs / racer-focused views multiple times.
- During API outage: assert no fresh updates.
- After restore: assert missed history appears (laps from the outage window visible).

### Chaos: reader crash

- Close reader browser context (or equivalent hard stop).
- Reopen reader, re-auth/re-arm if required.
- Confirm previously scored laps remain.
- Use manual entry UI/API to enter taps missed while down; verify they appear quickly and correctly.

### Chaos: API outage (~5 minutes)

- Block Playwright routes / client networking so browsers cannot reach the API (spectators stale; reader sync to hosted/API path blocked as applicable).
- Reader continues local scoring.
- Restore connectivity; assert automatic sync and spectator catch-up of missed history.

## Agent contract

**Start command (natural language):** “start the East Bluffet e2e test” (or equivalent).

**Agent responsibilities:**

1. Ensure Docker stack + seed + hardware mode prerequisites.
2. Launch the Playwright hardware harness.
3. Every **60 seconds**: read `status.json` and open issues; summarize health; classify new problems as critical vs minor.
4. On **critical**: stop harness early, implement fix, **commit**, restart full test.
5. On **minor only**: let the run finish; then address all listed bugs/limitations/ideas, **commit**, restart.
6. Exit when a full run completes with **zero** remaining issues/improvements from that run.

**Harness responsibilities:** own the deterministic timeline, Proxmark sequencing, chaos windows, assertions, screenshots, and video artifacts so the agent does not invent mid-race taps ad hoc.

## Artifacts

Per run under `e2e-artifacts/bluffet-hardware/<run-id>/`:

| Artifact | Purpose |
|---|---|
| `status.json` | Live phase / metrics for 1‑minute agent polls |
| `issues.jsonl` + `issues.md` | Timestamped bugs, limitations, improvement ideas with screenshot paths and context |
| `reader.webm`, `spectator-laptop.webm`, `spectator-iphone.webm` | 1080p source videos |
| `side-by-side-1440p.mp4` | Composed deliverable |
| `run.log` | Harness stdout/stderr |

Issue entries must be list-formatted with relevant details, screenshots, and context (phase, clocks, racer/tag ids when applicable).

## Non-goals

- Replacing the existing mock-inject CI Playwright suite.
- Physical iPhone automation.
- Running the full 12h/6h/90m production Bluffet durations.
- Unsupervised infinite agent creativity mid-race (timeline stays harness-owned).

## Success criteria

1. Full timeline executes on real Proxmark3 with randomized lap traffic as specified.
2. Late signups, no-shows, and DNFs behave correctly in UI and scoring.
3. Reader crash recovery loses no committed data; missed taps are manually enterable quickly.
4. 5‑minute API outage leaves spectators stale while reader continues; restore syncs history automatically.
5. Three 1080p videos + one 1440p side-by-side artifact produced.
6. Monitoring agent can start the run and assess health every minute from status/issues.
7. Iterate until a clean run reports no bugs, limitations, or improvement ideas.

## Open implementation notes (for the plan, not TBD product decisions)

- Prefer a generator flag `--durations=30,15,5` that keeps fixture event/race UUIDs; start times overwritten at runtime.
- Exact Playwright `context.setOffline` for “client cannot hit API” (local Docker with empty `HOSTED_API_URL`).
- Proxmark CLI command strings and timeouts live in the **Proxmark prerequisite plan**, tuned on the attached reader before this dress rehearsal runs.
