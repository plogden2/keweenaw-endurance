<template>
  <section class="race-flow-chart" data-testid="race-flow-chart">
    <div v-if="loading" class="status">Loading race flow…</div>
    <div v-else-if="error" class="status error">{{ error }}</div>
    <p v-else-if="!hasData" class="empty" data-testid="race-flow-empty">
      Not enough timing data to render race flow yet.
    </p>
    <canvas v-else ref="canvasRef" data-testid="race-flow-canvas" />
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
  Legend,
  type Plugin,
} from 'chart.js'
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { timingApi } from '@/services/api'
import type { RaceStatus, RaceType, TimingRecord } from '@/types/models'
import {
  buildExtrapolationPoint,
  buildParticipantFlows,
  getCurrentElapsedMinutes,
  getFlowChartTitle,
  getFlowYAxisLabel,
  resolveRaceStartMs,
} from '@/utils/raceFlowData'
import { getErrorMessage } from '@/utils/error'

const FLOW_LINE_COLORS = [
  '#3498db',
  '#e74c3c',
  '#2ecc71',
  '#9b59b6',
  '#f39c12',
  '#1abc9c',
  '#e91e63',
  '#16a085',
  '#d35400',
  '#8e44ad',
]

const CURRENT_TIME_LINE_COLOR = '#e74c3c'
const LIVE_REFRESH_MS = 30_000

function flowLineColor(index: number): string {
  return FLOW_LINE_COLORS[index % FLOW_LINE_COLORS.length]
}

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
  Legend,
  currentTimeLinePlugin,
)

const props = defineProps<{
  raceId: string
  raceStatus?: RaceStatus
  raceStartTime?: string
  raceType?: RaceType
}>()

const chartRaceType = computed(() => props.raceType ?? 'time_based')

const canvasRef = ref<HTMLCanvasElement | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)
const records = ref<TimingRecord[]>([])
const chartInstance = ref<Chart | null>(null)
const nowMs = ref(Date.now())
let liveRefreshTimer: ReturnType<typeof setInterval> | null = null

const isActiveRace = computed(() => props.raceStatus === 'active')
const flows = computed(() =>
  buildParticipantFlows(records.value, props.raceStartTime, chartRaceType.value),
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

function renderChart(): void {
  destroyChart()
  if (!canvasRef.value || !hasData.value) {
    return
  }

  const showCurrentTime = currentElapsedMinutes.value != null
  const maxElapsedMinutes = flows.value.reduce((max, flow) => {
    const extrapolation = showCurrentTime
      ? buildExtrapolationPoint(flow, currentElapsedMinutes.value!)
      : null
    const lastElapsed = extrapolation?.elapsedMinutes ?? flow.points.at(-1)?.elapsedMinutes ?? 0
    return Math.max(max, lastElapsed)
  }, 0)

  chartInstance.value = new Chart(canvasRef.value, {
    type: 'line',
    data: {
      datasets: flows.value.map((flow, index) => {
        const color = flowLineColor(index)
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
      }),
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
          title: { display: true, text: getFlowYAxisLabel(chartRaceType.value) },
          ticks: chartRaceType.value === 'lap_based' ? { stepSize: 1 } : undefined,
        },
      },
      plugins: {
        currentTimeLine: { xMinutes: currentElapsedMinutes.value },
        legend: { display: true, position: 'bottom' },
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

watch([flows, loading, currentElapsedMinutes, chartRaceType], async () => {
  if (!loading.value) {
    await nextTick()
    renderChart()
  }
})

onBeforeUnmount(() => {
  clearLiveRefreshTimer()
  destroyChart()
})

defineExpose({ loadRecords, records, flows, currentElapsedMinutes })
</script>

<style scoped>
.race-flow-chart {
  min-height: 320px;
}

canvas {
  width: 100% !important;
  height: 320px !important;
}

.status,
.empty {
  color: #6c757d;
}

.status.error {
  color: #c0392b;
}
</style>
