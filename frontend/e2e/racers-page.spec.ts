import { test, expect } from '@playwright/test'
import { BLUFFET, pinLogin } from './fixtures/rfid'

const RACE_ID = BLUFFET.races.twelveHour.id

/**
 * T017 / US3 — Racers page (debounced search, add, bib edit, multi-tag program).
 * Intentionally red until Racers.vue + routes land (T049 / T050).
 */
test.describe('Racers page [US3]', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/pin')
    await pinLogin(page)
  })

  test('PIN unlock navigates to Racers for a race', async ({ page }) => {
    await page.goto(`/races/${RACE_ID}/racers`)
    await expect(page.getByTestId('racers-list')).toBeVisible()
    await expect(page.getByTestId('racers-search')).toBeVisible()
  })

  test('debounced search filters list within 300ms after typing pause (SC-013)', async ({
    page,
  }) => {
    await page.goto(`/races/${RACE_ID}/racers`)
    const search = page.getByTestId('racers-search')
    const list = page.getByTestId('racers-list')
    await expect(list).toBeVisible()

    const beforeCount = await list.getByTestId('racer-row').count()
    expect(beforeCount).toBeGreaterThan(1)

    await search.fill('zzzz-no-match-sc013')
    // SC-013: filter updates within ~300ms of typing pause (no Search button).
    await expect
      .poll(async () => list.getByTestId('racer-row').count(), {
        timeout: 300,
        intervals: [50, 100, 150],
      })
      .toBe(0)

    await search.fill('')
    await expect
      .poll(async () => list.getByTestId('racer-row').count(), {
        timeout: 300,
        intervals: [50, 100, 150],
      })
      .toBe(beforeCount)
  })

  test('adds a racer with category', async ({ page }) => {
    await page.goto(`/races/${RACE_ID}/racers`)
    await page.getByTestId('add-racer').click()

    await page.getByTestId('racer-first-name').fill('E2E')
    await page.getByTestId('racer-last-name').fill('Racer')
    await page.getByTestId('racer-category').selectOption({ label: /Advanced Men/i })
    await page.getByTestId('racer-save').click()

    const list = page.getByTestId('racers-list')
    await expect(list.getByText('E2E Racer')).toBeVisible()
    await expect(list.getByText(/Advanced Men/i)).toBeVisible()
  })

  test('click-to-edit bib shows save when dirty and persists', async ({ page }) => {
    await page.goto(`/races/${RACE_ID}/racers`)
    const row = page.getByTestId('racer-row').first()
    await row.getByTestId('bib-edit').click()

    const bibInput = page.getByTestId('bib-edit-input')
    await expect(bibInput).toBeVisible()
    // Save control hidden until dirty.
    await expect(page.getByTestId('bib-save')).toBeHidden()

    await bibInput.fill('9999')
    await expect(page.getByTestId('bib-save')).toBeVisible()
    await page.getByTestId('bib-save').click()

    await expect(row.getByTestId('bib-edit')).toHaveText('9999')
  })

  test('inline multi-tag program associates two tags', async ({ page }) => {
    await page.goto(`/races/${RACE_ID}/racers`)
    const row = page.getByTestId('racer-row').first()
    await row.getByTestId('program-tag').click()

    const program = page.getByTestId('program-tag-panel')
    await expect(program).toBeVisible()

    await program.getByTestId('program-tag-uid').fill('E2E-TAG-A')
    await program.getByTestId('program-tag-write').click()
    await expect(program.getByText('E2E-TAG-A')).toBeVisible()

    await program.getByTestId('program-tag-uid').fill('E2E-TAG-B')
    await program.getByTestId('program-tag-write').click()
    await expect(program.getByText('E2E-TAG-B')).toBeVisible()

    await expect(row.getByText(/2 tags/i)).toBeVisible()
  })
})
