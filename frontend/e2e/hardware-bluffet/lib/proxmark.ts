import type { APIRequestContext, Page } from '@playwright/test'
import { API_BASE, pinToken } from '../../fixtures/rfid'
import { BRIDGE_LOCAL_URL, fetchLocalBridgeStatus } from './chaos'

async function resolveLogicalUuid(
  request: APIRequestContext,
  participantId: string,
  raceId?: string,
  logicalTagUuid?: string,
): Promise<string | undefined> {
  if (logicalTagUuid) return logicalTagUuid
  if (!raceId) return undefined
  const token = await pinToken(request)
  const res = await request.get(
    `${API_BASE}/api/races/${raceId}/participants/${participantId}/tags`,
    { headers: { Authorization: `Bearer ${token}` } },
  )
  if (!res.ok()) return undefined
  const body = await res.json()
  const tags = Array.isArray(body) ? body : (body.data ?? [])
  const uid = tags[0]?.tag_uid
  return uid ? String(uid).toLowerCase() : undefined
}

/**
 * Single physical chip dress rehearsal:
 * WriteTag(participant) overwrites chip user memory with that racer's permanent logical UUID,
 * then a real Proxmark Poll/WS read scores the lap. No silicon UID reassignment.
 *
 * During a hosted partition (`useLocalBridge`), write-tag goes to loopback bridge HTTP
 * and success is confirmed when the bridge pending queue grows (offline poll enqueue).
 */
export async function programRacerAndAwaitLap(opts: {
  request: APIRequestContext
  readerPage: Page
  participantId: string
  logicalTagUuid?: string
  raceId?: string
  useLocalBridge?: boolean
  timeoutMs?: number
  dismissAfter?: boolean
}) {
  const popup = opts.readerPage.getByTestId('scan-popup')
  const testRead = opts.readerPage.getByTestId('test-read-message')
  const feedback = popup.or(testRead)

  // A popup/toast left over from the previous lap would let us "await visible"
  // on a stale element and score a false-positive. Wait for hidden first.
  await feedback.waitFor({ state: 'hidden', timeout: 5_000 }).catch(() => {})

  const timeoutMs = opts.timeoutMs ?? 60_000

  if (opts.useLocalBridge) {
    const pendingBefore = (await fetchLocalBridgeStatus()).pending_count
    const logicalUuid = await resolveLogicalUuid(
      opts.request,
      opts.participantId,
      opts.raceId,
      opts.logicalTagUuid,
    )
    const body: Record<string, string> = {}
    if (logicalUuid) {
      body.logical_uuid = logicalUuid
    } else {
      body.participant_id = opts.participantId
      if (opts.raceId) body.race_id = opts.raceId
    }

    const write = await fetch(`${BRIDGE_LOCAL_URL}/write-tag`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
      signal: AbortSignal.timeout(90_000),
    })
    if (!write.ok) {
      throw new Error(`local bridge write-tag failed: ${write.status} ${await write.text()}`)
    }

    const deadline = Date.now() + timeoutMs
    while (Date.now() < deadline) {
      const pendingAfter = (await fetchLocalBridgeStatus()).pending_count
      if (pendingAfter > pendingBefore) {
        if (opts.dismissAfter) {
          await popup
            .getByTestId('scan-popup-dismiss')
            .click({ timeout: 2_000 })
            .catch(() => {})
        }
        return
      }

      await new Promise((r) => setTimeout(r, 500))
    }
    throw new Error('offline lap not enqueued on local bridge before timeout')
  }

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
    timeout: timeoutMs,
  })

  if (opts.dismissAfter) {
    await popup
      .getByTestId('scan-popup-dismiss')
      .click({ timeout: 2_000 })
      .catch(() => {})
  }
}
