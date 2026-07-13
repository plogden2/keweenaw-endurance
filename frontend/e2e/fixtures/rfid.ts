/**
 * T014 e2e helpers for RFID race scanner Playwright suite.
 * PIN login, mock inject, station arming, and AYCEB 2026 seed assumptions.
 */
import type { APIRequestContext, APIResponse, Page } from '@playwright/test'

export const ORGANIZER_PIN = '1738'
export const PIN = ORGANIZER_PIN
export const API_BASE = process.env.E2E_API_URL ?? 'http://localhost:8080'

export function apiBase(): string {
  return API_BASE
}

/** Seeded All You Can East Bluffet 2026 (database/seed/03-bluffet-2026.sql) */
export const BLUFFET = {
  eventId: '1441674d-a011-471a-a601-722b88b117f5',
  eventName: 'All You Can East Bluffet',
  eventDate: '2026-08-01',
  races: {
    twelveHour: {
      id: '17da3ba1-2e09-4eb1-aeb3-d9dd5b6a394e',
      name: '12 Hour',
      startTimeLocal: '08:00',
      categoryNames: [
        'Intermediate Men',
        'Intermediate Women',
        'Advanced Men',
        'Advanced Women',
      ],
    },
    sixHour: {
      id: '209769a1-f723-4f70-ae90-466a46338684',
      name: '6 Hour',
      startTimeLocal: '08:00',
      categoryNames: [
        'Intermediate Men',
        'Intermediate Women',
        'Advanced Men',
        'Advanced Women',
      ],
    },
    kids: {
      id: '0e45ee85-800c-4e1f-a95b-4b92462e790a',
      name: '90-Minute Kids',
      startTimeLocal: '15:00',
      categoryNames: ['Men', 'Women'],
    },
  },
  expectedRacerCount: 100,
  demoTags: ['DEMO-TAG-0001', 'DEMO-TAG-0002', 'DEMO-TAG-0003'] as const,
} as const

export const DEMO_TAG_12H = 'DEMO-TAG-0001'
export const DEMO_TAG_6H = 'DEMO-TAG-0002'
export const DEMO_TAG_KIDS = 'DEMO-TAG-0003'

/** Display names matched by seed generator for first tags — adjust if seed names change */
export const DEMO_RACER_NAMES = {
  twelveHour: /./, // any name until seed names are asserted in UI
  sixHour: /./,
  kids: /./,
} as const

export type FinishDeviceId =
  | 'laptop-finish-1'
  | 'laptop-finish-2'
  | 'laptop-finish-3'

/** Unlock management via UI PIN field. */
export async function pinLogin(
  pageOrRequest: Page | APIRequestContext,
  pin: string = ORGANIZER_PIN,
): Promise<string | void> {
  // APIRequestContext has post(); Page has goto/getByTestId
  if ('post' in pageOrRequest && typeof pageOrRequest.post === 'function') {
    return pinToken(pageOrRequest as APIRequestContext, pin)
  }
  const page = pageOrRequest as Page
  const pinInput = page.getByTestId('pin-input').or(page.locator('#pin-input'))
  await pinInput.fill(pin)
  const submit = page.getByTestId('pin-submit').or(page.locator('#submit'))
  await submit.click()
}

/** Exchange organizer PIN for a management JWT via API. */
export async function pinToken(
  request: APIRequestContext,
  pin: string = ORGANIZER_PIN,
): Promise<string> {
  const res = await request.post(`${API_BASE}/api/auth/pin`, {
    data: { pin },
  })
  if (!res.ok()) {
    throw new Error(`PIN auth failed: ${res.status()} ${await res.text()}`)
  }
  const body = (await res.json()) as { token: string }
  return body.token
}

/** Resolve Bluffet event from API (falls back to seeded id). */
export async function getBluffetEvent(
  request: APIRequestContext,
): Promise<{ id: string; name: string }> {
  const res = await request.get(`${API_BASE}/api/events`)
  if (res.ok()) {
    const body = (await res.json()) as {
      data?: Array<{ id: string; name: string }>
    }
    const found = (body.data ?? []).find((e) => e.name === BLUFFET.eventName)
    if (found) return found
  }
  return { id: BLUFFET.eventId, name: BLUFFET.eventName }
}

/** Inject a tag UID into the mock Proxmark3 stream. */
export async function inject(
  request: APIRequestContext,
  tagUid: string,
): Promise<APIResponse> {
  return request.post(`${API_BASE}/api/rfid/inject`, {
    data: { tag_uid: tagUid },
  })
}

export async function injectTag(
  request: APIRequestContext,
  tagUid: string,
): Promise<APIResponse> {
  return inject(request, tagUid)
}

/** Arm this station as a finish reader for an event. */
export async function armFinishStation(
  request: APIRequestContext,
  token: string,
  eventId: string,
  deviceId: string = 'laptop-finish-1',
): Promise<APIResponse> {
  return request.put(`${API_BASE}/api/stations/current`, {
    headers: { Authorization: `Bearer ${token}` },
    data: {
      event_id: eventId,
      mode: 'finish',
      checkpoint_id: null,
      device_id: deviceId,
      name: deviceId,
    },
  })
}

export async function configureStation(
  request: APIRequestContext,
  opts: {
    deviceId: string
    mode?: 'finish' | 'checkpoint'
    checkpointId?: string | null
    name?: string
    eventId?: string
    token?: string
  },
): Promise<void> {
  const token = opts.token ?? (await pinToken(request))
  const res = await request.put(`${API_BASE}/api/stations/current`, {
    headers: { Authorization: `Bearer ${token}` },
    data: {
      event_id: opts.eventId ?? BLUFFET.eventId,
      mode: opts.mode ?? 'finish',
      checkpoint_id: opts.checkpointId ?? null,
      device_id: opts.deviceId,
      name: opts.name ?? opts.deviceId,
    },
  })
  if (!res.ok()) {
    throw new Error(`configureStation failed: ${res.status()} ${await res.text()}`)
  }
}

export async function postScan(
  request: APIRequestContext,
  opts: {
    tagUid: string
    deviceId: string
    eventId?: string
    localTimestamp?: string
    token?: string
  },
): Promise<unknown> {
  const token = opts.token ?? (await pinToken(request))
  const eventId = opts.eventId ?? BLUFFET.eventId
  const res = await request.post(`${API_BASE}/api/events/${eventId}/scans`, {
    headers: { Authorization: `Bearer ${token}` },
    data: {
      tag_uid: opts.tagUid,
      device_id: opts.deviceId,
      local_timestamp: opts.localTimestamp ?? new Date().toISOString(),
    },
  })
  if (!res.ok()) {
    throw new Error(`postScan failed: ${res.status()} ${await res.text()}`)
  }
  return res.json()
}
