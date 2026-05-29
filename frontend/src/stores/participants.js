import { defineStore } from 'pinia'
import { participantsApi } from '../services/api.js'

export const useParticipantsStore = defineStore('participants', {
  state: () => ({
    participants: [],
    currentParticipant: null,
    total: 0,
    page: 1,
    limit: 20,
    loading: false,
    error: null,
  }),

  actions: {
    async fetchParticipants(params = {}) {
      this.loading = true
      this.error = null
      try {
        const { data } = await participantsApi.list(params)
        this.participants = data.data ?? []
        this.total = data.total ?? 0
        this.page = data.page ?? this.page
        this.limit = data.limit ?? this.limit
      } catch (err) {
        this.error = err.message ?? 'Failed to fetch participants'
      } finally {
        this.loading = false
      }
    },

    async fetchParticipant(id) {
      this.loading = true
      this.error = null
      try {
        const { data } = await participantsApi.get(id)
        this.currentParticipant = data
      } catch (err) {
        this.error = err.message ?? 'Failed to fetch participant'
      } finally {
        this.loading = false
      }
    },

    async createParticipant(payload) {
      const { data } = await participantsApi.create(payload)
      this.participants.push(data)
      this.total += 1
      return data
    },

    async updateParticipant(id, payload) {
      const { data } = await participantsApi.update(id, payload)
      const index = this.participants.findIndex((p) => p.id === id)
      if (index !== -1) {
        this.participants[index] = data
      }
      if (this.currentParticipant?.id === id) {
        this.currentParticipant = data
      }
      return data
    },

    async deleteParticipant(id) {
      await participantsApi.remove(id)
      this.participants = this.participants.filter((p) => p.id !== id)
      this.total = Math.max(0, this.total - 1)
      if (this.currentParticipant?.id === id) {
        this.currentParticipant = null
      }
    },
  },
})
