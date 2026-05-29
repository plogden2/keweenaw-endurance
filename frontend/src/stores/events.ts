import { defineStore } from 'pinia'
import { eventsApi } from '@/services/api'
import type {
  CreateEventPayload,
  Event,
  ListParams,
  UpdateEventPayload,
} from '@/types/models'
import { getErrorMessage } from '@/utils/error'

interface EventsState {
  events: Event[]
  currentEvent: Event | null
  total: number
  page: number
  limit: number
  loading: boolean
  error: string | null
}

export const useEventsStore = defineStore('events', {
  state: (): EventsState => ({
    events: [],
    currentEvent: null,
    total: 0,
    page: 1,
    limit: 20,
    loading: false,
    error: null,
  }),

  getters: {
    upcomingEvents: (state): Event[] =>
      state.events.filter((e) => e.status === 'upcoming'),
    activeEvents: (state): Event[] =>
      state.events.filter((e) => e.status === 'active'),
    pastEvents: (state): Event[] =>
      state.events.filter((e) => e.status === 'completed'),
  },

  actions: {
    async fetchEvents(params: ListParams = {}) {
      this.loading = true
      this.error = null
      try {
        const { data } = await eventsApi.list(params)
        this.events = data.data ?? []
        this.total = data.total ?? 0
        this.page = data.page ?? this.page
        this.limit = data.limit ?? this.limit
      } catch (err) {
        this.error = getErrorMessage(err, 'Failed to fetch events')
      } finally {
        this.loading = false
      }
    },

    async fetchEvent(id: string) {
      this.loading = true
      this.error = null
      try {
        const { data } = await eventsApi.get(id)
        this.currentEvent = data
      } catch (err) {
        this.error = getErrorMessage(err, 'Failed to fetch event')
      } finally {
        this.loading = false
      }
    },

    async createEvent(payload: CreateEventPayload) {
      const { data } = await eventsApi.create(payload)
      this.events.push(data)
      this.total += 1
      return data
    },

    async updateEvent(id: string, payload: UpdateEventPayload) {
      const { data } = await eventsApi.update(id, payload)
      const index = this.events.findIndex((e) => e.id === id)
      if (index !== -1) {
        this.events[index] = data
      }
      if (this.currentEvent?.id === id) {
        this.currentEvent = data
      }
      return data
    },

    async deleteEvent(id: string) {
      await eventsApi.remove(id)
      this.events = this.events.filter((e) => e.id !== id)
      this.total = Math.max(0, this.total - 1)
      if (this.currentEvent?.id === id) {
        this.currentEvent = null
      }
    },
  },
})
