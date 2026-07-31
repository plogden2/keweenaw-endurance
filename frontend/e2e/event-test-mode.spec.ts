import { expect, test } from '@playwright/test'
import {
  API_BASE,
  BLUFFET,
  DEMO_BIB_12H,
  DEMO_TAG_12H,
  armFinishStation,
  getBluffetEvent,
  injectTag,
  pinLogin,
} from './fixtures/rfid'

/**
 * Event Test Mode — intercepts RFID/manual into ephemeral board; no production pollution.
 */
test.describe('Event test mode', () => {
  test('open test mode, inject tag, assert live standings unchanged, discard on close', async ({
    page,
    request,
  }) => {
    const event = await getBluffetEvent(request)
    const token = await pinLogin(request)
    await armFinishStation(request, token, event.id)

    await page.goto('/pin')
    await pinLogin(page)
    await expect(page.getByTestId('footer-pin')).toContainText(/Unlocked/i, { timeout: 10_000 })

    await page.goto(`/events/${event.id}/live`)
    await expect(page.getByTestId('live-view')).toBeVisible()
    await expect(page.getByTestId('live-ops-links')).toBeVisible()

    // Capture a baseline lap cell if present (may be empty).
    const liveLapsBefore = await page
      .getByTestId('leaderboard-overall')
      .locator('[data-testid="leaderboard-laps"]')
      .allTextContents()
      .catch(() => [] as string[])

    await page.getByTestId('live-open-test-mode').click()
    const dialog = page.getByTestId('event-test-mode-dialog')
    await expect(dialog).toBeVisible()
    await expect(page.getByTestId('test-mode-banner')).toBeVisible()

    const injectRes = await injectTag(request, DEMO_TAG_12H)
    expect(injectRes.ok()).toBeTruthy()

    await expect(page.getByTestId('test-mode-feedback')).toBeVisible({ timeout: 10_000 })
    await expect(page.getByTestId('test-mode-leaderboard-row').first()).toBeVisible()
    await expect(page.getByTestId('scan-popup')).toHaveCount(0)

    // Production live board must not change from this inject.
    const liveLapsAfter = await page
      .getByTestId('leaderboard-overall')
      .locator('[data-testid="leaderboard-laps"]')
      .allTextContents()
      .catch(() => [] as string[])
    expect(liveLapsAfter).toEqual(liveLapsBefore)

    // Rapid second tap still accepted in test mode (no cooldown).
    await injectTag(request, DEMO_TAG_12H)
    await expect
      .poll(async () => {
        const laps = await page.getByTestId('test-mode-leaderboard-laps').first().innerText()
        return Number(laps)
      })
      .toBeGreaterThanOrEqual(2)

    await page.getByTestId('test-mode-close').click()
    await page.getByTestId('test-mode-discard-confirm-btn').click()
    await expect(dialog).toHaveCount(0)

    // Re-open starts empty.
    await page.getByTestId('live-open-test-mode').click()
    await expect(page.getByTestId('event-test-mode-dialog')).toBeVisible()
    await expect(page.getByTestId('test-mode-leaderboard')).toContainText('No test taps yet')
  })

  test('manual bib in test mode updates board without creating event taps', async ({
    page,
    request,
  }) => {
    const event = await getBluffetEvent(request)
    const token = await pinLogin(request)
    await armFinishStation(request, token, event.id)

    await page.goto('/pin')
    await pinLogin(page)
    await expect(page.getByTestId('footer-pin')).toContainText(/Unlocked/i, { timeout: 10_000 })

    const tapsBefore = await request.get(
      `${API_BASE}/api/events/${event.id}/taps?limit=1`,
      { headers: { Authorization: `Bearer ${token}` } },
    )
    const tapsBeforeJson = await tapsBefore.json()
    const totalBefore = tapsBeforeJson.total ?? 0

    await page.goto(`/events/${event.id}/live`)
    await expect(page.getByTestId('live-ops-links')).toBeVisible()
    // Prefer 12 Hour tab so ambiguous seed bibs resolve to Alex Rivera.
    await page.getByTestId('race-tab-12h').click()
    await page.getByTestId('live-open-test-mode').click()
    await expect(page.getByTestId('event-test-mode-dialog')).toBeVisible({ timeout: 15_000 })

    await page.getByTestId('test-mode-bib-input').fill(DEMO_BIB_12H)
    await page.getByTestId('test-mode-bib-submit').click()
    await expect(page.getByTestId('test-mode-feedback')).toContainText(DEMO_BIB_12H)
    // Sanity: 12 Hour race from fixtures still exists for preferred-race resolution.
    expect(BLUFFET.races.twelveHour.name).toMatch(/12/)

    const tapsAfter = await request.get(
      `${API_BASE}/api/events/${event.id}/taps?limit=1`,
      { headers: { Authorization: `Bearer ${token}` } },
    )
    const tapsAfterJson = await tapsAfter.json()
    expect(tapsAfterJson.total ?? 0).toBe(totalBefore)
  })
})
