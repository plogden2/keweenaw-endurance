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
} from 'chart.js'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { timingApi } from '@/services/api'
import type { TimingRecord } from '@/types/models'
import { buildParticipantFlows } from '@/utils/raceFlowData'
import { getErrorMessage } from '@/utils/error'

Chart.register(
  LineController,
  LineElement,
  PointElement,
  LinearScale,
  CategoryScale,
  Title,
  Tooltip,
  Legend,
)

const props = defineProps<{
  raceId: string
}>()

const canvasRef = ref<HTMLCanvasElement | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)
const records = ref<TimingRecord[]>([])
const chartInstance = ref<Chart | null>(null)

const flows = computed(() => buildParticipantFlows(records.value))
const hasData = computed(() => flows.value.length > 0)

async function loadRecords(): Promise<void> {
  loading.value = true
  error.value = null
  try {
    const { data } = await timingApi.getLive(props.raceId)
    records.value = data.records ?? []
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

  chartInstance.value = new Chart(canvasRef.value, {
    type: 'line',
    data: {
      datasets: flows.value.map((flow) => ({
        label: flow.label,
        data: flow.points.map((point) => ({
          x: point.elapsedMinutes,
          y: point.position,
        })),
        tension: 0.2,
      })),
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
          reverse: true,
          title: { display: true, text: 'Position' },
          ticks: { stepSize: 1 },
        },
      },
      plugins: {
        legend: { display: true, position: 'bottom' },
        title: { display: true, text: 'Finish order over elapsed time' },
      },
    },
  })
}

onMounted(async () => {
  await loadRecords()
})

watch(
  () => props.raceId,
  async () => {
    await loadRecords()
  },
)

watch([flows, loading], () => {
  if (!loading.value) {
    renderChart()
  }
})

onBeforeUnmount(() => {
  destroyChart()
})

defineExpose({ loadRecords, records, flows })
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
