export type EventStatus = 'upcoming' | 'active' | 'completed' | 'cancelled'
export type RaceStatus = 'scheduled' | 'active' | 'finished' | 'cancelled'
export type RaceType = 'time_based' | 'lap_based'
export type ParticipantStatus = 'registered' | 'started' | 'finished' | 'dnf' | 'dns'

export interface Event {
  id: string
  name: string
  description?: string
  event_date: string
  location?: string
  website_url?: string
  status: EventStatus
  created_at?: string
  updated_at?: string
}

export interface Race {
  id: string
  event_id: string
  name: string
  race_type: RaceType
  distance_km?: number
  duration_minutes?: number
  start_time?: string
  status: RaceStatus
  created_at?: string
}

export interface Participant {
  id: string
  race_id: string
  bib_number: string
  first_name: string
  last_name: string
  gender?: string
  age?: number
  rfid_tag_uid?: string
  status: ParticipantStatus
  created_at?: string
}

export interface LeaderboardEntry {
  position: number
  participant_id: string
  bib_number: string
  first_name: string
  last_name: string
  total_time_seconds: number
  laps?: number
  status: string
}

export interface PaginatedResponse<T> {
  data: T[]
  total: number
  page?: number
  limit?: number
}

export interface LeaderboardResponse {
  data: LeaderboardEntry[]
}

export type CreateEventPayload = Pick<
  Event,
  'name' | 'description' | 'event_date' | 'location' | 'website_url' | 'status'
>

export type UpdateEventPayload = Partial<CreateEventPayload>

export type CreateRacePayload = Pick<
  Race,
  'event_id' | 'name' | 'race_type' | 'distance_km' | 'duration_minutes' | 'start_time' | 'status'
>

export type UpdateRacePayload = Partial<Omit<CreateRacePayload, 'event_id'>>

export type CreateParticipantPayload = Pick<
  Participant,
  'race_id' | 'bib_number' | 'first_name' | 'last_name' | 'gender' | 'age' | 'rfid_tag_uid' | 'status'
>

export type UpdateParticipantPayload = Partial<Omit<CreateParticipantPayload, 'race_id'>>

export type CheckpointType = 'start' | 'finish' | 'intermediate'
export type SyncStatus = 'synced' | 'pending_sync' | 'failed_sync'

export interface Checkpoint {
  id: string
  race_id: string
  name: string
  checkpoint_type: CheckpointType
  distance_from_start_km?: number
  location_description?: string
  is_active: boolean
}

export interface TimingRecord {
  id: string
  participant_id: string
  checkpoint_id: string
  timestamp: string
  local_timestamp: string
  device_id?: string
  sync_status: SyncStatus
  participant?: Participant
  checkpoint?: Checkpoint
}

export interface LiveTimingResponse {
  race_id: string
  records: TimingRecord[]
}

export interface SyncStatusResponse {
  pending_count: number
  failed_count: number
  synced_count: number
}

export interface ManualTimingEntryPayload {
  race_id: string
  checkpoint_id: string
  bib_number?: string
  rfid_tag_uid?: string
  timestamp: string
  device_id?: string
  sync_status?: SyncStatus
}

export interface ListParams {
  page?: number
  limit?: number
  event_id?: string
  race_id?: string
  category_id?: string
  [key: string]: string | number | undefined
}
