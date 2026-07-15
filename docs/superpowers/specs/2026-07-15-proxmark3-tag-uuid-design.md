# Proxmark3 driver + logical tag UUID — design

**Date:** 2026-07-15  
**Status:** Approved for planning (feeds implementation plan)  
**Depends on:** nothing  
**Blocks:** `docs/superpowers/plans/2026-07-15-hardware-bluffet-e2e.md`

## Goal

Ship a real Proxmark3 read/write path and rework RFID identity so scoring keys off a **logical UUID written into tag user memory**, not the chip’s silicon UID.

## Identity model

| Concept | Role |
|---|---|
| Silicon UID | Hardware serial only — **not** used for racer identity or `tag_uid` lookups |
| Logical RFID UUID | Stable UUID per racer (generated at registration / seed). Stored in `rfid_tag_associations.tag_uid` and mirrored on `participants.rfid_tag_uid`. **Never changes** for that racer. |
| Program tag | Write the racer’s logical RFID UUID into NTAG/MIFARE user memory on whatever chip is on the antenna |
| Read / Poll | Read user-memory payload → return logical UUID string on the WebSocket / scan path |
| Lost / replacement tag | Place new blank chip → Program tag again → **same** logical UUID is written. No DB reassignment of silicon IDs. |
| Dress rehearsal (1 physical chip) | Before each simulated lap, Program tag for the next racer (overwrite user memory with that racer’s logical UUID), then Poll/read to score |

## API / UX changes

- `POST /api/rfid/write-tag` body: `{ "participant_id": "…" }` only (optional `association` later). No client-supplied silicon UID required.
- Racers “Program tag” UI: remove mandatory Tag UID field; button writes racer UUID to the chip on the reader.
- Mock / inject path continues to use logical UUIDs (`DEMO-TAG-*` seeds become real UUIDs or stay as string keys that are the logical id).

## Hardware

- `RFID_HARDWARE=true` selects a real `CLIProxmarkReader` (pm3 CLI) instead of `MockReader`.
- `PROXMARK3_CLI`, `PROXMARK3_PORT` configure the bridge.
- Compose overlay documents USBIPD / native Windows host options.

## Non-goals

- Supporting non-Proxmark RFID hardware
- Using silicon UID as the identity key
- Reassigning one silicon UID across racers in the DB (unnecessary under this model)
