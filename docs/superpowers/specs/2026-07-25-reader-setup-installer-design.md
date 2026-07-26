# Reader laptop setup.exe — Design

**Date:** 2026-07-25  
**Status:** Approved (user) + advisor review incorporated  
**Related:** `docs/production-reader.md`, `backend/cmd/reader-gui`

## Goal

Ship a single Windows `setup.exe` that installs a working finish-line reader on a clean laptop without the full git repo or ProxSpace (~5.7 GB).

## Decisions

| Topic | Choice |
|-------|--------|
| Delivery | `setup.exe` (Go self-extracting installer) |
| Default install dir | `%LOCALAPPDATA%\KeweenawReader\` (no admin) |
| Proxmark runtime | Slim bundle: `proxmark3.exe` + recursively walked MinGW DLLs + `platforms/qwindows.dll` |
| Python stdlib | Not bundled (bridge uses built-in `hf` CLI only) |
| Desktop shortcut | Yes, created by installer |
| Secrets | Not prefilled; operator enters PIN/token in GUI |
| Git | Pack scripts + docs only; `dist/` and `payload.zip` gitignored |
| Drivers | Document Zadig/WinUSB; do not install drivers |

## Install layout

```
%LOCALAPPDATA%\KeweenawReader\
  reader-gui.exe
  SETUP.txt
  Uninstall.cmd          # deletes install dir + desktop shortcut
  proxmark\
    proxmark3.exe
    *.dll
    platforms\
      qwindows.dll
```

## Operator flow

1. Copy `KeweenawReader-Setup.exe` from USB (or run in place).
2. Double-click → installs under `%LOCALAPPDATA%\KeweenawReader\`.
3. Launch **Keweenaw Reader** (shortcut or `reader-gui.exe`).
4. Confirm Proxmark CLI defaults to `{install}\proxmark\proxmark3.exe`.
5. Set API URL / PIN or bridge token / device + event IDs → **Test Proxmark** → **Start bridge**.
6. Browser: https://www.keweenawendurance.com → PIN → Station arm.

## Build (dev machine)

`scripts/pack-reader-setup.ps1`:

1. Build `reader-gui.exe` (existing MinGW/CGO path).
2. Stage slim Proxmark from ProxSpace via `objdump` dependency walk (+ Qt `platforms\qwindows.dll`).
3. Zip stage → `backend/cmd/reader-setup/payload.zip`.
4. `go build` → `dist/reader-setup/KeweenawReader-Setup.exe`.
5. Write USB folder: setup exe + short `README-USB.txt`.

## Code changes

- `reader-gui`: if Proxmark CLI unset/`pm3`, prefer `{exeDir}\proxmark\proxmark3.exe` before repo `scripts\pm3.cmd`.
- RFID runner: when DLLs live beside the CLI, prepend that directory to `PATH` (and set `QT_PLUGIN_PATH` to `{cliDir}` if `platforms` exists) so spawn works outside ProxSpace.

## Out of scope

Full ProxSpace, firmware flashing, macOS/Linux, Inno Setup/WiX, auto-filled secrets, silent driver install.

## Acceptance

- [ ] Clean Win10/11 laptop (no ProxSpace): run setup → Test Proxmark → Start bridge → arm station
- [ ] No admin / UAC for default path
- [ ] Upgrade overwrites binaries; AppData config preserved
- [ ] SETUP.txt covers SmartScreen, COM port, Zadig
- [ ] `dist/` and Proxmark binaries not committed
