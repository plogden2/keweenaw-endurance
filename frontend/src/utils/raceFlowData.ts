import type { RaceType, TimingRecord } from '@/types/models'

export interface FlowPoint {
  elapsedMinutes: number
  value: number
}

export interface ParticipantFlow {
  participantId: string
  label: string
  points: FlowPoint[]
}

export interface RaceStatistics {
  totalParticipants: number
  finished: number
  started: number
  registered: number
  dnf: number
  averageFinishMinutes: number | null
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

export function buildExtrapolationPoint(
  flow: ParticipantFlow,
  currentElapsedMinutes: number,
): FlowPoint | null {
  const lastPoint = flow.points.at(-1)
  if (!lastPoint || currentElapsedMinutes <= lastPoint.elapsedMinutes) {
    return null
  }

  return {
    elapsedMinutes: currentElapsedMinutes,
    value: lastPoint.value,
  }
}

function buildLapFlows(records: TimingRecord[], raceStartMs: number): ParticipantFlow[] {
  const finishRecords = records
    .filter((record) => record.checkpoint?.checkpoint_type === 'finish' && record.participant)
    .sort(
      (a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime(),
    )

  const lapCounts = new Map<string, number>()
  const flows = new Map<string, ParticipantFlow>()

  for (const record of finishRecords) {
    const participant = record.participant!
    const laps = (lapCounts.get(participant.id) ?? 0) + 1
    lapCounts.set(participant.id, laps)
    const elapsedMinutes =
      (new Date(record.timestamp).getTime() - raceStartMs) / 60000

    const existingFlow = flows.get(participant.id) ?? {
      participantId: participant.id,
      label: `#${participant.bib_number} ${participant.first_name}`,
      points: [],
    }
    existingFlow.points.push({ elapsedMinutes, value: laps })
    flows.set(participant.id, existingFlow)
  }

  return [...flows.values()]
}

function buildDistanceFlows(records: TimingRecord[], raceStartMs: number): ParticipantFlow[] {
  const checkpointRecords = records
    .filter(
      (record) =>
        record.participant &&
        record.checkpoint &&
        DISTANCE_CHECKPOINT_TYPES.has(record.checkpoint.checkpoint_type),
    )
    .sort(
      (a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime(),
    )

  const flows = new Map<string, ParticipantFlow>()

  for (const record of checkpointRecords) {
    const participant = record.participant!
    const elapsedMinutes =
      (new Date(record.timestamp).getTime() - raceStartMs) / 60000
    const value = record.checkpoint?.distance_from_start_km ?? 0

    const existingFlow = flows.get(participant.id) ?? {
      participantId: participant.id,
      label: `#${participant.bib_number} ${participant.first_name}`,
      points: [],
    }
    existingFlow.points.push({ elapsedMinutes, value })
    flows.set(participant.id, existingFlow)
  }

  return [...flows.values()]
}

export function buildParticipantFlows(
  records: TimingRecord[],
  raceStartTime?: string,
  raceType: RaceType = 'time_based',
): ParticipantFlow[] {
  const raceStartMs = resolveRaceStartMs(records, raceStartTime)
  if (raceStartMs === null) {
    return []
  }

  if (raceType === 'lap_based') {
    return buildLapFlows(records, raceStartMs)
  }

  return buildDistanceFlows(records, raceStartMs)
}

export function getFlowYAxisLabel(raceType: RaceType): string {
  return raceType === 'lap_based' ? 'Laps' : 'Distance (km)'
}

export function getFlowChartTitle(raceType: RaceType, showCurrentTime: boolean): string {
  const metric = raceType === 'lap_based' ? 'Laps' : 'Distance'
  if (showCurrentTime) {
    return `${metric} over elapsed time (dotted = projected since last tap)`
  }
  return `${metric} over elapsed time`
}

export function buildRaceStatistics(records: TimingRecord[]): RaceStatistics {
  const participants = new Map<
    string,
    { status: string; finishMinutes: number | null }
  >()

  for (const record of records) {
    const participant = record.participant
    if (!participant) {
      continue
    }

    const existing = participants.get(participant.id) ?? {
      status: participant.status,
      finishMinutes: null,
    }
    existing.status = participant.status

    if (record.checkpoint?.checkpoint_type === 'finish') {
      existing.finishMinutes = new Date(record.timestamp).getTime() / 60000
    }

    participants.set(participant.id, existing)
  }

  const values = [...participants.values()]
  const finished = values.filter((entry) => entry.status === 'finished').length
  const started = values.filter((entry) => entry.status === 'started').length
  const registered = values.filter((entry) => entry.status === 'registered').length
  const dnf = values.filter((entry) => entry.status === 'dnf').length
  const finishTimes = values
    .map((entry) => entry.finishMinutes)
    .filter((value): value is number => value !== null)

  return {
    totalParticipants: values.length,
    finished,
    started,
    registered,
    dnf,
    averageFinishMinutes:
      finishTimes.length > 0
        ? finishTimes.reduce((sum, value) => sum + value, 0) / finishTimes.length
        : null,
  }
}
