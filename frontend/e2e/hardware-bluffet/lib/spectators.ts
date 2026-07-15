import type { APIRequestContext, Page } from '@playwright/test'
import { API_BASE, pinToken } from '../../fixtures/rfid'
import type { Racer } from './roster'

export type Spectator = {
  name: string
  eventId: string
  /** Race whose live-view tab is checked for catch-up (must be the 12h tab —
   *  it's the only panel wired with `leaderboard-overall`/`leaderboard-laps`
   *  testids in EventLive.vue; the 6h/90m panels render plain tables). */
  raceId: string
  friends: Racer[]
  page: Page
}

/** Deterministic per-spectator pick of 5 "friends" to search/track. */
export function pickFriends(racers: Racer[], seed: number, n = 5): Racer[] {
  const arr = [...racers].sort((a, b) => a.id.localeCompare(b.id))
  const out: Racer[] = []
  let x = seed
  const used = new Set<number>()
  while (out.length < Math.min(n, arr.length)) {
    x = (x * 1103515245 + 12345) & 0x7fffffff
    const idx = x % arr.length
    if (used.has(idx)) continue
    used.add(idx)
    out.push(arr[idx])
  }
  return out
}

export function createSpectator(opts: {
  name: string
  eventId: string
  raceId: string
  page: Page
  friends: Racer[]
}): Spectator {
  return { ...opts }
}

const RACE_TABS = ['race-tab-12h', 'race-tab-6h', 'race-tab-90m'] as const

/**
 * One "churn" tick: search a random friend on the racers page, then bounce
 * through live-view race tabs / leaderboard / fullscreen rotator. Best-effort —
 * swallows navigation errors so one flaky tick doesn't kill the whole run.
 */
export async function churnOnce(spec: Spectator): Promise<void> {
  const { page, friends, eventId } = spec
  try {
    const friend = friends[Math.floor(Math.random() * friends.length)]
    await page.goto(`/races/${friend.raceId}/racers`, { timeout: 15_000 })
    const search = page.getByTestId('racers-search')
    await search.waitFor({ state: 'visible', timeout: 10_000 })
    await search.fill(friend.lastName)
    await page.waitForTimeout(400)
    await search.fill('')

    await page.goto(`/events/${eventId}/live`, { timeout: 15_000 })
    await page.getByTestId('live-view').waitFor({ state: 'visible', timeout: 10_000 })

    const tab = RACE_TABS[Math.floor(Math.random() * RACE_TABS.length)]
    await page.getByTestId(tab).click({ timeout: 5_000 }).catch(() => {})

    if (Math.random() < 0.3) {
      await page.getByTestId('fullscreen-rotator-toggle').click({ timeout: 5_000 }).catch(() => {})
      await page.waitForTimeout(300)
      await page.keyboard.press('Escape').catch(() => {})
    }
  } catch {
    // Best-effort churn — caller/orchestrator logs issues from higher-level assertions instead.
  }
}

/** Runs churn ticks roughly every `intervalMs` until `stop()` resolves true. */
export async function runChurnLoop(
  spec: Spectator,
  opts: { intervalMs: number; shouldStop: () => boolean },
): Promise<void> {
  while (!opts.shouldStop()) {
    await churnOnce(spec)
    await new Promise((r) => setTimeout(r, opts.intervalMs))
  }
}

/**
 * Snapshot of the visible leaderboard-overall lap counts. `leaderboard-overall`
 * / `leaderboard-laps` testids only exist in EventLive.vue's 12h panel (the
 * 6h/90m panels render plain, untagged tables) — force that tab first so this
 * works regardless of where spectator churn last left the page.
 */
export async function snapshotVisibleLaps(page: Page): Promise<number> {
  await page
    .getByTestId('race-tab-12h')
    .click({ timeout: 5_000 })
    .catch(() => {})
  const cells = page.getByTestId('leaderboard-laps')
  const count = await cells.count()
  let total = 0
  for (let i = 0; i < count; i++) {
    const text = (await cells.nth(i).innerText()).trim()
    total += Number(text) || 0
  }
  return total
}

/** Always-online ground truth (via the `request` fixture, never taken offline). */
export async function serverLapsTotal(request: APIRequestContext, raceId: string): Promise<number> {
  const token = await pinToken(request)
  const res = await request.get(`${API_BASE}/api/timing/leaderboard/${raceId}`, {
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!res.ok()) throw new Error(`leaderboard ${raceId}: ${res.status()} ${await res.text()}`)
  const body = await res.json()
  const rows = body.data ?? body
  return (Array.isArray(rows) ? rows : []).reduce(
    (sum: number, r: { laps?: number }) => sum + (r.laps ?? 0),
    0,
  )
}

/**
 * After restoring connectivity, poll the spectator's live view until its
 * visible lap total catches up to the always-online server total (or timeout).
 * Returns { caughtUp, uiTotal, serverTotal } for the caller to log/assert.
 */
export async function awaitCatchUp(
  spec: Spectator,
  request: APIRequestContext,
  opts: { timeoutMs?: number; pollMs?: number } = {},
): Promise<{ caughtUp: boolean; uiTotal: number; serverTotal: number }> {
  const timeoutMs = opts.timeoutMs ?? 30_000
  const pollMs = opts.pollMs ?? 2_000
  const deadline = Date.now() + timeoutMs
  const serverTotal = await serverLapsTotal(request, spec.raceId)
  let uiTotal = 0
  while (Date.now() < deadline) {
    await spec.page.reload({ timeout: 10_000 }).catch(() => {})
    await spec.page
      .getByTestId('live-view')
      .waitFor({ state: 'visible', timeout: 10_000 })
      .catch(() => {})
    uiTotal = await snapshotVisibleLaps(spec.page).catch(() => 0)
    if (uiTotal >= serverTotal) return { caughtUp: true, uiTotal, serverTotal }
    await new Promise((r) => setTimeout(r, pollMs))
  }
  return { caughtUp: false, uiTotal, serverTotal }
}
