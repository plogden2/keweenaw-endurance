import type { ParticipantStatus, RaceType, TimingRecord } from '@/types/models'
import {
  convertDistanceFromKm,
  getDistanceAxisLabel,
  getDistanceUnitAbbreviation,
  type UnitSystem,
} from '@/utils/units'

export interface FlowPoint {
  elapsedMinutes: number
  value: number
  /** Present on real scored taps; omitted on gun-start origin. */
  kind?: 'rfid' | 'karaoke'
}

export interface ParticipantFlow {
  participantId: string
  label: string
  bibNumber: string
  firstName: string
  lastName: string
  gender?: string
  genderKey: string
  age?: number
  ageGroup: string
  status: ParticipantStatus
  points: FlowPoint[]
}

export interface ParticipantFlowTooltip {
  fullName: string
  bibNumber: string
  status: ParticipantStatus
  gender?: string
  age?: number
  ageGroup?: string
  progress: string
}

const FLOW_COLOR_SATURATION = 70
const FLOW_COLOR_LIGHTNESS = 45

export function getFlowLineColor(participantId: string): string {
  let hash = 0
  for (let index = 0; index < participantId.length; index += 1) {
    hash = participantId.charCodeAt(index) + ((hash << 5) - hash)
  }
  const hue = Math.abs(hash) % 360
  return `hsl(${hue}, ${FLOW_COLOR_SATURATION}%, ${FLOW_COLOR_LIGHTNESS}%)`
}

export function assignContrastFlowColors(participantIds: string[]): Map<string, string> {
  const sortedIds = [...participantIds].sort()
  const colors = new Map<string, string>()

  if (sortedIds.length === 0) {
    return colors
  }

  sortedIds.forEach((participantId, index) => {
    const hue = Math.round((index * 360) / sortedIds.length)
    colors.set(
      participantId,
      `hsl(${hue}, ${FLOW_COLOR_SATURATION}%, ${FLOW_COLOR_LIGHTNESS}%)`,
    )
  })

  return colors
}

export function getParticipantStatusLabel(status: ParticipantStatus): string {
  return status.toUpperCase()
}

const GENDER_ORDER = ['male', 'female', 'other', 'unknown']

export function getParticipantGenderKey(gender?: string): string {
  if (!gender) {
    return 'unknown'
  }

  return gender.toLowerCase()
}

export function getParticipantGenderLabel(genderKey: string): string {
  const labels: Record<string, string> = {
    male: 'Male',
    female: 'Female',
    other: 'Other',
    unknown: 'Unknown',
  }

  return labels[genderKey] ?? genderKey
}

export function compareGenderKeys(a: string, b: string): number {
  const indexA = GENDER_ORDER.indexOf(a)
  const indexB = GENDER_ORDER.indexOf(b)
  const orderA = indexA === -1 ? GENDER_ORDER.length : indexA
  const orderB = indexB === -1 ? GENDER_ORDER.length : indexB
  return orderA - orderB
}

export function getParticipantAgeGroupKey(age?: number): string {
  if (age == null || age < 0 || Number.isNaN(age)) {
    return 'unknown'
  }

  if (age < 20) {
    return 'under-20'
  }

  if (age >= 80) {
    return '80-plus'
  }

  const lower = Math.floor(age / 5) * 5
  return `${lower}-${lower + 4}`
}

export function getParticipantAgeGroupLabel(ageGroupKey: string): string {
  if (ageGroupKey === 'unknown') {
    return 'Unknown'
  }

  if (ageGroupKey === 'under-20') {
    return 'Under 20'
  }

  if (ageGroupKey === '80-plus') {
    return '80+'
  }

  return ageGroupKey.replace('-', '–')
}

export function compareAgeGroupKeys(a: string, b: string): number {
  const order = (key: string): number => {
    if (key === 'unknown') {
      return Number.POSITIVE_INFINITY
    }

    if (key === 'under-20') {
      return -1
    }

    if (key === '80-plus') {
      return 1000
    }

    const lower = Number.parseInt(key, 10)
    return Number.isNaN(lower) ? Number.POSITIVE_INFINITY - 1 : lower
  }

  return order(a) - order(b)
}

function createParticipantFlow(participant: NonNullable<TimingRecord['participant']>): ParticipantFlow {
  return {
    participantId: participant.id,
    label: `#${participant.bib_number} ${participant.first_name} ${participant.last_name}`.trim(),
    bibNumber: participant.bib_number,
    firstName: participant.first_name,
    lastName: participant.last_name,
    gender: participant.gender,
    genderKey: getParticipantGenderKey(participant.gender),
    age: participant.age,
    ageGroup: getParticipantAgeGroupKey(participant.age),
    status: participant.status,
    points: [],
  }
}

function formatElapsedMinutes(minutes: number): string {
  const total = Math.max(0, Math.round(minutes))
  const hours = Math.floor(total / 60)
  const mins = total % 60

  if (hours > 0) {
    return `${hours}h ${mins}m`
  }

  return `${mins}m`
}

export function buildParticipantFlowTooltip(
  flow: ParticipantFlow,
  raceType: RaceType,
  unitSystem: UnitSystem = 'imperial',
): ParticipantFlowTooltip {
  const lastPoint = flow.points.at(-1)
  let progress = 'No timing data yet'

  if (lastPoint) {
    const elapsed = formatElapsedMinutes(lastPoint.elapsedMinutes)
    if (raceType === 'lap_based') {
      const laps = Number.isInteger(lastPoint.value)
        ? String(lastPoint.value)
        : lastPoint.value.toFixed(1)
      progress = `${laps} laps at ${elapsed}`
    } else {
      const unit = getDistanceUnitAbbreviation(unitSystem)
      const formatted =
        lastPoint.value >= 10 ? lastPoint.value.toFixed(1) : lastPoint.value.toFixed(2)
      progress = `${formatted} ${unit} at ${elapsed}`
    }
  }

  return {
    fullName: `${flow.firstName} ${flow.lastName}`.trim(),
    bibNumber: flow.bibNumber,
    status: flow.status,
    gender: flow.genderKey,
    age: flow.age,
    ageGroup: flow.ageGroup,
    progress,
  }
}

export interface RaceStatistics {
  totalParticipants: number
  finished: number
  started: number
  registered: number
  dnf: number
  averageFinishSeconds: number | null
  averageLaps: number | null
}

const DISTANCE_CHECKPOINT_TYPES = new Set(['start', 'intermediate', 'finish'])

export function resolveRaceStartMs(
  records: TimingRecord[],
  raceStartTime?: string,
): number | null {
  if (raceStartTime) {
    const startMs = new Date(raceStartTime).getTime()
    if (!Number.isNaN(startMs)) {
      return startMs
    }
  }

  const timedRecords = records
    .filter((record) => record.participant && record.checkpoint)
    .sort(
      (a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime(),
    )

  if (timedRecords.length === 0) {
    return null
  }

  return timedRecords.reduce((earliest, record) => {
    const ts = new Date(record.timestamp).getTime()
    return ts < earliest ? ts : earliest
  }, Number.POSITIVE_INFINITY)
}

export function getCurrentElapsedMinutes(
  raceStartMs: number,
  nowMs = Date.now(),
): number {
  return Math.max(0, (nowMs - raceStartMs) / 60000)
}

export function clampElapsedToDuration(
  elapsedMinutes: number,
  durationMinutes?: number | null,
): number {
  if (durationMinutes != null && durationMinutes > 0) {
    return Math.min(elapsedMinutes, durationMinutes)
  }
  return elapsedMinutes
}

export function resolveRaceFlowAxisMaxMinutes(
  durationMinutes: number | null | undefined,
  recordedMaxMinutes: number,
  currentElapsedMinutes: number | null,
): number {
  if (durationMinutes != null && durationMinutes > 0) {
    return durationMinutes
  }
  return Math.max(recordedMaxMinutes, currentElapsedMinutes ?? 0)
}

export function resolveRaceFlowXAxisMax(
  durationMinutes: number | null | undefined,
  recordedMaxMinutes: number,
  currentElapsedMinutes: number | null,
  showCurrentTime: boolean,
): number | undefined {
  const axisMax = resolveRaceFlowAxisMaxMinutes(
    durationMinutes,
    recordedMaxMinutes,
    currentElapsedMinutes,
  )
  if (durationMinutes != null && durationMinutes > 0) {
    return axisMax
  }
  if (showCurrentTime) {
    return Math.ceil(axisMax * 1.05)
  }
  return undefined
}

export function buildExtrapolationPoint(
  flow: ParticipantFlow,
  currentElapsedMinutes: number,
): FlowPoint | null {
  const lastPoint = flow.points.at(-1)
  if (
    !lastPoint ||
    lastPoint.value <= 0 ||
    currentElapsedMinutes <= lastPoint.elapsedMinutes
  ) {
    return null
  }

  return {
    elapsedMinutes: currentElapsedMinutes,
    value: lastPoint.value,
  }
}

/** Collapse duplicate RFID taps inside the cooldown window (not karaoke). */
const LAP_POINT_MERGE_MINUTES = 1

export type LapPointKind = 'rfid' | 'karaoke'

function isScoredLapRecord(record: TimingRecord): boolean {
  const recordType = record.record_type ?? 'rfid_lap'
  return recordType === 'rfid_lap' || recordType === 'karaoke_bonus'
}

function lapKindForRecord(record: TimingRecord): LapPointKind {
  return record.record_type === 'karaoke_bonus' ? 'karaoke' : 'rfid'
}

function pushLapPoint(
  flow: ParticipantFlow,
  elapsedMinutes: number,
  laps: number,
  kind: LapPointKind,
): void {
  if (flow.points.length === 0) {
    flow.points.push({ elapsedMinutes: 0, value: 0 })
  }

  const last = flow.points.at(-1)!
  const x = Math.max(0, last.elapsedMinutes, elapsedMinutes)

  // Never merge into the gun-start origin — every series must keep (0, 0).
  if (last.elapsedMinutes === 0 && last.value === 0) {
    flow.points.push({ elapsedMinutes: x, value: laps, kind })
    return
  }

  // Karaoke bonuses are always their own plotted point (music note).
  if (kind === 'karaoke' || last.kind === 'karaoke') {
    flow.points.push({ elapsedMinutes: x, value: laps, kind })
    return
  }

  if (x - last.elapsedMinutes <= LAP_POINT_MERGE_MINUTES) {
    last.elapsedMinutes = x
    last.value = Math.max(last.value, laps)
    last.kind = 'rfid'
    return
  }

  flow.points.push({ elapsedMinutes: x, value: laps, kind })
}

/** Build axis-aligned step vertices so Chart.js cannot bezier between taps. */
export function expandSteppedLapPoints(
  points: Array<{ x: number; y: number; kind?: LapPointKind }>,
): Array<{ x: number; y: number; kind?: LapPointKind }> {
  if (points.length === 0) {
    return []
  }

  const out: Array<{ x: number; y: number; kind?: LapPointKind }> = [
    { x: points[0].x, y: points[0].y, kind: points[0].kind },
  ]
  for (let index = 1; index < points.length; index += 1) {
    const prev = points[index - 1]
    const curr = points[index]
    if (curr.x !== prev.x) {
      out.push({ x: curr.x, y: prev.y })
    }
    const last = out.at(-1)!
    if (last.x !== curr.x || last.y !== curr.y) {
      out.push({ x: curr.x, y: curr.y, kind: curr.kind })
    } else if (curr.kind) {
      last.kind = curr.kind
    }
  }
  return out
}

function buildLapFlows(
  records: TimingRecord[],
  raceStartMs: number,
  registeredParticipants?: Array<{
    id: string
    bib_number: string
    first_name: string
    last_name: string
    gender?: string
    age?: number
    status: ParticipantStatus
  }>,
): ParticipantFlow[] {
  const finishRecords = records
    .filter((record) => record.checkpoint?.checkpoint_type === 'finish' && record.participant)
    .filter((record) => isScoredLapRecord(record))
    .filter((record) => new Date(record.timestamp).getTime() >= raceStartMs)
    .sort(
      (a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime(),
    )

  const lapCounts = new Map<string, number>()
  const flows = new Map<string, ParticipantFlow>()

  for (const participant of registeredParticipants ?? []) {
    const flow = createParticipantFlow({
      id: participant.id,
      race_id: '',
      bib_number: participant.bib_number,
      first_name: participant.first_name,
      last_name: participant.last_name,
      gender: participant.gender,
      age: participant.age,
      status: participant.status,
    })
    flow.points.push({ elapsedMinutes: 0, value: 0 })
    flows.set(participant.id, flow)
  }

  for (const record of finishRecords) {
    const participant = record.participant!
    const laps = (lapCounts.get(participant.id) ?? 0) + 1
    lapCounts.set(participant.id, laps)
    const elapsedMinutes =
      (new Date(record.timestamp).getTime() - raceStartMs) / 60000

    const existingFlow = flows.get(participant.id) ?? createParticipantFlow(participant)
    if (!flows.has(participant.id)) {
      existingFlow.points.push({ elapsedMinutes: 0, value: 0 })
    }
    existingFlow.status = participant.status
    existingFlow.gender = participant.gender
    existingFlow.genderKey = getParticipantGenderKey(participant.gender)
    existingFlow.age = participant.age
    existingFlow.ageGroup = getParticipantAgeGroupKey(participant.age)
    existingFlow.lastName = participant.last_name
    pushLapPoint(existingFlow, elapsedMinutes, laps, lapKindForRecord(record))
    flows.set(participant.id, existingFlow)
  }

  return [...flows.values()]
}

function buildDistanceFlows(
  records: TimingRecord[],
  raceStartMs: number,
  unitSystem: UnitSystem,
): ParticipantFlow[] {
  const checkpointRecords = records
    .filter(
      (record) =>
        record.participant &&
        record.checkpoint &&
        DISTANCE_CHECKPOINT_TYPES.has(record.checkpoint.checkpoint_type),
    )
    .filter((record) => new Date(record.timestamp).getTime() >= raceStartMs)
    .sort(
      (a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime(),
    )

  const flows = new Map<string, ParticipantFlow>()

  for (const record of checkpointRecords) {
    const participant = record.participant!
    const elapsedMinutes =
      (new Date(record.timestamp).getTime() - raceStartMs) / 60000
    const distanceKm = record.checkpoint?.distance_from_start_km ?? 0
    const value = convertDistanceFromKm(distanceKm, unitSystem)

    const existingFlow = flows.get(participant.id) ?? createParticipantFlow(participant)
    existingFlow.status = participant.status
    existingFlow.gender = participant.gender
    existingFlow.genderKey = getParticipantGenderKey(participant.gender)
    existingFlow.age = participant.age
    existingFlow.ageGroup = getParticipantAgeGroupKey(participant.age)
    existingFlow.lastName = participant.last_name
    existingFlow.points.push({ elapsedMinutes, value })
    flows.set(participant.id, existingFlow)
  }

  return [...flows.values()]
}

export function buildParticipantFlows(
  records: TimingRecord[],
  raceStartTime?: string,
  raceType: RaceType = 'time_based',
  unitSystem: UnitSystem = 'imperial',
  registeredParticipants?: Array<{
    id: string
    bib_number: string
    first_name: string
    last_name: string
    gender?: string
    age?: number
    status: ParticipantStatus
  }>,
): ParticipantFlow[] {
  const raceStartMs = resolveRaceStartMs(records, raceStartTime)
  if (raceStartMs === null) {
    return []
  }

  if (raceType === 'lap_based') {
    return buildLapFlows(records, raceStartMs, registeredParticipants)
  }

  return buildDistanceFlows(records, raceStartMs, unitSystem)
}

export function getFlowYAxisLabel(
  raceType: RaceType,
  unitSystem: UnitSystem = 'imperial',
): string {
  return raceType === 'lap_based' ? 'Laps' : getDistanceAxisLabel(unitSystem)
}

export function getFlowChartTitle(raceType: RaceType, showCurrentTime: boolean): string {
  const metric = raceType === 'lap_based' ? 'Laps' : 'Distance'
  if (showCurrentTime) {
    return `${metric} over elapsed time (dotted = projected since last tap)`
  }
  return `${metric} over elapsed time`
}

export function formatDuration(seconds: number): string {
  const total = Math.max(0, Math.round(seconds))
  const h = Math.floor(total / 3600)
  const m = Math.floor((total % 3600) / 60)
  const s = total % 60

  const parts: string[] = []
  if (h > 0) {
    parts.push(`${h}h`)
  }
  if (m > 0 || h > 0) {
    parts.push(`${m}m`)
  }
  parts.push(`${s}s`)

  return parts.join(' ')
}

export function getAverageResultLabel(raceType: RaceType): string {
  return raceType === 'lap_based' ? 'Avg laps' : 'Avg finish'
}

export function formatAverageResult(
  raceType: RaceType,
  statistics: Pick<RaceStatistics, 'averageFinishSeconds' | 'averageLaps'>,
): string {
  if (raceType === 'lap_based') {
    if (statistics.averageLaps == null) {
      return '—'
    }

    const rounded = statistics.averageLaps
    return Number.isInteger(rounded) ? String(rounded) : rounded.toFixed(1)
  }

  if (statistics.averageFinishSeconds == null) {
    return '—'
  }

  return formatDuration(statistics.averageFinishSeconds)
}

export function buildRaceStatistics(
  records: TimingRecord[],
  raceStartTime?: string,
  raceType: RaceType = 'time_based',
): RaceStatistics {
  const raceStartMs = resolveRaceStartMs(records, raceStartTime)
  const participants = new Map<
    string,
    { status: string; startMs: number | null; finishMs: number | null; laps: number }
  >()

  for (const record of records) {
    const participant = record.participant
    if (!participant) {
      continue
    }

    const checkpointType = record.checkpoint?.checkpoint_type
    const crossingMs = new Date(record.timestamp).getTime()
    const existing = participants.get(participant.id) ?? {
      status: participant.status,
      startMs: null,
      finishMs: null,
      laps: 0,
    }
    existing.status = participant.status

    if (checkpointType === 'start') {
      existing.startMs = crossingMs
    }

    if (checkpointType === 'finish') {
      existing.finishMs = crossingMs
      existing.laps += 1
    }

    participants.set(participant.id, existing)
  }

  const values = [...participants.values()]
  const finished = values.filter((entry) => entry.status === 'finished').length
  const started = values.filter((entry) => entry.status === 'started').length
  const registered = values.filter((entry) => entry.status === 'registered').length
  const dnf = values.filter((entry) => entry.status === 'dnf').length

  const finishDurations = values
    .filter((entry) => entry.status === 'finished' && entry.startMs != null && entry.finishMs != null)
    .map((entry) => (entry.finishMs! - entry.startMs!) / 1000)

  const fallbackFinishDurations =
    finishDurations.length > 0 || raceStartMs == null
      ? finishDurations
      : values
          .filter((entry) => entry.status === 'finished' && entry.finishMs != null)
          .map((entry) => (entry.finishMs! - raceStartMs) / 1000)

  const lapCounts = values
    .map((entry) => entry.laps)
    .filter((laps) => laps > 0)

  return {
    totalParticipants: values.length,
    finished,
    started,
    registered,
    dnf,
    averageFinishSeconds:
      raceType === 'time_based' && fallbackFinishDurations.length > 0
        ? fallbackFinishDurations.reduce((sum, value) => sum + value, 0) /
          fallbackFinishDurations.length
        : null,
    averageLaps:
      raceType === 'lap_based' && lapCounts.length > 0
        ? lapCounts.reduce((sum, value) => sum + value, 0) / lapCounts.length
        : null,
  }
}
