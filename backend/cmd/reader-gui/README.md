# Reader GUI (Proxmark operator panel)

Windows desktop app that runs the device bridge in-process: start/stop Proxmark polling, live status, and **manual bib entry** when hardware misses a tap.

## Build (Windows)

Requires a **64-bit** MinGW (ProxSpace ships one):

```powershell
$mingw = "C:\Users\gener\sdk\ProxSpace\msys2\mingw64\bin"
$env:Path = "$mingw;" + $env:Path
$env:CGO_ENABLED = "1"
$env:CC = "$mingw\gcc.exe"
cd C:\Users\gener\Documents\keweenaw-endurance\backend
go build -o reader-gui.exe ./cmd/reader-gui
```

Or: `..\scripts\build-reader-gui.ps1`

## Race day

1. Double-click `reader-gui.exe` (or run from `backend\`).
2. Set Hosted API URL (`https://www.keweenawendurance.com`), bridge token and/or organizer PIN, Event ID, Device ID, Race ID + Checkpoint ID (for manual entry), COM port.
3. Enable **Use Proxmark hardware**, Save config, **Test Proxmark**, then **Start bridge**.
4. Arm the station in the website as usual. Tag programming still goes through the site → local `:8091`.
5. If a finish is missed: type bib → **Record lap**. Online posts to hosted; offline queues into the bridge pending file and flushes on reconnect.

Config defaults to `%LOCALAPPDATA%\KeweenawEndurance\bridge-data\reader-gui-config.json`.

Headless fallback remains: `device-bridge.exe` (see `cmd/device-bridge/README.md`).
