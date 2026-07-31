# Station under PIN management

**Date:** 2026-07-31  
**Status:** Approved

## Goal

Stop advertising Station config in global chrome. Operators reach it from PIN management only.

## Changes

1. Remove header **Station** link (`AppHeader.vue`).
2. Remove footer **Station** link (`App.vue`).
3. Keep **Station config** under PIN → Other management (`PinUnlock.vue`); add `data-testid="mgmt-station-config"` for tests.
4. Keep `/station` route and `StationConfig.vue` public (direct URL / e2e unchanged).
5. Update unit tests and production-reader docs that say header → Station.

## Out of scope

- PIN-gating the `/station` route
- Changing StationConfig form behavior
