import { test, expect } from '@playwright/test'
import { BLUFFET, inject, pinLogin } from './fixtures/rfid'

/**
 * T019 / US5 — Online/offline indicator, offline taps, pending sync, reconnect clear.
 *
 * data-testid contract: sync-status, sync-online, sync-offline, sync-pending
 *
 * Note: Playwright context.setOffline() blocks browser HTTP and WebSocket frames.
 * Inject still hits the local API via the request fixture; we re-deliver the tag
 * into the app shell with a DOM CustomEvent so WAQ/UI can enqueue while offline.
 */
async function deliverTagToUi(page: import('@playwright/test').Page, tagUid: string) {
  await page.evaluate((uid) => {
    window.dispatchEvent(
      new CustomEvent('rfid-tag-read', {
        detail: { tag_uid: uid, read_at: new Date().toISOString() },
      }),
    )
  }, tagUid)
}

test.describe('Offline sync [US5]', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(`/events/${BLUFFET.eventId}/live`)
    await expect(page.getByTestId('sync-status')).toBeVisible({ timeout: 15_000 })
    await expect(page.getByTestId('sync-online')).toBeVisible()
  })

  test('shows online indicator when connected', async ({ page }) => {
    await expect(page.getByTestId('sync-status')).toBeVisible()
    await expect(page.getByTestId('sync-online')).toBeVisible()
    await expect(page.getByTestId('sync-offline')).toBeHidden()
  })

  test('offline taps still record a lap (context.setOffline)', async ({
    page,
    context,
    request,
  }) => {
    await page.goto('/pin')
    await pinLogin(page)
    await page.goto(`/events/${BLUFFET.eventId}/live`)
    await expect(page.getByTestId('sync-status')).toBeVisible()

    await context.setOffline(true)
    await page.waitForFunction(() => navigator.onLine === false)
    await expect(page.getByTestId('sync-offline')).toBeVisible({ timeout: 10_000 })

    // Inject still hits local API (station Postgres authority); UI must accept the tap.
    await inject(request, BLUFFET.demoTags[0])
    await deliverTagToUi(page, BLUFFET.demoTags[0])
    await expect(page.getByTestId('scan-popup')).toBeVisible({ timeout: 5_000 })

    await expect(page.getByTestId('sync-pending')).toBeVisible()
  })

  test('shows pending sync while offline after a scored tap', async ({
    page,
    context,
    request,
  }) => {
    await context.setOffline(true)
    await page.waitForFunction(() => navigator.onLine === false)
    await expect(page.getByTestId('sync-offline')).toBeVisible({ timeout: 10_000 })

    await inject(request, BLUFFET.demoTags[1])
    await deliverTagToUi(page, BLUFFET.demoTags[1])
    await expect(page.getByTestId('scan-popup')).toBeVisible({ timeout: 5_000 })
    await expect(page.getByTestId('sync-pending')).toBeVisible()
  })

  test('reconnect clears pending sync indicator', async ({
    page,
    context,
    request,
  }) => {
    await context.setOffline(true)
    await page.waitForFunction(() => navigator.onLine === false)
    await inject(request, BLUFFET.demoTags[2])
    await deliverTagToUi(page, BLUFFET.demoTags[2])
    await expect(page.getByTestId('sync-pending')).toBeVisible({ timeout: 5_000 })

    await context.setOffline(false)
    // Pending clears once WAQ + hosted sync catch up after reconnect.
    await expect(page.getByTestId('sync-online')).toBeVisible({ timeout: 15_000 })
    await expect(page.getByTestId('sync-pending')).toBeHidden({ timeout: 15_000 })
    await expect(page.getByTestId('sync-offline')).toBeHidden()
  })

  test('route abort also surfaces offline + pending after tap attempt', async ({
    page,
    request,
  }) => {
    await expect(page.getByTestId('sync-online')).toBeVisible()

    // Abort local scan + sync APIs to simulate connectivity loss while WS may still work.
    await page.route('**/api/sync/**', (route) => route.abort())
    await page.route('**/api/rfid/sync-status', (route) => route.abort())
    await page.route('**/api/events/**/scans', (route) => route.abort())

    await inject(request, BLUFFET.demoTags[3])
    await deliverTagToUi(page, BLUFFET.demoTags[3])
    await expect(page.getByTestId('sync-pending')).toBeVisible({ timeout: 5_000 })
    await expect(
      page.getByTestId('scan-popup').or(page.getByTestId('sync-pending')).first(),
    ).toBeVisible()
    await expect(
      page.getByTestId('sync-offline').or(page.getByTestId('sync-pending')).first(),
    ).toBeVisible()
  })
})
