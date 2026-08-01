import type { RotatorMode, RotatorRaceKey } from '@/composables/useFullscreenRotator'
import { publicIdsMatch } from '@/utils/id'

/** True when the participant is assigned to a cooperative team. */
export function participantHasTeam(
  participantId: string,
  teamByParticipantId: ReadonlyMap<string, boolean>,
): boolean {
  for (const [id, onTeam] of teamByParticipantId) {
    if (publicIdsMatch(id, participantId)) return onTeam
  }
  return false
}

export function preferredRotatorMode(
  participantId: string,
  teamByParticipantId: ReadonlyMap<string, boolean>,
): RotatorMode {
  return participantHasTeam(participantId, teamByParticipantId) ? 'teams' : 'individuals'
}

export function rotatorKeyForRaceId(
  raceId: string,
  races: { '12h'?: string; '6h'?: string },
): RotatorRaceKey | null {
  if (races['12h'] && publicIdsMatch(races['12h'], raceId)) return '12h'
  if (races['6h'] && publicIdsMatch(races['6h'], raceId)) return '6h'
  return null
}

/** Build participant_id → on-team map from roster rows. */
export function buildTeamMembershipMap(
  participants: Array<{ id: string; team_id?: string | null }>,
): Map<string, boolean> {
  const map = new Map<string, boolean>()
  for (const p of participants) {
    const id = String(p.id || '').trim()
    if (!id) continue
    const team = p.team_id != null ? String(p.team_id).trim() : ''
    map.set(id, team.length > 0)
  }
  return map
}
