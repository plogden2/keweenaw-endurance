import axios from 'axios'

const baseURL = import.meta.env.VITE_API_URL || ''

export const apiClient = axios.create({
  baseURL,
  headers: { 'Content-Type': 'application/json' },
})

function createResourceApi(basePath) {
  return {
    list: (params) => apiClient.get(basePath, { params }),
    get: (id) => apiClient.get(`${basePath}/${id}`),
    create: (data) => apiClient.post(basePath, data),
    update: (id, data) => apiClient.put(`${basePath}/${id}`, data),
    remove: (id) => apiClient.delete(`${basePath}/${id}`),
  }
}

export const eventsApi = createResourceApi('/api/events')
export const racesApi = createResourceApi('/api/races')
export const participantsApi = createResourceApi('/api/participants')

export const timingApi = {
  getResults: (raceId, params) =>
    apiClient.get(`/api/timing/results/${raceId}`, { params }),
  getLeaderboard: (raceId, params) =>
    apiClient.get(`/api/timing/leaderboard/${raceId}`, { params }),
  getLive: (raceId) => apiClient.get(`/api/timing/live/${raceId}`),
}
