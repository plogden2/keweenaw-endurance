import { defineStore } from 'pinia'
import type { LiveLeaderboardEntry } from '@/services/api'
import type { Participant, TimingRecord } from '@/types/models'
import {
  buildTestModeLeaderboard,
  buildTestModeTimingRecords,
  findParticipantByTag,
  findParticipantsByBib,
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

/** Snapshot of the local Proxmark bridge last-tap fields used for test-mode ingest. */
export interface LocalBridgeTapSnapshot {
  last_tap_uuid?: string
  last_tap_at?: string | null
  last_tap_bib?: string
  last_tap_race_id?: string
}

interface EventTestModeState {
  isOpen: boolean
  eventId: string | null
  roster: Participant[]
  taps: TestModeTap[]
  sessionStartedAt: string | null
  lastFeedback: TestModeFeedback | null
  /** Dedupe key for the most recently ingested local-bridge tap. */
  lastBridgeTapKey: string | null
}

function bridgeTapKey(snap: LocalBridgeTapSnapshot): string | null {
  const at = snap.last_tap_at?.trim()
  if (!at) return null
  const id = (snap.last_tap_uuid || snap.last_tap_bib || '').trim().toLowerCase()
  if (!id) return null
  return `${id}|${at}`
}

/** True when an RFID tap for this participant was already stored near `at`. */
function hasNearbyRfidTap(
  taps: TestModeTap[],
  participantId: string,
  at: string,
  windowMs = 500,
): boolean {
  const target = Date.parse(at)
  if (Number.isNaN(target)) return false
  return taps.some((tap) => {
    if (tap.participant_id !== participantId || tap.source !== 'rfid') return false
    const ts = Date.parse(tap.recorded_at)
    return !Number.isNaN(ts) && Math.abs(ts - target) <= windowMs
  })
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
    lastBridgeTapKey: null,
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
      this.lastBridgeTapKey = null
    },

    close() {
      this.isOpen = false
      this.eventId = null
      this.roster = []
      this.taps = []
      this.sessionStartedAt = null
      this.lastFeedback = null
      this.lastBridgeTapKey = null
    },

    /** Mark the current bridge last-tap as already seen so it is not ingested. */
    noteBridgeTapBaseline(snap: LocalBridgeTapSnapshot) {
      this.lastBridgeTapKey = bridgeTapKey(snap)
    },

    /**
     * Ingest a tap from the on-laptop Reader bridge `/status` poll.
     * Returns null when the snapshot is empty or already ingested.
     */
    ingestLocalBridgeTap(snap: LocalBridgeTapSnapshot): TestModeFeedback | null {
      if (!this.isOpen) return null
      const key = bridgeTapKey(snap)
      if (!key || key === this.lastBridgeTapKey) return null

      const at = snap.last_tap_at!.trim()
      const uid = snap.last_tap_uuid?.trim()
      if (uid) {
        const byTag = findParticipantByTag(this.roster, uid)
        if (byTag) {
          this.lastBridgeTapKey = key
          if (hasNearbyRfidTap(this.taps, byTag.id, at)) {
            return null
          }
          return recordFor(this, byTag, 'rfid', at)
        }
      }

      // Always advance the dedupe key so a failed resolve does not retry-spam.
      this.lastBridgeTapKey = key

      const bib = snap.last_tap_bib?.trim()
      if (bib) {
        const preferredRaceId = snap.last_tap_race_id?.trim() || undefined
        let matches = findParticipantsByBib(this.roster, bib)
        if (matches.length > 1 && preferredRaceId) {
          const preferred = matches.filter((p) => p.race_id === preferredRaceId)
          if (preferred.length === 1) matches = preferred
        }
        if (matches.length === 1 && hasNearbyRfidTap(this.taps, matches[0].id, at)) {
          return null
        }
        const feedback = this.recordBibTap(bib, at, preferredRaceId)
        if (feedback.ok) {
          // recordBibTap sets source 'manual'; keep RFID semantics for bridge.
          const last = this.taps.at(-1)
          if (last) last.source = 'rfid'
          feedback.source = 'rfid'
          this.lastFeedback = feedback
        }
        return feedback
      }

      const feedback: TestModeFeedback = {
        ok: false,
        message: 'Unknown tag',
        source: 'rfid',
      }
      this.lastFeedback = feedback
      return feedback
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

    recordBibTap(bib: string, recordedAt?: string, preferredRaceId?: string): TestModeFeedback {
      let matches = findParticipantsByBib(this.roster, bib)
      if (matches.length === 0) {
        const feedback: TestModeFeedback = {
          ok: false,
          message: 'Bib not found',
          source: 'manual',
        }
        this.lastFeedback = feedback
        return feedback
      }
      if (matches.length > 1 && preferredRaceId) {
        const preferred = matches.filter((p) => p.race_id === preferredRaceId)
        if (preferred.length === 1) {
          matches = preferred
        }
      }
      if (matches.length > 1) {
        const feedback: TestModeFeedback = {
          ok: false,
          message: 'Multiple matches',
          source: 'manual',
        }
        this.lastFeedback = feedback
        return feedback
      }
      return recordFor(this, matches[0], 'manual', recordedAt)
    },
  },
})
