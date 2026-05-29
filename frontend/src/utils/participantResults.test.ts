import { describe, it, expect } from 'vitest'
import type { LeaderboardEntry } from '@/types/models'
import {
  buildParticipantDetailsMap,
  computeParticipantRanks,
  formatAverageSpeedMph,
  formatCategoryLabel,
  formatCertificateFinishTime,
  formatOrdinal,
  formatRankSummary,
} from '@/utils/participantResults'

describe('participantResults', () => {
  it('formats certificate finish time with tenths', () => {
    expect(formatCertificateFinishTime(7829)).toBe('2:10:29.0')
    expect(formatCertificateFinishTime(45.7)).toBe('0:45.7')
  })

  it('formats ordinals and rank summaries', () => {
    expect(formatOrdinal(1)).toBe('1st')
    expect(formatOrdinal(11)).toBe('11th')
    expect(formatOrdinal(22)).toBe('22nd')
    expect(formatRankSummary({ position: 1, total: 154 })).toBe('1st out of 154')
  })

  it('computes average speed in mph', () => {
    expect(formatAverageSpeedMph(16.0934, 3600)).toBe('10.0')
    expect(formatAverageSpeedMph(null, 7200)).toBeNull()
  })

  it('formats category labels', () => {
    expect(formatCategoryLabel({ gender: 'male', age: 27 })).toBe('M25–29')
    expect(formatCategoryLabel({ gender: 'female', age: 17 })).toBe('FUnder 20')
  })

  it('computes overall, gender, and category ranks', () => {
    const finishedEntries: LeaderboardEntry[] = [
      {
        position: 1,
        participant_id: 'p1',
        bib_number: '7',
        first_name: 'Alex',
        last_name: 'Runner',
        total_time_seconds: 3600,
        status: 'finished',
      },
      {
        position: 2,
        participant_id: 'p2',
        bib_number: '12',
        first_name: 'Sam',
        last_name: 'Trail',
        total_time_seconds: 3900,
        status: 'finished',
      },
      {
        position: 3,
        participant_id: 'p3',
        bib_number: '15',
        first_name: 'Jordan',
        last_name: 'Peak',
        total_time_seconds: 4200,
        status: 'finished',
      },
    ]

    const details = buildParticipantDetailsMap([
      { id: 'p1', gender: 'male', age: 27 },
      { id: 'p2', gender: 'female', age: 28 },
      { id: 'p3', gender: 'male', age: 31 },
    ])

    const ranks = computeParticipantRanks(finishedEntries[0], finishedEntries, details)

    expect(ranks.overall).toEqual({ position: 1, total: 3 })
    expect(ranks.gender).toEqual({ position: 1, total: 2 })
    expect(ranks.category).toEqual({ position: 1, total: 1 })
  })
})
