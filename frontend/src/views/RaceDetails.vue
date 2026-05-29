<template>
  <div class="race-details">
    <AppHeader />

    <div v-if="racesStore.loading" class="status">Loading race…</div>
    <div v-else-if="racesStore.error" class="status error">{{ racesStore.error }}</div>

    <template v-else-if="racesStore.currentRace">
      <router-link :to="`/timing/${eventId}`" class="back-link">← Back to event</router-link>
      <h1 class="page-title">{{ racesStore.currentRace.name }}</h1>
      <p class="meta">
        {{ racesStore.currentRace.race_type }} ·
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
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import AppHeader from '@/components/AppHeader.vue'
import { useRacesStore } from '@/stores/races'
import { timingApi } from '@/services/api'
import type { LeaderboardEntry } from '@/types/models'
import { getErrorMessage } from '@/utils/error'

const route = useRoute()
const racesStore = useRacesStore()

const eventId = computed(() => String(route.params.eventId))
const raceId = computed(() => String(route.params.raceId))
const activeTab = ref('leaderboard')
const leaderboard = ref<LeaderboardEntry[]>([])
const leaderboardLoading = ref(false)
const leaderboardError = ref<string | null>(null)

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

onMounted(async () => {
  await loadRace()
  await loadLeaderboard()
})

watch(raceId, async () => {
  await loadRace()
  await loadLeaderboard()
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
</style>
