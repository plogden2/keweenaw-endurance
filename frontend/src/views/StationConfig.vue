<template>
  <div class="station-config" data-testid="station-config">
    <h1 class="page-title">Reader station</h1>
    <p class="lead">
      Bind this laptop to an <strong>event</strong> (not a single race). Default mode is
      Finish (+1 lap).
    </p>

    <p
      v-if="station.isConfigured"
      class="armed"
      data-testid="station-armed-indicator"
      role="status"
    >
      Armed — {{ station.name || station.deviceId }} · {{ station.mode }} · event
      {{ station.eventId }}
    </p>

    <form class="panel" data-testid="station-form" @submit.prevent="onSave">
      <label>
        Event
        <select v-model="form.event_id" required data-testid="station-event-select">
          <option disabled value="">Select event…</option>
          <option v-for="ev in events" :key="ev.id" :value="ev.id">
            {{ ev.name }}
          </option>
        </select>
      </label>
      <label>
        Station name
        <input v-model="form.name" required data-testid="station-name" />
      </label>
      <label>
        Device ID
        <input v-model="form.device_id" required data-testid="station-device-id" />
      </label>

      <label>
        Mode
        <select v-model="form.mode" required data-testid="station-mode">
          <option value="finish">Finish station (default) — each valid tap = +1 lap</option>
          <option value="checkpoint">Checkpoint — advances when sequence is satisfied</option>
        </select>
      </label>

      <label v-if="form.mode === 'checkpoint'">
        Checkpoint
        <select
          v-model="form.checkpoint_id"
          required
          data-testid="checkpoint-picker"
        >
          <option disabled value="">Select checkpoint…</option>
          <option
            v-for="cp in checkpointOptions"
            :key="cp.id"
            :value="cp.id"
          >
            {{ cp.label }}
          </option>
        </select>
      </label>

      <p v-if="station.error || localError" class="error" role="alert">
        {{ station.error || localError }}
      </p>

      <div class="row">
        <button
          type="submit"
          class="btn"
          data-testid="station-save"
          :disabled="station.loading"
        >
          {{ station.loading ? 'Saving…' : 'Save & arm reader' }}
        </button>
        <router-link
          v-if="form.event_id"
          class="btn secondary"
          :to="`/events/${form.event_id}/live`"
        >
          Go to live view
        </router-link>
      </div>
    </form>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { checkpointsApi, eventsApi, racesApi } from '@/services/api'
import { useStationStore } from '@/stores/station'
import { usePinAuthStore } from '@/stores/pinAuth'
import type { Event } from '@/types/models'
import { getErrorMessage } from '@/utils/error'

const station = useStationStore()
const pinAuth = usePinAuthStore()
const router = useRouter()

const events = ref<Event[]>([])
const localError = ref<string | null>(null)
const checkpointOptions = ref<Array<{ id: string; label: string }>>([])
let pollTimer: number | undefined

const form = reactive({
  event_id: '' as string,
  name: 'Finish Mat A',
  device_id: 'laptop-finish-1',
  mode: 'finish' as 'finish' | 'checkpoint',
  checkpoint_id: '' as string,
})

const needsCheckpoint = computed(() => form.mode === 'checkpoint')

async function loadCheckpointsForEvent(eventId: string) {
  checkpointOptions.value = []
  if (!eventId) return
  try {
    const { data: racesData } = await racesApi.list({ event_id: eventId, limit: 100 })
    const races = racesData.data ?? []
    const options: Array<{ id: string; label: string }> = []
    for (const race of races) {
      const { data: cpData } = await checkpointsApi.listByRace(race.id, { limit: 100 })
      for (const cp of cpData.data ?? []) {
        options.push({
          id: cp.id,
          label: `${race.name} — ${cp.name} (${cp.checkpoint_type})`,
        })
      }
    }
    // Prefer race order then start → intermediate → finish within each race group as returned.
    checkpointOptions.value = options
    if (
      form.checkpoint_id &&
      !options.some((o) => o.id === form.checkpoint_id)
    ) {
      form.checkpoint_id = ''
    }
  } catch {
    checkpointOptions.value = []
  }
}

async function onSave() {
  localError.value = null
  if (!pinAuth.isAuthenticated) {
    localError.value = 'Unlock with organizer PIN first'
    await router.push('/pin')
    return
  }
  if (needsCheckpoint.value && !form.checkpoint_id) {
    localError.value = 'Select a checkpoint for checkpoint mode'
    return
  }
  try {
    await station.saveCurrent({
      event_id: form.event_id,
      name: form.name,
      device_id: form.device_id,
      mode: form.mode,
      checkpoint_id: form.mode === 'checkpoint' ? form.checkpoint_id : null,
    })
    await router.push(`/events/${form.event_id}/live`)
  } catch (err) {
    localError.value = getErrorMessage(err, 'Failed to arm station')
  }
}

async function refreshStation() {
  try {
    await station.fetchCurrent()
    if (station.eventId) form.event_id = station.eventId
    if (station.name) form.name = station.name
    if (station.deviceId) form.device_id = station.deviceId
    form.mode = station.mode
    form.checkpoint_id = station.checkpointId ?? ''
  } catch {
    /* unconfigured */
  }
}

watch(
  () => form.event_id,
  (id) => {
    void loadCheckpointsForEvent(id)
  },
)

onMounted(async () => {
  try {
    const { data } = await eventsApi.list({ limit: 100 })
    events.value = data.data ?? []
  } catch {
    events.value = []
  }

  await refreshStation()
  if (form.event_id) {
    await loadCheckpointsForEvent(form.event_id)
  }
  // Poll so e2e (and multi-tab) API arming updates the armed indicator
  pollTimer = window.setInterval(() => {
    void refreshStation()
  }, 1500)
})

onUnmounted(() => {
  if (pollTimer) window.clearInterval(pollTimer)
})
</script>

<style scoped>
.station-config {
  max-width: 40rem;
  margin: 0 auto;
  padding: 0 2rem;
  --line: var(--border);
}

.page-title {
  color: var(--ink);
}

.lead {
  color: var(--muted);
}

.armed {
  background: color-mix(in srgb, var(--success) 15%, var(--surface));
  color: var(--success);
  padding: 0.65rem 0.85rem;
  border-radius: 6px;
  margin-bottom: 1rem;
  font-weight: 600;
}

.panel {
  background: var(--surface);
  border: 1px solid var(--line);
  border-radius: 6px;
  padding: 1.25rem;
}

label {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  margin-bottom: 1rem;
  font-weight: 600;
  color: var(--ink);
}

label.inline {
  flex-direction: row;
  align-items: center;
  gap: 0.5rem;
  font-weight: 400;
}

input,
select {
  padding: 0.55rem 0.75rem;
  border: 1px solid var(--line);
  border-radius: 4px;
  font: inherit;
  font-weight: 400;
}

.row {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.btn {
  border: none;
  border-radius: 4px;
  padding: 0.55rem 1rem;
  font: inherit;
  cursor: pointer;
  background: var(--accent-link);
  color: #fff;
  text-decoration: none;
  display: inline-block;
}

.btn.secondary {
  background: var(--mist);
  color: var(--ink);
}

.error {
  color: var(--signal);
}
</style>
