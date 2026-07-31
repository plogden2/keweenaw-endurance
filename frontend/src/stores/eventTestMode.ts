import { defineStore } from 'pinia'
import type { LiveLeaderboardEntry } from '@/services/api'
import type { Participant, TimingRecord } from '@/types/models'
import {
  buildTestModeLeaderboard,
  buildTestModeTimingRecords,
  findParticipantByBib,
  findParticipantByTag,
  type TestModeTap,
  type TestModeTapSource,
} from '@/utils/eventTestModeFlow'

export interface TestModeFeedback {
  ok: boolean
  message: string
  participant_name?: string
  bib_number?: string
  lap_count?: number
  source?: TestModeTapSource
}

interface EventTestModeState {
  isOpen: boolean
  eventId: string | null
  roster: Participant[]
  taps: TestModeTap[]
  sessionStartedAt: string | null
  lastFeedback: TestModeFeedback | null
}

function recordFor(
  state: EventTestModeState,
  participant: Participant,
  source: TestModeTapSource,
  recordedAt?: string,
): TestModeFeedback {
  const at = recordedAt || new Date().toISOString()
  state.taps.push({
    participant_id: participant.id,
    recorded_at: at,
    source,
  })
  const laps = state.taps.filter((t) => t.participant_id === participant.id).length
  const feedback: TestModeFeedback = {
    ok: true,
    message: `Test lap recorded`,
    participant_name: `${participant.first_name} ${participant.last_name}`.trim(),
    bib_number: participant.bib_number,
    lap_count: laps,
    source,
  }
  state.lastFeedback = feedback
  return feedback
}

export const useEventTestModeStore = defineStore('eventTestMode', {
  state: (): EventTestModeState => ({
    isOpen: false,
    eventId: null,
    roster: [],
    taps: [],
    sessionStartedAt: null,
    lastFeedback: null,
  }),

  getters: {
    leaderboard(state): LiveLeaderboardEntry[] {
      return buildTestModeLeaderboard(state.taps, state.roster)
    },
    timingRecords(state): TimingRecord[] {
      return buildTestModeTimingRecords(state.taps, state.roster)
    },
    hasTaps(state): boolean {
      return state.taps.length > 0
    },
  },

  actions: {
    isActiveForEvent(eventId: string): boolean {
      return this.isOpen && this.eventId === eventId
    },

    open(eventId: string, roster: Participant[]) {
      this.isOpen = true
      this.eventId = eventId
      this.roster = roster
      this.taps = []
      this.sessionStartedAt = new Date().toISOString()
      this.lastFeedback = null
    },

    close() {
      this.isOpen = false
      this.eventId = null
      this.roster = []
      this.taps = []
      this.sessionStartedAt = null
      this.lastFeedback = null
    },

    recordTagTap(tagUid: string, recordedAt?: string): TestModeFeedback {
      const participant = findParticipantByTag(this.roster, tagUid)
      if (!participant) {
        const feedback: TestModeFeedback = {
          ok: false,
          message: 'Unknown tag',
          source: 'rfid',
        }
        this.lastFeedback = feedback
        return feedback
      }
      return recordFor(this, participant, 'rfid', recordedAt)
    },

    recordBibTap(bib: string, recordedAt?: string): TestModeFeedback {
      const participant = findParticipantByBib(this.roster, bib)
      if (!participant) {
        const feedback: TestModeFeedback = {
          ok: false,
          message: 'Bib not found',
          source: 'manual',
        }
        this.lastFeedback = feedback
        return feedback
      }
      return recordFor(this, participant, 'manual', recordedAt)
    },
  },
})
