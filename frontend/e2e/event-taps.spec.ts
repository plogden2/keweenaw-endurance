import { expect, test } from '@playwright/test'
import {
  BLUFFET,
  DEMO_BIB_12H,
  DEMO_PARTICIPANT_12H_ID,
  DEMO_RACER_NAMES,
  ensureUniqueEventBib,
  getBluffetEvent,
  pinLogin,
} from './fixtures/rfid'

/**
 * Event taps editor — PIN → inline bib Enter → void → restore.
 */
test.describe('Event taps editor', () => {
  test('PIN unlock, inline bib Enter, void, then restore', async ({ page }) => {
    const event = await getBluffetEvent(page.request)
    // Seed reuses bibs across races; keep DEMO_BIB_12H → Alex Rivera only.
    await ensureUniqueEventBib(
      page.request,
      event.id,
      DEMO_BIB_12H,
      DEMO_PARTICIPANT_12H_ID,
    )

    await page.goto('/pin')
    await pinLogin(page)

    await page.goto(`/events/${event.id}/taps`)
    // Table is always rendered (empty state is a row inside it).
    await expect(page.getByTestId('event-taps-table')).toBeVisible({ timeout: 15_000 })

    page.on('dialog', (dialog) => dialog.accept())

    const bibInput = page.getByTestId('inline-bib-input')
    await expect(bibInput).toBeVisible()
    await bibInput.fill(DEMO_BIB_12H)
    await bibInput.press('Enter')

    const table = page.getByTestId('event-taps-table')
    const firstRow = table.locator('tbody tr').first()
    await expect(firstRow).toBeVisible({ timeout: 10_000 })
    await expect(firstRow).toContainText(DEMO_BIB_12H)
    await expect(firstRow).toContainText(DEMO_RACER_NAMES.twelveHour)
    await expect(firstRow.getByTestId('voided-badge')).toHaveCount(0)

    await firstRow.getByTestId('void-tap-btn').click()
    await expect(firstRow.getByTestId('voided-badge')).toBeVisible()
    await expect(firstRow).toHaveClass(/voided/)

    await firstRow.getByTestId('restore-tap-btn').click()
    await expect(firstRow.getByTestId('voided-badge')).toHaveCount(0)
    await expect(firstRow).not.toHaveClass(/voided/)
  })

  test('legacy /timing/live/:raceId redirects to event taps', async ({ page }) => {
    await page.goto(`/timing/live/${BLUFFET.races.twelveHour.id}`)
    await expect(page).toHaveURL(new RegExp(`/events/${BLUFFET.eventId}/taps`))
  })
})
