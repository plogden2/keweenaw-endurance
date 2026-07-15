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

  // A popup left over from the previous lap (auto-dismiss timer, or the
  // reader page having just navigated) would let us "await visible" on a
  // stale element and score a false-positive lap. Wait for hidden first.
  await popup.waitFor({ state: 'hidden', timeout: 5_000 }).catch(() => {})

  const token = await pinToken(opts.request)
  const write = await opts.request.post(`${API_BASE}/api/rfid/write-tag`, {
    headers: { Authorization: `Bearer ${token}` },
    data: { participant_id: opts.participantId },
  })
  if (!write.ok()) {
    throw new Error(`write-tag failed: ${write.status()} ${await write.text()}`)
  }

  await popup.waitFor({
    state: 'visible',
    timeout: opts.timeoutMs ?? 30_000,
  })

  if (opts.dismissAfter) {
    await popup
      .getByTestId('scan-popup-dismiss')
      .click({ timeout: 2_000 })
      .catch(() => {})
  }
}
