import { defineConfig, devices } from '@playwright/test'
import path from 'node:path'

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
  },
  projects: [{ name: 'hardware-bluffet', use: { ...devices['Desktop Chrome'] } }],
})
