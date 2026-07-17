<template>
  <div class="pin-unlock">
    <h1 class="page-title">Organizer PIN</h1>
    <p class="lead">
      Unlocks management: racers, station config, race create/delete, CSV. Live
      leaderboard stays public without PIN.
    </p>

    <form
      v-if="!pinAuth.isAuthenticated"
      class="panel pin-shell"
      data-testid="pin-form"
      @submit.prevent="onSubmit"
    >
      <p class="muted">Enter organizer PIN</p>
      <div class="pin-display" aria-live="polite" aria-label="PIN digits entered">
        {{ pinDisplay }}
      </div>
      <p v-if="pinAuth.error || localError" class="err" role="alert">
        {{ pinAuth.error || localError }}
      </p>
      <div class="pad" aria-label="PIN keypad">
        <button
          v-for="k in keys"
          :key="k"
          type="button"
          @click="onKey(k)"
        >
          {{ k === 'clear' ? 'C' : k === 'ok' ? 'OK' : k }}
        </button>
      </div>
      <label class="sr-only" for="pin-input">PIN</label>
      <input
        id="pin-input"
        v-model="pin"
        data-testid="pin-input"
        type="password"
        inputmode="numeric"
        maxlength="8"
        autocomplete="off"
        placeholder="Or type PIN"
      />
      <button
        type="submit"
        class="btn"
        data-testid="pin-submit"
        :disabled="pinAuth.loading"
      >
        {{ pinAuth.loading ? 'Unlocking…' : 'Unlock management' }}
      </button>
      <p class="muted link-row">
        <router-link v-if="managementEventId" :to="`/events/${managementEventId}/live`">
          Continue to live view without unlocking
        </router-link>
        <router-link v-else to="/">Continue without unlocking</router-link>
      </p>
    </form>

    <div v-else class="mgmt">
      <p class="meta-bar">
        <span class="badge online">Management unlocked</span>
        <span class="muted">PIN session active — race create/delete lives here</span>
        <button type="button" class="btn secondary" @click="pinAuth.logout()">
          Lock
        </button>
      </p>

      <section
        class="panel"
        data-testid="race-management"
        aria-labelledby="race-mgmt-heading"
      >
        <h2 id="race-mgmt-heading">
          Races{{ managementEventName ? ` — ${managementEventName}` : '' }}
        </h2>
        <p class="muted intro">
          Create or delete races under this event. Delete asks for confirmation.
        </p>
        <p v-if="mgmtError" class="err" role="alert">{{ mgmtError }}</p>
        <div v-if="racesLoading" class="muted">Loading races…</div>
        <div v-else data-testid="race-list" class="race-list">
          <div
            v-for="race in racesStore.races"
            :key="race.id"
            class="race-row"
            data-testid="race-row"
          >
            <div>
              <strong>{{ race.name }}</strong>
              <div class="race-meta">
                {{ formatRaceMeta(race) }}
              </div>
            </div>
            <button
              type="button"
              class="btn secondary"
              data-testid="delete-race"
              @click="pendingDelete = race"
            >
              Delete
            </button>
          </div>
          <p v-if="!racesStore.races.length" class="muted">No races yet.</p>
        </div>
      </section>

      <form
        class="panel"
        aria-labelledby="create-race-heading"
        @submit.prevent="onCreateRace"
      >
        <h2 id="create-race-heading">Create race</h2>
        <label
          >Name
          <input
            v-model="createForm.name"
            data-testid="create-race-name"
            required
            placeholder="e.g. 4 Hour Fun Run"
          />
        </label>
        <div class="grid-2">
          <label
            >Duration
            <select
              v-model="createForm.duration"
              data-testid="create-race-duration"
              required
            >
              <option value="90">90 minutes</option>
              <option value="360">6 hours</option>
              <option value="720">12 hours</option>
            </select>
          </label>
          <label
            >Start time
            <input
              v-model="createForm.startTime"
              data-testid="create-race-start-time"
              type="time"
              required
            />
          </label>
        </div>
        <div class="row">
          <button
            type="submit"
            class="btn"
            data-testid="create-race"
            :disabled="creating || !managementEventId"
          >
            {{ creating ? 'Creating…' : 'Create race' }}
          </button>
        </div>
      </form>

      <div
        v-if="pendingDelete"
        class="confirm-overlay"
        role="dialog"
        aria-modal="true"
        aria-labelledby="delete-race-title"
      >
        <div class="confirm-panel panel">
          <h2 id="delete-race-title">Delete race?</h2>
          <p>
            Delete “{{ pendingDelete.name }}”? Racers in this race will no longer
            score laps. This cannot be undone from the list.
          </p>
          <div class="row">
            <button
              type="button"
              class="btn secondary"
              data-testid="delete-race-cancel"
              @click="pendingDelete = null"
            >
              Cancel
            </button>
            <button
              type="button"
              class="btn danger"
              data-testid="delete-race-confirm"
              :disabled="deleting"
              @click="onConfirmDelete"
            >
              {{ deleting ? 'Deleting…' : 'Delete' }}
            </button>
          </div>
        </div>
      </div>

      <section class="panel">
        <h2>Other management</h2>
        <div class="row">
          <router-link class="btn secondary" to="/station">Station config</router-link>
          <router-link
            class="btn secondary"
            :to="managementEventId ? `/csv?eventId=${managementEventId}` : '/csv'"
          >
            CSV recovery
          </router-link>
          <router-link
            v-if="managementEventId"
            class="btn secondary"
            :to="`/events/${managementEventId}/live`"
          >
            Live view
          </router-link>
          <router-link class="btn secondary" to="/">Home</router-link>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { eventsApi, setAuthToken } from '@/services/api'
import { usePinAuthStore } from '@/stores/pinAuth'
import { useRacesStore } from '@/stores/races'
import { useStationStore } from '@/stores/station'
import { BLUFFET_EVENT_NAME } from '@/themes/bluffetConstants'
import type { Race } from '@/types/models'
import { getErrorMessage } from '@/utils/error'

const pinAuth = usePinAuthStore()
const station = useStationStore()
const racesStore = useRacesStore()

const pin = ref('')
const localError = ref<string | null>(null)
const mgmtError = ref<string | null>(null)
const keys = ['1', '2', '3', '4', '5', '6', '7', '8', '9', 'clear', '0', 'ok'] as const

const managementEventId = ref<string | null>(null)
const managementEventName = ref<string | null>(null)
const managementEventDate = ref<string>('2026-08-01')
const racesLoading = ref(false)
const creating = ref(false)
const deleting = ref(false)
const pendingDelete = ref<Race | null>(null)

const createForm = ref({
  name: '',
  duration: '720',
  startTime: '08:00',
})

const pinDisplay = computed(() => (pin.value ? '•'.repeat(pin.value.length) : '••••'))

function onKey(k: (typeof keys)[number]) {
  if (k === 'clear') {
    pin.value = ''
    localError.value = null
    return
  }
  if (k === 'ok') {
    void onSubmit()
    return
  }
  if (pin.value.length < 8) pin.value += k
}

async function resolveManagementEvent() {
  await station.fetchCurrent().catch(() => {
    /* station may be unconfigured */
  })

  const { data } = await eventsApi.list({ limit: 100 })
  const events = data.data ?? []
  const byStation = station.eventId
    ? events.find((e) => e.id === station.eventId)
    : undefined
  const bluffet = events.find((e) => e.name === BLUFFET_EVENT_NAME)
  const chosen = byStation ?? bluffet ?? events[0]
  if (!chosen) {
    managementEventId.value = null
    managementEventName.value = null
    return
  }
  managementEventId.value = chosen.id
  managementEventName.value = chosen.name
  managementEventDate.value = (chosen.event_date || '2026-08-01').slice(0, 10)
}

async function loadRaces() {
  if (!managementEventId.value) return
  racesLoading.value = true
  mgmtError.value = null
  try {
    await racesStore.fetchRaces({
      event_id: managementEventId.value,
      limit: 100,
    })
  } catch (err) {
    mgmtError.value = getErrorMessage(err, 'Failed to load races')
  } finally {
    racesLoading.value = false
  }
}

async function onSubmit() {
  localError.value = null
  if (!pin.value) {
    localError.value = 'Enter PIN'
    return
  }
  try {
    await pinAuth.loginWithPin(pin.value)
    setAuthToken(pinAuth.token)
    pin.value = ''
    await resolveManagementEvent()
    await loadRaces()
  } catch {
    pin.value = ''
  }
}

function durationLabel(minutes: number | undefined): string {
  if (!minutes) return ''
  if (minutes === 90) return '90 minutes'
  if (minutes === 360) return '6 hours'
  if (minutes === 720) return '12 hours'
  if (minutes % 60 === 0) return `${minutes / 60} hours`
  return `${minutes} minutes`
}

function formatRaceMeta(race: Race): string {
  const parts: string[] = []
  if (race.start_time) {
    const d = new Date(race.start_time)
    if (!Number.isNaN(d.getTime())) {
      const partsFmt = new Intl.DateTimeFormat('en-US', {
        timeZone: 'America/Detroit',
        hour: '2-digit',
        minute: '2-digit',
        hour12: false,
      }).formatToParts(d)
      const hour = partsFmt.find((p) => p.type === 'hour')?.value ?? ''
      const minute = partsFmt.find((p) => p.type === 'minute')?.value ?? ''
      parts.push(`Starts ${hour.padStart(2, '0')}:${minute.padStart(2, '0')}`)
    }
  }
  const dur = durationLabel(race.duration_minutes)
  if (dur) parts.push(dur)
  return parts.join(' · ') || race.race_type
}

/** Combine event date + HH:MM as America/Detroit wall time (EDT offset for seed season). */
function toStartRFC3339(eventDate: string, timeHHMM: string): string {
  return `${eventDate}T${timeHHMM}:00-04:00`
}

async function onCreateRace() {
  if (!managementEventId.value) {
    mgmtError.value = 'No event available for race create'
    return
  }
  creating.value = true
  mgmtError.value = null
  try {
    await racesStore.createRace({
      event_id: managementEventId.value,
      name: createForm.value.name.trim(),
      race_type: 'lap_based',
      duration_minutes: Number(createForm.value.duration),
      start_time: toStartRFC3339(
        managementEventDate.value,
        createForm.value.startTime,
      ),
      status: 'scheduled',
    })
    createForm.value = { name: '', duration: '720', startTime: '08:00' }
    await loadRaces()
  } catch (err) {
    mgmtError.value = getErrorMessage(err, 'Failed to create race')
  } finally {
    creating.value = false
  }
}

async function onConfirmDelete() {
  if (!pendingDelete.value) return
  deleting.value = true
  mgmtError.value = null
  try {
    await racesStore.deleteRace(pendingDelete.value.id)
    pendingDelete.value = null
    await loadRaces()
  } catch (err) {
    mgmtError.value = getErrorMessage(err, 'Failed to delete race')
  } finally {
    deleting.value = false
  }
}

watch(
  () => pinAuth.isAuthenticated,
  async (ok) => {
    if (ok) {
      await resolveManagementEvent()
      await loadRaces()
    }
  },
)

onMounted(async () => {
  await resolveManagementEvent()
  if (pinAuth.isAuthenticated) {
    setAuthToken(pinAuth.token)
    await loadRaces()
  }
})
</script>

<style scoped>
.pin-unlock {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 2rem;
  --line: var(--border);
}

.page-title {
  color: var(--ink);
  text-align: center;
}

.lead {
  text-align: center;
  color: var(--muted);
  max-width: 36rem;
  margin: 0 auto 1.5rem;
}

.panel {
  background: var(--surface);
  border: 1px solid var(--line);
  border-radius: 6px;
  padding: 1.25rem;
  margin-bottom: 1rem;
}

.pin-shell {
  max-width: 24rem;
  margin: 0 auto;
  text-align: center;
}

.pin-display {
  font-size: 2rem;
  letter-spacing: 0.45em;
  font-variant-numeric: tabular-nums;
  padding: 0.75rem;
  border: 1px solid var(--line);
  border-radius: 6px;
  background: var(--surface);
  margin: 1rem 0;
  min-height: 3rem;
}

.pad {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 0.5rem;
  max-width: 16rem;
  margin: 0 auto 1rem;
}

.pad button {
  padding: 1rem;
  font-size: 1.25rem;
  border: 1px solid var(--line);
  border-radius: 6px;
  background: var(--surface);
  cursor: pointer;
}

.pad button:hover {
  background: var(--mist);
}

.err {
  color: var(--signal);
  min-height: 1.25rem;
}

input[type='password'],
input[type='text'],
input[type='time'],
input:not([type]),
select {
  width: 100%;
  margin-top: 0.35rem;
  margin-bottom: 0.75rem;
  padding: 0.55rem 0.75rem;
  border: 1px solid var(--line);
  border-radius: 4px;
  font: inherit;
  box-sizing: border-box;
}

label {
  display: block;
  font-weight: 600;
  color: var(--ink);
  margin-bottom: 0.5rem;
}

.btn {
  border: none;
  border-radius: 4px;
  padding: 0.55rem 1rem;
  font: inherit;
  cursor: pointer;
  background: var(--accent-link);
  color: #fff;
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn.secondary {
  background: var(--mist);
  color: var(--ink);
}

.btn.danger {
  background: var(--signal);
  color: #fff;
}

.pin-shell .btn {
  width: 100%;
}

.muted {
  color: var(--muted);
}

.intro {
  margin-top: 0;
}

.link-row {
  margin-top: 1rem;
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  border: 0;
}

.meta-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  align-items: center;
  margin-bottom: 1rem;
}

.badge.online {
  background: color-mix(in srgb, var(--success) 15%, var(--surface));
  color: var(--success);
  padding: 0.2rem 0.55rem;
  border-radius: 4px;
  font-size: 0.85rem;
}

.row {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.grid-2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
}

.mgmt {
  max-width: 40rem;
  margin: 0 auto;
}

.race-row {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  align-items: center;
  justify-content: space-between;
  padding: 0.65rem 0;
  border-bottom: 1px solid var(--line);
}

.race-row:last-child {
  border-bottom: none;
}

.race-meta {
  color: var(--muted);
  font-size: 0.9rem;
}

.confirm-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.35);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 50;
  padding: 1rem;
}

.confirm-panel {
  max-width: 24rem;
  width: 100%;
  margin: 0;
}

@media (max-width: 640px) {
  .grid-2 {
    grid-template-columns: 1fr;
  }
}
</style>
