import type { APIRequestContext, Page } from '@playwright/test'
import { API_BASE, pinToken } from '../../fixtures/rfid'

/**
 * Single physical chip dress rehearsal:
 * WriteTag(participant) overwrites chip user memory with that racer's permanent logical UUID,
 * then a real Proxmark Poll/WS read scores the lap. No silicon UID reassignment.
 */
export async function programRacerAndAwaitLap(opts: {
  request: APIRequestContext
  readerPage: Page
  participantId: string
  timeoutMs?: number
  dismissAfter?: boolean
}) {
  const popup = opts.readerPage.getByTestId('scan-popup')
  const testRead = opts.readerPage.getByTestId('test-read-message')
  const feedback = popup.or(testRead)

  // A popup/toast left over from the previous lap would let us "await visible"
  // on a stale element and score a false-positive. Wait for hidden first.
  await feedback.waitFor({ state: 'hidden', timeout: 5_000 }).catch(() => {})

  const token = await pinToken(opts.request)
  // Hardware write waits on the Proxmark mutex (and may queue behind a poll).
  // Playwright's default API timeout (15s) is too tight for real CLI round-trips.
  const write = await opts.request.post(`${API_BASE}/api/rfid/write-tag`, {
    headers: { Authorization: `Bearer ${token}` },
    data: { participant_id: opts.participantId },
    timeout: 90_000,
  })
  if (!write.ok()) {
    throw new Error(`write-tag failed: ${write.status()} ${await write.text()}`)
  }

  // Pre-start races return `test_read` (toast), not a scored `lap` modal.
  // Either proves write→Poll→WS→scan worked.
  await feedback.waitFor({
    state: 'visible',
    timeout: opts.timeoutMs ?? 60_000,
  })

  if (opts.dismissAfter) {
    await popup
      .getByTestId('scan-popup-dismiss')
      .click({ timeout: 2_000 })
      .catch(() => {})
  }
}
