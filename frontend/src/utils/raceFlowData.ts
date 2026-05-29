import type { TimingRecord } from '@/types/models'

export interface FlowPoint {
  elapsedMinutes: number
  position: number
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

export function buildParticipantFlows(records: TimingRecord[]): ParticipantFlow[] {
  const finishRecords = records
    .filter((record) => record.checkpoint?.checkpoint_type === 'finish' && record.participant)
    .sort(
      (a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime(),
    )

  if (finishRecords.length === 0) {
    return []
  }

  const raceStartMs = finishRecords.reduce((earliest, record) => {
    const ts = new Date(record.timestamp).getTime()
    return ts < earliest ? ts : earliest
  }, Number.POSITIVE_INFINITY)

  const flows: ParticipantFlow[] = finishRecords.map((record, index) => {
    const participant = record.participant!
    const elapsedMinutes =
      (new Date(record.timestamp).getTime() - raceStartMs) / 60000

    return {
      participantId: participant.id,
      label: `#${participant.bib_number} ${participant.first_name}`,
      points: [{ elapsedMinutes, position: index + 1 }],
    }
  })

  return flows
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
