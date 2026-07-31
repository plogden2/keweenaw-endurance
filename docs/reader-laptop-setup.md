# Reader laptop — minimal setup

Install a finish-line laptop with one executable. You do **not** need the full git repo or ProxSpace on race day.

**Production site:** https://www.keweenawendurance.com

## What you need

| Item | Notes |
|------|--------|
| `KeweenawReader-Setup.exe` | From `dist/reader-setup/` after packing, or the USB stick |
| Proxmark3 | Already flashed; plug in (usually **COM3**) |
| Internet | For hosted API sync (offline queue still works) |
| Organizer PIN / bridge token | Entered in the GUI |

## Install (once per laptop)

1. Copy `KeweenawReader-Setup.exe` from the USB pack (folder `dist/reader-setup/`).
2. Double-click it (no admin). Installs to `%LOCALAPPDATA%\KeweenawReader\`.
3. If SmartScreen blocks: **More info** → **Run anyway**.
4. Desktop shortcut **Keweenaw Reader** is created.

Contents installed:

- `reader-gui.exe` — operator UI + bridge  
- `proxmark\` — slim Proxmark client + DLLs  
- `SETUP.txt` — short day-of checklist  
- `Uninstall.cmd`

## First-run GUI

1. Open **Keweenaw Reader**.
2. Confirm **Proxmark CLI** is `%LOCALAPPDATA%\KeweenawReader\proxmark\proxmark3.exe`.
3. Set:
   - Hosted API URL: `https://www.keweenawendurance.com`
   - Bridge token and/or organizer PIN
   - Device ID, Event / Race / Checkpoint IDs
   - COM port (usually `COM3`)
   - **Use Proxmark hardware**
4. **Save config** → **Test Proxmark** → **Start bridge**.
5. Status: `ONLINE_SYNCED` / `online_synced`.

Then in the browser: unlock PIN → **Station config** → arm finish with the same Device ID.

Missed finish: type **bib** in the GUI → **Record lap**.

## Drivers

If no COM port appears the first time this laptop sees the Proxmark, install **WinUSB** with [Zadig](https://zadig.akeo.ie/) once. This pack does **not** flash firmware.

## Rebuild the USB pack (dev machine)

Requires ProxSpace (for `proxmark3.exe` + MinGW) and 64-bit MinGW to build the GUI:

```powershell
powershell -File scripts\pack-reader-setup.ps1
```

Output folder (copy to USB):

```
dist\reader-setup\
  KeweenawReader-Setup.exe
  README-USB.txt
  SETUP.txt
```

## Full race-day ops

See [production-reader.md](./production-reader.md) for tag programming, manual entry, and troubleshooting.
