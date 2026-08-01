<template>
  <section class="participant-flow-chart" data-testid="participant-flow-chart">
    <div v-if="loading" class="status">Loading race flow…</div>
    <div v-else-if="error" class="status error">{{ error }}</div>
    <p v-else-if="!hasData" class="empty" data-testid="participant-flow-empty">
      Not enough timing data to render race flow yet.
    </p>
    <div v-else class="chart-panel">
      <div class="chart-canvas-host">
        <canvas ref="canvasRef" data-testid="participant-flow-canvas" />
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
} from 'chart.js'
import {
  computed,
  markRaw,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  shallowRef,
  watch,
} from 'vue'
import { timingApi } from '@/services/api'
import { useUnitsStore } from '@/stores/units'
import type { RaceStatus, RaceType, TimingRecord } from '@/types/models'
import {
  buildExtrapolationPoint,
  buildParticipantFlows,
  clampElapsedToDuration,
  expandSteppedLapPoints,
  formatElapsedClock,
  getCurrentElapsedMinutes,
  getFlowLineColor,
  getFlowYAxisLabel,
  resolveRaceFlowXAxisMax,
  resolveRaceStartMs,
} from '@/utils/raceFlowData'
import { resolveChartDevicePixelRatio } from '@/utils/raceFlowZoom'
import { getErrorMessage } from '@/utils/error'

Chart.register(
  LineController,
  LineElement,
  PointElement,
  LinearScale,
  CategoryScale,
  Title,
  Tooltip,
)

const LIVE_REFRESH_MS = 30_000

const props = defineProps<{
  raceId: string
  participantId: string
  raceStatus?: RaceStatus
  raceStartTime?: string
  raceType?: RaceType
  durationMinutes?: number
}>()

const unitsStore = useUnitsStore()
const chartRaceType = computed(() => props.raceType ?? 'time_based')
const isActiveRace = computed(() => props.raceStatus === 'active')

const canvasRef = ref<HTMLCanvasElement | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)
const records = ref<TimingRecord[]>([])
/** shallowRef + markRaw: Chart.js must not be deeply proxied (update() stack overflows). */
const chartInstance = shallowRef<Chart | null>(null)
const nowMs = ref(Date.now())
let liveRefreshTimer: ReturnType<typeof setInterval> | null = null

const flows = computed(() =>
  buildParticipantFlows(
    records.value,
    props.raceStartTime,
    chartRaceType.value,
    unitsStore.unitSystem,
  ),
)

const participantFlow = computed(() =>
  flows.value.find((flow) => flow.participantId === props.participantId) ?? null,
)

const hasData = computed(() => participantFlow.value != null && participantFlow.value.points.length > 0)

const raceStartMs = computed(() => resolveRaceStartMs(records.value, props.raceStartTime))

const currentElapsedMinutes = computed(() => {
  if (!isActiveRace.value || raceStartMs.value === null) {
    return null
  }

  return clampElapsedToDuration(
    getCurrentElapsedMinutes(raceStartMs.value, nowMs.value),
    props.durationMinutes,
  )
})

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
  const showLoading = !hasData.value
  if (showLoading) {
    loading.value = true
  }
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

function restoreChartAfterVisible(): void {
  if (typeof document !== 'undefined' && document.visibilityState !== 'visible') {
    return
  }
  if (loading.value || !hasData.value) {
    return
  }

  void nextTick(() => {
    const chart = chartInstance.value
    if (chart) {
      chart.resize()
      chart.update('none')
      return
    }
    renderChart()
  })
}

function handleVisibilityChange(): void {
  if (document.visibilityState === 'visible') {
    restoreChartAfterVisible()
  }
}

function handlePageShow(event: PageTransitionEvent): void {
  if (event.persisted) {
    restoreChartAfterVisible()
  }
}

function buildChartDataset() {
  const flow = participantFlow.value
  if (!flow) {
    return null
  }

  const color = getFlowLineColor(flow.participantId)
  const showCurrentTime = currentElapsedMinutes.value != null
  const extrapolation = showCurrentTime
    ? buildExtrapolationPoint(flow, currentElapsedMinutes.value!)
    : null
  const rawPoints = flow.points.map((point) => ({
    x: point.elapsedMinutes,
    y: point.value,
  }))

  if (extrapolation) {
    rawPoints.push({
      x: extrapolation.elapsedMinutes,
      y: extrapolation.value,
    })
  }

  const isLapChart = chartRaceType.value === 'lap_based'
  const chartPoints = isLapChart ? expandSteppedLapPoints(rawPoints) : rawPoints

  const recordedMaxMinutes = flow.points.reduce(
    (max, point) => Math.max(max, point.elapsedMinutes),
    0,
  )
  const xAxisMax = resolveRaceFlowXAxisMax(
    props.durationMinutes,
    extrapolation?.elapsedMinutes ?? recordedMaxMinutes,
    currentElapsedMinutes.value,
    showCurrentTime,
  )

  const pointRadius = chartPoints.map((point, pointIndex) => {
    if (extrapolation != null && pointIndex === chartPoints.length - 1) {
      return 0
    }
    if (!isLapChart) {
      return 4
    }
    return rawPoints.some((raw) => raw.x === point.x && raw.y === point.y) ? 4 : 0
  })

  return {
    dataset: {
      label: flow.label,
      data: chartPoints,
      borderColor: color,
      backgroundColor: color,
      pointBackgroundColor: color,
      pointBorderColor: color,
      borderWidth: 3,
      tension: 0,
      stepped: false as const,
      pointHitRadius: 12,
      pointRadius,
      ...(extrapolation
        ? {
            segment: {
              borderDash: (ctx: { p1DataIndex: number }) =>
                ctx.p1DataIndex === chartPoints.length - 1 ? [6, 6] : undefined,
            },
          }
        : {}),
    },
    xAxisMax,
  }
}

/** Move the live "now" extrapolation without destroy()+new Chart() (iOS Safari OOM). */
function updateLiveProgress(): void {
  const chart = chartInstance.value
  const built = buildChartDataset()
  if (!chart || !built) {
    return
  }

  chart.data.datasets = [built.dataset]
  const xScale = chart.options.scales?.x
  if (xScale && typeof xScale === 'object') {
    ;(xScale as { max?: number }).max = built.xAxisMax
  }
  chart.update('none')
}

function renderChart(): void {
  destroyChart()

  const built = buildChartDataset()
  if (!canvasRef.value || !built) {
    return
  }

  chartInstance.value = markRaw(
    new Chart(canvasRef.value, {
      type: 'line',
      data: {
        datasets: [built.dataset],
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        animation: false,
        devicePixelRatio: resolveChartDevicePixelRatio(),
        interaction: {
          mode: 'nearest',
          intersect: false,
          axis: 'xy',
        },
        scales: {
          x: {
            type: 'linear',
            min: Math.min(0, ...(built.dataset.data as Array<{ x: number }>).map((p) => p.x)),
            title: {
              display: true,
              text: 'Elapsed time',
              color: '#1a3f3d',
              font: { size: 9, weight: 'bold' },
            },
            ticks: {
              color: '#1a3f3d',
              font: { size: 8 },
              callback: (value: string | number) => formatElapsedClock(Number(value)),
            },
            max: built.xAxisMax,
          },
          y: {
            beginAtZero: true,
            title: {
              display: true,
              text: getFlowYAxisLabel(chartRaceType.value, unitsStore.unitSystem),
              color: '#1a3f3d',
              font: { size: 9, weight: 'bold' },
            },
            ticks: {
              color: '#1a3f3d',
              font: { size: 8 },
              ...(chartRaceType.value === 'lap_based' ? { stepSize: 1 } : {}),
            },
          },
        },
        plugins: {
          legend: { display: false },
          title: { display: false },
        },
      },
    }),
  )
}

onMounted(async () => {
  document.addEventListener('visibilitychange', handleVisibilityChange)
  window.addEventListener('pageshow', handlePageShow)
  await loadRecords()
  startLiveRefreshTimer()
})

watch(
  () => [props.raceId, props.participantId],
  async () => {
    await loadRecords()
    startLiveRefreshTimer()
  },
)

watch(
  () => [props.raceStatus, props.raceStartTime, props.raceType, props.durationMinutes],
  () => {
    startLiveRefreshTimer()
  },
)

watch(
  [participantFlow, loading, chartRaceType, () => props.durationMinutes, () => unitsStore.unitSystem],
  async () => {
    if (!loading.value) {
      await nextTick()
      renderChart()
    }
  },
)

watch(currentElapsedMinutes, () => {
  if (loading.value) {
    return
  }
  if (chartInstance.value) {
    updateLiveProgress()
    return
  }
  void nextTick().then(() => {
    if (!loading.value && !chartInstance.value) {
      renderChart()
    } else if (chartInstance.value) {
      updateLiveProgress()
    }
  })
})

onBeforeUnmount(() => {
  document.removeEventListener('visibilitychange', handleVisibilityChange)
  window.removeEventListener('pageshow', handlePageShow)
  clearLiveRefreshTimer()
  destroyChart()
})

defineExpose({
  loadRecords,
  currentElapsedMinutes,
})
</script>

<style scoped>
.participant-flow-chart {
  min-height: 280px;
  max-width: 100%;
  min-width: 0;
}

.chart-panel {
  background: white;
  border-radius: 8px;
  padding: 1rem;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.08);
  max-width: 100%;
  min-width: 0;
}

.chart-canvas-host {
  position: relative;
  width: 100%;
  max-width: 100%;
  min-width: 0;
  height: 280px;
  overflow: hidden;
}

.status,
.empty {
  color: #6c757d;
}

.status.error {
  color: #c0392b;
}
</style>
