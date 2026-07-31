# Racers inline edit + unassigned bib — Design

**Date:** 2026-07-31  
**Status:** Approved for implementation  
**Surface:** `/races/:raceId/racers` (`Racers.vue`)

## Goal

Race-day ops can assign bibs at pickup via an always-visible textbox for unassigned racers, edit name/category in an inline panel, and remove racers (hard delete if no taps; DNS if they have history).

## Decisions

| Topic | Choice |
|-------|--------|
| Unassigned bib UI | Permanent textbox when `bib_number` blank; Enter assigns |
| Assigned bib UI | Keep click-to-edit (not a permanent textbox) |
| Create without bib | Allowed — empty stays empty (no auto sequential) |
| Edit UX | Row **Edit** expands inline panel (like Program tag): name, category, Save, Cancel, Delete |
| Delete | Confirm → hard delete if no timing records; else set `status=dns` |
| DNS on list | Remain visible with muted DNS badge |

## Backend

1. **Create:** omit/blank `bib_number` → store `""`; skip `EnsureBib` / uniqueness for empty.
2. **Update:** assigning a bib from empty works; uniqueness still enforced for non-empty.
3. **Delete:** if any `timing_records` for participant → update `status=dns`, return participant JSON `{ action: "dns", participant }`; else hard delete `{ action: "deleted" }`.
4. **validateParticipantInput:** `bib_number` not required when empty string.

## Frontend

- Bib cell: input if blank; else existing click-to-edit.
- Actions: Edit + Program tag; mutually exclusive expand panels.
- Edit panel: first/last name, category, Save (PUT), Delete (DELETE + confirm).
- Add form: blank bib = unassigned (hint only).

## Out of scope

- Gender edit in panel
- Filtering DNS off live boards (follow-up if needed)
- Event Bibs inventory changes
