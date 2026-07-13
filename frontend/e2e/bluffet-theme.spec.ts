import { test, expect } from '@playwright/test'
import { BLUFFET } from './fixtures/rfid'

test.describe('Bluffet theme', () => {
  test('home featured poster assets resolve and section is themed', async ({ page, request }) => {
    const avif = await request.get('/images/bluffet-2026-poster.avif')
    const png = await request.get('/images/bluffet-2026-poster.png')
    const logo = await request.get('/images/bluffet-2026-logo.png')
    expect(avif.ok()).toBeTruthy()
    expect(png.ok()).toBeTruthy()
    expect(logo.ok()).toBeTruthy()

    await page.goto('/')
    await expect(page.getByTestId('bluffet-poster')).toBeVisible()
    await expect(page.locator('.featured-event.bluffet-theme')).toBeVisible()
  })

  test('Bluffet live view gets theme-bluffet on app root', async ({ page }) => {
    await page.goto(`/events/${BLUFFET.eventId}/live`)
    await expect(page.locator('#app')).toHaveClass(/theme-bluffet/)
  })

  test('non-Bluffet timing list does not force theme from station alone', async ({ page }) => {
    await page.goto('/timing')
    await expect(page.locator('#app')).not.toHaveClass(/theme-bluffet/)
  })
})
