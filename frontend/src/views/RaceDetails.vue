<template>
  <div class="race-details">
    <div v-if="racesStore.loading" class="status">Loading race…</div>
    <div v-else-if="racesStore.error" class="status error">{{ racesStore.error }}</div>

    <template v-else-if="racesStore.currentRace">
      <router-link :to="`/timing/${eventId}`" class="back-link">← Back to event</router-link>
      <h1 class="page-title">{{ racesStore.currentRace.name }}</h1>
      <p class="meta">
        {{ racesStore.currentRace.race_type }}
        <template v-if="racesStore.currentRace.distance_km != null">
          · {{ formatRaceDistance(racesStore.currentRace.distance_km) }}
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
        <div v-if="leaderboardLoading" class="status">Loading leaderboard…</div>
        <div v-else-if="leaderboardError" class="status error">{{ leaderboardError }}</div>
        <table v-else-if="leaderboard.length" class="leaderboard-table">
          <thead>
            <tr>
              <th>Pos</th>
              <th>Bib</th>
              <th>Name</th>
              <th>Result</th>
              <th>Status</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="entry in leaderboard" :key="entry.participant_id">
              <td>{{ entry.position }}</td>
              <td>{{ entry.bib_number }}</td>
              <td>{{ entry.first_name }} {{ entry.last_name }}</td>
              <td>{{ formatResult(entry) }}</td>
              <td>{{ entry.status }}</td>
            </tr>
          </tbody>
        </table>
        <p v-else class="empty">No results yet.</p>
      </section>

      <section v-else-if="activeTab === 'race-flow'" class="race-flow">
        <RaceFlowChart
          :race-id="raceId"
          :race-status="racesStore.currentRace.status"
          :race-start-time="racesStore.currentRace.start_time"
          :race-type="racesStore.currentRace.race_type"
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
import RaceFlowChart from '@/components/RaceFlowChart.vue'
import { useRacesStore } from '@/stores/races'
import { useUnitsStore } from '@/stores/units'
import { timingApi } from '@/services/api'
import type { LeaderboardEntry, TimingRecord } from '@/types/models'
import {
  buildRaceStatistics,
  formatAverageResult,
  getAverageResultLabel,
  type RaceStatistics,
} from '@/utils/raceFlowData'
import { getErrorMessage } from '@/utils/error'
import { formatDistance } from '@/utils/units'

const route = useRoute()
const racesStore = useRacesStore()
const unitsStore = useUnitsStore()

const eventId = computed(() => String(route.params.eventId))
const raceId = computed(() => String(route.params.raceId))
const activeTab = ref('leaderboard')
const leaderboard = ref<LeaderboardEntry[]>([])
const leaderboardLoading = ref(false)
const leaderboardError = ref<string | null>(null)
const statsLoading = ref(false)
const statsError = ref<string | null>(null)
const statistics = ref<RaceStatistics>({
  totalParticipants: 0,
  finished: 0,
  started: 0,
  registered: 0,
  dnf: 0,
  averageFinishSeconds: null,
  averageLaps: null,
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

async function loadRace(): Promise<void> {
  await racesStore.fetchRace(raceId.value)
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
    statistics.value = buildRaceStatistics(
      (data.records ?? []) as TimingRecord[],
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
  await loadRace()
  await loadLeaderboard()
  await loadStatistics()
})

watch(raceId, async () => {
  await loadRace()
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
  color: #3498db;
  text-decoration: none;
}

.page-title {
  margin-bottom: 0.5rem;
  color: #2c3e50;
}

.meta {
  color: #6c757d;
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
  background: #e9ecef;
  border-radius: 4px;
  cursor: pointer;
}

.tab.active {
  background: #2c3e50;
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
  border-bottom: 1px solid #e9ecef;
}

.leaderboard-table th {
  background: #f8f9fa;
  font-weight: 600;
}

.status {
  color: #6c757d;
}

.status.error {
  color: #dc3545;
}

.empty {
  color: #6c757d;
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
  background: #f8f9fa;
  border-radius: 8px;
  padding: 1rem;
}

.stats-grid .label {
  display: block;
  font-size: 0.85rem;
  color: #6c757d;
}

.stats-grid .value {
  font-size: 1.5rem;
  font-weight: 600;
  color: #2c3e50;
}
</style>
