<template>
  <section class="participant-flow-chart" data-testid="participant-flow-chart">
    <div v-if="loading" class="status">Loading race flow…</div>
    <div v-else-if="error" class="status error">{{ error }}</div>
    <p v-else-if="!hasData" class="empty" data-testid="participant-flow-empty">
      Not enough timing data to render race flow yet.
    </p>
    <div v-else class="chart-panel">
      <canvas ref="canvasRef" data-testid="participant-flow-canvas" />
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
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { timingApi } from '@/services/api'
import { useUnitsStore } from '@/stores/units'
import type { RaceStatus, RaceType, TimingRecord } from '@/types/models'
import {
  buildExtrapolationPoint,
  buildParticipantFlows,
  getCurrentElapsedMinutes,
  getFlowChartTitle,
  getFlowLineColor,
  getFlowYAxisLabel,
  resolveRaceStartMs,
} from '@/utils/raceFlowData'
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

const props = defineProps<{
  raceId: string
  participantId: string
  raceStatus?: RaceStatus
  raceStartTime?: string
  raceType?: RaceType
}>()

const unitsStore = useUnitsStore()
const chartRaceType = computed(() => props.raceType ?? 'time_based')
const isActiveRace = computed(() => props.raceStatus === 'active')

const canvasRef = ref<HTMLCanvasElement | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)
const records = ref<TimingRecord[]>([])
const chartInstance = ref<Chart | null>(null)
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

  const elapsed = getCurrentElapsedMinutes(raceStartMs.value, nowMs.value)
  const lastPoint = participantFlow.value?.points.at(-1)
  const latestRecordedMinute = lastPoint?.elapsedMinutes ?? 0

  return elapsed > latestRecordedMinute ? elapsed : null
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
  }, 30_000)
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

function renderChart(): void {
  destroyChart()

  const flow = participantFlow.value
  if (!canvasRef.value || !flow || flow.points.length === 0) {
    return
  }

  const color = getFlowLineColor(flow.participantId)
  const showCurrentTime = currentElapsedMinutes.value != null
  const extrapolation = showCurrentTime
    ? buildExtrapolationPoint(flow, currentElapsedMinutes.value!)
    : null
  const chartPoints = flow.points.map((point) => ({
    x: point.elapsedMinutes,
    y: point.value,
  }))

  if (extrapolation) {
    chartPoints.push({
      x: extrapolation.elapsedMinutes,
      y: extrapolation.value,
    })
  }

  chartInstance.value = new Chart(canvasRef.value, {
    type: 'line',
    data: {
      datasets: [
        {
          label: flow.label,
          data: chartPoints,
          borderColor: color,
          backgroundColor: color,
          pointBackgroundColor: color,
          pointBorderColor: color,
          borderWidth: 3,
          tension: 0.2,
          ...(extrapolation
            ? {
                segment: {
                  borderDash: (ctx: { p1DataIndex: number }) =>
                    ctx.p1DataIndex === chartPoints.length - 1 ? [6, 6] : undefined,
                },
                pointRadius: chartPoints.map((_point, pointIndex) =>
                  pointIndex === chartPoints.length - 1 ? 0 : 4,
                ),
              }
            : {}),
        },
      ],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      scales: {
        x: {
          type: 'linear',
          title: { display: true, text: 'Elapsed time (minutes)' },
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
        legend: { display: false },
        title: {
          display: true,
          text: getFlowChartTitle(chartRaceType.value, showCurrentTime),
        },
      },
    },
  })
}

onMounted(async () => {
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
  () => [props.raceStatus, props.raceStartTime, props.raceType],
  () => {
    startLiveRefreshTimer()
  },
)

watch(
  [participantFlow, loading, currentElapsedMinutes, chartRaceType, () => unitsStore.unitSystem],
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
</script>

<style scoped>
.participant-flow-chart {
  min-height: 280px;
}

.chart-panel {
  background: white;
  border-radius: 8px;
  padding: 1rem;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.08);
}

canvas {
  width: 100% !important;
  height: 280px !important;
}

.status,
.empty {
  color: #6c757d;
}

.status.error {
  color: #c0392b;
}
</style>
