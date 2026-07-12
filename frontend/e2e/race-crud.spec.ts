import { test, expect } from '@playwright/test'
import { BLUFFET, pinLogin } from './fixtures/rfid'

/**
 * T018 / T063 / US4 — PIN-gated race create/delete on `/pin` management;
 * public live view without PIN. Matches pin-unlock prototype race-management.
 */
test.describe('Race CRUD [US4]', () => {
  test('create and delete require PIN (management locked without unlock)', async ({
    page,
  }) => {
    await page.goto('/pin')
    await expect(page.getByTestId('pin-input')).toBeVisible()
    await expect(page.getByTestId('race-management')).toBeHidden()
    await expect(page.getByTestId('create-race')).toBeHidden()
  })

  test('creates a lap race under the event after PIN unlock', async ({ page }) => {
    await page.goto('/pin')
    await pinLogin(page)
    await expect(page.getByTestId('race-management')).toBeVisible()

    const uniqueName = `E2E Lap Race ${Date.now()}`
    await page.getByTestId('create-race-name').fill(uniqueName)
    await page.getByTestId('create-race-duration').selectOption({ label: /12 hours/i })
    await page.getByTestId('create-race-start-time').fill('09:00')
    await page.getByTestId('create-race').click()

    const list = page.getByTestId('race-list')
    await expect(list.getByText(uniqueName)).toBeVisible()
  })

  test('delete cancel leaves race; confirm removes it', async ({ page }) => {
    await page.goto('/pin')
    await pinLogin(page)
    await expect(page.getByTestId('race-management')).toBeVisible()

    const uniqueName = `E2E Delete Me ${Date.now()}`
    await page.getByTestId('create-race-name').fill(uniqueName)
    await page.getByTestId('create-race-duration').selectOption({ label: /6 hours/i })
    await page.getByTestId('create-race-start-time').fill('10:00')
    await page.getByTestId('create-race').click()

    const list = page.getByTestId('race-list')
    const row = list.getByTestId('race-row').filter({ hasText: uniqueName })
    await expect(row).toBeVisible()

    await row.getByTestId('delete-race').click()
    await page.getByTestId('delete-race-cancel').click()
    await expect(list.getByText(uniqueName)).toBeVisible()

    await row.getByTestId('delete-race').click()
    await page.getByTestId('delete-race-confirm').click()
    await expect(list.getByText(uniqueName)).toHaveCount(0)
  })

  test('live view is public without PIN', async ({ page }) => {
    await page.goto(`/events/${BLUFFET.eventId}/live`)
    await expect(page.getByTestId('live-view')).toBeVisible()
    await expect(page.getByTestId('leaderboard-overall')).toBeVisible()
    // Must not force PIN gate for read-only live.
    await expect(page.getByTestId('pin-input')).toHaveCount(0)
  })

  test('seeded race remains visible on live after management session', async ({
    page,
  }) => {
    await page.goto('/pin')
    await pinLogin(page)
    await page.goto(`/events/${BLUFFET.eventId}/live`)
    await expect(page.getByTestId('live-view')).toBeVisible()
    await expect(page.getByText(BLUFFET.races.twelveHour.name)).toBeVisible()
  })
})
