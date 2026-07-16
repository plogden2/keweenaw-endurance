<template>
  <div class="event-live" data-testid="live-view">
    <p v-if="isReaderSession" class="meta-bar sync-bar">
      <span class="sync-group" data-testid="sync-status">
        <span
          v-if="online"
          class="badge online"
          data-testid="sync-online"
        >
          Station online
        </span>
        <span
          v-else
          class="badge offline"
          data-testid="sync-offline"
        >
          Station offline
        </span>
        <span
          v-if="pendingSync > 0"
          class="badge pending"
          data-testid="sync-pending"
        >
          {{ pendingSync }} pending sync
        </span>
      </span>
    </p>

    <div v-if="loading" class="status">Loading live view…</div>
    <div v-else-if="error" class="status error">{{ error }}</div>

    <template v-else-if="live">
      <p class="meta-bar">
        <span>{{ live.event.name }}</span>
      </p>

      <h1 class="page-title">Live race flow</h1>

      <div class="toolbar">
        <div class="race-tabs" role="tablist" aria-label="Races">
          <button
            type="button"
            role="tab"
            data-testid="race-tab-12h"
            :aria-selected="activeTab === '12h'"
            @click="activeTab = '12h'"
          >
            12 Hour
          </button>
          <button
            type="button"
            role="tab"
            data-testid="race-tab-6h"
            :aria-selected="activeTab === '6h'"
            @click="activeTab = '6h'"
          >
            6 Hour
          </button>
          <button
            type="button"
            role="tab"
            data-testid="race-tab-90m"
            :aria-selected="activeTab === '90m'"
            @click="activeTab = '90m'"
          >
            90 Minute
          </button>
          <button
            type="button"
            role="tab"
            data-testid="overlap-chart-toggle"
            :aria-selected="activeTab === 'overlap'"
            @click="activeTab = 'overlap'"
          >
            Overlap (12h + 6h)
          </button>
        </div>
        <button
          type="button"
          class="btn"
          data-testid="fullscreen-rotator-toggle"
          @click="rotatorOpen = true"
        >
          Fullscreen rotate
        </button>
      </div>

      <div class="legend" data-testid="category-legend" role="list" aria-label="Category legend">
        <strong>Overall</strong>
        <span
          v-for="item in live.category_legend"
          :key="item.key"
          class="legend-item"
          role="listitem"
        >
          <i :style="{ background: item.color }" aria-hidden="true" />
          {{ item.label }}
        </span>
      </div>

      <div v-show="activeTab === '12h'" data-testid="race-panel-12h">
        <section class="panel">
          <h2>{{ race12?.name || '12 Hour' }}</h2>
          <p id="countdown-label-12h" class="countdown-label">Countdown</p>
          <p
            class="countdown"
            data-testid="live-countdown"
            role="timer"
            aria-live="polite"
            aria-labelledby="countdown-label-12h"
          >
            {{ formatCountdown(race12?.countdown_seconds ?? 0) }}
          </p>
        </section>
        <div class="chart-wrap" aria-label="Lap progress chart">
          <RaceFlowChart v-if="race12?.id" :race-id="race12.id" />
        </div>
        <section class="panel">
          <h2>Leaderboard — Combined overall</h2>
          <table data-testid="leaderboard-overall">
            <thead>
              <tr>
                <th>Place</th>
                <th>Bib</th>
                <th>Name</th>
                <th>Laps</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="e in race12?.leaderboard_overall || []"
                :key="e.participant_id"
                data-testid="leaderboard-row"
              >
                <td class="place">{{ e.place }}</td>
                <td>{{ e.bib_number }}</td>
                <td>
                  <span
                    class="cat-dot"
                    :style="{ background: colorFor(e.category_key) }"
                  />
                  {{ e.name }}
                </td>
                <td data-testid="leaderboard-laps">{{ e.laps }}</td>
              </tr>
              <tr v-if="!(race12?.leaderboard_overall?.length)">
                <td colspan="4">No results yet</td>
              </tr>
            </tbody>
          </table>
        </section>
      </div>

      <div v-show="activeTab === '6h'" data-testid="race-panel-6h">
        <section class="panel">
          <h2>{{ race6?.name || '6 Hour' }}</h2>
          <p id="countdown-label-6h" class="countdown-label">Countdown</p>
          <p
            class="countdown"
            role="timer"
            aria-live="polite"
            aria-labelledby="countdown-label-6h"
          >
            {{ formatCountdown(race6?.countdown_seconds ?? 0) }}
          </p>
        </section>
        <div class="chart-wrap">
          <RaceFlowChart v-if="race6?.id" :race-id="race6.id" />
        </div>
        <section class="panel">
          <h2>Leaderboard — Combined overall</h2>
          <table>
            <thead>
              <tr>
                <th>Place</th>
                <th>Bib</th>
                <th>Name</th>
                <th>Laps</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="e in race6?.leaderboard_overall || []" :key="e.participant_id">
                <td class="place">{{ e.place }}</td>
                <td>{{ e.bib_number }}</td>
                <td>{{ e.name }}</td>
                <td>{{ e.laps }}</td>
              </tr>
              <tr v-if="!(race6?.leaderboard_overall?.length)">
                <td colspan="4">No results yet</td>
              </tr>
            </tbody>
          </table>
        </section>
      </div>

      <div v-show="activeTab === '90m'" data-testid="race-panel-90m">
        <section class="panel">
          <h2>{{ race90?.name || '90 Minute' }}</h2>
          <p id="countdown-label-90m" class="countdown-label">Countdown</p>
          <p
            class="countdown"
            role="timer"
            aria-live="polite"
            aria-labelledby="countdown-label-90m"
          >
            {{ formatCountdown(race90?.countdown_seconds ?? 0) }}
          </p>
        </section>
        <div class="chart-wrap">
          <RaceFlowChart v-if="race90?.id" :race-id="race90.id" />
        </div>
        <section class="panel">
          <h2>Leaderboard — Combined overall</h2>
          <table>
            <thead>
              <tr>
                <th>Place</th>
                <th>Bib</th>
                <th>Name</th>
                <th>Laps</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="e in race90?.leaderboard_overall || []" :key="e.participant_id">
                <td class="place">{{ e.place }}</td>
                <td>{{ e.bib_number }}</td>
                <td>{{ e.name }}</td>
                <td>{{ e.laps }}</td>
              </tr>
              <tr v-if="!(race90?.leaderboard_overall?.length)">
                <td colspan="4">No results yet</td>
              </tr>
            </tbody>
          </table>
        </section>
      </div>

      <div v-show="activeTab === 'overlap'" data-testid="overlap-chart">
        <section class="panel">
          <h2>Overlapping races — 12 Hour + 6 Hour</h2>
          <p class="muted">Combined flow chart for concurrent races.</p>
        </section>
        <div class="chart-wrap overlap-charts" aria-label="Overlap race flow">
          <RaceFlowChart v-if="race12?.id" :race-id="race12.id" />
          <RaceFlowChart v-if="race6?.id" :race-id="race6.id" />
        </div>
      </div>

      <div
        v-if="rotatorOpen"
        class="fs-root"
        data-testid="fullscreen-rotator"
        aria-label="Fullscreen rotating race display"
      >
        <button type="button" class="btn secondary fs-exit" @click="rotatorOpen = false">
          Exit (Esc)
        </button>
        <div class="fs-top">
          <div>
            <h1>{{ race12?.name || '12 Hour' }}</h1>
            <p class="fs-meta">Fullscreen rotator · combined overall</p>
          </div>
        </div>
        <div class="fs-grid">
          <div class="fs-panel" data-testid="rotator-flow">
            <h2>Race flow</h2>
            <div class="chart-wrap dark">
              <RaceFlowChart v-if="race12?.id" :race-id="race12.id" />
            </div>
          </div>
          <div class="fs-panel" data-testid="rotator-leaderboard">
            <h2>Leaderboard — Combined overall</h2>
            <table>
              <thead>
                <tr>
                  <th>#</th>
                  <th>Bib</th>
                  <th>Name</th>
                  <th>Laps</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="e in race12?.leaderboard_overall || []"
                  :key="'fs-' + e.participant_id"
                >
                  <td>{{ e.place }}</td>
                  <td>{{ e.bib_number }}</td>
                  <td>{{ e.name }}</td>
                  <td>{{ e.laps }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import RaceFlowChart from '@/components/RaceFlowChart.vue'
import {
  eventsLiveApi,
  rfidApi,
  type EventLiveRace,
  type EventLiveResponse,
} from '@/services/api'
import {
  getLocalPendingCount,
  onOnline,
  onPendingChange,
  syncAll,
} from '@/services/offlineQueue'
import { setDisplayCache } from '@/services/timingStorage'
import { useReaderStation } from '@/composables/useReaderStation'
import { usePinAuthStore } from '@/stores/pinAuth'
import { useStationStore } from '@/stores/station'
import { getErrorMessage } from '@/utils/error'

const route = useRoute()
const eventId = computed(() => String(route.params.eventId))
const station = useStationStore()
const pinAuth = usePinAuthStore()
const { lastScan } = useReaderStation()
const isReaderSession = computed(() => pinAuth.isAuthenticated)

const live = ref<EventLiveResponse | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)
const activeTab = ref<'12h' | '6h' | '90m' | 'overlap'>('12h')
const rotatorOpen = ref(false)
const online = ref(typeof navigator !== 'undefined' ? navigator.onLine : true)
const pendingSync = ref(0)
let pollTimer: number | undefined
let removeOnlineListener: (() => void) | undefined
let removePendingListener: (() => void) | undefined

function formatCountdown(seconds: number): string {
  const s = Math.max(0, Math.floor(seconds))
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  const r = s % 60
  return [h, m, r].map((n) => String(n).padStart(2, '0')).join(':')
}

function matchRace(predicate: (name: string) => boolean): EventLiveRace | undefined {
  return live.value?.races.find((r) => predicate(r.name.toLowerCase()))
}

const race12 = computed(
  () =>
    matchRace((n) => n.includes('12')) ??
    // Compressed hardware dress-rehearsal names (30-minute stand-in for 12h).
    matchRace((n) => /\b30\b/.test(n) && n.includes('minute')),
)
const race6 = computed(
  () =>
    matchRace((n) => n.includes('6') && !n.includes('12')) ??
    matchRace((n) => /\b15\b/.test(n) && n.includes('minute')),
)
const race90 = computed(
  () =>
    matchRace((n) => n.includes('90') || n.includes('kids')) ??
    matchRace((n) => /\b5\b/.test(n) && n.includes('minute')),
)

function colorFor(key: string): string {
  return live.value?.category_legend.find((l) => l.key === key)?.color || '#6c757d'
}

async function refreshPending() {
  let local = 0
  try {
    local = await getLocalPendingCount()
  } catch {
    local = 0
  }
  let server = 0
  if (navigator.onLine) {
    try {
      const { data } = await rfidApi.getSyncStatus()
      server = data.pending_count || 0
    } catch {
      // ignore — may be unreachable while still "online"
    }
  }
  pendingSync.value = local + server
}

async function cacheLiveLabels(data: EventLiveResponse) {
  await setDisplayCache({
    event_id: data.event.id,
    event_name: data.event.name,
    races: data.races.map((r) => ({ id: r.id, name: r.name })),
    tags: {},
  })
}

async function loadLive() {
  loading.value = true
  error.value = null
  online.value = navigator.onLine
  try {
    const { data } = await eventsLiveApi.getLive(eventId.value)
    live.value = data
    online.value = navigator.onLine
    void cacheLiveLabels(data)
  } catch (err) {
    error.value = getErrorMessage(err, 'Failed to load live view')
    online.value = navigator.onLine
  } finally {
    loading.value = false
    await refreshPending()
  }
}

function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape') rotatorOpen.value = false
}

function onBrowserOnline() {
  online.value = true
  void syncAll().then(() => refreshPending())
}

function onBrowserOffline() {
  online.value = false
}

watch(eventId, () => {
  void loadLive()
})

watch(
  () => [lastScan.value?.timing_record_id, lastScan.value?.lap_count],
  () => {
    if (lastScan.value?.result === 'lap') {
      void loadLive()
    }
  },
)

onMounted(() => {
  if (isReaderSession.value) {
    void station.fetchCurrent().catch(() => {
      /* offline / unconfigured */
    })
  }

  void loadLive()
  online.value = typeof navigator !== 'undefined' ? navigator.onLine : true

  const syncOnlineState = () => {
    const next = typeof navigator !== 'undefined' ? navigator.onLine : true
    if (online.value !== next) {
      online.value = next
      if (next) {
        void syncAll().then(() => refreshPending())
      } else {
        void refreshPending()
      }
    }
  }

  pollTimer = window.setInterval(() => {
    syncOnlineState()
    if (!navigator.onLine) {
      void refreshPending()
      return
    }
    void eventsLiveApi
      .getLive(eventId.value)
      .then(({ data }) => {
        live.value = data
        online.value = navigator.onLine
        void refreshPending()
      })
      .catch(() => {
        online.value = navigator.onLine
        void refreshPending()
      })
  }, 2000)
  window.addEventListener('keydown', onKey)
  window.addEventListener('online', onBrowserOnline)
  window.addEventListener('offline', onBrowserOffline)
  removeOnlineListener = onOnline(() => {
    online.value = true
    void refreshPending()
  })
  removePendingListener = onPendingChange((count) => {
    if (count > 0) {
      pendingSync.value = count
    }
    void refreshPending()
  })
})

onUnmounted(() => {
  if (pollTimer) window.clearInterval(pollTimer)
  window.removeEventListener('keydown', onKey)
  window.removeEventListener('online', onBrowserOnline)
  window.removeEventListener('offline', onBrowserOffline)
  removeOnlineListener?.()
  removePendingListener?.()
})
</script>

<style scoped>
.event-live {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 2rem;
  --line: #dee2e6;
  --muted: #6c757d;
  --accent: #1a5276;
}

.page-title {
  color: #2c3e50;
  margin: 0 0 1rem;
}

.meta-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  align-items: center;
  color: var(--muted);
  font-size: 0.95rem;
}

.badge {
  display: inline-block;
  padding: 0.2rem 0.55rem;
  border-radius: 4px;
  font-size: 0.8rem;
  background: #ecf0f1;
}

.badge.online {
  background: #d5f5e3;
  color: #1e8449;
}

.badge.offline {
  background: #fadbd8;
  color: #922b21;
}

.badge.pending {
  background: #fdebd0;
  color: #7d6608;
}

.sync-group {
  display: inline-flex;
  flex-wrap: wrap;
  gap: 0.4rem;
  align-items: center;
}

.toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  align-items: center;
  margin-bottom: 1rem;
}

.race-tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem;
}

.race-tabs button {
  border: 1px solid var(--line);
  background: #fff;
  padding: 0.45rem 0.85rem;
  border-radius: 4px;
  cursor: pointer;
  font: inherit;
}

.race-tabs button[aria-selected='true'] {
  background: var(--accent);
  color: #fff;
  border-color: var(--accent);
}

.btn {
  border: none;
  border-radius: 4px;
  padding: 0.45rem 0.85rem;
  font: inherit;
  cursor: pointer;
  background: var(--accent);
  color: #fff;
}

.btn.secondary {
  background: #ecf0f1;
  color: #2c3e50;
}

.legend {
  display: flex;
  flex-wrap: wrap;
  gap: 0.65rem 1.1rem;
  font-size: 0.85rem;
  margin: 0.5rem 0 1rem;
  padding: 0.65rem 0.85rem;
  background: #fff;
  border: 1px solid var(--line);
  border-radius: 6px;
  align-items: center;
}

.legend-item {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
}

.legend i {
  display: inline-block;
  width: 0.7rem;
  height: 0.7rem;
  border-radius: 50%;
}

.panel {
  background: #fff;
  border: 1px solid var(--line);
  border-radius: 6px;
  padding: 1rem 1.15rem;
  margin-bottom: 1rem;
}

.countdown-label {
  color: var(--muted);
  font-size: 0.85rem;
  margin: 0 0 0.2rem;
}

.countdown {
  font-variant-numeric: tabular-nums;
  font-size: clamp(1.5rem, 3.5vw, 2.25rem);
  font-weight: 700;
  letter-spacing: 0.04em;
  margin: 0;
}

.chart-wrap {
  background: #fff;
  border: 1px solid var(--line);
  border-radius: 6px;
  padding: 0.75rem;
  margin-bottom: 1rem;
  overflow-x: auto;
}

.overlap-charts {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.chart-wrap.dark {
  background: #1c2833;
  border-color: #34495e;
}

.muted {
  color: var(--muted);
}

.status {
  text-align: center;
  color: var(--muted);
}

.status.error {
  color: #dc3545;
}

table {
  width: 100%;
  border-collapse: collapse;
}

th,
td {
  text-align: left;
  padding: 0.5rem 0.4rem;
  border-bottom: 1px solid var(--line);
}

.place {
  font-weight: 700;
  width: 2.5rem;
}

.cat-dot {
  display: inline-block;
  width: 0.65rem;
  height: 0.65rem;
  border-radius: 50%;
  margin-right: 0.35rem;
  vertical-align: middle;
}

.fs-root {
  position: fixed;
  inset: 0;
  z-index: 1000;
  background: #1c2833;
  color: #ecf0f1;
  padding: 1.5rem 2rem;
  display: flex;
  flex-direction: column;
}

.fs-top {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  margin-bottom: 1rem;
}

.fs-root h1 {
  color: #fff;
  font-size: 1.75rem;
  margin: 0;
}

.fs-meta {
  color: #aeb6bf;
  font-size: 0.95rem;
}

.fs-grid {
  flex: 1;
  display: grid;
  grid-template-columns: 1.1fr 1fr;
  gap: 1.25rem;
  min-height: 0;
}

.fs-panel {
  background: #273746;
  border-radius: 8px;
  padding: 1rem 1.15rem;
  overflow: auto;
}

.fs-panel h2 {
  margin: 0 0 0.75rem;
  font-size: 1.2rem;
  color: #fff;
}

.fs-exit {
  position: absolute;
  top: 1rem;
  right: 1rem;
}

@media (max-width: 900px) {
  .fs-grid {
    grid-template-columns: 1fr;
  }

  .event-live {
    padding: 0 1rem;
  }
}
</style>
