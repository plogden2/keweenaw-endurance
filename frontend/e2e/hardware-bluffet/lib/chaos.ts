import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import type { Browser, BrowserContext, Page } from '@playwright/test'
import { armFinishStation, pinLogin, pinToken } from '../../fixtures/rfid'
import type { APIRequestContext } from '@playwright/test'

export const BRIDGE_LOCAL_URL = (
  process.env.VITE_BRIDGE_LOCAL_URL || 'http://127.0.0.1:8091'
).replace(/\/$/, '')

const DEFAULT_PARTITION_SIGNAL = path.join(os.tmpdir(), 'keweenaw-bridge-partition.signal')

/** Path watched by device-bridge (`BRIDGE_PARTITION_SIGNAL`) to force offline mode. */
export function partitionSignalPath(): string {
  return process.env.BRIDGE_PARTITION_SIGNAL || DEFAULT_PARTITION_SIGNAL
}

export type LocalBridgeStatus = {
  connected: boolean
  pending_count: number
  syncing: boolean
  mode?: string
  csv_path?: string
  pending_path?: string
}

export type SyncChipExpectation = 'offline' | 'syncing' | 'online_synced'

const CHIP_TEST_IDS: Record<SyncChipExpectation, string> = {
  offline: 'sync-offline',
  syncing: 'sync-syncing',
  online_synced: 'sync-online',
}

const CHIP_LABELS: Record<SyncChipExpectation, string> = {
  offline: 'Offline',
  syncing: 'Syncing',
  online_synced: 'Online · Synced',
}

/** Poll loopback bridge status (always reachable during a hosted partition). */
export async function fetchLocalBridgeStatus(): Promise<LocalBridgeStatus> {
  const res = await fetch(`${BRIDGE_LOCAL_URL}/status`)
  if (!res.ok) {
    throw new Error(`local bridge status ${res.status} ${await res.text()}`)
  }
  return (await res.json()) as LocalBridgeStatus
}

/** Count non-empty lines in bridge `pending.jsonl` (fallback when HTTP is unavailable). */
export function countPendingLines(pendingPath: string): number {
  if (!fs.existsSync(pendingPath)) return 0
  return fs
    .readFileSync(pendingPath, 'utf8')
    .split('\n')
    .filter((line) => line.trim().length > 0).length
}

/**
 * Real outage: cut bridge→hosted (signal file) and take browser contexts offline.
 * Local bridge loopback (write-tag, poll, pending queue) keeps running.
 */
export async function partitionFromHosted(
  contexts: BrowserContext[],
  opts: { signalPath?: string } = {},
) {
  const signal = opts.signalPath ?? partitionSignalPath()
  fs.mkdirSync(path.dirname(signal), { recursive: true })
  fs.writeFileSync(signal, `${new Date().toISOString()}\n`, 'utf8')
  process.env.BRIDGE_PARTITION_SIGNAL = signal
  for (const ctx of contexts) await ctx.setOffline(true)
}

/** Restore connectivity; bridge auto-flushes pending laps — no import.csv. */
export async function healPartition(
  contexts: BrowserContext[],
  opts: { signalPath?: string } = {},
) {
  const signal = opts.signalPath ?? partitionSignalPath()
  try {
    fs.unlinkSync(signal)
  } catch {
    // already removed
  }
  delete process.env.BRIDGE_PARTITION_SIGNAL
  for (const ctx of contexts) await ctx.setOffline(false)
}

/** Wait for EventLive reader sync chip to reach the expected state. */
export async function waitForReaderChip(
  page: Page,
  expected: SyncChipExpectation,
  timeoutMs = 60_000,
): Promise<void> {
  const testId = CHIP_TEST_IDS[expected]
  const label = CHIP_LABELS[expected]
  const chip = page.getByTestId(testId)
  await chip.waitFor({ state: 'visible', timeout: timeoutMs })
  const text = (await chip.innerText()).trim()
  if (!text.includes(label.split(' · ')[0])) {
    throw new Error(`expected sync chip "${label}", got "${text}"`)
  }
}

export type VideoContext = {
  context: BrowserContext
  page: Page
}

/**
 * Simulates a reader browser crash: hard-close the context (killing the
 * in-flight video), then open a fresh context + page with a NEW video
 * recording, re-arm the finish station (station config is server-side and
 * persists, but the UI needs a fresh mount), and re-open the live view.
 *
 * Returns the new { context, page } — caller must swap its `reader` reference
 * and remember to merge the old + new video files at the end (see artifacts).
 */
export async function crashAndReopenReader(opts: {
  browser: Browser
  request: APIRequestContext
  eventId: string
  videoDir: string
  deviceId?: string
}): Promise<VideoContext> {
  const token = await pinToken(opts.request)
  await armFinishStation(opts.request, token, opts.eventId, opts.deviceId ?? 'laptop-finish-1')

  const context = await opts.browser.newContext({
    viewport: { width: 1920, height: 1080 },
    recordVideo: { dir: opts.videoDir, size: { width: 1920, height: 1080 } },
  })
  const page = await context.newPage()
  // Reader RFID/ScanPopup is gated on PIN session — re-unlock after crash.
  await page.goto('/pin')
  await pinLogin(page)
  await page.goto(`/events/${opts.eventId}/live`)
  await page.getByTestId('live-view').waitFor({ state: 'visible', timeout: 30_000 })
  return { context, page }
}
