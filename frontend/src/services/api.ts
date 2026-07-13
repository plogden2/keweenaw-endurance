import axios, { type AxiosInstance, type AxiosResponse } from 'axios'
import type {
  Checkpoint,
  CreateEventPayload,
  CreateParticipantPayload,
  CreateRacePayload,
  Event,
  LeaderboardResponse,
  ListParams,
  LiveTimingResponse,
  ManualTimingEntryPayload,
  PaginatedResponse,
  Participant,
  Race,
  RfidTagAssociation,
  Category,
  SyncStatusResponse,
  TimingRecord,
  UpdateEventPayload,
  UpdateParticipantPayload,
  UpdateRacePayload,
} from '@/types/models'

const baseURL = import.meta.env.VITE_API_URL || ''

export const apiClient: AxiosInstance = axios.create({
  baseURL,
  headers: { 'Content-Type': 'application/json' },
})

interface ResourceApi<T, CreateT, UpdateT> {
  list: (params?: ListParams) => Promise<AxiosResponse<PaginatedResponse<T>>>
  get: (id: string) => Promise<AxiosResponse<T>>
  create: (data: CreateT) => Promise<AxiosResponse<T>>
  update: (id: string, data: UpdateT) => Promise<AxiosResponse<T>>
  remove: (id: string) => Promise<AxiosResponse<void>>
}

function createResourceApi<T, CreateT = Partial<T>, UpdateT = Partial<T>>(
  basePath: string,
): ResourceApi<T, CreateT, UpdateT> {
  return {
    list: (params) => apiClient.get<PaginatedResponse<T>>(basePath, { params }),
    get: (id) => apiClient.get<T>(`${basePath}/${id}`),
    create: (data) => apiClient.post<T>(basePath, data),
    update: (id, data) => apiClient.put<T>(`${basePath}/${id}`, data),
    remove: (id) => apiClient.delete<void>(`${basePath}/${id}`),
  }
}

export const eventsApi = createResourceApi<
  Event,
  CreateEventPayload,
  UpdateEventPayload
>('/api/events')

export const racesApi = createResourceApi<
  Race,
  CreateRacePayload,
  UpdateRacePayload
>('/api/races')

export const participantsApi = createResourceApi<
  Participant,
  CreateParticipantPayload,
  UpdateParticipantPayload
>('/api/participants')

/** Race-scoped participants + tags (US3 contract). */
export const raceParticipantsApi = {
  list: (raceId: string, params?: ListParams & { q?: string }) =>
    apiClient.get<PaginatedResponse<Participant>>(`/api/races/${raceId}/participants`, {
      params,
    }),
  create: (raceId: string, data: CreateParticipantPayload) =>
    apiClient.post<Participant>(`/api/races/${raceId}/participants`, data),
  update: (id: string, data: UpdateParticipantPayload) =>
    apiClient.put<Participant>(`/api/participants/${id}`, data),
  listCategories: (raceId: string, params?: ListParams) =>
    apiClient.get<PaginatedResponse<Category>>(`/api/races/${raceId}/categories`, {
      params: { limit: 100, ...params },
    }),
  listTags: (raceId: string, participantId: string) =>
    apiClient.get<{ data: RfidTagAssociation[] }>(
      `/api/races/${raceId}/participants/${participantId}/tags`,
    ),
  addTag: (raceId: string, participantId: string, tagUid: string) =>
    apiClient.post(`/api/races/${raceId}/participants/${participantId}/tags`, {
      tag_uid: tagUid,
    }),
}
export const checkpointsApi = {
  listByRace: (raceId: string, params?: ListParams) =>
    apiClient.get<PaginatedResponse<Checkpoint>>(`/api/races/${raceId}/checkpoints`, {
      params,
    }),
}

export const timingApi = {
  getResults: (raceId: string, params?: ListParams) =>
    apiClient.get<LeaderboardResponse>(`/api/timing/results/${raceId}`, {
      params,
    }),
  getLeaderboard: (raceId: string, params?: ListParams) =>
    apiClient.get<LeaderboardResponse>(`/api/timing/leaderboard/${raceId}`, {
      params,
    }),
  getLive: (raceId: string) =>
    apiClient.get<LiveTimingResponse>(`/api/timing/live/${raceId}`),
}

export const rfidApi = {
  scan: (uid: string) =>
    apiClient.get<Participant>(`/api/rfid/scan/${encodeURIComponent(uid)}`),
  manualEntry: (payload: ManualTimingEntryPayload) =>
    apiClient.post<TimingRecord>('/api/rfid/manual-entry', payload),
  getSyncStatus: () => apiClient.get<SyncStatusResponse>('/api/rfid/sync-status'),
  syncPending: () =>
    apiClient.post<{ synced_count: number }>('/api/rfid/sync-pending'),
}

export const syncApi = {
  push: () => apiClient.post<{ pushed: number; duplicates: number }>('/api/sync/push'),
  pull: () => apiClient.post<{ imported: number; duplicates: number }>('/api/sync/pull'),
}

export interface PinLoginResponse {
  token: string
  role: string
  expires_at: number
}

export interface StationConfig {
  event_id: string | null
  mode: 'finish' | 'checkpoint'
  checkpoint_id?: string | null
  device_id: string
  name: string
  online?: boolean
  pending_sync_count?: number
}

export type ScanResultKind =
  | 'lap'
  | 'test_read'
  | 'cooldown'
  | 'unknown_tag'
  | 'out_of_order'
  | 'checkpoint_pass'

export interface ScanResult {
  result: ScanResultKind
  participant_name?: string
  race_name?: string
  placement?: number
  placement_category?: number
  lap_count?: number
  timing_record_id?: string
  karaoke_available?: boolean
  bib_number?: string
  category_label?: string
  retry_after_seconds?: number
  message?: string
}

export interface PostScanPayload {
  tag_uid: string
  device_id: string
  local_timestamp: string
}

export interface CategoryLegendItem {
  key: string
  label: string
  color: string
}

export interface LiveLeaderboardEntry {
  place: number
  participant_id: string
  bib_number: string
  name: string
  category_key: string
  laps: number
  last_lap_at?: string
}

export interface EventLiveRace {
  id: string
  name: string
  status: string
  start_time: string
  countdown_seconds: number
  leaderboard_overall: LiveLeaderboardEntry[]
  flow_series: unknown[]
}

export interface EventLiveResponse {
  event: { id: string; name: string }
  category_legend: CategoryLegendItem[]
  races: EventLiveRace[]
}

/** Convert HTTP(S) API base to WebSocket URL for the RFID tag stream. */
export function rfidStreamUrl(apiBase: string = baseURL): string {
  const trimmed = (apiBase || '').replace(/\/$/, '')
  if (!trimmed) {
    const proto = typeof window !== 'undefined' && window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = typeof window !== 'undefined' ? window.location.host : 'localhost:8080'
    return `${proto}//${host}/api/rfid/stream`
  }
  if (trimmed.startsWith('https://')) {
    return `wss://${trimmed.slice('https://'.length)}/api/rfid/stream`
  }
  if (trimmed.startsWith('http://')) {
    return `ws://${trimmed.slice('http://'.length)}/api/rfid/stream`
  }
  if (trimmed.startsWith('wss://') || trimmed.startsWith('ws://')) {
    return `${trimmed}/api/rfid/stream`
  }
  return `ws://${trimmed}/api/rfid/stream`
}

export const authApi = {
  loginWithPin: (pin: string) =>
    apiClient.post<PinLoginResponse>('/api/auth/pin', { pin }),
}

export const stationsApi = {
  getCurrent: () => apiClient.get<StationConfig>('/api/stations/current'),
  putCurrent: (config: StationConfig) =>
    apiClient.put<StationConfig>('/api/stations/current', config),
}

export interface KaraokeBonusResponse {
  lap_count: number
  placement: number
  placement_category?: number
  timing_record_id?: string
  record?: TimingRecord & { record_type?: string; source_lap_id?: string }
}

export const scansApi = {
  postScan: (eventId: string, payload: PostScanPayload) =>
    apiClient.post<ScanResult>(`/api/events/${eventId}/scans`, payload),
}

export const timingRecordsApi = {
  karaokeBonus: (timingRecordId: string) =>
    apiClient.post<KaraokeBonusResponse>(
      `/api/timing-records/${timingRecordId}/karaoke-bonus`,
    ),
}

export const eventsLiveApi = {
  getLive: (eventId: string, params?: { category_id?: string }) =>
    apiClient.get<EventLiveResponse>(`/api/events/${eventId}/live`, { params }),
}

export interface LiveCSVStatus {
  path: string
  exists: boolean
  updated_at: string
  size_bytes: number
}

export interface CSVImportSummary {
  event_id: string
  event_name: string
  races: number
  racers: number
  tag_associations: number
  timing_records: number
  categories?: number
  checkpoints?: number
  imported_at: string
}

export const csvApi = {
  getLiveStatus: (eventId: string) =>
    apiClient.get<LiveCSVStatus>(`/api/events/${eventId}/live-csv/status`),
  downloadLiveCsv: (eventId: string) =>
    apiClient.get<Blob>(`/api/events/${eventId}/live-csv`, {
      responseType: 'blob',
    }),
  importCsv: (eventId: string, file: File) => {
    const form = new FormData()
    form.append('file', file)
    return apiClient.post<CSVImportSummary>(`/api/events/${eventId}/import.csv`, form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
  },
}

/** Attach Bearer token for PIN-gated management calls. */
export function setAuthToken(token: string | null) {
  if (token) {
    apiClient.defaults.headers.common.Authorization = `Bearer ${token}`
  } else {
    delete apiClient.defaults.headers.common.Authorization
  }
}
