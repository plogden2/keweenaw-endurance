import { expect, test } from '@playwright/test'
import {
  DEMO_RACER_NAMES,
  DEMO_TAG_12H,
  DEMO_TAG_6H,
  DEMO_TAG_KIDS,
  armFinishStation,
  getBluffetEvent,
  injectTag,
  pinLogin,
} from './fixtures/rfid'

/**
 * US1 — Live RFID lap timing at reader station.
 * Frontend hooks (routes / data-testid) are implemented; full pass still needs
 * backend scan stream + live endpoints (T027–T030).
 */
test.describe('US1 live lap timing', () => {
  test('arm station, inject tag → scan popup with name, place, laps; no playing-sound label', async ({
    page,
    request,
  }) => {
    const event = await getBluffetEvent(request)
    const token = await pinLogin(request)

    await page.goto('/pin')
    await expect(page.getByTestId('pin-form')).toBeVisible()
    await page.getByTestId('pin-input').fill('1738')
    await page.getByTestId('pin-submit').click()

    await page.goto('/station')
    await expect(page.getByTestId('station-config')).toBeVisible()
    await armFinishStation(request, token, event.id)
    await page.getByTestId('station-armed-indicator').waitFor({ state: 'visible' })

    await page.goto(`/events/${event.id}/live`)
    await expect(page.getByTestId('live-view')).toBeVisible()

    const injectRes = await injectTag(request, DEMO_TAG_12H)
    expect(injectRes.ok()).toBeTruthy()

    const popup = page.getByTestId('scan-popup')
    await expect(popup).toBeVisible({ timeout: 10_000 })
    await expect(popup.getByTestId('scan-racer-name')).toContainText(
      DEMO_RACER_NAMES.twelveHour,
    )
    await expect(popup.getByTestId('scan-placement')).toBeVisible()
    await expect(popup.getByTestId('scan-lap-count')).toBeVisible()

    // Mario Kart sound plays, but UI must not show a "playing sound" label
    await expect(page.getByTestId('scan-sound-playing')).toHaveCount(0)
    await expect(page.getByText(/playing sound/i)).toHaveCount(0)
  })

  test('same tag within cooldown shows cooldown message and does not add a lap', async ({
    page,
    request,
  }) => {
    const event = await getBluffetEvent(request)
    const token = await pinLogin(request)
    await armFinishStation(request, token, event.id)

    await page.goto('/pin')
    await pinLogin(page)
    await page.goto(`/events/${event.id}/live`)
    await expect(page.getByTestId('live-view')).toBeVisible()

    await injectTag(request, DEMO_TAG_12H)
    await expect(page.getByTestId('scan-popup')).toBeVisible({ timeout: 10_000 })
    const lapsAfterFirst = await page.getByTestId('scan-lap-count').innerText()

    await injectTag(request, DEMO_TAG_12H)
    await expect(page.getByTestId('cooldown-message')).toBeVisible({
      timeout: 5_000,
    })
    await expect(page.getByTestId('scan-lap-count')).toHaveText(lapsAfterFirst)
  })

  test('navigating away from live still processes injected tags', async ({
    page,
    request,
  }) => {
    const event = await getBluffetEvent(request)
    const token = await pinLogin(request)
    await armFinishStation(request, token, event.id)

    await page.goto('/pin')
    await pinLogin(page)
    await page.goto(`/events/${event.id}/live`)
    await expect(page.getByTestId('live-view')).toBeVisible()

    // Background reader must keep working on non-live routes
    await page.goto(`/events/${event.id}/racers`)
    await expect(page.getByTestId('racers-page')).toBeVisible()

    await injectTag(request, DEMO_TAG_12H)
    await expect(page.getByTestId('scan-popup')).toBeVisible({ timeout: 10_000 })
    await expect(page.getByTestId('scan-racer-name')).toContainText(
      DEMO_RACER_NAMES.twelveHour,
    )
  })

  test('pre-start live view shows countdown; kids tag is test-read only before start', async ({
    page,
    request,
  }) => {
    const event = await getBluffetEvent(request)
    const token = await pinLogin(request)
    await armFinishStation(request, token, event.id)

    await page.goto('/pin')
    await pinLogin(page)
    await page.goto(`/events/${event.id}/live`)
    await expect(page.getByTestId('live-view')).toBeVisible()
    await expect(page.getByTestId('live-countdown')).toBeVisible()

    // 90-Minute Kids starts later — inject should be test_read, not a scored lap
    await injectTag(request, DEMO_TAG_KIDS)
    await expect(page.getByTestId('test-read-message')).toBeVisible({
      timeout: 10_000,
    })
    await expect(page.getByTestId('scan-popup')).toHaveCount(0)
  })

  test('multi-race attribution: 12h and 6h tags score into their own races', async ({
    page,
    request,
  }) => {
    const event = await getBluffetEvent(request)
    const token = await pinLogin(request)
    await armFinishStation(request, token, event.id)

    await page.goto('/pin')
    await pinLogin(page)
    await page.goto(`/events/${event.id}/live`)
    await expect(page.getByTestId('live-view')).toBeVisible()

    await injectTag(request, DEMO_TAG_12H)
    await expect(page.getByTestId('scan-popup')).toBeVisible({ timeout: 10_000 })
    await expect(page.getByTestId('scan-race-name')).toContainText('12 Hour')
    await page.getByTestId('scan-popup-dismiss').click()

    await injectTag(request, DEMO_TAG_6H)
    await expect(page.getByTestId('scan-popup')).toBeVisible({ timeout: 10_000 })
    await expect(page.getByTestId('scan-racer-name')).toContainText(
      DEMO_RACER_NAMES.sixHour,
    )
    await expect(page.getByTestId('scan-race-name')).toContainText('6 Hour')
  })

  test('live tabs 12h / 6h / 90m, overlap chart, fullscreen rotator, overall + legend', async ({
    page,
    request,
  }) => {
    const event = await getBluffetEvent(request)

    await page.goto(`/events/${event.id}/live`)
    await expect(page.getByTestId('live-view')).toBeVisible()

    await page.getByTestId('race-tab-12h').click()
    await expect(page.getByTestId('race-panel-12h')).toBeVisible()

    await page.getByTestId('race-tab-6h').click()
    await expect(page.getByTestId('race-panel-6h')).toBeVisible()

    await page.getByTestId('race-tab-90m').click()
    await expect(page.getByTestId('race-panel-90m')).toBeVisible()

    await page.getByTestId('overlap-chart-toggle').click()
    await expect(page.getByTestId('overlap-chart')).toBeVisible()

    await page.getByTestId('fullscreen-rotator-toggle').click()
    await expect(page.getByTestId('fullscreen-rotator')).toBeVisible()
    await expect(page.getByTestId('rotator-flow')).toBeVisible()
    await expect(page.getByTestId('rotator-leaderboard')).toBeVisible()

    await expect(page.getByTestId('leaderboard-overall')).toBeVisible()
    await expect(page.getByTestId('category-legend')).toBeVisible()
  })

  test('opening app during active races lands on live race flow by default', async ({
    page,
    request,
  }) => {
    const event = await getBluffetEvent(request)
    const token = await pinLogin(request)
    await armFinishStation(request, token, event.id)

    await page.goto('/')
    await expect(page).toHaveURL(new RegExp(`/events/${event.id}/live`))
    await expect(page.getByTestId('live-view')).toBeVisible()
    await expect(page.getByTestId('leaderboard-overall')).toBeVisible()
  })
})
