# GCP production deploy + prod-like Bluffet dress rehearsal — design

**Date:** 2026-07-16  
**Status:** Implemented — prod-like dress rehearsal green (`e2e-artifacts/bluffet-hardware/2026-07-17T01-21-24-591Z`, 77 laps, exit 0); Syncing-chip hold + compose retry pending on main  

**Domain:** `keweenawendurance.com`  
**Related:** `docs/superpowers/specs/2026-07-15-hardware-bluffet-e2e-design.md`  
**Feature context:** Speckit `002-rfid-race-scanner` / All You Can East Bluffet  
**Supersedes:** Speckit plan storage decision that “per-station Docker Postgres is authority + HOSTED_API_URL sync.” This design makes **hosted Postgres the sole online authority**; the reader laptop’s **device-bridge** owns Proxmark + a **local live CSV** for offline continuity with **automatic sync** when connectivity returns.

## Goal

1. Get the repo **ready to deploy** to Google Cloud Platform at `keweenawendurance.com`.
2. Update the East Bluffet hardware dress rehearsal to **mimic the production topology** (including the on-laptop Proxmark device-bridge).
3. Add a **`--prod` flag** so the harness can target real production when live.
4. Run the dress rehearsal against a **local prod-like stack** (not real GCP today).

## Decisions (locked with user)

| Topic | Choice |
|---|---|
| Topology | **Fully hosted online authority** — Cloud Run–style origin for UI + API + Cloud SQL |
| Proxmark / RFID | **Local device-bridge** on the reader laptop (owns `pm3`; relays to hosted when online) |
| Offline resilience | **Keep scoring offline into local live CSV**, then **auto-sync to hosted DB** when back online — **no manual intervention** |
| Operator UX | Status indicator: **Offline** → **Syncing** → **Online · Synced** (reader UI) |
| Today's rehearsal | **Local prod-like Compose** + device-bridge path |
| API outage chaos | Real continuity: bridge keeps scoring to CSV; on restore auto-sync; assert UI status + spectator catch-up |
| Approach | Prod-like Compose + GCP deploy kit + harness `--prod` |
| Manual `import.csv` | **Emergency-only** disaster recovery (corrupt DB / laptop lost mid-race), not the normal outage path |

## Advisor review history

- First review: **APPROVE WITH CHANGES** — remote-USB, false confidence, CSV continuity, destructive import, ephemeral DATA_DIR.
- User chose **device-bridge (A)**.
- User corrected offline model: **automatic CSV sync, keep scoring offline (A)** — not harness-fabricated manual import.

## §1 — Production topology

```text
                     Reader laptop                          Cloud (keweenawendurance.com)
                 ┌──────────────────────────┐            ┌──────────────────────────────┐
                 │ Reader UI                │──HTTPS────▶│ Frontend (static)            │
                 │  status: Offline|Syncing │            │ Backend (Go API)             │
                 │         |Online·Synced   │            │   └─ Cloud SQL (online auth) │
                 │ Device-bridge            │──WS/HTTPS─▶│   └─ live CSV → GCS          │
                 │  pm3 USB                 │  when up   │                              │
                 │  local live CSV + queue  │◀─auto sync─│                              │
                 └──────────────────────────┘            └──────────────────────────────┘
                 Spectator browsers ─────────────────────▶ same hosted origin
                                                           (stale while offline)
```

### Authority and clients

- **When online:** Cloud SQL is the system of record; bridge relays every write/poll immediately.
- **When offline:** Bridge continues Proxmark write/poll and appends scored laps to a **local live CSV** (and an in-memory/disk queue). Hosted DB does not grow until reconnect.
- Spectator and reader **UIs** talk to the hosted origin; reader UI also reflects bridge connectivity/sync state (Offline / Syncing / Online · Synced).
- Manual `POST /api/events/:id/import.csv` remains available for **emergency** full replace only — not used by the automatic outage path.

### Proxmark device-bridge (required)

1. Owns the `pm3` serial port (mutex: Poll and WriteTag never race).
2. When hosted is reachable: relays write commands (downstream WS) and read results (upstream) to the hosted API.
3. When hosted is unreachable: **keeps scoring** — write tags locally, record laps into **local live CSV** + durable queue. No operator action required.
4. On reconnect: **automatically** flushes the queue / syncs local CSV deltas into hosted DB (idempotent upsert of timing records). UI transitions Offline → syncing → online · synced.
5. Short blips (≤30s) and prolonged outages (e.g. 5 minutes) use the **same** automatic path; 5‑minute dress-rehearsal outage must exercise real offline scoring + auto-sync.

**Prod-like local stack:** same bridge binary against Compose “hosted” services (no USB into the hosted backend container).

### Sync status UX (reader)

Visible connection/sync chip (extend existing `SyncStatus` / EventLive sync group):

| State | Meaning |
|---|---|
| **Offline** | Hosted unreachable; local CSV/queue accepting new laps |
| **Syncing** | Connectivity restored; auto-flush in progress |
| **Online · Synced** | Hosted reachable; queue empty; DB caught up |

No “Sync pending” button required for the happy path — flush is automatic on reconnect. Emergency manual sync may remain as a fallback control.

### Durable CSV copies

- Bridge local live CSV is the **offline continuity** artifact on the laptop.
- Hosted also maintains a snapshot mirrored to **GCS** (Cloud Run) / compose volume (prod-like) as secondary DR.
- Automatic sync prefers **incremental queue flush** over destructive full `import.csv`. Destructive import is emergency-only and all-stop / single-operator.

## §2 — Deploy readiness + local prod-like stack

*(Unchanged in spirit from prior approval: GCP deploy kit, frontend/backend image fixes, `docker-compose.prod.yml`, nginx `keweenawendurance.com`, secrets via `.env.prod.example`, bridge native on Windows for COM ports.)*

Additional: bridge data dir for local CSV/queue on the laptop; document in `deploy/README.md` and bridge README. Cloud SQL backups/PITR remain primary hosted DR; GCS CSV is secondary; laptop CSV enables automatic outage continuity.

## §3 — Dress rehearsal harness (`--prod` + auto sync outage)

### CLI

```text
npm run test:e2e:bluffet-hardware                  # prod-like default
npm run test:e2e:bluffet-hardware -- --prod        # https://keweenawendurance.com (needs BLUFFET_HW_ALLOW_PROD=1)
npm run test:e2e:bluffet-hardware -- --prod https://staging.example.com
```

Safety gate for live domain: `BLUFFET_HW_ALLOW_PROD=1`.  
Destructive emergency import against prod (if ever used by harness): `BLUFFET_HW_ALLOW_PROD_IMPORT=1` — **not** part of normal outage chaos.

### Chaos: 5-minute outage → automatic sync (no manual import)

1. Snapshot hosted lap totals and reader UI status (**Online · Synced**).
2. Cut connectivity between bridge/browsers and hosted (network partition — not merely `setOffline` on spectators while scoring continues on hosted).
3. During outage: harness **continues** Proxmark write→read via bridge; assert laps land in **local CSV/queue**; assert reader UI shows **Offline**; assert hosted totals **do not** rise; spectators may go stale.
4. Restore connectivity — **do not** call `import.csv`.
5. Assert UI shows **Syncing** then **Online · Synced**; hosted totals catch up to pre + outage laps; spectators catch up.
6. Log critical issues if sync does not complete without intervention.

Unchanged chaos: reader crash + manual entry, late signups, DNFs, spectator friends, video compose.

## Out of scope

- Creating the GCP project / public DNS cutover today.
- Terraform / full IaC.
- Deleting dormant `HOSTED_API_URL` station-sync code.

## Success criteria

- [ ] Deploy kit documented for `keweenawendurance.com`.
- [ ] Device-bridge is sole USB owner; Cloud Run RFID hardware off.
- [ ] Prod-like stack + bridge→hosted path (no USB into hosted backend).
- [ ] Offline: bridge keeps scoring to local CSV; UI shows Offline.
- [ ] Reconnect: automatic sync to hosted DB; UI Syncing → Online · Synced; no manual import.
- [ ] `--prod` flag + safety gates.
- [ ] Dress rehearsal outage chaos proves the automatic path.
- [ ] Full prod-like dress rehearsal run completes.

## Self-review checklist

- [x] Device-bridge chosen for Proxmark
- [x] Offline = keep scoring to local CSV (user choice A)
- [x] Restore = automatic sync; no manual intervention on happy path
- [x] UI states Offline / Syncing / Online · Synced specified
- [x] Manual `import.csv` emergency-only
- [x] Prod-like exercises bridge path
- [x] `--prod` vs prod-like distinguished
