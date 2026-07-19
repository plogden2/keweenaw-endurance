# Production reader laptop — race-day instructions

Short setup and ops guide for the finish-line laptop against live production.

**Production site:** https://www.keweenawendurance.com  
(Use the public domain, not a raw Cloud Run frontend URL — `/api` is routed through the load balancer.)

**Organizer PIN:** `1738` (unless rotated in Secret Manager)

### Where to find controls

| Control | Where |
|---------|--------|
| **PIN** / **Manage** | Top header (every page) · also footer |
| **Station** | Top header · also footer |
| **CSV recovery** | Footer · also PIN page after unlock |
| **Racers** | After PIN unlock: each race row on **Manage** · live view toolbar · race details · event page |
| **Manual entry** | Same places as Racers (opens Live timing) |
| **Add racer** / **Program tag** / **Write tag** | On the Racers page for that race |
| **Save & arm reader** | Station page |
| **Record time** | Manual entry (Live timing) page |
| **Online · Synced** chip | Live race flow page (when PIN unlocked) |

---

## 1. One-time hardware setup

1. Plug in the Proxmark3 (usually **COM3** on Windows).
2. Confirm ProxSpace / `pm3` works. This repo’s wrapper is:

   `scripts\pm3.cmd` → ProxSpace client on COM3

3. Build the device-bridge once:

```powershell
cd C:\Users\gener\Documents\keweenaw-endurance\backend
go build -o device-bridge.exe ./cmd/device-bridge
```

---

## 2. Start the reader against production

Run this **on the reader laptop** (outside Docker) every race day:

```powershell
$env:HOSTED_API_URL = "https://www.keweenawendurance.com"
$env:BRIDGE_TOKEN   = "<keweenaw-bridge-token from Secret Manager>"
$env:ORGANIZER_PIN  = "1738"   # fallback if token unset
$env:DEVICE_ID      = "laptop-finish-1"
$env:EVENT_ID       = "1441674d-a011-471a-a601-722b88b117f5"   # Bluffet 2026
$env:BRIDGE_DATA_DIR = "C:\Users\gener\Documents\keweenaw-endurance\bridge-data"
$env:RFID_HARDWARE  = "true"
$env:PROXMARK3_PORT = "COM3"
$env:PROXMARK3_CLI  = "C:\Users\gener\Documents\keweenaw-endurance\scripts\pm3.cmd"

cd C:\Users\gener\Documents\keweenaw-endurance\backend
.\device-bridge.exe
```

Healthy bridge checklist:

```powershell
curl.exe http://127.0.0.1:8091/status
```

Expect `"mode":"online_synced"` and `"connected":true`.

Then in Chrome:

1. Open https://www.keweenawendurance.com
2. Top header → **PIN** → enter organizer PIN → **Unlock management**  
   (header label becomes **Manage**; footer shows **PIN · Unlocked**)
3. Top header → **Station**
4. Select the event, keep **Finish station**, set **Device ID** to `laptop-finish-1`
5. Click **Save & arm reader**
6. Header → **Manage** (or Timing → event → **Open live race flow**) to work the race

Watch the sync chip on the live view: **Online · Synced**. If it shows **Offline**, the bridge is down or unreachable; keep scoring locally — it will auto-sync when back.

---

## 3. Program tags for racers

1. Unlock with PIN (header **PIN** / **Manage**).
2. On the Manage page, find the race → click **Racers**  
   (or from live view: select the race tab → **Racers**)
3. Search by name or bib.
4. On the racer row, click **Program tag**.
5. Place the physical chip on the Proxmark.
6. Click **Write tag**.
7. Wait for success, then **Done**.

Notes:

- Each racer has a permanent logical RFID UUID. Replacement chips get the **same** UUID.
- You can program multiple physical tags for one racer (lost-tag replacements).
- Bridge must be running; write goes hosted → bridge → Proxmark.

---

## 4. Last-minute sign-ups

1. Unlock with PIN.
2. Open that race’s **Racers** page (Manage → race → **Racers**).
3. Click **Add racer**.
4. Fill first/last name, gender, category. Bib is optional (defaults to next sequential).
5. **Save racer**.
6. Immediately **Program tag** for their chip (section 3).

They can race as soon as the tag is written and the station is armed.

---

## 5. Manual taps (add a lap without a chip read)

Use this when a racer finishes but the tag didn’t read.

1. Unlock with PIN.
2. Open **Manual entry** for that race (Manage → race → **Manual entry**, or live view → **Manual entry**).
3. Under **Manual Timing Entry**:
   - Select the **Finish** (or correct) checkpoint
   - Enter **bib number** (preferred) *or* RFID tag UID
4. Click **Record time**

The lap is stored with the current timestamp and shows under **Recent Records**.

**Karaoke bonus lap:** after a normal RFID lap popup, use the karaoke control on the scan popup to add one bonus lap (when available).

---

## 6. Removing / fixing bad taps

There is **no “delete this lap” button** in the UI today.

What you can do on race day:

| Situation | What to do |
|-----------|------------|
| Accidental double-tap | Usually blocked by the **1-minute cooldown** — no action |
| Missed tap | **Manual entry** (section 5) |
| Wrong racer credited | Add the correct lap via manual entry; note the bad tap for post-race cleanup |
| Need to wipe/rebuild timing data | Emergency only: footer **CSV recovery** (PIN required). This replaces event timing data — stop all scoring first |

Do **not** call CSV import for normal outages. Offline scoring uses the device-bridge queue and auto-flushes when production is reachable again.

---

## 7. Quick day-of checklist

- [ ] Proxmark on COM3; `scripts\pm3.cmd` works
- [ ] `device-bridge.exe` running → `http://127.0.0.1:8091/status` = `online_synced`
- [ ] Browser on https://www.keweenawendurance.com
- [ ] Header shows **PIN** / **Station**; unlock PIN; arm station as finish / `laptop-finish-1`
- [ ] From Manage, open **Racers** for a race; spot-check program + tap
- [ ] Know **Manual entry** on Manage / live view for missed taps

---

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| Write tag / 500 / `pm3` not found | Set `PROXMARK3_CLI` to `scripts\pm3.cmd` and restart the bridge |
| Sync chip offline | Restart bridge; check internet; confirm `HOSTED_API_URL` is the public domain |
| Frontend loads but API broken | Don’t use the raw `*.run.app` frontend URL; use https://www.keweenawendurance.com |
| Local `keweenawendurance.com` hits nothing | Your hosts file may map it to `127.0.0.1` — use `www.` or fix hosts to the LB IP |
| Can’t find Racers | Unlock PIN first → **Manage** → race row **Racers** (also on live view when unlocked) |

More detail: `backend/cmd/device-bridge/README.md`, `deploy/README.md`.
