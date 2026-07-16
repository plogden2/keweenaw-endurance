# Device bridge (Proxmark owner)

Native laptop process that owns the Proxmark3 USB serial port, relays RFID reads/writes to the hosted API over WebSocket, and keeps scoring during outages by appending to a local live CSV + durable pending queue. On reconnect it automatically flushes pending laps — operators never call `import.csv` for normal outages.

## When to run natively (Windows)

Run the bridge **on the reader laptop outside Docker** so it can open the Proxmark COM port (`PROXMARK3_PORT=COM3`). The hosted backend container sets `RFID_HARDWARE=false`; only this binary talks to USB.

## Environment

| Variable | Default | Purpose |
|---|---|---|
| `HOSTED_API_URL` | *(required)* | Hosted origin, e.g. `http://keweenawendurance.com` |
| `BRIDGE_TOKEN` | — | Preferred auth: `X-Bridge-Token` on WebSocket dial |
| `ORGANIZER_PIN` | — | Fallback: exchange PIN for admin JWT via `POST /api/auth/pin` |
| `DEVICE_ID` | `laptop-finish-1` | Bridge device id (must match reader station) |
| `EVENT_ID` | *(required)* | Bluffet event UUID for local CSV path |
| `BRIDGE_DATA_DIR` | `./bridge-data` | Local `events/{EVENT_ID}/live-snapshot.csv` + `pending.jsonl` |
| `BRIDGE_LOCAL_ADDR` | `127.0.0.1:8091` | Loopback HTTP for offline write-tag + status |
| `RFID_HARDWARE` | `false` | `true` → use Proxmark CLI reader |
| `PROXMARK3_CLI` | `pm3` | Path to Proxmark client |
| `PROXMARK3_PORT` | — | Serial port, e.g. `COM3` on Windows |
| `BRIDGE_MOCK` | `false` | `true` → `MockReader` (CI / no hardware smoke) |
| `BRIDGE_POLL_MS` | `500` | Poll interval in milliseconds |
| `BRIDGE_PARTITION_SIGNAL` | `%TEMP%\keweenaw-bridge-partition.signal` | When this file **exists**, bridge treats hosted as unreachable (dress-rehearsal outage chaos) |

## Behavior

1. Authenticate to hosted (`BRIDGE_TOKEN` or `ORGANIZER_PIN`).
2. Dial `ws(s)://{host}/api/rfid/bridge?device_id=...`.
3. **Online:** hosted `write` → local `WriteTag` → `write_ack`; poll UUID → `read` upstream.
4. **Offline:** local `POST http://127.0.0.1:8091/write-tag` still programs the chip; poll enqueues laps to `pending.jsonl` and appends audit rows to `live-snapshot.csv`; status `offline`.
5. **Reconnect:** status `syncing`; flush `pending.jsonl` as `read` messages; clear queue; status `online_synced`.
6. Reports `{type:status, pending_count, syncing, last_sync_at}` on the WebSocket for hosted `GET /api/rfid/bridge/status`.

## Local HTTP (offline write-tag)

```http
GET http://127.0.0.1:8091/status
POST http://127.0.0.1:8091/write-tag
Content-Type: application/json

{"logical_uuid":"9fe78eeb-a21c-594a-acc2-7e1efe378201"}
```

When hosted is reachable, you may instead send `participant_id` + `race_id` and the bridge will fetch the active tag from hosted before writing.

## Windows launch (prod-like dress rehearsal)

```powershell
$env:HOSTED_API_URL="http://keweenawendurance.com"
$env:ORGANIZER_PIN="1738"
$env:DEVICE_ID="laptop-finish-1"
$env:EVENT_ID="1441674d-a011-471a-a601-722b88b117f5"
$env:BRIDGE_DATA_DIR="$PWD\bridge-data"
$env:RFID_HARDWARE="true"
$env:PROXMARK3_PORT="COM3"
cd backend
go run ./cmd/device-bridge
```

Add `127.0.0.1 keweenawendurance.com` to `C:\Windows\System32\drivers\etc\hosts` when using the local prod-like Compose stack.

## No-hardware smoke

```powershell
$env:HOSTED_API_URL="http://keweenawendurance.com"
$env:ORGANIZER_PIN="1738"
$env:DEVICE_ID="laptop-finish-1"
$env:EVENT_ID="1441674d-a011-471a-a601-722b88b117f5"
$env:BRIDGE_MOCK="true"
cd backend
go run ./cmd/device-bridge
```

Inject poll results in tests via `MockReader.Enqueue(logicalUUID)` when building a custom harness; the stock binary uses env-driven `BRIDGE_MOCK`.

## Tests

```powershell
cd backend
go test ./internal/bridge/...
```

Unit tests cover local CSV append and idempotent pending flush.

## Data layout

```text
bridge-data/
  events/
    {EVENT_ID}/
      live-snapshot.csv   # append-only timing_records audit rows while offline
      pending.jsonl       # one JSON lap per line; flushed on reconnect
```

## Emergency import

Normal outages use automatic bridge sync. `POST /api/events/:id/import.csv` is **emergency-only** (destructive full replace) — not part of this path.
