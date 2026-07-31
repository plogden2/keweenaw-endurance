import { expect, test, type Page, type Route } from '@playwright/test'

/**
 * Plot reliability e2e — API-mocked so CI/local can verify chart UX without Docker.
 * Covers sticky highlight, toolbar zoom, overlap dual-mount, participant chart,
 * and mobile viewport stability.
 */

const EVENT_ID = 'evt-plot-e2e'
const RACE_12H = 'race-12h'
const RACE_6H = 'race-6h'
const RACE_90M = 'race-90m'
const P1 = 'p-alex'
const P2 = 'p-sam'

const startIso = '2026-08-01T08:00:00.000Z'
const lap1Iso = '2026-08-01T09:00:00.000Z'
const lap2Iso = '2026-08-01T10:00:00.000Z'

function json(route: Route, body: unknown, status = 200): Promise<void> {
  return route.fulfill({
    status,
    contentType: 'application/json',
    body: JSON.stringify(body),
  })
}

function participant(id: string, bib: string, first: string, last: string) {
  return {
    id,
    race_id: RACE_12H,
    bib_number: bib,
    first_name: first,
    last_name: last,
    gender: 'male',
    age: 32,
    status: 'started',
  }
}

function timingRecord(
  id: string,
  participantId: string,
  timestamp: string,
  person: ReturnType<typeof participant>,
) {
  return {
    id,
    participant_id: participantId,
    checkpoint_id: 'cp-finish',
    timestamp,
    local_timestamp: timestamp,
    sync_status: 'synced',
    participant: person,
    checkpoint: {
      id: 'cp-finish',
      race_id: RACE_12H,
      name: 'Finish',
      checkpoint_type: 'finish',
      distance_from_start_km: 0,
      is_active: true,
    },
  }
}

const alex = participant(P1, '1', 'Alex', 'Rivera')
const sam = participant(P2, '2', 'Sam', 'Trail')

const liveEventPayload = {
  event: { id: EVENT_ID, name: 'Plot Reliability Cup' },
  category_legend: [
    { key: 'advanced_men', label: 'Advanced Men', color: '#1a5276' },
  ],
  races: [
    {
      id: RACE_12H,
      name: '12 Hour',
      race_type: 'lap_based',
      status: 'active',
      start_time: startIso,
      duration_minutes: 720,
      countdown_seconds: 0,
      leaderboard_overall: [
        {
          place: 1,
          participant_id: P1,
          bib_number: '1',
          name: 'Alex Rivera',
          category_key: 'advanced_men',
          laps: 2,
          last_lap_at: lap2Iso,
        },
        {
          place: 2,
          participant_id: P2,
          bib_number: '2',
          name: 'Sam Trail',
          category_key: 'advanced_men',
          laps: 1,
          last_lap_at: lap1Iso,
        },
      ],
      leaderboard_teams: [],
      flow_series: [],
    },
    {
      id: RACE_6H,
      name: '6 Hour',
      race_type: 'lap_based',
      status: 'active',
      start_time: startIso,
      duration_minutes: 360,
      countdown_seconds: 0,
      leaderboard_overall: [],
      leaderboard_teams: [],
      flow_series: [],
    },
    {
      id: RACE_90M,
      name: '90-Minute Kids',
      race_type: 'lap_based',
      status: 'scheduled',
      start_time: '2026-08-01T15:00:00.000Z',
      duration_minutes: 90,
      countdown_seconds: 10_000,
      leaderboard_overall: [],
      leaderboard_teams: [],
      flow_series: [],
    },
  ],
}

const timingLiveByRace: Record<string, unknown> = {
  [RACE_12H]: {
    race_id: RACE_12H,
    records: [
      timingRecord('t1', P1, lap1Iso, alex),
      timingRecord('t2', P1, lap2Iso, alex),
      timingRecord('t3', P2, lap1Iso, { ...sam, race_id: RACE_12H }),
    ],
  },
  [RACE_6H]: {
    race_id: RACE_6H,
    records: [
      timingRecord('t6', P1, lap1Iso, { ...alex, id: 'p6', race_id: RACE_6H }),
    ],
  },
  [RACE_90M]: { race_id: RACE_90M, records: [] },
}

async function installPlotMocks(page: Page): Promise<void> {
  await page.route('**/api/**', async (route) => {
    const url = new URL(route.request().url())
    const path = url.pathname
    const method = route.request().method()

    if (method === 'GET' && path === `/api/events/${EVENT_ID}/live`) {
      return json(route, liveEventPayload)
    }
    if (method === 'GET' && /^\/api\/timing\/live\//.test(path)) {
      const raceId = path.split('/').pop()!
      return json(route, timingLiveByRace[raceId] ?? { race_id: raceId, records: [] })
    }
    if (method === 'GET' && /^\/api\/races\/[^/]+\/participants$/.test(path)) {
      const raceId = path.split('/')[3]
      const people =
        raceId === RACE_12H
          ? [alex, sam]
          : raceId === RACE_6H
            ? [{ ...alex, id: 'p6', race_id: RACE_6H }]
            : []
      return json(route, { data: people, total: people.length })
    }
    if (method === 'GET' && path === `/api/races/${RACE_12H}`) {
      return json(route, {
        id: RACE_12H,
        event_id: EVENT_ID,
        name: '12 Hour',
        race_type: 'lap_based',
        status: 'active',
        start_time: startIso,
        duration_minutes: 720,
      })
    }
    if (method === 'GET' && path === `/api/timing/leaderboard/${RACE_12H}`) {
      return json(route, {
        data: [
          {
            participant_id: P1,
            bib_number: '1',
            first_name: 'Alex',
            last_name: 'Rivera',
            position: 1,
            status: 'finished',
            result_value: 2,
          },
        ],
        total: 1,
      })
    }
    if (method === 'GET' && path.includes('/rfid/sync-status')) {
      return json(route, { pending_count: 0, failed_count: 0, synced_count: 0 })
    }
    if (method === 'GET' && path.includes('/rfid/bridge')) {
      return json(route, { connected: true, pending_count: 0, syncing: false })
    }
    if (method === 'GET' && path.startsWith('/api/events/') && path.endsWith('/live-csv')) {
      return json(route, { path: '', updated_at: null })
    }

    // Default: empty success for incidental calls (auth ping, etc.)
    if (method === 'GET') {
      return json(route, { data: [], total: 0 })
    }
    return json(route, { ok: true })
  })
}

async function openLiveRaceFlow(page: Page): Promise<void> {
  await installPlotMocks(page)
  await page.goto(`/events/${EVENT_ID}/live`)
  await expect(page.getByTestId('live-view')).toBeVisible()
  await expect(page.getByTestId('race-tab-12h')).toBeVisible()
  await page.getByTestId('race-tab-12h').click()
  await expect(page.getByTestId('race-panel-12h')).toBeVisible()
  await expect(page.getByTestId('race-flow-chart')).toBeVisible({ timeout: 20_000 })
  await expect(page.getByTestId('race-flow-canvas')).toBeVisible()
}

async function expectCanvasDrawn(page: Page, testId = 'race-flow-canvas'): Promise<void> {
  const canvas = page.getByTestId(testId).first()
  await expect(canvas).toBeVisible()
  await expect
    .poll(async () => {
      return canvas.evaluate((node) => {
        const el = node as HTMLCanvasElement
        if (el.width < 8 || el.height < 8) {
          return false
        }
        const ctx = el.getContext('2d')
        if (!ctx) {
          return false
        }
        const sample = ctx.getImageData(
          0,
          0,
          Math.min(el.width, 64),
          Math.min(el.height, 64),
        ).data
        for (let i = 3; i < sample.length; i += 4) {
          if (sample[i] > 0) {
            return true
          }
        }
        return false
      })
    }, { timeout: 15_000 })
    .toBe(true)
}

async function readZoomDomain(page: Page): Promise<{ min: number; max: number }> {
  const host = page.getByTestId('race-flow-zoom-domain').first()
  await expect(host).toBeVisible()
  const min = Number(await host.getAttribute('data-zoom-min'))
  const max = Number(await host.getAttribute('data-zoom-max'))
  expect(Number.isFinite(min)).toBe(true)
  expect(Number.isFinite(max)).toBe(true)
  return { min, max }
}

test.describe('Race flow plots', () => {
  test('legend sticky highlight toggles via select control', async ({ page }) => {
    await openLiveRaceFlow(page)
    await expect(page.getByTestId('race-flow-legend')).toBeVisible()

    const chart = page.getByTestId('race-flow-chart')
    const select = page.getByRole('button', { name: /#1 Alex Rivera/ })
    await expect(select).toBeVisible()
    await select.click()
    await expect(chart).toHaveAttribute('data-highlight-participant-id', P1)
    await expect(select).toHaveAttribute('aria-pressed', 'true')

    await select.click()
    await expect(chart).toHaveAttribute('data-highlight-participant-id', '')
    await expect(select).toHaveAttribute('aria-pressed', 'false')
  })

  test('toolbar zoom changes x window and reset restores full domain', async ({ page }) => {
    await openLiveRaceFlow(page)
    await expect(page.getByTestId('race-flow-zoom-toolbar')).toBeVisible()
    await expectCanvasDrawn(page)

    const before = await readZoomDomain(page)
    expect(before.max - before.min).toBeGreaterThan(1)

    await page.getByTestId('race-flow-zoom-in').click()
    await expect
      .poll(async () => {
        const after = await readZoomDomain(page)
        return after.max - after.min
      })
      .toBeLessThan(before.max - before.min)

    await page.getByTestId('race-flow-zoom-reset').click()
    await expect
      .poll(async () => {
        const reset = await readZoomDomain(page)
        return Math.abs(reset.min - before.min) < 0.01 && Math.abs(reset.max - before.max) < 0.01
      })
      .toBe(true)
  })

  test('overlap mounts two charts without crashing', async ({ page }) => {
    await installPlotMocks(page)
    await page.goto(`/events/${EVENT_ID}/live`)
    await expect(page.getByTestId('live-view')).toBeVisible()

    await page.getByTestId('overlap-chart-toggle').click()
    await expect(page.getByTestId('overlap-chart')).toBeVisible()

    const charts = page.getByTestId('race-flow-chart')
    await expect(charts).toHaveCount(2)
    await expect(page.getByTestId('race-flow-canvas')).toHaveCount(2)

    const chart = page.getByTestId('race-flow-chart').first()
    const select = chart.getByRole('button', { name: /#1 Alex Rivera/ })
    await select.click()
    await expect(chart).toHaveAttribute('data-highlight-participant-id', P1)
    await expect(page.getByTestId('live-view')).toBeVisible()
  })

  test('race details race flow chart stays drawn with zoom toolbar', async ({ page }) => {
    await installPlotMocks(page)
    await page.goto(`/timing/${EVENT_ID}/race/${RACE_12H}`)
    await expect(page.locator('.race-details')).toBeVisible({ timeout: 20_000 })

    await page.getByRole('button', { name: 'Race Flow' }).click()
    await expect(page.getByTestId('race-flow-chart')).toBeVisible({ timeout: 20_000 })
    await expect(page.getByTestId('race-flow-zoom-toolbar')).toBeVisible()
    await expectCanvasDrawn(page)
  })

  test('mobile viewport: legend sticky works and canvas stays drawn', async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 })
    await openLiveRaceFlow(page)
    await expectCanvasDrawn(page)

    const chart = page.getByTestId('race-flow-chart')
    const select = page.getByRole('button', { name: /#1 Alex Rivera/ })
    await select.scrollIntoViewIfNeeded()
    await select.click()
    await expect(chart).toHaveAttribute('data-highlight-participant-id', P1)
    await expect(chart).toBeVisible()
    await expectCanvasDrawn(page)
  })
})
