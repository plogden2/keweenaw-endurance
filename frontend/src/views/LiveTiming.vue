<template>
  <div class="live-timing">
    <div v-if="racesStore.loading" class="status">Loading race…</div>
    <div v-else-if="racesStore.error" class="status error">{{ racesStore.error }}</div>

    <template v-else-if="racesStore.currentRace">
      <router-link to="/timing" class="back-link">← Back to timing</router-link>
      <h1 class="page-title">{{ racesStore.currentRace.name }}</h1>
      <p class="meta">Live timing station</p>

      <div class="layout">
        <section class="lookup-panel">
          <h2 class="section-title">Participant Lookup</h2>

          <div class="lookup-row">
            <label>
              Bib number
              <input
                v-model="bibLookup"
                type="text"
                data-testid="bib-lookup"
                placeholder="Enter bib"
                @keyup.enter="lookupByBib"
              />
            </label>
            <button
              type="button"
              class="action-btn"
              data-testid="bib-lookup-btn"
              @click="lookupByBib"
            >
              Look up
            </button>
          </div>

          <div class="lookup-row">
            <label>
              RFID tag
              <input
                v-model="rfidLookup"
                type="text"
                data-testid="rfid-lookup"
                placeholder="Scan tag UID"
                @keyup.enter="lookupByRfid"
              />
            </label>
            <button
              type="button"
              class="action-btn"
              data-testid="rfid-lookup-btn"
              @click="lookupByRfid"
            >
              Scan
            </button>
          </div>

          <p v-if="lookupError" class="error">{{ lookupError }}</p>

          <div v-if="selectedParticipant" class="participant-card" data-testid="selected-participant">
            <strong>#{{ selectedParticipant.bib_number }}</strong>
            {{ selectedParticipant.first_name }}
            {{ selectedParticipant.last_name }}
            <span v-if="selectedParticipant.rfid_tag_uid" class="tag">
              {{ selectedParticipant.rfid_tag_uid }}
            </span>
          </div>
        </section>

        <section class="form-panel">
          <ManualTimingForm
            :race-id="raceId"
            :checkpoints="checkpoints"
            :submitting="submitting"
            @submit="onManualSubmit"
          />
        </section>

        <div class="sync-panel">
          <SyncStatus ref="syncStatusRef" @synced="refreshLive" />
        </div>
      </div>

      <section class="recent-records">
        <h2 class="section-title">Recent Records</h2>
        <div v-if="liveLoading" class="status">Loading records…</div>
        <div v-else-if="liveError" class="status error">{{ liveError }}</div>
        <table v-else-if="liveRecords.length" class="records-table">
          <thead>
            <tr>
              <th>Time</th>
              <th>Participant</th>
              <th>Checkpoint</th>
              <th>Sync</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="record in liveRecords" :key="record.id">
              <td>{{ formatTime(record.timestamp) }}</td>
              <td>
                <template v-if="record.participant">
                  #{{ record.participant.bib_number }}
                  {{ record.participant.first_name }}
                  {{ record.participant.last_name }}
                </template>
                <template v-else>{{ formatShortId(record.participant_id) }}</template>
              </td>
              <td>{{ record.checkpoint?.name ?? formatShortId(record.checkpoint_id) }}</td>
              <td>{{ record.sync_status }}</td>
            </tr>
          </tbody>
        </table>
        <p v-else class="empty">No timing records yet.</p>
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import ManualTimingForm from '@/components/ManualTimingForm.vue'
import SyncStatus from '@/components/SyncStatus.vue'
import { useRacesStore } from '@/stores/races'
import {
  checkpointsApi,
  participantsApi,
  rfidApi,
  timingApi,
} from '@/services/api'
import { enqueue } from '@/services/offlineQueue'
import type {
  Checkpoint,
  ManualTimingEntryPayload,
  Participant,
  TimingRecord,
} from '@/types/models'
import { getErrorMessage } from '@/utils/error'
import { formatShortId } from '@/utils/id'

const route = useRoute()
const racesStore = useRacesStore()
const syncStatusRef = ref<InstanceType<typeof SyncStatus> | null>(null)

const raceId = computed(() => String(route.params.raceId))
const checkpoints = ref<Checkpoint[]>([])
const liveRecords = ref<TimingRecord[]>([])
const liveLoading = ref(false)
const liveError = ref<string | null>(null)
const selectedParticipant = ref<Participant | null>(null)
const bibLookup = ref('')
const rfidLookup = ref('')
const lookupError = ref<string | null>(null)
const submitting = ref(false)

function formatTime(iso: string): string {
  try {
    return new Date(iso).toLocaleString()
  } catch {
    return iso
  }
}

async function loadRace(): Promise<void> {
  await racesStore.fetchRace(raceId.value)
}

async function loadCheckpoints(): Promise<void> {
  const { data } = await checkpointsApi.listByRace(raceId.value, { limit: 100 })
  checkpoints.value = data.data ?? []
}

async function refreshLive(): Promise<void> {
  liveLoading.value = true
  liveError.value = null
  try {
    const { data } = await timingApi.getLive(raceId.value)
    liveRecords.value = data.records ?? []
  } catch (err) {
    liveError.value = getErrorMessage(err, 'Failed to load live timing')
  } finally {
    liveLoading.value = false
  }
}

async function lookupByBib(): Promise<void> {
  lookupError.value = null
  selectedParticipant.value = null
  const bib = bibLookup.value.trim()
  if (!bib) {
    lookupError.value = 'Enter a bib number.'
    return
  }
  try {
    const { data } = await participantsApi.list({
      race_id: raceId.value,
      limit: 500,
    })
    const match = (data.data ?? []).find((p) => p.bib_number === bib)
    if (!match) {
      lookupError.value = `No participant found with bib ${bib}.`
      return
    }
    selectedParticipant.value = match
  } catch (err) {
    lookupError.value = getErrorMessage(err, 'Bib lookup failed')
  }
}

async function lookupByRfid(): Promise<void> {
  lookupError.value = null
  selectedParticipant.value = null
  const uid = rfidLookup.value.trim()
  if (!uid) {
    lookupError.value = 'Enter an RFID tag UID.'
    return
  }
  try {
    const { data } = await rfidApi.scan(uid)
    if (data.race_id !== raceId.value) {
      lookupError.value = 'Participant is not registered for this race.'
      return
    }
    selectedParticipant.value = data
  } catch (err) {
    lookupError.value = getErrorMessage(err, 'RFID scan failed')
  }
}

async function onManualSubmit(payload: ManualTimingEntryPayload): Promise<void> {
  submitting.value = true
  lookupError.value = null
  try {
    const result = await enqueue(payload)
    if (result === 'queued') {
      lookupError.value = 'Recorded locally — will sync when online.'
    }
    bibLookup.value = ''
    rfidLookup.value = ''
    selectedParticipant.value = null
    if (result === 'synced') {
      await refreshLive()
    }
    await syncStatusRef.value?.loadStatus()
  } catch (err) {
    lookupError.value = getErrorMessage(err, 'Failed to record timing')
  } finally {
    submitting.value = false
  }
}

onMounted(async () => {
  await loadRace()
  await loadCheckpoints()
  await refreshLive()
})

watch(raceId, async () => {
  await loadRace()
  await loadCheckpoints()
  await refreshLive()
})
</script>

<style scoped>
.live-timing {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 2rem 2rem;
}

.page-title {
  color: var(--ink);
  margin-bottom: 0.25rem;
}

.meta {
  color: var(--muted);
  margin-bottom: 1.5rem;
}

.back-link {
  display: inline-block;
  margin-bottom: 1rem;
  color: var(--accent-link);
  text-decoration: none;
}

.layout {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1.5rem;
  margin-bottom: 1.5rem;
}

.lookup-panel,
.form-panel {
  min-width: 0;
}

.section-title {
  margin: 0 0 1rem;
  font-size: 1.1rem;
  color: var(--ink);
}

.lookup-row {
  display: flex;
  gap: 0.75rem;
  align-items: flex-end;
  margin-bottom: 1rem;
}

.lookup-row label {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  font-size: 0.9rem;
  color: var(--muted);
}

.lookup-row input {
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--border);
  border-radius: 4px;
}

.action-btn {
  padding: 0.5rem 1rem;
  background: var(--accent-link);
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  white-space: nowrap;
}

.participant-card {
  margin-top: 1rem;
  padding: 1rem;
  background: var(--mist);
  border-radius: 6px;
  color: var(--ink);
}

.participant-card .tag {
  display: block;
  font-size: 0.85rem;
  color: var(--muted);
  margin-top: 0.25rem;
}

.recent-records {
  margin-top: 1rem;
}

.records-table {
  width: 100%;
  border-collapse: collapse;
}

.records-table th,
.records-table td {
  padding: 0.5rem 0.75rem;
  text-align: left;
  border-bottom: 1px solid var(--border);
}

.records-table th {
  color: var(--muted);
  font-weight: 600;
}

.status {
  color: var(--muted);
}

.status.error,
.error {
  color: var(--signal);
}

.empty {
  color: var(--muted);
}

.sync-panel {
  grid-column: 1 / -1;
}

@media (max-width: 900px) {
  .layout {
    grid-template-columns: 1fr;
  }
}
</style>
