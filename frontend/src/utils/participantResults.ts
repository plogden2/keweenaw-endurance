import type { LeaderboardEntry, Participant } from '@/types/models'
import {
  getParticipantAgeGroupKey,
  getParticipantAgeGroupLabel,
  getParticipantGenderKey,
} from '@/utils/raceFlowData'
import { KM_TO_MILES } from '@/utils/units'

export interface ParticipantRankInfo {
  position: number
  total: number
}

export interface ParticipantResultRanks {
  overall: ParticipantRankInfo
  gender: ParticipantRankInfo | null
  category: ParticipantRankInfo | null
}

export function formatOrdinal(value: number): string {
  const remainder = value % 100
  if (remainder >= 11 && remainder <= 13) {
    return `${value}th`
  }

  switch (value % 10) {
    case 1:
      return `${value}st`
    case 2:
      return `${value}nd`
    case 3:
      return `${value}rd`
    default:
      return `${value}th`
  }
}

export function formatRankSummary(rank: ParticipantRankInfo): string {
  return `${formatOrdinal(rank.position)} out of ${rank.total}`
}

export function formatCertificateFinishTime(seconds: number): string {
  const totalTenths = Math.max(0, Math.round(seconds * 10))
  const totalSeconds = Math.floor(totalTenths / 10)
  const tenths = totalTenths % 10
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const secs = totalSeconds % 60

  const timeParts = [
    String(minutes).padStart(2, '0'),
    `${String(secs).padStart(2, '0')}.${tenths}`,
  ]

  if (hours > 0) {
    return `${hours}:${timeParts.join(':')}`
  }

  return `${minutes}:${String(secs).padStart(2, '0')}.${tenths}`
}

export function formatAverageSpeedMph(
  distanceKm: number | undefined | null,
  totalTimeSeconds: number,
): string | null {
  if (distanceKm == null || distanceKm <= 0 || totalTimeSeconds <= 0) {
    return null
  }

  const miles = distanceKm * KM_TO_MILES
  const hours = totalTimeSeconds / 3600
  const mph = miles / hours

  return mph.toFixed(1)
}

export function formatCategoryLabel(participant: Pick<Participant, 'gender' | 'age'>): string {
  const genderKey = getParticipantGenderKey(participant.gender)
  const genderPrefix =
    genderKey === 'male' ? 'M' : genderKey === 'female' ? 'F' : genderKey === 'other' ? 'O' : ''
  const ageGroupKey = getParticipantAgeGroupKey(participant.age)

  if (ageGroupKey === 'unknown') {
    return genderPrefix || '—'
  }

  if (ageGroupKey === 'under-20') {
    return `${genderPrefix}Under 20`
  }

  if (ageGroupKey === '80-plus') {
    return `${genderPrefix}80+`
  }

  return `${genderPrefix}${ageGroupKey.replace('-', '–')}`
}

export function formatEventDate(dateValue: string | undefined): string {
  if (!dateValue) {
    return '—'
  }

  const date = new Date(`${dateValue}T00:00:00`)
  if (Number.isNaN(date.getTime())) {
    return dateValue
  }

  return date.toLocaleDateString('en-US', {
    month: '2-digit',
    day: '2-digit',
    year: 'numeric',
  })
}

function rankWithinGroup(
  entries: LeaderboardEntry[],
  participantId: string,
): ParticipantRankInfo | null {
  const sorted = [...entries].sort(
    (a, b) => a.total_time_seconds - b.total_time_seconds || a.position - b.position,
  )
  const index = sorted.findIndex((entry) => entry.participant_id === participantId)

  if (index === -1) {
    return null
  }

  return {
    position: index + 1,
    total: sorted.length,
  }
}

export function computeParticipantRanks(
  entry: LeaderboardEntry,
  finishedEntries: LeaderboardEntry[],
  participantDetails: Map<string, Pick<Participant, 'gender' | 'age'>>,
): ParticipantResultRanks {
  const participant = participantDetails.get(entry.participant_id)
  const overall = rankWithinGroup(finishedEntries, entry.participant_id) ?? {
    position: entry.position,
    total: finishedEntries.length,
  }

  if (!participant) {
    return {
      overall,
      gender: null,
      category: null,
    }
  }

  const genderKey = getParticipantGenderKey(participant.gender)
  const ageGroupKey = getParticipantAgeGroupKey(participant.age)

  const genderEntries = finishedEntries.filter((finishedEntry) => {
    const details = participantDetails.get(finishedEntry.participant_id)
    return details && getParticipantGenderKey(details.gender) === genderKey
  })

  const categoryEntries = finishedEntries.filter((finishedEntry) => {
    const details = participantDetails.get(finishedEntry.participant_id)
    return (
      details &&
      getParticipantGenderKey(details.gender) === genderKey &&
      getParticipantAgeGroupKey(details.age) === ageGroupKey
    )
  })

  return {
    overall,
    gender: rankWithinGroup(genderEntries, entry.participant_id),
    category: rankWithinGroup(categoryEntries, entry.participant_id),
  }
}

export function buildParticipantDetailsMap(
  participants: Array<Pick<Participant, 'id' | 'gender' | 'age'>>,
): Map<string, Pick<Participant, 'gender' | 'age'>> {
  return new Map(
    participants.map((participant) => [
      participant.id,
      { gender: participant.gender, age: participant.age },
    ]),
  )
}

export function getGenderRankLabel(gender?: string): string {
  const genderKey = getParticipantGenderKey(gender)
  if (genderKey === 'male') {
    return 'Male Rank'
  }
  if (genderKey === 'female') {
    return 'Female Rank'
  }
  return 'Gender Rank'
}

export function getCategoryRankLabel(participant: Pick<Participant, 'gender' | 'age'>): string {
  return `${getParticipantAgeGroupLabel(getParticipantAgeGroupKey(participant.age))} Rank`
}
