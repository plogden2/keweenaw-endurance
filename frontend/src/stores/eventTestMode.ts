import { defineStore } from 'pinia'
import type { LiveLeaderboardEntry } from '@/services/api'
import type { BibListItem, Participant, TimingRecord } from '@/types/models'
import {
  buildTestModeLeaderboard,
  buildTestModeTimingRecords,
  findBibByNumber,
  findBibByTag,
  findParticipantByTag,
  findParticipantsByBib,
  participantFromUnassignedBib,
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
  /** Event bib inventory for resolving tags/bibs not assigned to a racer. */
  bibInventory: BibListItem[]
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

function feedbackFor(
  state: EventTestModeState,
  participant: Participant,
  source: TestModeTapSource,
): TestModeFeedback {
  const laps = state.taps.filter((t) => t.participant_id === participant.id).length
  return {
    ok: true,
    message: `Test lap recorded`,
    participant_name: `${participant.first_name} ${participant.last_name}`.trim() || 'Unassigned',
    bib_number: participant.bib_number,
    lap_count: laps,
    source,
  }
}

function recordFor(
  state: EventTestModeState,
  participant: Participant,
  source: TestModeTapSource,
  recordedAt?: string,
): TestModeFeedback {
  const at = recordedAt || new Date().toISOString()
  if (source === 'rfid' && hasNearbyRfidTap(state.taps, participant.id, at)) {
    const feedback = feedbackFor(state, participant, source)
    state.lastFeedback = feedback
    return feedback
  }
  state.taps.push({
    participant_id: participant.id,
    recorded_at: at,
    source,
  })
  const feedback = feedbackFor(state, participant, source)
  state.lastFeedback = feedback
  return feedback
}

export const useEventTestModeStore = defineStore('eventTestMode', {
  state: (): EventTestModeState => ({
    isOpen: false,
    eventId: null,
    roster: [],
    bibInventory: [],
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

    open(eventId: string, roster: Participant[], bibInventory: BibListItem[] = []) {
      this.isOpen = true
      this.eventId = eventId
      this.roster = roster
      this.bibInventory = bibInventory
      this.taps = []
      this.sessionStartedAt = new Date().toISOString()
      this.lastFeedback = null
      this.lastBridgeTapKey = null
    },

    close() {
      this.isOpen = false
      this.eventId = null
      this.roster = []
      this.bibInventory = []
      this.taps = []
      this.sessionStartedAt = null
      this.lastFeedback = null
      this.lastBridgeTapKey = null
    },

    /** Ensure an unassigned bib has a synthetic roster entry; return it. */
    ensureUnassignedBibParticipant(bib: BibListItem): Participant {
      const existing = this.roster.find((p) => p.id === `unassigned-bib:${bib.id}`)
      if (existing) return existing
      const synthetic = participantFromUnassignedBib(bib)
      this.roster.push(synthetic)
      return synthetic
    },

    resolveParticipantForTag(tagUid: string): Participant | undefined {
      const byTag = findParticipantByTag(this.roster, tagUid)
      if (byTag) return byTag
      const bib = findBibByTag(this.bibInventory, tagUid)
      if (!bib || bib.participant_id) return undefined
      return this.ensureUnassignedBibParticipant(bib)
    },

    resolveParticipantForBib(
      bibNumber: string,
      preferredRaceId?: string,
    ): { participant?: Participant; error?: string } {
      let matches = findParticipantsByBib(this.roster, bibNumber)
      if (matches.length > 1 && preferredRaceId) {
        const preferred = matches.filter((p) => p.race_id === preferredRaceId)
        if (preferred.length === 1) matches = preferred
      }
      if (matches.length === 1) return { participant: matches[0] }
      if (matches.length > 1) return { error: 'Multiple matches' }

      const bib = findBibByNumber(this.bibInventory, bibNumber)
      if (bib && !bib.participant_id) {
        return { participant: this.ensureUnassignedBibParticipant(bib) }
      }
      return { error: 'Bib not found' }
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
        const byTag = this.resolveParticipantForTag(uid)
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
        const resolved = this.resolveParticipantForBib(bib, preferredRaceId)
        if (resolved.participant) {
          if (hasNearbyRfidTap(this.taps, resolved.participant.id, at)) {
            return null
          }
          const feedback = recordFor(this, resolved.participant, 'rfid', at)
          return feedback
        }
        const feedback: TestModeFeedback = {
          ok: false,
          message: resolved.error || 'Bib not found',
          source: 'rfid',
        }
        this.lastFeedback = feedback
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
      const participant = this.resolveParticipantForTag(tagUid)
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
      const resolved = this.resolveParticipantForBib(bib, preferredRaceId)
      if (!resolved.participant) {
        const feedback: TestModeFeedback = {
          ok: false,
          message: resolved.error || 'Bib not found',
          source: 'manual',
        }
        this.lastFeedback = feedback
        return feedback
      }
      return recordFor(this, resolved.participant, 'manual', recordedAt)
    },
  },
})
