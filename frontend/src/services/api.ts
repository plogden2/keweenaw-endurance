import axios, { type AxiosInstance, type AxiosResponse } from 'axios'
import type {
  CreateEventPayload,
  CreateParticipantPayload,
  CreateRacePayload,
  Event,
  LeaderboardResponse,
  ListParams,
  PaginatedResponse,
  Participant,
  Race,
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
    apiClient.get(`/api/timing/live/${raceId}`),
}
