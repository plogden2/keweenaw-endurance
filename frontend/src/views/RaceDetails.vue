<template>
  <div class="race-details">
    <div v-if="racesStore.loading" class="status">Loading race…</div>
    <div v-else-if="racesStore.error" class="status error">{{ racesStore.error }}</div>

    <template v-else-if="racesStore.currentRace">
      <router-link :to="`/timing/${eventId}`" class="back-link">← Back to event</router-link>
      <h1 class="page-title">{{ racesStore.currentRace.name }}</h1>
      <p class="meta">
        {{ racesStore.currentRace.race_type }}
        <template v-if="showRaceDistance">
          · {{ formatRaceDistance(racesStore.currentRace.distance_km!) }}
        </template>
        ·
        <span :class="`status-${racesStore.currentRace.status}`">
          {{ racesStore.currentRace.status }}
        </span>
      </p>

      <div class="tabs">
        <button
          type="button"
          class="tab"
          :class="{ active: activeTab === 'leaderboard' }"
          @click="activeTab = 'leaderboard'"
        >
          Leaderboard
        </button>
        <button
          type="button"
          class="tab"
          :class="{ active: activeTab === 'race-flow' }"
          @click="activeTab = 'race-flow'"
        >
          Race Flow
        </button>
        <button
          type="button"
          class="tab"
          :class="{ active: activeTab === 'statistics' }"
          @click="activeTab = 'statistics'"
        >
          Statistics
        </button>
      </div>

      <section v-if="activeTab === 'leaderboard'" class="leaderboard">
        <div v-if="selectedEntry" class="participant-detail">
          <div class="participant-detail-header">
            <button type="button" class="back-to-leaderboard" @click="clearSelectedParticipant">
              ← Back to leaderboard
            </button>
          </div>

          <ResultCertificate
            v-if="certificateData"
            :event-title="certificateData.eventTitle"
            :event-name="certificateData.eventName"
            :event-date="certificateData.eventDate"
            :logo-url="certificateData.logoUrl"
            :participant-name="certificateData.participantName"
            :location="certificateData.location"
            :bib-number="certificateData.bibNumber"
            :race-name="certificateData.raceName"
            :category-label="certificateData.categoryLabel"
            :finish-time="certificateData.finishTime"
            :mph="certificateData.mph"
            :overall-rank="certificateData.overallRank"
            :gender-rank="certificateData.genderRank"
            :category-rank="certificateData.categoryRank"
            :gender-rank-label="certificateData.genderRankLabel"
            :category-rank-label="certificateData.categoryRankLabel"
            :leaderboard-to="leaderboardRoute"
            @view-leaderboard="clearSelectedParticipant"
          />

          <ParticipantFlowChart
            :race-id="raceId"
            :participant-id="selectedEntry.participant_id"
            :race-status="racesStore.currentRace.status"
            :race-start-time="racesStore.currentRace.start_time"
            :race-type="racesStore.currentRace.race_type"
          />

          <div class="participant-actions">
            <button type="button" class="compare-btn" @click="compareInRaceFlow">
              Compare in Race Flow
            </button>
          </div>
        </div>

        <template v-else>
          <div v-if="leaderboardLoading" class="status">Loading leaderboard…</div>
          <div v-else-if="leaderboardError" class="status error">{{ leaderboardError }}</div>
          <table v-else-if="leaderboard.length" class="leaderboard-table">
            <thead>
              <tr>
                <th>Pos</th>
                <th>Bib</th>
                <th>Name</th>
                <th>Location</th>
                <th>Result</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="entry in leaderboard"
                :key="entry.participant_id"
                :class="{ clickable: entry.status === 'finished' }"
                @click="selectParticipant(entry)"
              >
                <td>{{ entry.position }}</td>
                <td>{{ entry.bib_number }}</td>
                <td>{{ entry.first_name }} {{ entry.last_name }}</td>
                <td>{{ entry.location || '—' }}</td>
                <td>{{ formatResult(entry) }}</td>
                <td>{{ entry.status }}</td>
              </tr>
            </tbody>
          </table>
          <p v-else class="empty">No results yet.</p>
        </template>
      </section>

      <section v-else-if="activeTab === 'race-flow'" class="race-flow">
        <RaceFlowChart
          :race-id="raceId"
          :race-status="racesStore.currentRace.status"
          :race-start-time="racesStore.currentRace.start_time"
          :race-type="racesStore.currentRace.race_type"
          :highlight-participant-id="highlightParticipantId"
        />
      </section>

      <section v-else-if="activeTab === 'statistics'" class="statistics">
        <div v-if="statsLoading" class="status">Loading statistics…</div>
        <div v-else-if="statsError" class="status error">{{ statsError }}</div>
        <ul v-else class="stats-grid">
          <li>
            <span class="label">Participants</span>
            <span class="value">{{ statistics.totalParticipants }}</span>
          </li>
          <li>
            <span class="label">Finished</span>
            <span class="value">{{ statistics.finished }}</span>
          </li>
          <li>
            <span class="label">Started</span>
            <span class="value">{{ statistics.started }}</span>
          </li>
          <li>
            <span class="label">Registered</span>
            <span class="value">{{ statistics.registered }}</span>
          </li>
          <li>
            <span class="label">DNF</span>
            <span class="value">{{ statistics.dnf }}</span>
          </li>
          <li>
            <span class="label">{{ averageResultLabel }}</span>
            <span class="value">{{ averageResultValue }}</span>
          </li>
        </ul>
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import ParticipantFlowChart from '@/components/ParticipantFlowChart.vue'
import RaceFlowChart from '@/components/RaceFlowChart.vue'
import ResultCertificate from '@/components/ResultCertificate.vue'
import { useEventsStore } from '@/stores/events'
import { useRacesStore } from '@/stores/races'
import { useUnitsStore } from '@/stores/units'
import { participantsApi, timingApi } from '@/services/api'
import type { LeaderboardEntry, Participant, TimingRecord } from '@/types/models'
import {
  buildRaceStatistics,
  formatAverageResult,
  getAverageResultLabel,
  type RaceStatistics,
} from '@/utils/raceFlowData'
import {
  buildParticipantDetailsMap,
  computeParticipantRanks,
  formatAverageSpeedMph,
  formatCategoryLabel,
  formatCertificateFinishTime,
  getCategoryRankLabel,
  getGenderRankLabel,
  type ParticipantResultRanks,
} from '@/utils/participantResults'
import { getErrorMessage } from '@/utils/error'
import { formatDistance } from '@/utils/units'

const route = useRoute()
const racesStore = useRacesStore()
const eventsStore = useEventsStore()
const unitsStore = useUnitsStore()

const eventId = computed(() => String(route.params.eventId))
const raceId = computed(() => String(route.params.raceId))
const showRaceDistance = computed(() => {
  const race = racesStore.currentRace
  return race != null
    && race.race_type !== 'lap_based'
    && race.distance_km != null
    && race.distance_km > 0
})
const leaderboardRoute = computed(() => ({
  name: 'race-details',
  params: { eventId: eventId.value, raceId: raceId.value },
}))
const activeTab = ref('leaderboard')
const leaderboard = ref<LeaderboardEntry[]>([])
const leaderboardLoading = ref(false)
const leaderboardError = ref<string | null>(null)
const statsLoading = ref(false)
const statsError = ref<string | null>(null)
const selectedEntry = ref<LeaderboardEntry | null>(null)
const selectedParticipant = ref<Participant | null>(null)
const participantDetails = ref<Map<string, Pick<Participant, 'gender' | 'age'>>>(new Map())
const highlightParticipantId = ref<string | undefined>(undefined)
const statistics = ref<RaceStatistics>({
  totalParticipants: 0,
  finished: 0,
  started: 0,
  registered: 0,
  dnf: 0,
  averageFinishSeconds: null,
  averageLaps: null,
})

const finishedEntries = computed(() =>
  leaderboard.value.filter((entry) => entry.status === 'finished'),
)

const participantRanks = computed<ParticipantResultRanks | null>(() => {
  if (!selectedEntry.value) {
    return null
  }

  return computeParticipantRanks(
    selectedEntry.value,
    finishedEntries.value,
    participantDetails.value,
  )
})

const certificateData = computed(() => {
  if (!selectedEntry.value || !racesStore.currentRace || !participantRanks.value) {
    return null
  }

  const entry = selectedEntry.value
  const event = eventsStore.currentEvent
  const participant = selectedParticipant.value
  const eventName = event?.name ?? 'Race Event'
  const raceName = racesStore.currentRace.name

  return {
    eventTitle: event ? `${eventName} - ${raceName}` : raceName,
    eventName,
    eventDate: event?.event_date ?? racesStore.currentRace.start_time ?? '',
    participantName: `${entry.first_name} ${entry.last_name}`.trim(),
    location: participant?.location ?? entry.location ?? event?.location,
    bibNumber: entry.bib_number,
    raceName,
    categoryLabel: participant ? formatCategoryLabel(participant) : '—',
    finishTime: formatCertificateFinishTime(entry.total_time_seconds),
    mph: formatAverageSpeedMph(racesStore.currentRace.distance_km, entry.total_time_seconds),
    logoUrl: event?.logo_url || undefined,
    overallRank: participantRanks.value.overall,
    genderRank: participantRanks.value.gender,
    categoryRank: participantRanks.value.category,
    genderRankLabel: participant ? getGenderRankLabel(participant.gender) : 'Gender Rank',
    categoryRankLabel: participant ? getCategoryRankLabel(participant) : 'Category Rank',
  }
})

const averageResultLabel = computed(() =>
  getAverageResultLabel(racesStore.currentRace?.race_type ?? 'time_based'),
)
const averageResultValue = computed(() =>
  formatAverageResult(racesStore.currentRace?.race_type ?? 'time_based', statistics.value),
)

function formatRaceDistance(distanceKm: number): string {
  return formatDistance(distanceKm, unitsStore.unitSystem)
}

function formatResult(entry: LeaderboardEntry): string {
  if (entry.laps) {
    return `${entry.laps} laps`
  }
  if (entry.total_time_seconds) {
    const total = Math.round(entry.total_time_seconds)
    const h = Math.floor(total / 3600)
    const m = Math.floor((total % 3600) / 60)
    const s = total % 60
    return [h, m, s].map((n) => String(n).padStart(2, '0')).join(':')
  }
  return '—'
}

function clearSelectedParticipant(): void {
  selectedEntry.value = null
  selectedParticipant.value = null
}

async function selectParticipant(entry: LeaderboardEntry): Promise<void> {
  if (entry.status !== 'finished') {
    return
  }

  selectedEntry.value = entry
  selectedParticipant.value = null

  try {
    const { data } = await participantsApi.get(entry.participant_id)
    selectedParticipant.value = data
  } catch {
    selectedParticipant.value = null
  }
}

function compareInRaceFlow(): void {
  if (!selectedEntry.value) {
    return
  }

  highlightParticipantId.value = selectedEntry.value.participant_id
  clearSelectedParticipant()
  activeTab.value = 'race-flow'
}

function buildDetailsFromRecords(records: TimingRecord[]): void {
  const participants = records
    .map((record) => record.participant)
    .filter((participant): participant is Participant => participant != null)
    .map((participant) => ({
      id: participant.id,
      gender: participant.gender,
      age: participant.age,
    }))

  participantDetails.value = buildParticipantDetailsMap(participants)
}

async function loadRace(): Promise<void> {
  await racesStore.fetchRace(raceId.value)
}

async function loadEvent(): Promise<void> {
  await eventsStore.fetchEvent(eventId.value)
}

async function loadLeaderboard(): Promise<void> {
  leaderboardLoading.value = true
  leaderboardError.value = null
  try {
    const { data } = await timingApi.getLeaderboard(raceId.value)
    leaderboard.value = data.data ?? []
  } catch (err) {
    leaderboardError.value = getErrorMessage(err, 'Failed to load leaderboard')
  } finally {
    leaderboardLoading.value = false
  }
}

async function loadStatistics(): Promise<void> {
  statsLoading.value = true
  statsError.value = null
  try {
    const { data } = await timingApi.getLive(raceId.value)
    const records = (data.records ?? []) as TimingRecord[]
    buildDetailsFromRecords(records)
    statistics.value = buildRaceStatistics(
      records,
      racesStore.currentRace?.start_time,
      racesStore.currentRace?.race_type,
    )
  } catch (err) {
    statsError.value = getErrorMessage(err, 'Failed to load statistics')
  } finally {
    statsLoading.value = false
  }
}

onMounted(async () => {
  await Promise.all([loadRace(), loadEvent()])
  await loadLeaderboard()
  await loadStatistics()
})

watch(raceId, async () => {
  clearSelectedParticipant()
  highlightParticipantId.value = undefined
  await Promise.all([loadRace(), loadEvent()])
  await loadLeaderboard()
  await loadStatistics()
})

watch(activeTab, async (tab) => {
  if (tab === 'statistics') {
    await loadStatistics()
  }
})
</script>

<style scoped>
.race-details {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 2rem 2rem;
}

.back-link {
  display: inline-block;
  margin-bottom: 1rem;
  color: var(--accent-link);
  text-decoration: none;
}

.page-title {
  margin-bottom: 0.5rem;
  color: var(--ink);
}

.meta {
  color: var(--muted);
  margin-bottom: 1.5rem;
  text-transform: capitalize;
}

.tabs {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 1.5rem;
}

.tab {
  padding: 0.5rem 1rem;
  border: none;
  background: var(--border);
  border-radius: 4px;
  cursor: pointer;
}

.tab.active {
  background: var(--accent);
  color: white;
}

.leaderboard-table {
  width: 100%;
  border-collapse: collapse;
  background: white;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.08);
}

.leaderboard-table th,
.leaderboard-table td {
  padding: 0.75rem 1rem;
  text-align: left;
  border-bottom: 1px solid var(--border);
}

.leaderboard-table th {
  background: var(--mist);
  font-weight: 600;
}

.leaderboard-table tr.clickable {
  cursor: pointer;
}

.leaderboard-table tr.clickable:hover {
  background: color-mix(in srgb, var(--accent-link) 8%, var(--surface));
}

.participant-detail {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.participant-detail-header {
  display: flex;
  justify-content: flex-start;
}

.back-to-leaderboard {
  padding: 0.45rem 0.75rem;
  border: 1px solid var(--border);
  border-radius: 4px;
  background: var(--surface);
  color: var(--ink);
  cursor: pointer;
  font: inherit;
}

.back-to-leaderboard:hover {
  background: var(--mist);
}

.participant-actions {
  display: flex;
  justify-content: center;
}

.compare-btn {
  padding: 0.65rem 1.25rem;
  border: none;
  border-radius: 4px;
  background: var(--accent);
  color: white;
  cursor: pointer;
  font: inherit;
  font-weight: 600;
}

.compare-btn:hover {
  background: var(--ink-deep);
}

.status {
  color: var(--muted);
}

.status.error {
  color: var(--signal);
}

.empty {
  color: var(--muted);
}

.stats-grid {
  list-style: none;
  margin: 0;
  padding: 0;
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  gap: 1rem;
}

.stats-grid li {
  background: var(--mist);
  border-radius: 8px;
  padding: 1rem;
}

.stats-grid .label {
  display: block;
  font-size: 0.85rem;
  color: var(--muted);
}

.stats-grid .value {
  font-size: 1.5rem;
  font-weight: 600;
  color: var(--ink);
}
</style>
