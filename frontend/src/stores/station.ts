import { defineStore } from 'pinia'
import { stationsApi, type StationConfig } from '@/services/api'
import { getErrorMessage } from '@/utils/error'

interface StationState {
  eventId: string | null
  mode: 'finish' | 'checkpoint'
  deviceId: string
  name: string
  checkpointId: string | null
  loading: boolean
  error: string | null
}

function applyStationPayload(state: StationState, data: Partial<StationConfig> & {
  station?: Partial<StationConfig>
}) {
  const src = data.station ?? data
  state.eventId = src.event_id ?? null
  state.mode = src.mode === 'checkpoint' ? 'checkpoint' : 'finish'
  state.deviceId = src.device_id ?? ''
  state.name = src.name ?? ''
  state.checkpointId = src.checkpoint_id ?? null
}

function toConfig(state: StationState): StationConfig {
  return {
    event_id: state.eventId,
    mode: state.mode,
    checkpoint_id: state.checkpointId,
    device_id: state.deviceId,
    name: state.name,
  }
}

export const useStationStore = defineStore('station', {
  state: (): StationState => ({
    eventId: null,
    mode: 'finish',
    deviceId: '',
    name: '',
    checkpointId: null,
    loading: false,
    error: null,
  }),

  getters: {
    isConfigured: (state): boolean =>
      Boolean(state.eventId && state.deviceId && state.name),
    currentConfig: (state): StationConfig => toConfig(state),
  },

  actions: {
    async fetchCurrent() {
      this.loading = true
      this.error = null
      try {
        const { data } = await stationsApi.getCurrent()
        applyStationPayload(this, data)
        this.loading = false
        return data
      } catch (err) {
        this.error = getErrorMessage(err, 'Failed to load station config')
        this.loading = false
        throw err
      }
    },

    async saveCurrent(partial: Partial<StationConfig> = {}) {
      this.loading = true
      this.error = null
      try {
        const payload: StationConfig = {
          ...toConfig(this),
          ...partial,
        }
        const { data } = await stationsApi.putCurrent(payload)
        applyStationPayload(this, data)
        this.loading = false
        return data
      } catch (err) {
        this.error = getErrorMessage(err, 'Failed to save station config')
        this.loading = false
        throw err
      }
    },
  },
})
