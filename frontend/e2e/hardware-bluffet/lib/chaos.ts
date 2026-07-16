import type { Browser, BrowserContext, Page } from '@playwright/test'
import { armFinishStation, pinLogin, pinToken } from '../../fixtures/rfid'
import type { APIRequestContext } from '@playwright/test'

/**
 * "Client cannot reach the API" outage: block browser HTTP/WS on the given
 * contexts. The `request` fixture (used by the lap engine + assertions) stays
 * online, and the backend keeps polling Proxmark into local Postgres — only
 * these browser contexts (spectators, optionally reader UI) go stale.
 */
export async function startApiOutage(contexts: BrowserContext[]) {
  for (const ctx of contexts) await ctx.setOffline(true)
}

export async function endApiOutage(contexts: BrowserContext[]) {
  for (const ctx of contexts) await ctx.setOffline(false)
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
