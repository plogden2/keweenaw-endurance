import { defineStore } from 'pinia'
import { racesApi } from '@/services/api'
import type {
  CreateRacePayload,
  ListParams,
  Race,
  UpdateRacePayload,
} from '@/types/models'
import { getErrorMessage } from '@/utils/error'

interface RacesState {
  races: Race[]
  currentRace: Race | null
  total: number
  page: number
  limit: number
  loading: boolean
  error: string | null
}

export const useRacesStore = defineStore('races', {
  state: (): RacesState => ({
    races: [],
    currentRace: null,
    total: 0,
    page: 1,
    limit: 20,
    loading: false,
    error: null,
  }),

  actions: {
    async fetchRaces(params: ListParams = {}) {
      this.loading = true
      this.error = null
      try {
        const { data } = await racesApi.list(params)
        this.races = data.data ?? []
        this.total = data.total ?? 0
        this.page = data.page ?? this.page
        this.limit = data.limit ?? this.limit
      } catch (err) {
        this.error = getErrorMessage(err, 'Failed to fetch races')
      } finally {
        this.loading = false
      }
    },

    async fetchRace(id: string) {
      this.loading = true
      this.error = null
      try {
        const { data } = await racesApi.get(id)
        this.currentRace = data
      } catch (err) {
        this.error = getErrorMessage(err, 'Failed to fetch race')
      } finally {
        this.loading = false
      }
    },

    async createRace(payload: CreateRacePayload) {
      const { data } = await racesApi.create(payload)
      this.races.push(data)
      this.total += 1
      return data
    },

    async updateRace(id: string, payload: UpdateRacePayload) {
      const { data } = await racesApi.update(id, payload)
      const index = this.races.findIndex((r) => r.id === id)
      if (index !== -1) {
        this.races[index] = data
      }
      if (this.currentRace?.id === id) {
        this.currentRace = data
      }
      return data
    },

    async deleteRace(id: string) {
      await racesApi.remove(id)
      this.races = this.races.filter((r) => r.id !== id)
      this.total = Math.max(0, this.total - 1)
      if (this.currentRace?.id === id) {
        this.currentRace = null
      }
    },
  },
})
