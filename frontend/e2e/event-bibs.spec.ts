import { expect, test } from '@playwright/test'
import {
  API_BASE,
  BLUFFET,
  armFinishStation,
  getBluffetEvent,
  injectTag,
  pinLogin,
  pinToken,
} from './fixtures/rfid'

/**
 * Event → Bibs inventory: bulk create + associate (mock/CI path) + morning assign → scan.
 * Uses POST …/bibs/{id}/tags with tag_uid when Proxmark write is unavailable.
 */
test.describe('Event bibs inventory', () => {
  const bulkFrom = 8801
  const bulkTo = 8803

  test('bulk create 1–3, associate tag, assign on Racers, scan resolves via bib', async ({
    page,
    request,
  }) => {
    const event = await getBluffetEvent(request)
    const token = await pinToken(request)
    const tagUid = `e2e-bib-tag-${Date.now()}-${Math.floor(Math.random() * 1e6)}`
    const raceId = BLUFFET.races.twelveHour.id

    await page.goto('/pin')
    await pinLogin(page)

    await page.goto(`/events/${event.id}/bibs`)
    await expect(page.getByTestId('event-bibs-page')).toBeVisible({ timeout: 15_000 })
    await expect(page.getByTestId('mgmt-unlocked')).toBeVisible()
    await expect(page.getByTestId('bibs-bulk-create')).toBeVisible()

    await page.getByTestId('bibs-bulk-from').fill(String(bulkFrom))
    await page.getByTestId('bibs-bulk-to').fill(String(bulkTo))
    await page.getByTestId('bibs-bulk-submit').click()

    const table = page.getByTestId('event-bibs-table')
    await expect(table).toBeVisible()
    for (const n of [bulkFrom, bulkFrom + 1, bulkTo]) {
      await expect(table.getByText(String(n), { exact: true })).toBeVisible()
    }

    const bibsRes = await request.get(`${API_BASE}/api/events/${event.id}/bibs`)
    expect(bibsRes.ok()).toBeTruthy()
    const bibsBody = (await bibsRes.json()) as {
      data?: Array<{ id: string; bib_number: string | number }>
    }
    const targetBib = (bibsBody.data ?? []).find(
      (b) => String(b.bib_number) === String(bulkFrom),
    )
    expect(targetBib, `bib ${bulkFrom} after bulk create`).toBeTruthy()

    // CI-safe associate (no Proxmark hardware).
    const assoc = await request.post(
      `${API_BASE}/api/events/${event.id}/bibs/${targetBib!.id}/tags`,
      {
        headers: { Authorization: `Bearer ${token}` },
        data: { tag_uid: tagUid },
      },
    )
    expect(assoc.ok(), await assoc.text()).toBeTruthy()

    await page.reload()
    await expect(page.getByTestId('event-bibs-table')).toBeVisible()
    const bibRow = page.locator(`[data-testid="bib-row-${targetBib!.id}"]`)
    await expect(bibRow).toBeVisible()
    await expect(bibRow).toContainText(/unassigned/i)
    // Tag count column shows 1 after associate.
    await expect(bibRow.locator('td').nth(1)).toHaveText('1')

    // Morning path: assign this pre-tagged bib to a seeded racer.
    await page.goto(`/races/${raceId}/racers`)
    await expect(page.getByTestId('racers-list')).toBeVisible()

    const row = page.getByTestId('racer-row').first()
    await row.getByTestId('bib-edit').click()
    await page.getByTestId('bib-edit-input').fill(String(bulkFrom))
    page.once('dialog', (d) => d.accept())
    await page.getByTestId('bib-save').click()
    await expect(row.getByTestId('bib-edit')).toHaveText(String(bulkFrom))

    const startRes = await request.post(`${API_BASE}/api/races/${raceId}/start`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    // Already-active is fine.
    expect([200, 201, 400, 409]).toContain(startRes.status())

    const arm = await armFinishStation(request, token, event.id)
    expect(arm.ok()).toBeTruthy()

    await page.goto(`/events/${event.id}/live`)
    await expect(page.getByTestId('live-view')).toBeVisible({ timeout: 15_000 })

    const injectRes = await injectTag(request, tagUid)
    expect(injectRes.ok()).toBeTruthy()

    // Prefer live popup when WS stream is up; always assert scan resolve via API.
    const popup = page.getByTestId('scan-popup')
    const popupVisible = await popup
      .waitFor({ state: 'visible', timeout: 10_000 })
      .then(() => true)
      .catch(() => false)
    if (popupVisible) {
      await expect(page.getByText(/unknown tag/i)).toHaveCount(0)
    }

    const scan = await request.post(`${API_BASE}/api/events/${event.id}/scans`, {
      headers: { Authorization: `Bearer ${token}` },
      data: {
        tag_uid: tagUid,
        device_id: 'laptop-finish-1',
        local_timestamp: new Date().toISOString(),
      },
    })
    expect(scan.ok(), await scan.text()).toBeTruthy()
    const scanBody = (await scan.json()) as {
      result?: string
      participant?: { bib_number?: string }
    }
    expect(scanBody.result).not.toBe('unknown_tag')
    expect(['lap', 'test_read', 'cooldown']).toContain(scanBody.result)
    if (scanBody.participant?.bib_number != null) {
      expect(String(scanBody.participant.bib_number)).toBe(String(bulkFrom))
    }
  })
})
