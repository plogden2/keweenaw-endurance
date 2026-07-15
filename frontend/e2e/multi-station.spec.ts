/**
 * T021 / US7 — Three finish stations + shared cooldown + checkpoint out-of-order (failing until stories land).
 */
import { expect, test } from '@playwright/test'
import {
  API_BASE,
  BLUFFET,
  configureStation,
  inject,
  pinLogin,
  pinToken,
  postScan,
  type FinishDeviceId,
} from './fixtures/rfid'

const FINISH_STATIONS: { deviceId: FinishDeviceId; tag: string; name: string }[] = [
  { deviceId: 'laptop-finish-1', tag: BLUFFET.demoTags[0], name: 'Finish Mat A' },
  { deviceId: 'laptop-finish-2', tag: BLUFFET.demoTags[1], name: 'Finish Mat B' },
  { deviceId: 'laptop-finish-3', tag: BLUFFET.demoTags[2], name: 'Finish Mat C' },
]

async function ensureRacesActive(
  request: Parameters<typeof pinToken>[0],
  token: string,
): Promise<void> {
  for (const race of Object.values(BLUFFET.races)) {
    const startRes = await request.post(`${API_BASE}/api/races/${race.id}/start`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    // 200 when newly started; 400/409 if already active is fine for parallel workers.
    expect([200, 400, 409]).toContain(startRes.status())
  }
}

test.describe('US7 multi-station readers', () => {
  test('three finish stations record distinct racers (device_id laptop-finish-1/2/3)', async ({
    request,
  }) => {
    const token = await pinToken(request)
    await ensureRacesActive(request, token)
    const participantIds = new Set<string>()

    for (const station of FINISH_STATIONS) {
      await configureStation(request, {
        deviceId: station.deviceId,
        mode: 'finish',
        name: station.name,
        token,
      })

      const scan = (await postScan(request, {
        tagUid: station.tag,
        deviceId: station.deviceId,
        token,
      })) as { result: string; participant?: { id: string }; lap_count?: number }

      expect(scan.result).toBe('lap')
      expect(scan.participant?.id).toBeTruthy()
      participantIds.add(scan.participant!.id)
    }

    // Three finish stations → three distinct racers (first three 12-hour tag UUIDs).
    expect(participantIds.size).toBe(3)
  })

  test('shared cooldown is enforced at station B after sync from station A', async ({
    request,
  }) => {
    const token = await pinToken(request)
    await ensureRacesActive(request, token)
    const sharedTag = BLUFFET.demoTags[0]

    await configureStation(request, {
      deviceId: 'laptop-finish-1',
      mode: 'finish',
      name: 'Finish Mat A',
      token,
    })
    const first = (await postScan(request, {
      tagUid: sharedTag,
      deviceId: 'laptop-finish-1',
      token,
    })) as { result: string }
    expect(first.result).toBe('lap')

    // Sync so station B learns about the recent tap.
    const push = await request.post(`${API_BASE}/api/sync/push`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    expect(push.ok()).toBeTruthy()
    const pull = await request.post(`${API_BASE}/api/sync/pull`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    expect(pull.ok()).toBeTruthy()

    await configureStation(request, {
      deviceId: 'laptop-finish-2',
      mode: 'finish',
      name: 'Finish Mat B',
      token,
    })
    const second = (await postScan(request, {
      tagUid: sharedTag,
      deviceId: 'laptop-finish-2',
      token,
    })) as { result: string; retry_after_seconds?: number }

    expect(second.result).toBe('cooldown')
    expect(second.retry_after_seconds).toBeGreaterThan(0)
    expect(second.retry_after_seconds).toBeLessThanOrEqual(60)
  })

  test('checkpoint mode out-of-order tap does not complete a lap', async ({ page, request }) => {
    const token = await pinToken(request)
    await ensureRacesActive(request, token)

    await page.goto('/station')
    await pinLogin(page)

    // Ensure event is selected so checkpoint picker has options.
    const eventSelect = page.getByTestId('station-event-select')
    await expect(eventSelect).toBeVisible()
    const options = eventSelect.locator('option:not([disabled])')
    if ((await options.count()) > 0) {
      await eventSelect.selectOption({ index: 1 })
    }

    const mode = page.getByTestId('station-mode')
    await expect(mode).toBeVisible()
    await mode.selectOption('checkpoint').catch(async () => {
      await page.getByLabel(/checkpoint/i).check()
    })

    const picker = page.getByTestId('checkpoint-picker')
    await expect(picker).toBeVisible()
    // Mid-sequence (Lap Check), not Start Line — out-of-order for a fresh racer.
    await picker.selectOption({ label: /lap check/i }).catch(async () => {
      await picker.selectOption({ index: 2 })
    })

    const checkpointId = await picker.inputValue()
    await page.getByRole('button', { name: /save|arm/i }).click()

    await configureStation(request, {
      deviceId: 'laptop-checkpoint-1',
      mode: 'checkpoint',
      name: 'Mid-loop CP',
      checkpointId,
      token,
    })

    await page.goto(`/events/${BLUFFET.eventId}/live`)
    await inject(request, BLUFFET.demoTags[0])

    const ooo = page.getByTestId('out-of-order-message')
    await expect(ooo).toBeVisible({ timeout: 10_000 })
    await expect(ooo).toContainText(/out of (order|sequence)|not yet|sequence/i)

    // Must not award a completed lap from this tap alone.
    const popup = page.getByTestId('scan-popup')
    if (await popup.isVisible().catch(() => false)) {
      await expect(popup).not.toContainText(/karaoke/i)
    }
  })
})
