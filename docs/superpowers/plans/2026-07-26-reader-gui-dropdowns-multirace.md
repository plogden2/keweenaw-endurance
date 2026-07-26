# Reader GUI Dropdowns + Multi-Race Implementation Plan

> **For agentic workers:** Execute task-by-task. Advisor decisions baked in.

**Goal:** Event/race/checkpoint dropdowns, tap shows name+UUID, All-races finish mode, manual bib auto-resolve with race override.

**Architecture:** Expand event roster cache; bridge WS `scan_result` to device; Fyne selects + last-tap label.

---

### Task 1: Roster event-wide + ambiguity — done
### Task 2: BridgeMessage Scan + hub SendToDevice + handler — done
### Task 3: bridgeapp Status last_tap + ManualEntry + RefreshRoster — done
### Task 4: Catalog fetch + ApplyBluffetDefaults All-races — done
### Task 5: reader-gui UI selects + display — done
### Task 6: Tests + rebuild reader-gui — done
