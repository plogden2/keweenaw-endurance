import { describe, expect, it } from 'vitest'
import { formatShortId, publicIdsMatch } from './id'

describe('formatShortId', () => {
  it('returns last six characters of a canonical UUID', () => {
    expect(formatShortId('550e8400-e29b-41d4-a716-446655440000')).toBe('440000')
  })

  it('returns short ids unchanged', () => {
    expect(formatShortId('a1b2c3')).toBe('a1b2c3')
  })
})

describe('publicIdsMatch', () => {
  it('matches short id to full UUID suffix', () => {
    expect(publicIdsMatch('440000', '550e8400-e29b-41d4-a716-446655440000')).toBe(true)
    expect(publicIdsMatch('550e8400-e29b-41d4-a716-446655440000', '440000')).toBe(true)
  })

  it('rejects different ids', () => {
    expect(publicIdsMatch('440000', '440001')).toBe(false)
    expect(publicIdsMatch('', '440000')).toBe(false)
  })
})
