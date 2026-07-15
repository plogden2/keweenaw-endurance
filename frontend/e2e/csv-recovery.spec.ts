/**
 * T020 / US6 — Live CSV updates + PIN-gated import recovery (failing until stories land).
 */
import { expect, test } from '@playwright/test'
import {
  API_BASE,
  BLUFFET,
  inject,
  pinLogin,
  pinToken,
  configureStation,
} from './fixtures/rfid'

test.describe('US6 CSV disaster recovery', () => {
  test('live CSV updates automatically after inject/lap without an export click', async ({
    page,
    request,
  }) => {
    await configureStation(request, {
      deviceId: 'laptop-finish-1',
      mode: 'finish',
      name: 'Finish Mat A',
    })

    await page.goto(`/csv?eventId=${BLUFFET.eventId}`)
    await pinLogin(page)

    const status = page.getByTestId('live-csv-status')
    await expect(status).toBeVisible()
    const before = (await status.getAttribute('data-updated-at')) ?? (await status.textContent())

    // Record a change that must refresh the live snapshot (no export button required).
    await inject(request, BLUFFET.demoTags[0])
    await page.waitForTimeout(500)

    const afterUi =
      (await status.getAttribute('data-updated-at')) ?? (await status.textContent())
    expect(afterUi).not.toEqual(before)

    const token = await pinToken(request)
    const liveCsv = await request.get(
      `${API_BASE}/api/events/${BLUFFET.eventId}/live-csv`,
      { headers: { Authorization: `Bearer ${token}` } },
    )
    expect(liveCsv.ok()).toBeTruthy()
    const body = await liveCsv.text()
    expect(body.length).toBeGreaterThan(0)
    // Snapshot should reflect race activity (racer/tag or timing row).
    expect(body).toMatch(
      new RegExp(`${BLUFFET.demoTags[0]}|timing|participant`, 'i'),
    )
  })

  test('PIN-gated CSV import restores state on a fresh station via /csv', async ({
    page,
    request,
  }) => {
    const token = await pinToken(request)

    // Capture live snapshot from the "healthy" station.
    const exportRes = await request.get(
      `${API_BASE}/api/events/${BLUFFET.eventId}/live-csv`,
      { headers: { Authorization: `Bearer ${token}` } },
    )
    expect(exportRes.ok()).toBeTruthy()
    const csvBytes = await exportRes.body()

    // Fresh station UI: management locked until PIN, then import.
    await page.goto('/csv')
    await expect(page.getByTestId('pin-input').or(page.locator('#pin-input'))).toBeVisible()
    await pinLogin(page)

    await expect(page.getByRole('heading', { name: /live csv|csv recovery/i })).toBeVisible()

    const fileInput = page.getByTestId('csv-import-input').or(page.locator('input[type="file"]'))
    await fileInput.setInputFiles({
      name: 'bluffet-live.csv',
      mimeType: 'text/csv',
      buffer: Buffer.from(csvBytes),
    })

    const confirm = page.getByTestId('csv-import-confirm').or(
      page.locator('input[name="confirm"], input[type="checkbox"]'),
    )
    if (await confirm.count()) {
      await confirm.check()
    }

    await page.getByTestId('csv-import-submit').or(page.getByRole('button', { name: /import/i })).click()

    const summary = page.getByTestId('csv-import-summary')
    await expect(summary).toBeVisible()
    await expect(summary).toContainText(/100|racers/i)
  })

  test('scanning continues after CSV import on the replacement station', async ({
    page,
    request,
  }) => {
    const token = await pinToken(request)
    const exportRes = await request.get(
      `${API_BASE}/api/events/${BLUFFET.eventId}/live-csv`,
      { headers: { Authorization: `Bearer ${token}` } },
    )
    const csvBytes = await exportRes.body()

    // Import via API (fresh station replace-semantics), then arm and scan.
    const importRes = await request.post(
      `${API_BASE}/api/events/${BLUFFET.eventId}/import.csv`,
      {
        headers: { Authorization: `Bearer ${token}` },
        multipart: {
          file: {
            name: 'bluffet-live.csv',
            mimeType: 'text/csv',
            buffer: Buffer.from(csvBytes),
          },
        },
      },
    )
    expect(importRes.ok()).toBeTruthy()

    await configureStation(request, {
      deviceId: 'laptop-finish-recovery',
      mode: 'finish',
      name: 'Replacement Finish',
      token,
    })

    await page.goto(`/events/${BLUFFET.eventId}/live`)
    await inject(request, BLUFFET.demoTags[1])

    const popup = page.getByTestId('scan-popup')
    await expect(popup).toBeVisible({ timeout: 10_000 })
    await expect(popup).toContainText(/\d+\s*lap|place/i)
  })
})
