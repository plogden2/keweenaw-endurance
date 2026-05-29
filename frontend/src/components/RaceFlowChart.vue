<template>
  <section class="race-flow-chart" data-testid="race-flow-chart">
    <div v-if="loading" class="status">Loading race flow…</div>
    <div v-else-if="error" class="status error">{{ error }}</div>
    <p v-else-if="!hasData" class="empty" data-testid="race-flow-empty">
      Not enough timing data to render race flow yet.
    </p>
    <div v-else class="chart-panel">
      <canvas ref="canvasRef" data-testid="race-flow-canvas" />
      <div class="legend-panel" data-testid="race-flow-legend">
        <div class="legend-controls">
          <input
            v-model="searchQuery"
            type="search"
            class="legend-search"
            placeholder="Search by bib or name…"
            data-testid="race-flow-legend-search"
          />
          <div class="status-filters" data-testid="race-flow-status-filters">
            <label
              v-for="status in availableStatuses"
              :key="status"
              class="status-filter"
            >
              <input
                v-model="selectedStatuses"
                type="checkbox"
                :value="status"
              />
              {{ getParticipantStatusLabel(status) }}
            </label>
          </div>
          <button
            type="button"
            class="select-all-btn"
            data-testid="race-flow-select-all"
            @click="toggleSelectAllFiltered"
          >
            {{ allFilteredSelected ? 'Deselect all' : 'Select all' }}
          </button>
        </div>
        <p v-if="filteredLegendItems.length === 0" class="legend-empty">
          No participants match the current search or filters.
        </p>
        <div v-else class="legend-items">
          <label
            v-for="item in filteredLegendItems"
            :key="item.participantId"
            class="legend-item"
            :class="{ 'legend-item-hidden': !isParticipantVisible(item.participantId) }"
            @mouseenter="showTooltip(item, $event)"
            @mousemove="moveTooltip($event)"
            @mouseleave="hideTooltip"
          >
            <input
              type="checkbox"
              :checked="isParticipantVisible(item.participantId)"
              @change="toggleParticipantVisibility(item.participantId)"
            />
            <span
              class="color-swatch"
              :style="{ backgroundColor: item.color }"
              aria-hidden="true"
            />
            <span class="legend-label">{{ item.label }}</span>
          </label>
        </div>
      </div>
      <div
        v-if="activeTooltip"
        class="legend-tooltip"
        data-testid="race-flow-legend-tooltip"
        :style="tooltipStyle"
        role="tooltip"
      >
        <strong class="tooltip-name">{{ activeTooltip.fullName }}</strong>
        <span>Bib {{ activeTooltip.bibNumber }}</span>
        <span>Status: {{ getParticipantStatusLabel(activeTooltip.status) }}</span>
        <span v-if="activeTooltip.gender">Gender: {{ activeTooltip.gender }}</span>
        <span v-if="activeTooltip.age != null">Age: {{ activeTooltip.age }}</span>
        <span>{{ activeTooltip.progress }}</span>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import {
  Chart,
  LineController,
  LineElement,
  PointElement,
  LinearScale,
  CategoryScale,
  Title,
  Tooltip,
  type Plugin,
} from 'chart.js'
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { timingApi } from '@/services/api'
import { useUnitsStore } from '@/stores/units'
import type { ParticipantStatus, RaceStatus, RaceType, TimingRecord } from '@/types/models'
import {
  assignContrastFlowColors,
  buildExtrapolationPoint,
  buildParticipantFlowTooltip,
  buildParticipantFlows,
  getCurrentElapsedMinutes,
  getFlowChartTitle,
  getFlowLineColor,
  getFlowYAxisLabel,
  getParticipantStatusLabel,
  resolveRaceStartMs,
  type ParticipantFlowTooltip,
} from '@/utils/raceFlowData'
import { getErrorMessage } from '@/utils/error'

const CURRENT_TIME_LINE_COLOR = '#e74c3c'
const LIVE_REFRESH_MS = 30_000
const STATUS_ORDER: ParticipantStatus[] = [
  'finished',
  'started',
  'registered',
  'dnf',
  'dns',
]

interface FlowLineDataset {
  label: string
  data: Array<{ x: number; y: number }>
  borderColor: string
  backgroundColor: string
  pointBackgroundColor: string
  pointBorderColor: string
  borderWidth: number
  tension: number
  hasExtrapolation: boolean
  segment?: {
    borderDash: (ctx: { p1DataIndex: number }) => number[] | undefined
  }
  pointRadius?: number | number[]
}

interface LegendItem {
  participantId: string
  label: string
  color: string
  status: ParticipantStatus
  bibNumber: string
  tooltip: ParticipantFlowTooltip
}

const currentTimeLinePlugin: Plugin<'line'> = {
  id: 'currentTimeLine',
  afterDraw(chart, _args, options) {
    const xMinutes = (options as { xMinutes?: number | null }).xMinutes
    if (xMinutes == null) {
      return
    }

    const { ctx, chartArea, scales } = chart
    const xScale = scales.x
    if (!xScale || xMinutes < xScale.min || xMinutes > xScale.max) {
      return
    }

    const xPixel = xScale.getPixelForValue(xMinutes)
    ctx.save()
    ctx.strokeStyle = CURRENT_TIME_LINE_COLOR
    ctx.lineWidth = 2
    ctx.beginPath()
    ctx.moveTo(xPixel, chartArea.top)
    ctx.lineTo(xPixel, chartArea.bottom)
    ctx.stroke()
    ctx.restore()
  },
}

Chart.register(
  LineController,
  LineElement,
  PointElement,
  LinearScale,
  CategoryScale,
  Title,
  Tooltip,
  currentTimeLinePlugin,
)

const props = defineProps<{
  raceId: string
  raceStatus?: RaceStatus
  raceStartTime?: string
  raceType?: RaceType
}>()

const unitsStore = useUnitsStore()
const chartRaceType = computed(() => props.raceType ?? 'time_based')

const canvasRef = ref<HTMLCanvasElement | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)
const records = ref<TimingRecord[]>([])
const chartInstance = ref<Chart | null>(null)
const nowMs = ref(Date.now())
const searchQuery = ref('')
const selectedStatuses = ref<ParticipantStatus[]>([...STATUS_ORDER])
const visibleParticipantIds = ref<Set<string>>(new Set())
const activeTooltip = ref<ParticipantFlowTooltip | null>(null)
const tooltipPosition = ref({ x: 0, y: 0 })
let liveRefreshTimer: ReturnType<typeof setInterval> | null = null

const isActiveRace = computed(() => props.raceStatus === 'active')
const flows = computed(() =>
  buildParticipantFlows(
    records.value,
    props.raceStartTime,
    chartRaceType.value,
    unitsStore.unitSystem,
  ),
)
const hasData = computed(() => flows.value.length > 0)
const raceStartMs = computed(() => resolveRaceStartMs(records.value, props.raceStartTime))
const currentElapsedMinutes = computed(() => {
  if (!isActiveRace.value || raceStartMs.value === null) {
    return null
  }

  const elapsed = getCurrentElapsedMinutes(raceStartMs.value, nowMs.value)
  const latestRecordedMinute = flows.value.reduce((latest, flow) => {
    const lastPoint = flow.points.at(-1)
    return lastPoint ? Math.max(latest, lastPoint.elapsedMinutes) : latest
  }, 0)

  return elapsed > latestRecordedMinute ? elapsed : null
})

const availableStatuses = computed(() =>
  STATUS_ORDER.filter((status) => flows.value.some((flow) => flow.status === status)),
)

const visibleFlows = computed(() =>
  flows.value.filter((flow) => visibleParticipantIds.value.has(flow.participantId)),
)

const visibleFlowColors = computed(() =>
  assignContrastFlowColors(visibleFlows.value.map((flow) => flow.participantId)),
)

function getParticipantColor(participantId: string): string {
  return (
    visibleFlowColors.value.get(participantId) ??
    getFlowLineColor(participantId)
  )
}

const legendItems = computed<LegendItem[]>(() =>
  flows.value.map((flow) => ({
    participantId: flow.participantId,
    label: flow.label,
    color: getParticipantColor(flow.participantId),
    status: flow.status,
    bibNumber: flow.bibNumber,
    tooltip: buildParticipantFlowTooltip(flow, chartRaceType.value, unitsStore.unitSystem),
  })),
)

const tooltipStyle = computed(() => ({
  top: `${tooltipPosition.value.y}px`,
  left: `${tooltipPosition.value.x}px`,
}))

const filteredLegendItems = computed(() => {
  const query = searchQuery.value.trim().toLowerCase()

  return legendItems.value.filter((item) => {
    if (!selectedStatuses.value.includes(item.status)) {
      return false
    }

    if (!query) {
      return true
    }

    return (
      item.label.toLowerCase().includes(query) ||
      item.bibNumber.toLowerCase().includes(query)
    )
  })
})

const allFilteredSelected = computed(() =>
  filteredLegendItems.value.length > 0 &&
  filteredLegendItems.value.every((item) =>
    visibleParticipantIds.value.has(item.participantId),
  ),
)

function syncVisibleParticipants(): void {
  const nextVisibleIds = new Set(visibleParticipantIds.value)
  const currentIds = new Set(flows.value.map((flow) => flow.participantId))

  for (const id of nextVisibleIds) {
    if (!currentIds.has(id)) {
      nextVisibleIds.delete(id)
    }
  }

  for (const id of currentIds) {
    if (!nextVisibleIds.has(id)) {
      nextVisibleIds.add(id)
    }
  }

  visibleParticipantIds.value = nextVisibleIds
}

function isParticipantVisible(participantId: string): boolean {
  return visibleParticipantIds.value.has(participantId)
}

function toggleParticipantVisibility(participantId: string): void {
  const nextVisibleIds = new Set(visibleParticipantIds.value)
  if (nextVisibleIds.has(participantId)) {
    nextVisibleIds.delete(participantId)
  } else {
    nextVisibleIds.add(participantId)
  }
  visibleParticipantIds.value = nextVisibleIds
}

function toggleSelectAllFiltered(): void {
  const nextVisibleIds = new Set(visibleParticipantIds.value)
  const filteredIds = filteredLegendItems.value.map((item) => item.participantId)

  if (allFilteredSelected.value) {
    filteredIds.forEach((id) => nextVisibleIds.delete(id))
  } else {
    filteredIds.forEach((id) => nextVisibleIds.add(id))
  }

  visibleParticipantIds.value = nextVisibleIds
}

function showTooltip(item: LegendItem, event: MouseEvent): void {
  activeTooltip.value = item.tooltip
  moveTooltip(event)
}

function moveTooltip(event: MouseEvent): void {
  tooltipPosition.value = {
    x: event.clientX + 12,
    y: event.clientY + 12,
  }
}

function hideTooltip(): void {
  activeTooltip.value = null
}

function clearLiveRefreshTimer(): void {
  if (liveRefreshTimer) {
    clearInterval(liveRefreshTimer)
    liveRefreshTimer = null
  }
}

function startLiveRefreshTimer(): void {
  clearLiveRefreshTimer()
  if (!isActiveRace.value) {
    return
  }

  liveRefreshTimer = setInterval(() => {
    nowMs.value = Date.now()
  }, LIVE_REFRESH_MS)
}

async function loadRecords(): Promise<void> {
  loading.value = true
  error.value = null
  try {
    const { data } = await timingApi.getLive(props.raceId)
    records.value = data.records ?? []
    nowMs.value = Date.now()
  } catch (err) {
    error.value = getErrorMessage(err, 'Failed to load race flow data')
  } finally {
    loading.value = false
  }
}

function destroyChart(): void {
  chartInstance.value?.destroy()
  chartInstance.value = null
}

function buildDataset(flow: (typeof flows.value)[number]): FlowLineDataset {
  const color = getParticipantColor(flow.participantId)
  const showCurrentTime = currentElapsedMinutes.value != null
  const extrapolation = showCurrentTime
    ? buildExtrapolationPoint(flow, currentElapsedMinutes.value!)
    : null
  const chartPoints = [
    ...flow.points.map((point) => ({
      x: point.elapsedMinutes,
      y: point.value,
    })),
  ]

  if (extrapolation) {
    chartPoints.push({
      x: extrapolation.elapsedMinutes,
      y: extrapolation.value,
    })
  }

  const dataset: FlowLineDataset = {
    label: flow.label,
    data: chartPoints,
    borderColor: color,
    backgroundColor: color,
    pointBackgroundColor: color,
    pointBorderColor: color,
    borderWidth: 2,
    tension: 0.2,
    hasExtrapolation: extrapolation != null,
  }

  if (extrapolation) {
    dataset.segment = {
      borderDash: (ctx: { p1DataIndex: number }) =>
        ctx.p1DataIndex === chartPoints.length - 1 ? [6, 6] : undefined,
    }
    dataset.pointRadius = chartPoints.map((_point, pointIndex) =>
      pointIndex === chartPoints.length - 1 ? 0 : 4,
    )
  }

  return dataset
}

function renderChart(): void {
  destroyChart()
  if (!canvasRef.value || !hasData.value) {
    return
  }

  const showCurrentTime = currentElapsedMinutes.value != null
  const maxElapsedMinutes = visibleFlows.value.reduce((max, flow) => {
    const extrapolation = showCurrentTime
      ? buildExtrapolationPoint(flow, currentElapsedMinutes.value!)
      : null
    const lastElapsed = extrapolation?.elapsedMinutes ?? flow.points.at(-1)?.elapsedMinutes ?? 0
    return Math.max(max, lastElapsed)
  }, 0)

  chartInstance.value = new Chart(canvasRef.value, {
    type: 'line',
    data: {
      datasets: visibleFlows.value.map((flow) => buildDataset(flow)),
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      scales: {
        x: {
          type: 'linear',
          title: { display: true, text: 'Elapsed time (minutes)' },
          max: showCurrentTime ? Math.ceil(maxElapsedMinutes * 1.05) : undefined,
        },
        y: {
          title: {
            display: true,
            text: getFlowYAxisLabel(chartRaceType.value, unitsStore.unitSystem),
          },
          ticks: chartRaceType.value === 'lap_based' ? { stepSize: 1 } : undefined,
        },
      },
      plugins: {
        currentTimeLine: { xMinutes: currentElapsedMinutes.value },
        legend: { display: false },
        title: {
          display: true,
          text: getFlowChartTitle(chartRaceType.value, showCurrentTime),
        },
      } as Record<string, unknown>,
    },
  })
}

onMounted(async () => {
  await loadRecords()
  startLiveRefreshTimer()
})

watch(
  () => props.raceId,
  async () => {
    await loadRecords()
    startLiveRefreshTimer()
  },
)

watch(
  () => [props.raceStatus, props.raceStartTime, props.raceType],
  () => {
    startLiveRefreshTimer()
  },
)

watch(flows, () => {
  syncVisibleParticipants()
})

watch(
  [visibleFlows, loading, currentElapsedMinutes, chartRaceType, () => unitsStore.unitSystem],
  async () => {
    if (!loading.value) {
      await nextTick()
      renderChart()
    }
  },
)

onBeforeUnmount(() => {
  clearLiveRefreshTimer()
  destroyChart()
})

defineExpose({
  loadRecords,
  records,
  flows,
  currentElapsedMinutes,
  visibleParticipantIds,
  filteredLegendItems,
})
</script>

<style scoped>
.race-flow-chart {
  min-height: 320px;
}

.chart-panel {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

canvas {
  width: 100% !important;
  height: 320px !important;
}

.legend-panel {
  background: #f8f9fa;
  border-radius: 8px;
  padding: 1rem;
}

.legend-controls {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem 1rem;
  align-items: center;
  margin-bottom: 0.75rem;
}

.legend-search {
  flex: 1 1 220px;
  min-width: 180px;
  padding: 0.45rem 0.65rem;
  border: 1px solid #ced4da;
  border-radius: 4px;
  font: inherit;
}

.status-filters {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem 0.75rem;
}

.status-filter {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.85rem;
  color: #495057;
  cursor: pointer;
  text-transform: capitalize;
}

.select-all-btn {
  padding: 0.45rem 0.75rem;
  border: 1px solid #ced4da;
  border-radius: 4px;
  background: white;
  color: #2c3e50;
  cursor: pointer;
  font: inherit;
}

.select-all-btn:hover {
  background: #e9ecef;
}

.legend-empty {
  margin: 0;
  color: #6c757d;
  font-size: 0.9rem;
}

.legend-items {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 0.35rem 0.75rem;
  max-height: 220px;
  overflow-y: auto;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.85rem;
  color: #2c3e50;
  cursor: pointer;
}

.legend-item-hidden .legend-label {
  opacity: 0.55;
}

.color-swatch {
  width: 12px;
  height: 12px;
  border-radius: 2px;
  flex-shrink: 0;
}

.legend-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.legend-tooltip {
  position: fixed;
  z-index: 1000;
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
  min-width: 160px;
  max-width: 240px;
  padding: 0.55rem 0.7rem;
  border-radius: 6px;
  background: #2c3e50;
  color: #f8f9fa;
  font-size: 0.8rem;
  line-height: 1.35;
  pointer-events: none;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.18);
}

.tooltip-name {
  font-size: 0.85rem;
}

.status,
.empty {
  color: #6c757d;
}

.status.error {
  color: #c0392b;
}
</style>
