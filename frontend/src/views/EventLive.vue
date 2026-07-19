<template>
  <div class="event-live" data-testid="live-view">
    <LapCelebrationOverlay :visible="celebrationVisible" :name="celebrationName" />
    <p
      v-if="isReaderSession"
      class="meta-bar sync-bar"
      :class="{ 'sync-bar--overlay': rotatorOpen }"
    >
      <span class="sync-group" data-testid="sync-status">
        <span
          v-if="chipState === 'online_synced'"
          class="badge online"
          data-testid="sync-online"
        >
          {{ chipLabel }}
        </span>
        <span
          v-else-if="chipState === 'syncing'"
          class="badge syncing"
          data-testid="sync-syncing"
        >
          {{ chipLabel }}
        </span>
        <span
          v-else
          class="badge offline"
          data-testid="sync-offline"
        >
          {{ chipLabel }}
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
        <div
          v-if="isReaderSession && activeRaceId"
          class="ops-links"
          data-testid="live-ops-links"
        >
          <router-link
            class="ops-link"
            :to="`/races/${activeRaceId}/racers`"
            data-testid="live-open-racers"
          >
            Racers
          </router-link>
          <router-link
            class="ops-link"
            :to="`/timing/live/${activeRaceId}`"
            data-testid="live-open-manual"
          >
            Manual entry
          </router-link>
        </div>
      </div>

      <div class="legend" data-testid="category-legend" role="list" aria-label="Category legend">
        <strong>Overall</strong>
        <span
          v-for="item in live.category_legend"
          :key="item.key"
          class="legend-item"
          role="listitem"
        >
          <i :style="{ background: resolveCategoryColor(item.key, item.color) }" aria-hidden="true" />
          {{ item.label }}
        </span>
      </div>

      <div v-show="activeTab === '12h'" data-testid="race-panel-12h">
        <section class="panel">
          <h2>{{ race12?.name || '12 Hour' }}</h2>
          <template v-if="displayCountdown(race12) > 0">
            <p id="countdown-label-12h" class="countdown-label">Countdown</p>
            <p
              class="countdown"
              data-testid="live-countdown"
              role="timer"
              aria-live="polite"
              aria-labelledby="countdown-label-12h"
            >
              {{ formatCountdown(displayCountdown(race12)) }}
            </p>
          </template>
        </section>
        <div class="chart-wrap" aria-label="Lap progress chart">
          <RaceFlowChart
            v-if="race12?.id"
            ref="chart12hRef"
            :race-id="race12.id"
            :race-status="asRaceStatus(race12.status)"
            :race-start-time="race12.start_time"
            :race-type="race12.race_type"
            :duration-minutes="race12.duration_minutes"
            v-model:highlight-participant-id="highlightParticipantId"
          />
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
                :data-participant-id="e.participant_id"
                :class="{ 'leaderboard-row--focus': focusParticipantId === e.participant_id }"
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
          <template v-if="displayCountdown(race6) > 0">
            <p id="countdown-label-6h" class="countdown-label">Countdown</p>
            <p
              class="countdown"
              role="timer"
              aria-live="polite"
              aria-labelledby="countdown-label-6h"
            >
              {{ formatCountdown(displayCountdown(race6)) }}
            </p>
          </template>
        </section>
        <div class="chart-wrap">
          <RaceFlowChart
            v-if="race6?.id"
            ref="chart6hRef"
            :race-id="race6.id"
            :race-status="asRaceStatus(race6.status)"
            :race-start-time="race6.start_time"
            :race-type="race6.race_type"
            :duration-minutes="race6.duration_minutes"
            v-model:highlight-participant-id="highlightParticipantId"
          />
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
              <tr
                v-for="e in race6?.leaderboard_overall || []"
                :key="e.participant_id"
                data-testid="leaderboard-row"
                :data-participant-id="e.participant_id"
                :class="{ 'leaderboard-row--focus': focusParticipantId === e.participant_id }"
              >
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
          <template v-if="displayCountdown(race90) > 0">
            <p id="countdown-label-90m" class="countdown-label">Countdown</p>
            <p
              class="countdown"
              role="timer"
              aria-live="polite"
              aria-labelledby="countdown-label-90m"
            >
              {{ formatCountdown(displayCountdown(race90)) }}
            </p>
          </template>
        </section>
        <div class="chart-wrap">
          <RaceFlowChart
            v-if="race90?.id"
            ref="chart90mRef"
            :race-id="race90.id"
            :race-status="asRaceStatus(race90.status)"
            :race-start-time="race90.start_time"
            :race-type="race90.race_type"
            :duration-minutes="race90.duration_minutes"
            v-model:highlight-participant-id="highlightParticipantId"
          />
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
              <tr
                v-for="e in race90?.leaderboard_overall || []"
                :key="e.participant_id"
                data-testid="leaderboard-row"
                :data-participant-id="e.participant_id"
                :class="{ 'leaderboard-row--focus': focusParticipantId === e.participant_id }"
              >
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
          <RaceFlowChart
            v-if="race12?.id"
            ref="chartOverlap12Ref"
            :race-id="race12.id"
            :race-status="asRaceStatus(race12.status)"
            :race-start-time="race12.start_time"
            :race-type="race12.race_type"
            :duration-minutes="race12.duration_minutes"
            v-model:highlight-participant-id="highlightParticipantId"
          />
          <RaceFlowChart
            v-if="race6?.id"
            ref="chartOverlap6Ref"
            :race-id="race6.id"
            :race-status="asRaceStatus(race6.status)"
            :race-start-time="race6.start_time"
            :race-type="race6.race_type"
            :duration-minutes="race6.duration_minutes"
            v-model:highlight-participant-id="highlightParticipantId"
          />
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
        <div
          ref="fsGridRef"
          class="fs-grid"
          :style="{ '--fs-flow-width': fsFlowWidthPercent + '%' }"
        >
          <div class="fs-panel" data-testid="rotator-flow">
            <h2>Race flow</h2>
            <div class="chart-wrap">
              <RaceFlowChart
                v-if="race12?.id"
                ref="chartRotatorRef"
                :race-id="race12.id"
                :race-status="asRaceStatus(race12.status)"
                :race-start-time="race12.start_time"
                :race-type="race12.race_type"
                :duration-minutes="race12.duration_minutes"
                v-model:highlight-participant-id="highlightParticipantId"
              />
            </div>
          </div>
          <button
            type="button"
            class="fs-split"
            data-testid="rotator-split-handle"
            aria-label="Resize race flow and leaderboard"
            aria-orientation="vertical"
            @pointerdown="onFsSplitPointerDown"
          />
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
                  data-testid="leaderboard-row"
                  :data-participant-id="e.participant_id"
                  :class="{ 'leaderboard-row--focus': focusParticipantId === e.participant_id }"
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
import { computed, nextTick, onMounted, onUnmounted, ref, watch, watchEffect } from 'vue'
import { useRoute } from 'vue-router'
import LapCelebrationOverlay from '@/components/LapCelebrationOverlay.vue'
import RaceFlowChart from '@/components/RaceFlowChart.vue'
import {
  eventsLiveApi,
  rfidApi,
  type EventLiveRace,
  type EventLiveResponse,
  type LapRecordedEvent,
} from '@/services/api'
import {
  getLocalPendingCount,
  onOnline,
  onPendingChange,
} from '@/services/offlineQueue'
import { setDisplayCache } from '@/services/timingStorage'
import { useBridgeSyncStatus } from '@/composables/useBridgeSyncStatus'
import { useEventLiveStream } from '@/composables/useEventLiveStream'
import { useReaderStation } from '@/composables/useReaderStation'
import { useSpectatorIdle } from '@/composables/useSpectatorIdle'
import { usePinAuthStore } from '@/stores/pinAuth'
import { useEventsStore } from '@/stores/events'
import { useStationStore } from '@/stores/station'
import { getErrorMessage } from '@/utils/error'
import type { RaceStatus } from '@/types/models'
import { resolveCategoryColor } from '@/themes/defaultLegend'

type ChartRef = InstanceType<typeof RaceFlowChart> | null

const route = useRoute()
const eventId = computed(() => String(route.params.eventId))
const station = useStationStore()
const eventsStore = useEventsStore()
const pinAuth = usePinAuthStore()
const { lastScan } = useReaderStation()
const isReaderSession = computed(() => pinAuth.isAuthenticated)
const { chipState, chipLabel } = useBridgeSyncStatus()
const { lastLap } = useEventLiveStream(eventId)

const live = ref<EventLiveResponse | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)
const activeTab = ref<'12h' | '6h' | '90m' | 'overlap'>('12h')
const rotatorOpen = ref(false)
const FS_FLOW_WIDTH_KEY = 'event-live-fs-flow-width'
const FS_FLOW_WIDTH_MIN = 25
const FS_FLOW_WIDTH_MAX = 75
const fsFlowWidthPercent = ref(52)
const fsGridRef = ref<HTMLElement | null>(null)
const online = ref(typeof navigator !== 'undefined' ? navigator.onLine : true)
const pendingSync = ref(0)
const highlightParticipantId = ref<string | undefined>()
const focusParticipantId = ref<string | undefined>()
const celebrationVisible = ref(false)
const celebrationName = ref('')
const celebrationRaceId = ref<string | undefined>()
const legendBusy = ref(false)
const pageScrolledFromTop = ref(false)
const { isBusy } = useSpectatorIdle({ legendBusy, pageScrolledFromTop })

const chart12hRef = ref<ChartRef>(null)
const chart6hRef = ref<ChartRef>(null)
const chart90mRef = ref<ChartRef>(null)
const chartOverlap12Ref = ref<ChartRef>(null)
const chartOverlap6Ref = ref<ChartRef>(null)
const chartRotatorRef = ref<ChartRef>(null)

let celebrationTimer: number | undefined
let focusTimer: number | undefined
/** Wall clock for local countdown ticks between 2s live polls. */
const nowMs = ref(Date.now())
let pollTimer: number | undefined
let countdownTickTimer: number | undefined
let countdownAnchoredAt = 0
let removeOnlineListener: (() => void) | undefined
let removePendingListener: (() => void) | undefined
let removeScrollListener: (() => void) | undefined

function formatCountdown(seconds: number): string {
  const s = Math.max(0, Math.floor(seconds))
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  const r = s % 60
  return [h, m, r].map((n) => String(n).padStart(2, '0')).join(':')
}

/** Smooth 1s countdown from the last polled server value. */
function displayCountdown(race: EventLiveRace | undefined): number {
  if (!race) return 0
  const base = race.countdown_seconds ?? 0
  if (base <= 0 || countdownAnchoredAt <= 0) return Math.max(0, base)
  const elapsed = Math.floor((nowMs.value - countdownAnchoredAt) / 1000)
  return Math.max(0, base - elapsed)
}

function asRaceStatus(status: string): RaceStatus | undefined {
  if (
    status === 'scheduled' ||
    status === 'active' ||
    status === 'finished' ||
    status === 'cancelled'
  ) {
    return status
  }
  return undefined
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

const visibleRaceIds = computed(() => {
  if (rotatorOpen.value) {
    return race12.value?.id ? [race12.value.id] : []
  }
  if (activeTab.value === 'overlap') {
    return [race12.value?.id, race6.value?.id].filter(Boolean) as string[]
  }
  if (activeTab.value === '12h') return race12.value?.id ? [race12.value.id] : []
  if (activeTab.value === '6h') return race6.value?.id ? [race6.value.id] : []
  return race90.value?.id ? [race90.value.id] : []
})

/** Race selected by the live tab — used for Racers / Manual entry shortcuts. */
const activeRaceId = computed(() => {
  if (activeTab.value === '12h') return race12.value?.id
  if (activeTab.value === '6h') return race6.value?.id
  if (activeTab.value === '90m') return race90.value?.id
  return race12.value?.id ?? race6.value?.id ?? race90.value?.id
})

function chartBindings(): Array<{ raceId?: string; chart: typeof chart12hRef }> {
  return [
    { raceId: race12.value?.id, chart: chart12hRef },
    { raceId: race6.value?.id, chart: chart6hRef },
    { raceId: race90.value?.id, chart: chart90mRef },
    { raceId: race12.value?.id, chart: chartOverlap12Ref },
    { raceId: race6.value?.id, chart: chartOverlap6Ref },
    { raceId: race12.value?.id, chart: chartRotatorRef },
  ]
}

function refreshFlowForRace(raceId: string) {
  for (const { raceId: id, chart } of chartBindings()) {
    if (id === raceId) {
      void chart.value?.loadRecords?.()
    }
  }
}

function clearCelebrationTimers() {
  if (celebrationTimer) {
    window.clearTimeout(celebrationTimer)
    celebrationTimer = undefined
  }
  if (focusTimer) {
    window.clearTimeout(focusTimer)
    focusTimer = undefined
  }
}

function clearFocusState() {
  highlightParticipantId.value = undefined
  focusParticipantId.value = undefined
}

function startCelebration(ev: LapRecordedEvent) {
  clearCelebrationTimers()

  celebrationRaceId.value = ev.race_id
  celebrationName.value = ev.participant_name
  celebrationVisible.value = true
  celebrationTimer = window.setTimeout(() => {
    celebrationVisible.value = false
    celebrationTimer = undefined
  }, 2500)

  void refreshFlowForRace(ev.race_id)

  if (isBusy.value) {
    clearFocusState()
    return
  }

  highlightParticipantId.value = ev.participant_id
  focusParticipantId.value = ev.participant_id
  void nextTick(() => {
    const row = document.querySelector(
      `[data-testid="leaderboard-row"][data-participant-id="${ev.participant_id}"]`,
    )
    row?.scrollIntoView({ block: 'center', behavior: 'smooth' })
  })

  focusTimer = window.setTimeout(() => {
    clearFocusState()
    window.scrollTo({ top: 0, behavior: 'smooth' })
    for (const panel of document.querySelectorAll<HTMLElement>('.event-live .fs-panel')) {
      panel.scrollTop = 0
    }
    focusTimer = undefined
  }, 3000)
}

function updatePageScrolledFromTop() {
  if (window.scrollY > 0) {
    pageScrolledFromTop.value = true
    return
  }
  for (const panel of document.querySelectorAll<HTMLElement>('.event-live .fs-panel')) {
    if (panel.scrollTop > 0) {
      pageScrolledFromTop.value = true
      return
    }
  }
  pageScrolledFromTop.value = false
}

function colorFor(key: string): string {
  const apiColor = live.value?.category_legend.find((l) => l.key === key)?.color
  return resolveCategoryColor(key, apiColor)
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

async function applyLiveData(data: EventLiveResponse) {
  live.value = data
  countdownAnchoredAt = Date.now()
  nowMs.value = countdownAnchoredAt
  eventsStore.setCurrentEventSummary(data.event)
  void cacheLiveLabels(data)
}

async function loadLive() {
  loading.value = true
  error.value = null
  online.value = navigator.onLine
  try {
    const { data } = await eventsLiveApi.getLive(eventId.value)
    await applyLiveData(data)
    online.value = navigator.onLine
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

function clampFsFlowWidth(percent: number): number {
  return Math.min(FS_FLOW_WIDTH_MAX, Math.max(FS_FLOW_WIDTH_MIN, percent))
}

function readStoredFsFlowWidth(): number | null {
  try {
    const raw = sessionStorage.getItem(FS_FLOW_WIDTH_KEY)
    if (raw == null) return null
    const parsed = Number(raw)
    if (!Number.isFinite(parsed)) return null
    return clampFsFlowWidth(parsed)
  } catch {
    return null
  }
}

function restoreFsFlowWidth() {
  fsFlowWidthPercent.value = readStoredFsFlowWidth() ?? 52
}

function persistFsFlowWidth() {
  try {
    sessionStorage.setItem(FS_FLOW_WIDTH_KEY, String(fsFlowWidthPercent.value))
  } catch {
    // ignore storage failures
  }
}

function updateFsFlowWidthFromClientX(clientX: number) {
  const grid = fsGridRef.value
  if (!grid) return
  const rect = grid.getBoundingClientRect()
  if (rect.width <= 0) return
  const percent = ((clientX - rect.left) / rect.width) * 100
  fsFlowWidthPercent.value = clampFsFlowWidth(percent)
}

let fsSplitPointerId: number | null = null
let fsSplitMoveHandler: ((e: PointerEvent) => void) | null = null
let fsSplitUpHandler: ((e: PointerEvent) => void) | null = null

function cleanupFsSplitDrag(handle?: HTMLElement | null) {
  if (fsSplitMoveHandler && handle) {
    handle.removeEventListener('pointermove', fsSplitMoveHandler)
  }
  if (fsSplitUpHandler && handle) {
    handle.removeEventListener('pointerup', fsSplitUpHandler)
    handle.removeEventListener('pointercancel', fsSplitUpHandler)
  }
  fsSplitMoveHandler = null
  fsSplitUpHandler = null
  fsSplitPointerId = null
}

function onFsSplitPointerDown(e: PointerEvent) {
  if (e.button !== 0) return
  e.preventDefault()
  const handle = e.currentTarget as HTMLElement
  fsSplitPointerId = e.pointerId
  handle.setPointerCapture(e.pointerId)

  fsSplitMoveHandler = (ev: PointerEvent) => {
    if (ev.pointerId !== fsSplitPointerId) return
    updateFsFlowWidthFromClientX(ev.clientX)
  }
  fsSplitUpHandler = (ev: PointerEvent) => {
    if (ev.pointerId !== fsSplitPointerId) return
    persistFsFlowWidth()
    cleanupFsSplitDrag(handle)
    handle.releasePointerCapture(ev.pointerId)
  }

  handle.addEventListener('pointermove', fsSplitMoveHandler)
  handle.addEventListener('pointerup', fsSplitUpHandler)
  handle.addEventListener('pointercancel', fsSplitUpHandler)
  updateFsFlowWidthFromClientX(e.clientX)
}

function onBrowserOnline() {
  online.value = true
  void refreshPending()
}

function onBrowserOffline() {
  online.value = false
  void refreshPending()
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

watch(lastLap, (ev) => {
  if (!ev || ev.type !== 'lap_recorded') return
  if (!visibleRaceIds.value.includes(ev.race_id)) return
  startCelebration(ev)
})

watch(visibleRaceIds, (ids) => {
  if (celebrationRaceId.value && !ids.includes(celebrationRaceId.value)) {
    clearCelebrationTimers()
    clearFocusState()
  }
})

watch(highlightParticipantId, (participantId) => {
  if (!participantId) {
    focusParticipantId.value = undefined
  }
})

watch(pageScrolledFromTop, (scrolled) => {
  if (scrolled && focusTimer) {
    window.clearTimeout(focusTimer)
    focusTimer = undefined
  }
})

watch(rotatorOpen, (open) => {
  if (open) restoreFsFlowWidth()
})

watchEffect(() => {
  const visible = new Set(visibleRaceIds.value)
  legendBusy.value = chartBindings().some(
    ({ raceId, chart }) => Boolean(raceId && visible.has(raceId) && chart.value?.isLegendBusy),
  )
})

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
      void refreshPending()
    }
  }

  countdownTickTimer = window.setInterval(() => {
    nowMs.value = Date.now()
  }, 1000)

  pollTimer = window.setInterval(() => {
    syncOnlineState()
    if (!navigator.onLine) {
      void refreshPending()
      return
    }
    void eventsLiveApi
      .getLive(eventId.value)
      .then(({ data }) => {
        void applyLiveData(data)
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

  const onScrollCheck = () => updatePageScrolledFromTop()
  window.addEventListener('scroll', onScrollCheck, { passive: true })
  removeScrollListener = () => {
    window.removeEventListener('scroll', onScrollCheck)
  }
})

onUnmounted(() => {
  clearCelebrationTimers()
  cleanupFsSplitDrag(fsGridRef.value?.querySelector('.fs-split') as HTMLElement | null)
  if (pollTimer) window.clearInterval(pollTimer)
  if (countdownTickTimer) window.clearInterval(countdownTickTimer)
  window.removeEventListener('keydown', onKey)
  window.removeEventListener('online', onBrowserOnline)
  window.removeEventListener('offline', onBrowserOffline)
  removeOnlineListener?.()
  removePendingListener?.()
  removeScrollListener?.()
})
</script>

<style scoped>
.event-live {
  position: relative;
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 2rem;
  --line: var(--border);
  --live-display-scale: 1;
  --live-title-size: calc(1.75rem * var(--live-display-scale));
  --live-tab-size: calc(1rem * var(--live-display-scale));
  --live-body-size: calc(1rem * var(--live-display-scale));
}

.page-title {
  color: var(--ink);
  margin: 0 0 1rem;
  font-size: var(--live-title-size);
}

.meta-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  align-items: center;
  color: var(--muted);
  font-size: calc(0.95rem * var(--live-display-scale));
}

.badge {
  display: inline-block;
  padding: 0.2rem 0.55rem;
  border-radius: 4px;
  font-size: 0.8rem;
  background: color-mix(in srgb, var(--muted) 15%, var(--surface));
}

.badge.online {
  background: color-mix(in srgb, var(--success) 15%, var(--surface));
  color: var(--success);
}

.badge.offline {
  background: color-mix(in srgb, var(--signal) 15%, var(--surface));
  color: var(--signal);
}

.badge.syncing {
  background: #fdebd0;
  color: #7d6608;
}

.badge.pending {
  background: color-mix(in srgb, var(--copper) 20%, var(--surface));
  color: var(--copper);
}

.sync-group {
  display: inline-flex;
  flex-wrap: wrap;
  gap: 0.4rem;
  align-items: center;
}

/* Keep bridge sync chip visible above the fullscreen rotator (race-day critical). */
.sync-bar--overlay {
  position: fixed;
  top: 0.75rem;
  left: 0.75rem;
  z-index: 1100;
  margin: 0;
  padding: 0.35rem 0.55rem;
  border-radius: 6px;
  background: color-mix(in srgb, var(--ink-deep) 88%, transparent);
  box-shadow: 0 2px 10px color-mix(in srgb, var(--ink-deep) 35%, transparent);
}

.toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  align-items: center;
  margin-bottom: 1rem;
}

.ops-links {
  display: inline-flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-left: auto;
}

.ops-link {
  display: inline-block;
  padding: 0.4rem 0.85rem;
  border-radius: 4px;
  background: var(--mist);
  color: var(--ink);
  text-decoration: none;
  font-weight: 600;
  font-size: 0.9rem;
}

.ops-link:hover {
  background: var(--sage);
}

.race-tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem;
}

.race-tabs button {
  border: 1px solid var(--line);
  background: var(--surface);
  padding: 0.45rem 0.85rem;
  border-radius: 4px;
  cursor: pointer;
  font: inherit;
  font-size: var(--live-tab-size);
}

.race-tabs button[aria-selected='true'] {
  background: var(--accent);
  color: var(--surface);
  border-color: var(--accent);
}

.btn {
  border: none;
  border-radius: 4px;
  padding: 0.45rem 0.85rem;
  font: inherit;
  cursor: pointer;
  background: var(--accent);
  color: var(--surface);
}

.btn.secondary {
  background: var(--mist);
  color: var(--ink);
}

.legend {
  display: flex;
  flex-wrap: wrap;
  gap: 0.65rem 1.1rem;
  font-size: calc(0.85rem * var(--live-display-scale));
  margin: 0.5rem 0 1rem;
  padding: 0.65rem 0.85rem;
  background: var(--surface);
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
  background: var(--surface);
  border: 1px solid var(--line);
  border-radius: 6px;
  padding: 1rem 1.15rem;
  margin-bottom: 1rem;
  font-size: var(--live-body-size);
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
  background: var(--surface);
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
  background: var(--ink-deep);
  border-color: var(--ink);
}

.muted {
  color: var(--muted);
}

.status {
  text-align: center;
  color: var(--muted);
}

.status.error {
  color: var(--signal);
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

.leaderboard-row--focus {
  background: color-mix(in srgb, var(--accent) 12%, var(--surface));
  outline: 2px solid var(--accent);
  outline-offset: -2px;
}

.fs-root {
  position: fixed;
  inset: 0;
  z-index: 1000;
  background: var(--mist);
  color: var(--ink);
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
  color: var(--ink);
  font-size: 1.75rem;
  margin: 0;
}

.fs-meta {
  color: var(--muted);
  font-size: 0.95rem;
}

.fs-grid {
  flex: 1;
  display: grid;
  grid-template-columns: minmax(0, var(--fs-flow-width, 52%)) 10px minmax(0, 1fr);
  gap: 1.25rem;
  min-height: 0;
}

.fs-split {
  cursor: col-resize;
  width: 10px;
  padding: 0;
  border: none;
  background: transparent;
  align-self: stretch;
  position: relative;
}

.fs-split::before {
  content: '';
  position: absolute;
  left: 50%;
  top: 0;
  bottom: 0;
  width: 1px;
  transform: translateX(-50%);
  background: var(--line);
}

.fs-panel {
  background: var(--surface);
  border: 1px solid var(--line);
  border-radius: 8px;
  padding: 1rem 1.15rem;
  overflow: auto;
}

.fs-panel h2 {
  margin: 0 0 0.75rem;
  font-size: 1.2rem;
  color: var(--ink);
}

.fs-exit {
  position: absolute;
  top: 1rem;
  right: 1rem;
}

@media (min-width: 900px) {
  .event-live {
    --live-display-scale: 1.15;
  }
}

[data-testid='fullscreen-rotator'] {
  --live-display-scale: 1.35;
  font-size: calc(1rem * var(--live-display-scale));
}

@media (max-width: 900px) {
  .fs-grid {
    grid-template-columns: 1fr;
  }

  .fs-split {
    display: none;
  }

  .event-live {
    padding: 0 1rem;
  }
}
</style>
