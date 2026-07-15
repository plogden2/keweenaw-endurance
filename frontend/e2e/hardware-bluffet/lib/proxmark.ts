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
}) {
  const token = await pinToken(opts.request)
  const write = await opts.request.post(`${API_BASE}/api/rfid/write-tag`, {
    headers: { Authorization: `Bearer ${token}` },
    data: { participant_id: opts.participantId },
  })
  if (!write.ok()) {
    throw new Error(`write-tag failed: ${write.status()} ${await write.text()}`)
  }
  await opts.readerPage.getByTestId('scan-popup').waitFor({
    state: 'visible',
    timeout: opts.timeoutMs ?? 30_000,
  })
}
