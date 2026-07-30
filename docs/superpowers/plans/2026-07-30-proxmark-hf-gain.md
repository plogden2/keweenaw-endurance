# Proxmark HF Antenna Gain Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add adjustable Proxmark HF gain (default max) to reader config + GUI, applying `hw sethfthresh` for max sensitivity by default.

**Architecture:** Operator-facing gain 1–63 (default 63) maps to Proxmark reader threshold `64 - gain`. `CLIProxmarkReader` applies `hw sethfthresh -t <n>` on session open / first one-shot use and when `SetHFGain` is called. Config persists `hf_gain`; GUI slider updates live while the bridge runs (same pattern as write-only).

**Tech Stack:** Go, Fyne reader-gui, Proxmark3 CLI (`hw sethfthresh`), testify

**Spec:** `docs/superpowers/specs/2026-07-30-proxmark-hf-gain-design.md`

**Worktree:** `.worktrees/hf-antenna-gain` on branch `feat/hf-antenna-gain`

---

## File structure

| File | Responsibility |
|------|----------------|
| `backend/internal/rfid/hf_gain.go` | `HFGainMin/Max/Default`, `ClampHFGain`, `HFThreshFromGain` |
| `backend/internal/rfid/hf_gain_test.go` | Mapping + clamp tests |
| `backend/internal/rfid/cli_proxmark.go` | Store gain; apply thresh; `SetHFGain` |
| `backend/internal/rfid/cli_proxmark_test.go` | Assert `hw sethfthresh` commands |
| `backend/internal/bridgeapp/config.go` | `HFGain` field, default, env, normalize |
| `backend/internal/bridgeapp/config_test.go` | Default 63 + round-trip + env |
| `backend/internal/bridgeapp/app.go` | Pass gain; `SetHFGain` live |
| `backend/cmd/reader-gui/ui.go` | Slider + persist + live apply |

---

### Task 1: Gain ↔ threshold helpers

**Files:**
- Create: `backend/internal/rfid/hf_gain.go`
- Create: `backend/internal/rfid/hf_gain_test.go`

- [ ] **Step 1: Write failing tests**

```go
func TestHFThreshFromGain(t *testing.T) {
	assert.Equal(t, 1, HFThreshFromGain(63))
	assert.Equal(t, 63, HFThreshFromGain(1))
	assert.Equal(t, 7, HFThreshFromGain(57)) // 64-57
}

func TestClampHFGain(t *testing.T) {
	assert.Equal(t, 63, ClampHFGain(0))
	assert.Equal(t, 63, ClampHFGain(99))
	assert.Equal(t, 1, ClampHFGain(1))
	assert.Equal(t, HFGainDefault, ClampHFGain(-1)) // default when invalid low → actually: clamp 0 and below to Default OR to 1?
}
```

**Clamp rule (lock this):** values `< 1` or `> 63` become `HFGainDefault` (63). Valid 1–63 unchanged.

- [ ] **Step 2: Run tests — expect FAIL**
- [ ] **Step 3: Implement helpers**

```go
const (
	HFGainMin     = 1
	HFGainMax     = 63
	HFGainDefault = 63
)

func ClampHFGain(g int) int {
	if g < HFGainMin || g > HFGainMax {
		return HFGainDefault
	}
	return g
}

func HFThreshFromGain(g int) int {
	g = ClampHFGain(g)
	return 64 - g
}
```

- [ ] **Step 4: Run tests — expect PASS**
- [ ] **Step 5: Commit**

```
feat(rfid): add HF gain to Proxmark threshold mapping
```

---

### Task 2: CLIProxmarkReader applies threshold

**Files:**
- Modify: `backend/internal/rfid/cli_proxmark.go`
- Modify: `backend/internal/rfid/cli_proxmark_test.go`
- Modify: `backend/internal/rfid/session_test.go` (if session factory tests need thresh apply)

**Behavior:**
- Add `HFGain int` to `CLIProxmarkConfig` (0 → default 63 via Clamp).
- Reader stores `hfGain int` and `threshApplied bool` (for one-shot).
- After opening a new session in `ensureSessionLocked`, run `hw sethfthresh -t <n>` (ignore soft failure? **must succeed or log**; for tests with Runner, apply before first Poll/Write/Detect via `ensureThreshLocked`).
- For one-shot (`runner` mode): call `ensureThreshLocked` before other commands; run thresh command once until `SetHFGain` or flag cleared.
- `SetHFGain(g int)` clamps, stores, clears `threshApplied`, and if session/runner ready applies immediately under mutex.
- Helper formats: `fmt.Sprintf("hw sethfthresh -t %d", HFThreshFromGain(g))`

- [ ] **Step 1: Write failing tests**

```go
func TestCLIProxmarkReader_AppliesHFThreshBeforePoll(t *testing.T) {
	var cmds []string
	r := NewCLIProxmarkReader(CLIProxmarkConfig{
		Enabled: true,
		HFGain:  63,
		Runner: func(command string) (string, error) {
			cmds = append(cmds, command)
			if strings.HasPrefix(command, "hw sethfthresh") {
				return "Thresholds set.", nil
			}
			return "Data : 14 41 67 4D A0 11 47 1A A6 01 72 2B 88 B1 17 F5\n", nil
		},
	})
	_, err := r.Poll()
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(cmds), 2)
	assert.Equal(t, "hw sethfthresh -t 1", cmds[0])
	assert.Equal(t, "hf mfu rdbl -b 4", cmds[1])
}

func TestCLIProxmarkReader_SetHFGainReapplies(t *testing.T) {
	// After SetHFGain(50), next command path issues hw sethfthresh -t 14
}
```

Also update existing Poll tests: Runner may receive thresh command first — either allow any prefix order in assertions or have Runner ignore thresh and only assert rdbl when not thresh. **Preferred:** update Poll tests to accept/skip thresh as first command, OR configure tests with a runner that records and returns ok for thresh. Simplest: in existing tests that assert exact command, change assertion to only check non-thresh commands, or prepend expected thresh.

**Existing test update pattern:** when Runner asserts `command == "hf mfu rdbl -b 4"`, change to:
```go
if strings.HasPrefix(command, "hw sethfthresh") {
	return "Thresholds set.", nil
}
assert.Equal(t, "hf mfu rdbl -b 4", command)
```

- [ ] **Step 2: Run new tests — FAIL**
- [ ] **Step 3: Implement apply logic**
- [ ] **Step 4: Fix existing tests; all `go test ./internal/rfid/` PASS**
- [ ] **Step 5: Commit**

```
feat(rfid): apply Proxmark HF threshold from gain
```

---

### Task 3: Config + App wiring

**Files:**
- Modify: `backend/internal/bridgeapp/config.go`
- Modify: `backend/internal/bridgeapp/config_test.go`
- Modify: `backend/internal/bridgeapp/app.go`

- [ ] **Step 1: Failing config tests**
  - `DefaultConfig().HFGain == 63`
  - Save/Load round-trip includes `hf_gain`
  - `PROXMARK3_HF_GAIN=40` overrides
  - Invalid/missing clamps to 63

- [ ] **Step 2: Implement Config field**
  - `HFGain int \`json:"hf_gain"\`` on Config + configFile
  - DefaultConfig sets 63
  - Save/Load/merge/normalize (clamp via `rfid.ClampHFGain`)
  - ApplyEnv `PROXMARK3_HF_GAIN`

- [ ] **Step 3: Wire App**
  - `openReader` passes `HFGain: cfg.HFGain`
  - `SetHFGain(gain int)` updates cfg, calls `cli.SetHFGain`, publishStatus (optional)
  - Mirror `SetWriteOnly` style

- [ ] **Step 4: `go test ./internal/bridgeapp/ ./internal/rfid/` PASS**
- [ ] **Step 5: Commit**

```
feat(bridge): persist and live-update Proxmark HF gain
```

---

### Task 4: Reader GUI slider

**Files:**
- Modify: `backend/cmd/reader-gui/ui.go`

- [ ] **Step 1: Add HF gain slider**
  - Fyne `widget.NewSlider(1, 63)` (step 1)
  - Label showing current value e.g. `HF gain: 63 (max sensitivity)`
  - Place near Proxmark COM port / hardware checks
  - Init from `ui.cfg.HFGain` (default 63)

- [ ] **Step 2: OnChange**
  - Update cfg + label
  - `SaveConfig` (like write-only)
  - If bridge running: `ui.bridge.SetHFGain(int(value))`

- [ ] **Step 3: Include in `readForm()`**
- [ ] **Step 4: Build check** `go build -o NUL ./cmd/reader-gui` (may need CGO on Windows — if fails, at least ensure package compiles with existing build script patterns; `go test` of bridgeapp is enough if GUI build needs MinGW)
- [ ] **Step 5: Commit**

```
feat(reader-gui): add HF gain slider defaulting to max
```

---

### Task 5: Brief ops note

**Files:**
- Modify: `docs/production-reader.md` (short bullet under GUI start)

- [ ] **Step 1:** Document that HF gain defaults to max (63) and maps to Proxmark `hw sethfthresh`; lower if false triggers.
- [ ] **Step 2: Commit**

```
docs: note Proxmark HF gain control in production reader guide
```
