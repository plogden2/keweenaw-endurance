# Audio assets

## Files

- `new-lap.mp3` — short cue played by `ScanPopup` when a scored lap is recorded. There is **no** on-screen “playing sound” label (product requirement).

## Rights / licensing note

**Do not ship Nintendo / Mario Kart audio** (or other third-party game IP) in this repository or in production builds without an explicit written license.

The checked-in `new-lap.mp3` is an **approved equivalent placeholder** (silent or original short cue) suitable for demos and CI. To replace it with a custom cue:

1. Use original audio you own, or a clearly licensed stock/SFX asset with redistribution rights for this project.
2. Keep the filename `new-lap.mp3` (or update the import in `ScanPopup.vue`).
3. Prefer a short clip (&lt;1s) so feedback stays within the ≤2s lap confirmation window (SC-001).

If a future licensed Mario Kart–style pack is approved by the project owner, document the license file path and attribution here before swapping the binary.
