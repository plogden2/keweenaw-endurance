import { describe, it, expect, vi } from 'vitest'
import {
  buildTeamIdMap,
  findLeaderboardFocusRow,
  leaderboardFocusRootSelector,
  scrollLeaderboardFocusIntoView,
  teamIdForParticipant,
} from './leaderboardFocus'

describe('leaderboardFocus', () => {
  it('scopes focus root to the rotator leaderboard when open', () => {
    expect(
      leaderboardFocusRootSelector({ rotatorOpen: true, activeTab: '12h' }),
    ).toBe('[data-testid="rotator-leaderboard"]')
  })

  it('scopes focus root to the active race panel when rotator is closed', () => {
    expect(
      leaderboardFocusRootSelector({ rotatorOpen: false, activeTab: '6h' }),
    ).toBe('[data-testid="race-panel-6h"]')
  })

  it('finds the participant row inside the given root only', () => {
    const root = document.createElement('div')
    root.innerHTML = `
      <table><tbody>
        <tr data-testid="leaderboard-row" data-participant-id="other"></tr>
        <tr data-testid="leaderboard-row" data-participant-id="p1"></tr>
      </tbody></table>
    `
    const row = findLeaderboardFocusRow(root, {
      participantId: 'p1',
      mode: 'individuals',
    })
    expect(row?.getAttribute('data-participant-id')).toBe('p1')
  })

  it('finds the team row by team id in teams mode', () => {
    const root = document.createElement('div')
    root.innerHTML = `
      <table><tbody>
        <tr data-testid="leaderboard-team-row" data-team-id="team-b"></tr>
        <tr data-testid="leaderboard-team-row" data-team-id="team-a"></tr>
      </tbody></table>
    `
    const row = findLeaderboardFocusRow(root, {
      participantId: 'p1',
      teamId: 'team-a',
      mode: 'teams',
    })
    expect(row?.getAttribute('data-team-id')).toBe('team-a')
  })

  it('retries until the row mounts then scrolls it', async () => {
    const root = document.createElement('div')
    root.innerHTML = '<table><tbody></tbody></table>'
    const tbody = root.querySelector('tbody')!
    const scroll = vi.fn()
    let attempts = 0
    const wait = vi.fn(async () => {
      attempts += 1
      if (attempts === 2) {
        tbody.innerHTML = `<tr data-testid="leaderboard-row" data-participant-id="p1"></tr>`
      }
    })

    const ok = await scrollLeaderboardFocusIntoView(
      () => root,
      { participantId: 'p1', mode: 'individuals' },
      { attempts: 5, wait, scroll },
    )

    expect(ok).toBe(true)
    expect(scroll).toHaveBeenCalledTimes(1)
    expect(wait).toHaveBeenCalled()
  })

  it('resolves team ids from roster with public id suffix match', () => {
    const map = buildTeamIdMap([
      { id: 'ab12cd', team_id: 'team-a' },
      { id: 'p2', team_id: null },
    ])
    expect(teamIdForParticipant('550e8400-e29b-41d4-a716-446655ab12cd', map)).toBe('team-a')
    expect(teamIdForParticipant('p2', map)).toBeUndefined()
  })
})
