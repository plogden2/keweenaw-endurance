# Production reader laptop — race-day instructions

**New laptop / USB install:** use [reader-laptop-setup.md](./reader-laptop-setup.md) and `dist/reader-setup/KeweenawReader-Setup.exe` (build with `scripts/pack-reader-setup.ps1`). That path does not need the full repo or ProxSpace on the race machine.

Short setup and ops guide for the finish-line laptop against live production.

**Production site:** https://www.keweenawendurance.com  
(Use the public domain, not a raw Cloud Run frontend URL — `/api` is routed through the load balancer.)

**Organizer PIN:** `1738` (unless rotated in Secret Manager)

**Bluffet is finish-only:** All You Can East Bluffet has a single start/finish lap point. The **Station** page has no checkpoint mode toggle or checkpoint picker for this event (only **Finish station**), and manual lap entry (reader GUI or website) never asks for a checkpoint — it always uses each race's finish checkpoint automatically.

### Where to find controls

| Control | Where |
|---------|--------|
| **Start Proxmark bridge** | `reader-gui.exe` (preferred) · or headless `device-bridge.exe` |
| **Manual lap (reader miss)** | `reader-gui.exe` → Manual entry · or website Manual entry |
| **PIN** / **Manage** | Top header (every page) · also footer |
| **Station** | Top header · also footer |
| **CSV recovery** | Footer · also PIN page after unlock |
| **Racers** | After PIN unlock: each race row on **Manage** · live view toolbar · race details · event page |
| **Manual entry (website)** | Same places as Racers (opens the event **Taps** page at `/events/:eventId/taps`) |
| **Export Excel** | PIN-unlocked live race flow toolbar |
| **Add racer** / **Program tag** / **Write tag** | On the Racers page for that race |
| **Save & arm reader** | Station page |
| **Record time** | Event Taps page (**Add tap**) · or reader GUI |
| **Online · Synced** chip | Live race flow page (when PIN unlocked) |

---

## 1. One-time hardware setup

**Preferred (USB / clean laptop):** run `KeweenawReader-Setup.exe` from `dist/reader-setup/` (see [reader-laptop-setup.md](./reader-laptop-setup.md)). Then skip the ProxSpace/repo build steps below.

**Dev machine / from this repo:**

1. Plug in the Proxmark3 (usually **COM3** on Windows).
2. Confirm ProxSpace / `pm3` works. This repo’s wrapper is:

   `scripts\pm3.cmd` → ProxSpace client on COM3

   Finish reads use a continuous Proxmark arm. Same-chip cooldown after a successful scan is **1 second**.

3. Build the reader GUI once (64-bit MinGW required — ProxSpace provides it):

```powershell
powershell -File C:\Users\gener\Documents\keweenaw-endurance\scripts\build-reader-gui.ps1
```

Optional headless binary:

```powershell
cd C:\Users\gener\Documents\keweenaw-endurance\backend
go build -o device-bridge.exe ./cmd/device-bridge
```

Rebuild the USB installer after GUI or Proxmark updates:

```powershell
powershell -File scripts\pack-reader-setup.ps1
```

---

## 2. Start the reader against production (GUI — preferred)

1. Run `backend\reader-gui.exe` (double-click or from Explorer).
2. Fill in:
   - **Hosted API URL:** `https://www.keweenawendurance.com`
   - **Bridge token** (from Secret Manager `keweenaw-bridge-token`) and/or **Organizer PIN** `1738`
   - **Device ID:** `laptop-finish-1`
   - **Event ID:** `1441674d-a011-471a-a601-722b88b117f5` (Bluffet 2026)
   - **Race:** leave on **All races (event finish)** to score every distance from one mat, or pick a single distance. Bluffet has only one finish/lap point, so there is **no "Checkpoint (manual)" picker** — the GUI hides it and auto-uses each race's finish checkpoint.
   - **COM port:** usually `COM3`
   - **Proxmark CLI:** `C:\Users\gener\Documents\keweenaw-endurance\scripts\pm3.cmd`
   - Check **Use Proxmark hardware**
   - **HF gain:** defaults to **63** (max sensitivity); maps to Proxmark `hw sethfthresh` (floored at 3 — raw thresh 1/2 break anticollision on this client). Lower the slider if false triggers / noise
3. **Save config** → **Test Proxmark** → **Start bridge**
4. Confirm status shows **ONLINE_SYNCED** (or `online_synced`).

Then in Chrome:

1. Open https://www.keweenawendurance.com
2. Top header → **PIN** → enter organizer PIN → **Unlock management**
3. Top header → **Station** → event, **Finish station**, Device ID `laptop-finish-1` → **Save & arm reader**
4. Work the race from **Manage** / live race flow

If Proxmark misses a finish: in the GUI, type the **bib** → **Record lap** (works offline too — queues and syncs later).

Config is stored under `%LOCALAPPDATA%\KeweenawEndurance\bridge-data\reader-gui-config.json`.

### Headless alternative (PowerShell)

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

**Taps:** Keep the bridge/GUI running so it holds one Proxmark session open. A brief wave should register quickly; the **laptop beeps** immediately on a successful chip read. The GUI shows **racer name · bib · UUID**. The website still plays the Mario Kart sound when a lap is scored.

**All races:** In reader-gui, leave **Race** on **All races (event finish)** so one mat scores every distance. Arm the website Station to the event in finish mode (no race picker on Station). Manual bib entry auto-resolves across races; pick a race override only if a bib is shared.

**Write-only mode:** Check **Write-only mode** in reader-gui when programming tags. Taps still show name/UUID in the GUI but are **not** scored or queued. Uncheck to resume normal finish recording. Tag writes and manual bib entry still work while write-only is on.

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
- Bridge/GUI must be running; write goes hosted → bridge → Proxmark.

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

**Preferred on the reader laptop:** `reader-gui.exe` → enter bib → **Record lap**.

Website path — event-scoped **Taps** page (covers every race in the event, not just one):

1. Unlock with PIN.
2. Open **Manual entry** (Manage → race → **Manual entry**, or live view → **Manual entry**, or event page). All of these open `/events/:eventId/taps`.
3. Type the **bib number** in the inline field and press **Enter**. No dialog — the tap records that racer's race finish immediately.
4. Bibs must be unique across the event for Enter to resolve; if more than one racer shares the number, fix the duplicate on Racers first.

The tap is stored with the current timestamp and shows in the taps table (sorted newest first). There is no karaoke control on this page.

**Karaoke bonus lap:** after a normal RFID lap popup, use the karaoke control on the scan popup to add one bonus lap (when available).

---

## 6. Removing / fixing bad taps

Unlock **PIN** first. Scored laps can be soft-voided (and restored) without wiping the event.

| Situation | Fix |
|-----------|-----|
| Just scored the wrong lap | On the scan popup: **Discard lap** → confirm **Discard** |
| Older bad lap / karaoke | Open **Manual entry** (event **Taps** page) → find the row → **Void** (or **Restore** if already voided) |
| Accidental double-tap | Usually blocked by the **1-minute cooldown** — no action |
| Missed tap | **reader-gui** manual entry or website **Manual entry** (bib + Enter on the event Taps page, section 5) |
| Need to wipe/rebuild timing data | Emergency only: footer **CSV recovery** (PIN required). This replaces event timing data — stop all scoring first |

Voided laps stay in the database/CSV for audit but do not count toward standings or cooldown. Voiding an RFID lap also voids its karaoke bonus.

Do **not** call CSV import for normal outages. Offline scoring uses the device-bridge queue and auto-flushes when production is reachable again.

---

## 7. Quick day-of checklist

- [ ] Proxmark on COM3; `scripts\pm3.cmd` works
- [ ] `reader-gui.exe` running → status `ONLINE_SYNCED` (or `curl.exe http://127.0.0.1:8091/status`)
- [ ] Browser on https://www.keweenawendurance.com
- [ ] Header shows **PIN** / **Station**; unlock PIN; arm station as finish / `laptop-finish-1`
- [ ] From Manage, open **Racers** for a race; spot-check program + tap
- [ ] Know **GUI Record lap** / website **Manual entry** (event Taps page → bib + **Enter**) for missed taps

---

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| Write tag / 500 / `pm3` not found | Set Proxmark CLI to `scripts\pm3.cmd` in the GUI (or `PROXMARK3_CLI`) and restart |
| Sync chip offline | Restart bridge/GUI; check internet; confirm Hosted API URL is the public domain |
| Frontend loads but API broken | Don’t use the raw `*.run.app` frontend URL; use https://www.keweenawendurance.com |
| Local `keweenawendurance.com` hits nothing | Your hosts file may map it to `127.0.0.1` — use `www.` or fix hosts to the LB IP |
| Can’t find Racers | Unlock PIN first → **Manage** → race row **Racers** (also on live view when unlocked) |
| GUI build fails (`64-bit mode not compiled in`) | Use ProxSpace `mingw64\bin\gcc.exe`, not 32-bit `C:\MinGW` |

More detail: `backend/cmd/reader-gui/README.md`, `backend/cmd/device-bridge/README.md`, `deploy/README.md`.
