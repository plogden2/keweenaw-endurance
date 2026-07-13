# HTML prototypes — RFID Race Scanner (`002-rfid-race-scanner`)

Constitution (Principle V): HTML prototypes approved for Vue implementation with the decisions below.

## Open in browser

| Screen | File | Decision |
|--------|------|----------|
| Racers | [racers.html](racers.html) | **A** table-first; inline program; bib click-to-edit; **debounced live search** |
| Event live | [event-live.html](event-live.html) | Tabs 12h/6h/90m; overlap chart; **overall + colors/legend**; **fullscreen rotate** |
| Scan popup | [scan-popup.html](scan-popup.html) | No sound label; karaoke button → “recorded” state (one bonus) |
| PIN unlock | [pin-unlock.html](pin-unlock.html) | **A** keypad + field (`1738`); **race create/delete** on unlocked management (no separate race-crud page) |
| Station config | [station-config.html](station-config.html) | **A** single form |
| CSV recovery | [csv-import-export.html](csv-import-export.html) | **A** layout; **live CSV always maintained** |
| Status | — | **Approved for Vue implementation** (2026-07-12) |

Shared styles: [`prototype.css`](prototype.css)
