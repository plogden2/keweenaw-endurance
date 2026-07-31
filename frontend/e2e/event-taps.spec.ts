import { expect, test } from '@playwright/test'
import { BLUFFET, getBluffetEvent, pinLogin } from './fixtures/rfid'

/**
 * Event taps editor — PIN → add tap → void → restore.
 */
test.describe('Event taps editor', () => {
  test('PIN unlock, add tap, void, then restore', async ({ page }) => {
    const event = await getBluffetEvent(page.request)

    await page.goto('/pin')
    await pinLogin(page)

    await page.goto(`/events/${event.id}/taps`)
    // Table is always rendered (empty state is a row inside it).
    await expect(page.getByTestId('event-taps-table')).toBeVisible({ timeout: 15_000 })

    page.on('dialog', (dialog) => dialog.accept())

    await page.getByTestId('add-tap-btn').click()
    await expect(page.getByTestId('add-tap-dialog')).toBeVisible()

    const search = page.getByTestId('tap-participant-search')
    await search.fill('1')
    await expect(page.getByTestId('tap-participant-option').first()).toBeVisible({
      timeout: 10_000,
    })
    const option = page.getByTestId('tap-participant-option').first()
    const optionText = (await option.innerText()).trim()
    await option.click()
    await expect(page.getByTestId('tap-participant-selected')).toContainText(optionText)

    await page.getByTestId('add-tap-submit').click()
    await expect(page.getByTestId('add-tap-dialog')).toHaveCount(0)

    const table = page.getByTestId('event-taps-table')
    await expect(table).toBeVisible()
    const firstRow = table.locator('tbody tr').first()
    await expect(firstRow).toBeVisible()
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
