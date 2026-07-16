# Default site theme: Superior Forest + copper

**Date:** 2026-07-16  
**Status:** Draft — awaiting user review before implementation plan  

**Inspiration:** [Visit Keweenaw](https://www.visitkeweenaw.com/) outdoor greens (adapted, not copied) + copper rock hues from a Keweenaw shoreline photo. No Visit Keweenaw logo, fonts, photography, or layout recreation.

## Goal

Replace the current generic default UI (system fonts, `#3498db` / `#2c3e50` / `#f5f7fa`) with a Keweenaw-rooted **default** visual system: evergreen primary, shoreline copper accents, Outfit typography, and a soft landscape page wash. Event-specific themes (Bluffet) continue to override via `#app.theme-*` and must keep working unchanged in behavior.

## Approach

**Full restyle pass (Approach 2):** introduce CSS tokens as the source of truth, then replace hardcoded default chrome colors across shared styles and Vue SFCs in one implementation effort—not a token-only partial migration.

## Visual system

### Tokens

| Token | Value | Role |
|-------|-------|------|
| `--ink` / `--accent` | `#1a3f3d` | Primary brand, titles, primary buttons, header |
| `--ink-deep` | `#203429` | Deeper text / primary hover |
| `--accent-link` | `#2f6b5a` | Mid-tone links, selected chrome, former `#1a5276` UI accents |
| `--sage` | `#a1b383` | `:focus-visible` ring, soft accents |
| `--copper` | `#9b654e` | Warm accent (replaces Visit-inspired gold); wash tint source |
| `--mist` | `#eff1f0` | Default page base / input fills |
| `--signal` | `#c45c38` | Errors, destructive, cancelled, former CTA reds, chart “now” line |
| `--success` | `#27ae60` | Positive / sync OK / recorded — distinct from brand greens |
| `--muted` | `#6b7a76` | Secondary text, secondary buttons |
| `--border` | `#d4dad7` | Inputs, dividers, card edges |
| `--surface` | `#ffffff` | Cards/panels on the wash |

**Wash:** soft mist → copper-tint (`#e8d5c8` family) on **page background only** (`body` / `#app`). Panels stay `--surface` for readability.

**Typography:** Outfit 400 / 600 / 700 loaded once. Map existing `font-weight: 500` → `600` (do not add Outfit 500).

**No Superior blue** in the default brand palette (clashes with evergreen).

### Token roles in UI

- Primary actions / header / titles → `--accent` / `--ink`
- Links / mid interactive chrome → `--accent-link`
- Secondary / featured warm highlight → `--copper`
- Borders / focus → `--border` / `--sage`
- Errors / destructive / cancelled / unified former reds → `--signal`
- Success states → `--success`
- Body text → `--ink` with muted variants via `--muted`

## Architecture

1. Define the token block and base shell styles in `frontend/src/assets/main.css` (`:root` / `body` / shared `.btn*`, forms, cards, focus, loading spinner).
2. Apply soft landscape wash on the page shell only; panels use `--surface`.
3. Sweep Vue scoped styles and shared components that use **default chrome** hexes onto tokens.
4. Retint category legend / chart series colors from the old blue family into an evergreen / copper / sage set (distinct per category; no Superior blue). Exact series hexes are chosen in the implementation plan from this family so categories stay distinguishable.
5. Restyle result certificates and share/export chrome to default tokens.
6. Update PWA `theme_color` in `vite.config.ts` to evergreen (`--ink`).
7. Keep `frontend/src/themes/bluffet.css` and Bluffet activation as the event override layer; Bluffet must still win under `#app.theme-bluffet`.

### Hex → token map (implementation)

| Hex (typical use) | Map to |
|-------------------|--------|
| `#3498db`, `#2980b9` | `--ink` / `--accent` or `--accent-link` by role |
| `#2c3e50` chrome / header | `--ink` / `--ink-deep` |
| `#f5f7fa` | `--mist` |
| `#1a5276` UI accents / local `--accent` fallbacks | `--accent-link` |
| `#1a5276` category legend / series | Retint to evergreen/copper/sage family (new distinct values) |
| `#e74c3c`, `#c0392b`, and similar UI reds | `--signal` |
| `#27ae60` | `--success` |
| `#95a5a6`, `#7f8c8d` chrome | `--muted` |
| `#ddd` borders | `--border` |
| Certificate toolbar / export darks (`#2c3e50`, `#152536`) | `--ink` / `--ink-deep` |
| PWA `theme_color` | evergreen / `--ink` |

### Do not flatten

- Bluffet tokens and rules under `#app.theme-bluffet`
- Visit Keweenaw assets, fonts, or copied layout
- `frontend/prototypes/` unless explicitly included later
- Print `@media print` high-contrast black/white neutrals (on-screen certificate UI/export uses tokens; print may stay high-contrast)

## Scope

### In scope

- Token block + Outfit + page wash in `main.css`
- Shared chrome: `AppHeader`, buttons, forms, cards, spinner, focus rings
- Full default-color sweep across views/components listed in planning (Home, events, timing, racers, station/PIN, CSV recovery, ScanPopup, SyncStatus, RaceCard, ManualTimingForm, RaceFlowChart chrome + series retint, ResultCertificate default + export)
- Category legend / chart series retint into brand family
- PWA `theme_color`
- Tests: update assertions of default chrome and retinted legend colors; keep Bluffet CSS/e2e theme checks as guardrails
- Smoke: one non-Bluffet and one Bluffet path (shell + live/admin)

### Out of scope

- Visit Keweenaw logo, fonts, photography, copy, or layout recreation
- Changing Bluffet theme behavior beyond ensuring overrides still apply
- New per-event themes beyond Bluffet
- Layout/content restructuring unrelated to color/type/tokens
- Motion redesign beyond respecting existing `prefers-reduced-motion`
- Backend category color APIs beyond frontend retint of displayed legend/series values used by the UI

## Testing & constraints

**Verify**

- Bluffet unit/CSS expectations and Bluffet e2e theme checks still pass
- Update only tests that assert default chrome colors or retinted legend/series colors
- Smoke non-Bluffet vs Bluffet: green primary + copper accents on default; Bluffet red/display fonts still win when active

**Constraints**

- No Visit Keweenaw assets
- Focus via sage `:focus-visible`; check contrast for ink-on-mist, white-on-ink, copper accents, sage ring on mist and ink
- Soft wash is static atmosphere; do not introduce motion that bypasses `prefers-reduced-motion`
- `--success` stays a distinct green; brand evergreen is not used as the sole success cue
- All former UI reds unify to `--signal` this pass

## Decisions log

| Topic | Decision |
|-------|----------|
| Depth | Tokens + visual language (not a rip-off) |
| Implementation | Full restyle pass |
| Palette base | Superior Forest evergreen |
| Warm accent | Photo copper `#9b654e` (not gold; no blue brand accent) |
| Type | Outfit; map 500→600 |
| Surfaces | Soft landscape wash on page background only |
| Legend/series | Retint into evergreen/copper/sage |
| Accent fallbacks | Separate `--accent-link #2f6b5a` |
| Success | Distinct `--success #27ae60` |
| Reds | Unify to `--signal` |
| Certificates | In scope |
| PWA theme_color | Evergreen |
| Default coverage | All non-Bluffet routes/events |

## Success criteria

1. Non-Bluffet UI reads as evergreen + copper + Outfit with mist/copper wash behind surface panels.
2. Hardcoded default blues/slates for chrome are gone in favor of tokens.
3. Bluffet event views still apply `theme-bluffet` visuals.
4. Legends/charts no longer use Superior-blue chrome leftovers; series use the brand family.
5. Certificates/export and PWA chrome match the default brand.
6. Existing Bluffet theme tests/e2e remain green.
