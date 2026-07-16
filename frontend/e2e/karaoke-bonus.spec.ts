import { expect, test } from '@playwright/test'
import {
  API_BASE,
  BLUFFET,
  DEMO_RACER_NAMES,
  DEMO_TAG_12H,
  armFinishStation,
  getBluffetEvent,
  injectTag,
  pinLogin,
} from './fixtures/rfid'

/**
 * US2 — Karaoke bonus lap after a scored RFID scan.
 */
test.describe('US2 karaoke bonus', () => {
  test('after lap popup, one karaoke click records bonus; second click does not duplicate; leaderboard includes bonus', async ({
    page,
    request,
  }) => {
    const event = await getBluffetEvent(request)
    const token = await pinLogin(request)

    // Seed races start as scheduled — activate 12 Hour so inject scores a lap.
    const startRes = await request.post(
      `${API_BASE}/api/races/${BLUFFET.races.twelveHour.id}/start`,
      { headers: { Authorization: `Bearer ${token}` } },
    )
    expect(startRes.ok()).toBeTruthy()

    await armFinishStation(request, token, event.id)

    await page.goto('/pin')
    await pinLogin(page)
    await page.goto(`/events/${event.id}/live`)
    await expect(page.getByTestId('live-view')).toBeVisible()

    const injectRes = await injectTag(request, DEMO_TAG_12H)
    expect(injectRes.ok()).toBeTruthy()

    const popup = page.getByTestId('scan-popup')
    await expect(popup).toBeVisible({ timeout: 10_000 })
    await expect(popup.getByTestId('scan-racer-name')).toContainText(
      DEMO_RACER_NAMES.twelveHour,
    )

    // Karaoke only offered in scan-popup context
    await expect(page.getByTestId('karaoke-bonus-button')).toBeVisible()
    const lapsBefore = await popup.getByTestId('scan-lap-count').innerText()

    await page.getByTestId('karaoke-bonus-button').click()
    await expect(page.getByTestId('karaoke-bonus-recorded')).toBeVisible({
      timeout: 5_000,
    })
    await expect(page.getByTestId('karaoke-bonus-recorded')).toContainText(
      'Karaoke bonus lap recorded',
    )
    // Button becomes one-shot recorded state (no second add control)
    await expect(page.getByTestId('karaoke-bonus-button')).toHaveCount(0)

    const lapsAfterBonus = await popup.getByTestId('scan-lap-count').innerText()
    expect(Number.parseInt(lapsAfterBonus, 10)).toBe(
      Number.parseInt(lapsBefore, 10) + 1,
    )

    // Accidental second click must not create another bonus
    await page.getByTestId('karaoke-bonus-recorded').click({ force: true })
    await expect(popup.getByTestId('scan-lap-count')).toHaveText(lapsAfterBonus)

    await page.getByTestId('scan-popup-dismiss').click()
    await expect(popup).toBeHidden()

    // Karaoke is not a free-form action outside a just-completed scan
    await expect(page.getByTestId('karaoke-bonus-button')).toHaveCount(0)

    await expect(page.getByTestId('leaderboard-overall')).toBeVisible()
    const row = page.getByTestId('leaderboard-row').filter({
      hasText: DEMO_RACER_NAMES.twelveHour,
    })
    await expect(row).toBeVisible()
    await expect(row.getByTestId('leaderboard-laps')).toContainText(lapsAfterBonus)
  })
})
