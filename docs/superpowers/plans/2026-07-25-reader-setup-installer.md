# Reader setup.exe Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a per-user Windows `KeweenawReader-Setup.exe` plus docs that install `reader-gui` and a slim Proxmark runtime under `%LOCALAPPDATA%\KeweenawReader\`.

**Architecture:** Pack script stages binaries into a zip; `backend/cmd/reader-setup` embeds the zip and extracts it on run. Reader GUI defaults Proxmark CLI to `{install}\proxmark\proxmark3.exe`.

**Tech Stack:** Go (embed + archive/zip), PowerShell pack script, ProxSpace `objdump` for DLL walk, Fyne reader-gui (existing).

---

### Task 1: Prefer install-dir Proxmark CLI in reader-gui

**Files:**
- Modify: `backend/cmd/reader-gui/main.go`
- Modify: `backend/internal/rfid/cli_proxmark.go`
- Test: `backend/internal/rfid/cli_proxmark_path_test.go` (new)

- [ ] **Step 1: Add PATH helper coverage**

In `cli_proxmark.go`, change `proxmarkMingwBin` so when `platforms` or MinGW DLLs sit beside the CLI, it returns `filepath.Dir(cliPath)`. Also set `QT_PLUGIN_PATH` on the command env when `{cliDir}/platforms` exists.

- [ ] **Step 2: Default CLI in main.go**

Before repo `scripts\pm3.cmd` probes, if `cfg.ProxmarkCLI` is empty or `pm3`, use `{exeDir}\proxmark\proxmark3.exe` when that file exists.

- [ ] **Step 3: Unit test path helper**

Table-test: ProxSpace-shaped path still finds mingw; side-by-side DLL dir returns cli dir.

---

### Task 2: Go installer (`reader-setup`)

**Files:**
- Create: `backend/cmd/reader-setup/main.go`
- Create: `backend/cmd/reader-setup/install.go`
- Create: `backend/cmd/reader-setup/install_test.go`
- Create: `backend/cmd/reader-setup/.gitignore` (ignore `payload.zip`)

- [ ] **Step 1: Extract + install helpers with tests**

`Install(dest string, zipBytes []byte) error` extracts zip, writes `Uninstall.cmd`, creates desktop shortcut via PowerShell COM.

- [ ] **Step 2: main.go embeds payload.zip**

```go
//go:embed payload.zip
var payloadZIP []byte
```

Default dest: `filepath.Join(os.Getenv("LOCALAPPDATA"), "KeweenawReader")`. Print paths; optional `-dest` flag. Create placeholder empty zip for local `go test` if missing (test generates its own zip).

---

### Task 3: Pack script + USB folder

**Files:**
- Create: `scripts/pack-reader-setup.ps1`
- Create: `scripts/Copy-ProxmarkRuntime.ps1` (DLL walk reusable)
- Ensure: `.gitignore` covers `backend/cmd/reader-setup/payload.zip` (and existing `dist/`, `*.exe`)

- [ ] **Step 1: DLL walk copies proxmark runtime into stage\proxmark**
- [ ] **Step 2: Build reader-gui into stage**
- [ ] **Step 3: Zip → payload.zip → build setup.exe → dist/reader-setup/**

---

### Task 4: Operator docs

**Files:**
- Create: `docs/reader-laptop-setup.md`
- Create: staged `SETUP.txt` content in pack script (or `deploy/reader-setup/SETUP.txt` template)
- Modify: `docs/production-reader.md` — link to new doc at top
- Modify: `README.md` — one line under RFID section

---

### Task 5: Pack and smoke

- [ ] Run `scripts/pack-reader-setup.ps1`
- [ ] Confirm `dist/reader-setup/KeweenawReader-Setup.exe` exists
- [ ] Run setup to a temp dest; verify layout; `proxmark3.exe --help` works from install dir with clean PATH
