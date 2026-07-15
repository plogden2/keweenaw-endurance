import { defineConfig, devices } from '@playwright/test'
import path from 'node:path'

// `BLUFFET_HW_ARTIFACT_DIR` must be set BEFORE this config loads (env vars set
// from inside the spec, e.g. in a `test.beforeAll`, are too late for Node's
// module evaluation). `npm run test:e2e:bluffet-hardware` goes through
// `scripts/run-bluffet-hardware.mjs`, which creates the run dir and exports
// this var before spawning Playwright — the spec's `createRunDir` then reuses
// the same dir/runId instead of inventing a second one. Falls back to
// `current/` only for ad-hoc `npx playwright test` invocations.
const artifactDir =
  process.env.BLUFFET_HW_ARTIFACT_DIR ??
  path.join('..', '..', 'e2e-artifacts', 'bluffet-hardware', 'current')

export default defineConfig({
  testDir: './',
  fullyParallel: false,
  workers: 1,
  timeout: 45 * 60 * 1000,
  expect: { timeout: 15_000 },
  reporter: [['list'], ['json', { outputFile: path.join(artifactDir, 'playwright-report.json') }]],
  use: {
    baseURL: process.env.E2E_BASE_URL ?? 'http://localhost:3000',
    trace: 'retain-on-failure',
    screenshot: 'on',
    video: { mode: 'on', size: { width: 1920, height: 1080 } },
    // Long-running background loops (reader carousel, spectator churn) rely on
    // every action having a bound — without this, a single stuck locator would
    // otherwise block only on the 45min test timeout.
    actionTimeout: 15_000,
    navigationTimeout: 30_000,
  },
  projects: [{ name: 'hardware-bluffet', use: { ...devices['Desktop Chrome'] } }],
})
