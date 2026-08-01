import { describe, it, expect } from 'vitest'
import {
  buildTeamMembershipMap,
  preferredRotatorMode,
  rotatorKeyForRaceId,
} from './rotatorLapJump'

describe('rotatorLapJump', () => {
  it('prefers teams when participant has a team_id', () => {
    const map = buildTeamMembershipMap([
      { id: 'p1', team_id: 'team-a' },
      { id: 'p2', team_id: null },
      { id: 'p3', team_id: '' },
    ])
    expect(preferredRotatorMode('p1', map)).toBe('teams')
    expect(preferredRotatorMode('p2', map)).toBe('individuals')
    expect(preferredRotatorMode('p3', map)).toBe('individuals')
  })

  it('matches participant and race ids by public suffix', () => {
    const map = buildTeamMembershipMap([
      { id: 'ab12cd', team_id: 'team-a' },
    ])
    expect(
      preferredRotatorMode('550e8400-e29b-41d4-a716-446655ab12cd', map),
    ).toBe('teams')

    expect(
      rotatorKeyForRaceId('550e8400-e29b-41d4-a716-446655abcdef', {
        '12h': 'r-12',
        '6h': 'abcdef',
      }),
    ).toBe('6h')
  })

  it('returns null for races outside the rotator', () => {
    expect(
      rotatorKeyForRaceId('r-90', { '12h': 'r-12', '6h': 'r-6' }),
    ).toBeNull()
  })
})
