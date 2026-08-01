import { nextTick } from 'vue'
import { publicIdsMatch } from '@/utils/id'

export type LeaderboardFocusMode = 'individuals' | 'teams'

export interface LeaderboardFocusTarget {
  participantId: string
  teamId?: string
  mode: LeaderboardFocusMode
}

/** Prefer the visible rotator board; otherwise the active live race panel. */
export function leaderboardFocusRootSelector(opts: {
  rotatorOpen: boolean
  activeTab: '12h' | '6h' | '90m' | 'overlap'
}): string {
  if (opts.rotatorOpen) return '[data-testid="rotator-leaderboard"]'
  if (opts.activeTab === '12h') return '[data-testid="race-panel-12h"]'
  if (opts.activeTab === '6h') return '[data-testid="race-panel-6h"]'
  if (opts.activeTab === '90m') return '[data-testid="race-panel-90m"]'
  return '[data-testid="live-view"]'
}

export function findLeaderboardFocusRow(
  root: ParentNode | null | undefined,
  target: LeaderboardFocusTarget,
): HTMLElement | null {
  if (!root) return null
  if (target.mode === 'teams') {
    if (!target.teamId) return null
    const rows = root.querySelectorAll<HTMLElement>('[data-testid="leaderboard-team-row"]')
    for (const row of rows) {
      const id = row.getAttribute('data-team-id')
      if (id && publicIdsMatch(id, target.teamId)) return row
    }
    return null
  }
  const rows = root.querySelectorAll<HTMLElement>('[data-testid="leaderboard-row"]')
  for (const row of rows) {
    const id = row.getAttribute('data-participant-id')
    if (id && publicIdsMatch(id, target.participantId)) return row
  }
  return null
}

/**
 * Wait for the focused leaderboard row to mount (e.g. after a rotator race jump)
 * then scroll it into view. Returns true when a row was scrolled.
 */
export async function scrollLeaderboardFocusIntoView(
  resolveRoot: () => ParentNode | null | undefined,
  target: LeaderboardFocusTarget,
  opts?: {
    attempts?: number
    wait?: () => Promise<void>
    scroll?: (el: HTMLElement) => void
  },
): Promise<boolean> {
  const attempts = opts?.attempts ?? 16
  const wait =
    opts?.wait ??
    (async () => {
      await nextTick()
      await new Promise<void>((resolve) => {
        if (typeof requestAnimationFrame === 'function') {
          requestAnimationFrame(() => resolve())
        } else {
          setTimeout(resolve, 0)
        }
      })
    })
  const scroll =
    opts?.scroll ??
    ((el: HTMLElement) => {
      if (typeof el.scrollIntoView === 'function') {
        el.scrollIntoView({ block: 'center', behavior: 'smooth' })
      }
    })

  for (let i = 0; i < attempts; i++) {
    const row = findLeaderboardFocusRow(resolveRoot(), target)
    if (row) {
      scroll(row)
      return true
    }
    await wait()
  }
  return false
}

/** Look up a participant's team id from a roster map (supports public id suffixes). */
export function teamIdForParticipant(
  participantId: string,
  teamIdByParticipantId: ReadonlyMap<string, string>,
): string | undefined {
  for (const [id, teamId] of teamIdByParticipantId) {
    if (publicIdsMatch(id, participantId)) return teamId
  }
  return undefined
}

/** Build participant_id → team_id for non-empty team assignments. */
export function buildTeamIdMap(
  participants: Array<{ id: string; team_id?: string | null }>,
): Map<string, string> {
  const map = new Map<string, string>()
  for (const p of participants) {
    const id = String(p.id || '').trim()
    if (!id) continue
    const team = p.team_id != null ? String(p.team_id).trim() : ''
    if (team) map.set(id, team)
  }
  return map
}
