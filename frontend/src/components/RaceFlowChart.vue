<template>
  <section class="race-flow-chart" data-testid="race-flow-chart">
    <div v-if="loading" class="status">Loading race flow…</div>
    <div v-else-if="error" class="status error">{{ error }}</div>
    <p v-else-if="!hasData" class="empty" data-testid="race-flow-empty">
      Not enough timing data to render race flow yet.
    </p>
    <div v-else class="chart-panel">
      <div class="chart-canvas-host">
        <canvas ref="canvasRef" data-testid="race-flow-canvas" />
      </div>
      <div class="legend-panel" data-testid="race-flow-legend" aria-label="Participant legend">
        <div class="legend-controls">
          <div class="legend-controls-row">
            <label class="legend-search-label">
              <span class="sr-only">Search legend by bib or name</span>
              <input
                v-model="searchQuery"
                type="search"
                class="legend-search"
                placeholder="Search by bib or name…"
                data-testid="race-flow-legend-search"
                aria-label="Search legend by bib or name"
              />
            </label>
            <button
              type="button"
              class="select-all-btn"
              data-testid="race-flow-select-all"
              @click="toggleSelectAllFiltered"
            >
              {{ allFilteredSelected ? 'Deselect all' : 'Select all' }}
            </button>
          </div>
          <div class="legend-filters" data-testid="race-flow-filters">
            <div
              v-if="availableStatuses.length"
              class="filter-dropdown"
              data-testid="race-flow-status-filters"
              @click.stop
            >
              <button
                type="button"
                class="filter-dropdown-trigger"
                :aria-expanded="openFilter === 'status'"
                @click="toggleFilterDropdown('status')"
              >
                <span class="filter-dropdown-label">Status</span>
                <span class="filter-dropdown-value">
                  {{ formatFilterSummary(selectedStatuses, availableStatuses, getParticipantStatusLabel) }}
                </span>
              </button>
              <ul
                v-if="openFilter === 'status'"
                class="filter-dropdown-menu"
                role="listbox"
                aria-multiselectable="true"
              >
                <li
                  v-for="status in availableStatuses"
                  :key="status"
                  role="option"
                  :aria-selected="selectedStatuses.includes(status)"
                >
                  <button
                    type="button"
                    class="filter-dropdown-option"
                    :class="{ selected: selectedStatuses.includes(status) }"
                    @click="toggleStatusFilter(status)"
                  >
                    {{ getParticipantStatusLabel(status) }}
                  </button>
                </li>
              </ul>
            </div>
            <div
              v-if="availableGenders.length"
              class="filter-dropdown"
              data-testid="race-flow-gender-filters"
              @click.stop
            >
              <button
                type="button"
                class="filter-dropdown-trigger"
                :aria-expanded="openFilter === 'gender'"
                @click="toggleFilterDropdown('gender')"
              >
                <span class="filter-dropdown-label">Gender</span>
                <span class="filter-dropdown-value">
                  {{ formatFilterSummary(selectedGenders, availableGenders, getParticipantGenderLabel) }}
                </span>
              </button>
              <ul
                v-if="openFilter === 'gender'"
                class="filter-dropdown-menu"
                role="listbox"
                aria-multiselectable="true"
              >
                <li
                  v-for="genderKey in availableGenders"
                  :key="genderKey"
                  role="option"
                  :aria-selected="selectedGenders.includes(genderKey)"
                >
                  <button
                    type="button"
                    class="filter-dropdown-option"
                    :class="{ selected: selectedGenders.includes(genderKey) }"
                    @click="toggleGenderFilter(genderKey)"
                  >
                    {{ getParticipantGenderLabel(genderKey) }}
                  </button>
                </li>
              </ul>
            </div>
            <div
              v-if="availableAgeGroups.length"
              class="filter-dropdown"
              data-testid="race-flow-age-group-filters"
              @click.stop
            >
              <button
                type="button"
                class="filter-dropdown-trigger"
                :aria-expanded="openFilter === 'ageGroup'"
                @click="toggleFilterDropdown('ageGroup')"
              >
                <span class="filter-dropdown-label">Age group</span>
                <span class="filter-dropdown-value">
                  {{ formatFilterSummary(selectedAgeGroups, availableAgeGroups, getParticipantAgeGroupLabel) }}
                </span>
              </button>
              <ul
                v-if="openFilter === 'ageGroup'"
                class="filter-dropdown-menu"
                role="listbox"
                aria-multiselectable="true"
              >
                <li
                  v-for="ageGroup in availableAgeGroups"
                  :key="ageGroup"
                  role="option"
                  :aria-selected="selectedAgeGroups.includes(ageGroup)"
                >
                  <button
                    type="button"
                    class="filter-dropdown-option"
                    :class="{ selected: selectedAgeGroups.includes(ageGroup) }"
                    @click="toggleAgeGroupFilter(ageGroup)"
                  >
                    {{ getParticipantAgeGroupLabel(ageGroup) }}
                  </button>
                </li>
              </ul>
            </div>
          </div>
        </div>
        <p v-if="filteredLegendItems.length === 0" class="legend-empty">
          No participants match the current search or filters.
        </p>
        <div
          v-else
          ref="legendItemsRef"
          class="legend-items"
          @scroll="handleLegendScroll"
        >
          <div
            v-for="item in filteredLegendItems"
            :key="item.participantId"
            class="legend-item"
            :class="{
              'legend-item-hidden': !isParticipantVisible(item.participantId),
              'legend-item-hovered':
                (hoveredParticipantId === item.participantId && !highlightParticipantId) ||
                highlightParticipantId === item.participantId,
            }"
            @mouseenter="highlightParticipant(item, $event)"
            @mousemove="moveTooltip($event)"
            @mouseleave="unhighlightParticipant"
          >
            <input
              type="checkbox"
              :aria-label="`Toggle visibility for ${item.label}`"
              :checked="isParticipantVisible(item.participantId)"
              @change="toggleParticipantVisibility(item.participantId)"
              @click.stop
            />
            <button
              type="button"
              class="legend-select"
              data-testid="race-flow-legend-select"
              :aria-pressed="highlightParticipantId === item.participantId"
              @click="selectParticipant(item.participantId)"
            >
              <span
                class="color-swatch"
                :style="{ backgroundColor: item.color }"
                aria-hidden="true"
              />
              <span class="legend-label">{{ item.label }}</span>
            </button>
          </div>
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
        <span v-if="activeTooltip.gender">Gender: {{ getParticipantGenderLabel(activeTooltip.gender) }}</span>
        <span v-if="activeTooltip.ageGroup">Age group: {{ getParticipantAgeGroupLabel(activeTooltip.ageGroup) }}</span>
        <span v-else-if="activeTooltip.age != null">Age: {{ activeTooltip.age }}</span>
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
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch, type Ref } from 'vue'
import { timingApi, raceParticipantsApi } from '@/services/api'
import { useUnitsStore } from '@/stores/units'
import type { Participant, ParticipantStatus, RaceStatus, RaceType, TimingRecord } from '@/types/models'
import { resolveCategoryColor } from '@/themes/defaultLegend'
import {
  buildExtrapolationPoint,
  buildParticipantFlowTooltip,
  buildParticipantFlows,
  clampElapsedToDuration,
  compareAgeGroupKeys,
  compareGenderKeys,
  expandSteppedLapPoints,
  getCurrentElapsedMinutes,
  getFlowYAxisLabel,
  getParticipantAgeGroupLabel,
  getParticipantGenderLabel,
  getParticipantStatusLabel,
  resolveRaceFlowXAxisMax,
  resolveRaceStartMs,
  type ParticipantFlowTooltip,
} from '@/utils/raceFlowData'
import { getErrorMessage } from '@/utils/error'

const SIGNAL_COLOR = '#c45c38'
const LIVE_REFRESH_MS = 30_000
const STATUS_ORDER: ParticipantStatus[] = [
  'finished',
  'started',
  'registered',
  'dnf',
  'dns',
]

type FilterDropdownKey = 'status' | 'gender' | 'ageGroup'

interface FlowLineDataset {
  label: string
  data: Array<{ x: number; y: number; kind?: 'rfid' | 'karaoke' }>
  borderColor: string
  backgroundColor: string
  pointBackgroundColor: string
  pointBorderColor: string
  borderWidth: number
  tension: number
  stepped?: false | 'after'
  hasExtrapolation: boolean
  segment?: {
    borderDash: (ctx: { p1DataIndex: number }) => number[] | undefined
  }
  pointRadius?: number | number[]
  pointStyle?: Array<'circle' | HTMLCanvasElement | HTMLImageElement | false>
  participantId?: string
}

const musicNoteStyleCache = new Map<string, HTMLImageElement>()

function createMusicNotePointStyle(color: string): HTMLImageElement {
  const cached = musicNoteStyleCache.get(color)
  if (cached) {
    return cached
  }

  const safeColor = color.replace(/[<>"']/g, '')
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 20 20"><text x="10" y="14" text-anchor="middle" font-size="14" font-family="Georgia, serif" fill="${safeColor}">♪</text></svg>`
  const img = new Image(20, 20)
  img.src = `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`
  musicNoteStyleCache.set(color, img)
  return img
}

interface LegendItem {
  participantId: string
  label: string
  color: string
  status: ParticipantStatus
  genderKey: string
  ageGroup: string
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
    ctx.strokeStyle = SIGNAL_COLOR
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

const HIGHLIGHT_COLOR = '#9b654e'
const DIMMED_OPACITY = 0.2

const props = defineProps<{
  raceId: string
  raceStatus?: RaceStatus
  raceStartTime?: string
  raceType?: RaceType
  durationMinutes?: number
  highlightParticipantId?: string
}>()

const emit = defineEmits<{
  'update:highlightParticipantId': [value: string | undefined]
}>()

function selectParticipant(participantId: string | undefined): void {
  const next =
    participantId != null && participantId === props.highlightParticipantId
      ? undefined
      : participantId
  emit('update:highlightParticipantId', next)
}

function clearStickyHighlight(): void {
  if (props.highlightParticipantId != null) {
    emit('update:highlightParticipantId', undefined)
  }
}

const unitsStore = useUnitsStore()
const chartRaceType = computed(() => props.raceType ?? 'time_based')

const canvasRef = ref<HTMLCanvasElement | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)
const records = ref<TimingRecord[]>([])
const registeredParticipants = ref<Participant[]>([])
const chartInstance = ref<Chart | null>(null)
const nowMs = ref(Date.now())
const searchQuery = ref('')
const legendItemsRef = ref<HTMLElement | null>(null)
const legendScrollTop = ref(0)
const selectedStatuses = ref<ParticipantStatus[]>([])
const selectedGenders = ref<string[]>([])
const selectedAgeGroups = ref<string[]>([])
const openFilter = ref<FilterDropdownKey | null>(null)
const visibleParticipantIds = ref<Set<string>>(new Set())
const hoveredParticipantId = ref<string | null>(null)
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
    registeredParticipants.value,
  ),
)
const hasData = computed(() => flows.value.length > 0)
const raceStartMs = computed(() => resolveRaceStartMs(records.value, props.raceStartTime))
const currentElapsedMinutes = computed(() => {
  if (!isActiveRace.value || raceStartMs.value === null) {
    return null
  }

  const elapsed = clampElapsedToDuration(
    getCurrentElapsedMinutes(raceStartMs.value, nowMs.value),
    props.durationMinutes,
  )
  const latestRecordedMinute = flows.value.reduce((latest, flow) => {
    const lastPoint = flow.points.at(-1)
    return lastPoint ? Math.max(latest, lastPoint.elapsedMinutes) : latest
  }, 0)

  return elapsed > latestRecordedMinute ? elapsed : null
})

const availableStatuses = computed(() =>
  STATUS_ORDER.filter((status) => flows.value.some((flow) => flow.status === status)),
)

const availableGenders = computed(() =>
  [...new Set(flows.value.map((flow) => flow.genderKey))].sort(compareGenderKeys),
)

const availableAgeGroups = computed(() =>
  [...new Set(flows.value.map((flow) => flow.ageGroup))].sort(compareAgeGroupKeys),
)

const filteredFlows = computed(() => {
  const query = searchQuery.value.trim().toLowerCase()

  return flows.value.filter((flow) => {
    if (!selectedStatuses.value.includes(flow.status)) {
      return false
    }

    if (!selectedGenders.value.includes(flow.genderKey)) {
      return false
    }

    if (!selectedAgeGroups.value.includes(flow.ageGroup)) {
      return false
    }

    if (!query) {
      return true
    }

    return (
      flow.label.toLowerCase().includes(query) ||
      flow.bibNumber.toLowerCase().includes(query)
    )
  })
})

const visibleFlows = computed(() =>
  filteredFlows.value.filter((flow) => visibleParticipantIds.value.has(flow.participantId)),
)

const visibleFlowColors = computed(() => {
  const colors = new Map<string, string>()
  for (const participantId of visibleFlows.value.map((flow) => flow.participantId).sort()) {
    colors.set(participantId, resolveCategoryColor(participantId))
  }
  return colors
})

function getParticipantColor(participantId: string): string {
  return (
    visibleFlowColors.value.get(participantId) ??
    resolveCategoryColor(participantId)
  )
}

function buildLegendItem(flow: (typeof flows.value)[number]): LegendItem {
  return {
    participantId: flow.participantId,
    label: flow.label,
    color: getParticipantColor(flow.participantId),
    status: flow.status,
    genderKey: flow.genderKey,
    ageGroup: flow.ageGroup,
    bibNumber: flow.bibNumber,
    tooltip: buildParticipantFlowTooltip(flow, chartRaceType.value, unitsStore.unitSystem),
  }
}

const filteredLegendItems = computed(() => filteredFlows.value.map(buildLegendItem))

const tooltipStyle = computed(() => ({
  top: `${tooltipPosition.value.y}px`,
  left: `${tooltipPosition.value.x}px`,
}))

const allFilteredSelected = computed(() =>
  filteredLegendItems.value.length > 0 &&
  filteredLegendItems.value.every((item) =>
    visibleParticipantIds.value.has(item.participantId),
  ),
)

const isLegendBusy = computed(() => {
  if (searchQuery.value.trim() !== '') {
    return true
  }

  if (legendScrollTop.value > 0) {
    return true
  }

  if (
    availableStatuses.value.length > 0 &&
    selectedStatuses.value.length < availableStatuses.value.length
  ) {
    return true
  }

  if (
    availableGenders.value.length > 0 &&
    selectedGenders.value.length < availableGenders.value.length
  ) {
    return true
  }

  if (
    availableAgeGroups.value.length > 0 &&
    selectedAgeGroups.value.length < availableAgeGroups.value.length
  ) {
    return true
  }

  return false
})

function handleLegendScroll(event: Event): void {
  const target = event.target
  if (target instanceof HTMLElement) {
    legendScrollTop.value = target.scrollTop
  }
}

function syncFilterSelections(): void {
  selectedStatuses.value = mergeFilterSelections(
    selectedStatuses.value,
    availableStatuses.value,
  )
  selectedGenders.value = mergeFilterSelections(
    selectedGenders.value,
    availableGenders.value,
  )
  selectedAgeGroups.value = mergeFilterSelections(
    selectedAgeGroups.value,
    availableAgeGroups.value,
  )
}

function mergeFilterSelections<T extends string>(current: T[], available: T[]): T[] {
  if (available.length === 0) {
    return []
  }

  if (current.length === 0) {
    return [...available]
  }

  const preserved = current.filter((value) => available.includes(value))
  const added = available.filter((value) => !current.includes(value))
  return [...preserved, ...added]
}

function toggleFilterDropdown(key: FilterDropdownKey): void {
  openFilter.value = openFilter.value === key ? null : key
}

function toggleFilterValue<T>(selectedRef: Ref<T[]>, value: T): void {
  if (selectedRef.value.includes(value)) {
    selectedRef.value = selectedRef.value.filter((item) => item !== value)
    return
  }

  selectedRef.value = [...selectedRef.value, value]
}

function toggleStatusFilter(status: ParticipantStatus): void {
  toggleFilterValue(selectedStatuses, status)
}

function toggleGenderFilter(genderKey: string): void {
  toggleFilterValue(selectedGenders, genderKey)
}

function toggleAgeGroupFilter(ageGroup: string): void {
  toggleFilterValue(selectedAgeGroups, ageGroup)
}

function formatFilterSummary<T extends string>(
  selected: T[],
  available: T[],
  labelFn: (value: T) => string,
): string {
  const activeSelected = selected.filter((value) => available.includes(value))

  if (activeSelected.length === 0) {
    return 'None'
  }

  if (activeSelected.length === available.length) {
    return 'All'
  }

  if (activeSelected.length <= 2) {
    return activeSelected.map(labelFn).join(', ')
  }

  return `${activeSelected.length} selected`
}

function handleDocumentClick(event: MouseEvent): void {
  const target = event.target
  if (!(target instanceof Element)) {
    return
  }
  if (!target.closest('.filter-dropdown')) {
    openFilter.value = null
  }
  if (!target.closest('.race-flow-chart')) {
    clearStickyHighlight()
  }
}

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

function highlightParticipant(item: LegendItem, event: MouseEvent): void {
  if (!props.highlightParticipantId) {
    hoveredParticipantId.value = item.participantId
  }
  showTooltip(item, event)
}

function unhighlightParticipant(): void {
  hoveredParticipantId.value = null
  hideTooltip()
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
    const [live, participantsRes] = await Promise.all([
      timingApi.getLive(props.raceId),
      raceParticipantsApi.list(props.raceId, { limit: 500 }),
    ])
    records.value = live.data.records ?? []
    registeredParticipants.value = participantsRes.data.data ?? []
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

function getLiveDisplayScale(): number {
  if (typeof window === 'undefined') {
    return 1
  }

  const host = canvasRef.value?.closest('.event-live, [data-testid=fullscreen-rotator]')
  if (!host) {
    return 1
  }

  const raw = getComputedStyle(host).getPropertyValue('--live-display-scale').trim()
  const scale = Number.parseFloat(raw || '1')
  return Number.isFinite(scale) && scale > 0 ? scale : 1
}

function chartFontSize(base: number): number {
  return Math.round(base * getLiveDisplayScale())
}

function withAlpha(color: string, alpha: number): string {
  if (color.startsWith('hsl(')) {
    return color.replace(')', `, ${alpha})`).replace('hsl(', 'hsla(')
  }

  if (color.startsWith('#')) {
    const hex = color.slice(1)
    const r = Number.parseInt(hex.slice(0, 2), 16)
    const g = Number.parseInt(hex.slice(2, 4), 16)
    const b = Number.parseInt(hex.slice(4, 6), 16)
    return `rgba(${r}, ${g}, ${b}, ${alpha})`
  }

  return color
}

function getEffectiveHighlightId(): string | undefined {
  return props.highlightParticipantId ?? hoveredParticipantId.value ?? undefined
}

function getOrderedVisibleFlows(): (typeof flows.value)[number][] {
  // Only reorder for a transient hover preview so it draws on top; the sticky
  // (prop-driven) selection keeps a stable dataset order so click hit-testing
  // (by datasetIndex) stays valid while selected.
  const highlightId = hoveredParticipantId.value

  if (!highlightId) {
    return visibleFlows.value
  }

  return [
    ...visibleFlows.value.filter((flow) => flow.participantId !== highlightId),
    ...visibleFlows.value.filter((flow) => flow.participantId === highlightId),
  ]
}

function getLineStyle(flow: (typeof flows.value)[number]): Pick<
  FlowLineDataset,
  'borderColor' | 'backgroundColor' | 'pointBackgroundColor' | 'pointBorderColor' | 'borderWidth'
> {
  const highlightId = getEffectiveHighlightId()
  const isHighlighted = highlightId === flow.participantId
  const isDimmed = highlightId != null && !isHighlighted
  const baseColor = isHighlighted ? HIGHLIGHT_COLOR : getParticipantColor(flow.participantId)
  const color = isDimmed ? withAlpha(baseColor, DIMMED_OPACITY) : baseColor

  return {
    borderColor: color,
    backgroundColor: color,
    pointBackgroundColor: color,
    pointBorderColor: color,
    borderWidth: isHighlighted ? 4 : isDimmed ? 1 : 2,
  }
}

function buildDataset(flow: (typeof flows.value)[number]): FlowLineDataset {
  const showCurrentTime = currentElapsedMinutes.value != null
  const extrapolation = showCurrentTime
    ? buildExtrapolationPoint(flow, currentElapsedMinutes.value!)
    : null
  const rawPoints = [
    ...flow.points.map((point) => ({
      x: point.elapsedMinutes,
      y: point.value,
      kind: point.kind,
    })),
  ]

  if (extrapolation) {
    rawPoints.push({
      x: extrapolation.elapsedMinutes,
      y: extrapolation.value,
      kind: undefined,
    })
  }

  const isLapChart = chartRaceType.value === 'lap_based'
  const chartPoints = isLapChart ? expandSteppedLapPoints(rawPoints) : rawPoints
  const lineStyle = getLineStyle(flow)

  const dataset: FlowLineDataset = {
    label: flow.label,
    data: chartPoints,
    ...lineStyle,
    tension: 0,
    stepped: false,
    hasExtrapolation: extrapolation != null,
    participantId: flow.participantId,
  }

  const isExtrapolationIndex = (pointIndex: number) =>
    extrapolation != null && pointIndex === chartPoints.length - 1

  dataset.pointRadius = chartPoints.map((point, pointIndex) => {
    if (isExtrapolationIndex(pointIndex)) {
      return 0
    }
    if (point.kind === 'karaoke') {
      return 8
    }
    // Only show markers on real tap vertices, not the synthetic step corners.
    return rawPoints.some((raw) => raw.x === point.x && raw.y === point.y && raw.kind) ? 4 : 0
  })

  dataset.pointStyle = chartPoints.map((point, pointIndex) => {
    if (isExtrapolationIndex(pointIndex) || !point.kind) {
      return false
    }
    if (point.kind === 'karaoke') {
      return createMusicNotePointStyle(lineStyle.borderColor)
    }
    return 'circle'
  })

  if (extrapolation) {
    dataset.segment = {
      borderDash: (ctx: { p1DataIndex: number }) =>
        ctx.p1DataIndex === chartPoints.length - 1 ? [6, 6] : undefined,
    }
  }

  return dataset
}

function updateLineHighlight(): void {
  const chart = chartInstance.value
  if (!chart) {
    return
  }

  for (const dataset of chart.data.datasets) {
    const flowDataset = dataset as FlowLineDataset
    const flow = visibleFlows.value.find(
      (item) => item.participantId === flowDataset.participantId,
    )
    if (!flow) {
      continue
    }

    Object.assign(flowDataset, getLineStyle(flow))
  }

  chart.update('none')
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

  const orderedFlows = getOrderedVisibleFlows()
  const xAxisMax = resolveRaceFlowXAxisMax(
    props.durationMinutes,
    maxElapsedMinutes,
    currentElapsedMinutes.value,
    showCurrentTime,
  )

  const tickSize = chartFontSize(10)
  const axisTitleSize = chartFontSize(11)
  const axisInk = '#1a3f3d'

  chartInstance.value = new Chart(canvasRef.value, {
    type: 'line',
    data: {
      datasets: orderedFlows.map((flow) => buildDataset(flow)),
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      interaction: {
        mode: 'nearest',
        intersect: false,
        axis: 'x',
      },
      onHover: (_event, elements, chart) => {
        chart.canvas.style.cursor = elements.length > 0 ? 'pointer' : 'default'
        if (props.highlightParticipantId) {
          return
        }
        if (elements.length > 0) {
          const dataset = chart.data.datasets[elements[0].datasetIndex] as FlowLineDataset
          hoveredParticipantId.value = dataset.participantId ?? null
        } else {
          hoveredParticipantId.value = null
        }
      },
      onClick: (event, _elements, chart) => {
        const hits = chart.getElementsAtEventForMode(
          event as unknown as Event,
          'nearest',
          { intersect: true },
          true,
        )
        if (hits.length === 0) {
          clearStickyHighlight()
          return
        }
        const dataset = chart.data.datasets[hits[0].datasetIndex] as FlowLineDataset
        selectParticipant(dataset.participantId)
      },
      scales: {
        x: {
          type: 'linear',
          min: 0,
          title: {
            display: true,
            text: 'Elapsed time (minutes)',
            color: axisInk,
            font: { size: axisTitleSize, weight: 'bold' },
          },
          ticks: {
            color: axisInk,
            font: { size: tickSize },
          },
          max: xAxisMax,
        },
        y: {
          beginAtZero: true,
          title: {
            display: true,
            text: getFlowYAxisLabel(chartRaceType.value, unitsStore.unitSystem),
            color: axisInk,
            font: { size: axisTitleSize, weight: 'bold' },
          },
          ticks: {
            color: axisInk,
            ...(chartRaceType.value === 'lap_based' ? { stepSize: 1 } : {}),
            font: { size: tickSize },
          },
        },
      },
      plugins: {
        currentTimeLine: { xMinutes: currentElapsedMinutes.value },
        legend: { display: false },
        title: { display: false },
      } as Record<string, unknown>,
    },
  })
}

onMounted(async () => {
  document.addEventListener('click', handleDocumentClick)
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
  () => [props.raceStatus, props.raceStartTime, props.raceType, props.durationMinutes],
  () => {
    startLiveRefreshTimer()
  },
)

watch(flows, () => {
  syncFilterSelections()
  syncVisibleParticipants()
})

watch(
  [
    visibleFlows,
    loading,
    currentElapsedMinutes,
    chartRaceType,
    () => props.durationMinutes,
    () => unitsStore.unitSystem,
  ],
  async () => {
    if (!loading.value) {
      await nextTick()
      renderChart()
    }
  },
)

watch(hoveredParticipantId, () => {
  if (!loading.value && chartInstance.value) {
    updateLineHighlight()
  }
})

watch(
  () => props.highlightParticipantId,
  () => {
    if (!loading.value && chartInstance.value) {
      updateLineHighlight()
    }
  },
)

watch(
  () => props.highlightParticipantId,
  (participantId) => {
    if (!participantId || visibleParticipantIds.value.has(participantId)) {
      return
    }

    const nextVisibleIds = new Set(visibleParticipantIds.value)
    nextVisibleIds.add(participantId)
    visibleParticipantIds.value = nextVisibleIds
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  document.removeEventListener('click', handleDocumentClick)
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
  hoveredParticipantId,
  isLegendBusy,
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

.chart-canvas-host {
  position: relative;
  width: 100%;
  height: 320px;
}

.legend-panel {
  background: var(--mist);
  border-radius: 8px;
  padding: 1rem;
  font-size: calc(0.9rem * var(--live-display-scale, 1));
}

.legend-controls {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  margin-bottom: 0.75rem;
}

.legend-controls-row {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem 1rem;
  align-items: center;
}

.legend-search-label {
  flex: 1 1 220px;
  min-width: 180px;
  display: block;
}

.legend-search {
  width: 100%;
  padding: 0.45rem 0.65rem;
  border: 1px solid var(--border);
  border-radius: 4px;
  font: inherit;
}

.legend-filters {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.filter-dropdown {
  position: relative;
  min-width: 160px;
}

.filter-dropdown-trigger {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 0.1rem;
  width: 100%;
  padding: 0.45rem 2rem 0.45rem 0.65rem;
  border: 1px solid var(--border);
  border-radius: 4px;
  background: var(--surface);
  color: var(--ink);
  cursor: pointer;
  font: inherit;
  text-align: left;
}

.filter-dropdown-trigger::after {
  content: '▾';
  position: absolute;
  right: 0.65rem;
  top: 50%;
  transform: translateY(-50%);
  color: var(--muted);
  pointer-events: none;
}

.filter-dropdown-trigger:hover {
  background: var(--mist);
}

.filter-dropdown-label {
  font-size: calc(0.72rem * var(--live-display-scale, 1));
  font-weight: 600;
  color: var(--muted);
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.filter-dropdown-value {
  font-size: calc(0.85rem * var(--live-display-scale, 1));
  color: var(--ink);
}

.filter-dropdown-menu {
  position: absolute;
  z-index: 20;
  top: calc(100% + 0.25rem);
  left: 0;
  min-width: 100%;
  margin: 0;
  padding: 0.25rem 0;
  list-style: none;
  border: 1px solid var(--border);
  border-radius: 4px;
  background: var(--surface);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
  max-height: 220px;
  overflow-y: auto;
}

.filter-dropdown-option {
  display: block;
  width: 100%;
  padding: 0.45rem 0.75rem;
  border: none;
  background: transparent;
  color: var(--ink);
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.filter-dropdown-option:hover {
  background: var(--mist);
}

.filter-dropdown-option.selected {
  background: color-mix(in srgb, var(--accent-link) 15%, var(--surface));
  color: var(--accent-link);
  font-weight: 600;
}

.filter-dropdown-option.selected::before {
  content: '✓ ';
}

.select-all-btn {
  padding: 0.45rem 0.75rem;
  border: 1px solid var(--border);
  border-radius: 4px;
  background: var(--surface);
  color: var(--ink);
  cursor: pointer;
  font: inherit;
}

.select-all-btn:hover {
  background: var(--mist);
}

.legend-empty {
  margin: 0;
  color: var(--muted);
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
  font-size: calc(0.85rem * var(--live-display-scale, 1));
  color: var(--ink);
}

.legend-select {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  flex: 1;
  min-width: 0;
  padding: 0;
  border: none;
  background: none;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.legend-item-hidden .legend-label {
  opacity: 0.55;
}

.legend-item-hovered {
  background: color-mix(in srgb, var(--accent-link) 15%, var(--surface));
  border-radius: 4px;
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
  background: var(--ink-deep);
  color: var(--mist);
  font-size: calc(0.8rem * var(--live-display-scale, 1));
  line-height: 1.35;
  pointer-events: none;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.18);
}

.tooltip-name {
  font-size: calc(0.85rem * var(--live-display-scale, 1));
}

.status,
.empty {
  color: var(--muted);
}

.status.error {
  color: var(--signal);
}
</style>
