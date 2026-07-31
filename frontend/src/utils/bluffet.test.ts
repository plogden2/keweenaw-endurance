import { describe, expect, it } from 'vitest'
import { isBluffetEventId } from './bluffet'

describe('isBluffetEventId', () => {
  it('matches the full UUID', () => {
    expect(isBluffetEventId('1441674d-a011-471a-a601-722b88b117f5')).toBe(true)
  })

  it('matches the full UUID case-insensitively', () => {
    expect(isBluffetEventId('1441674D-A011-471A-A601-722B88B117F5')).toBe(true)
  })

  it('matches the short PublicUUID suffix returned by the API', () => {
    expect(isBluffetEventId('b117f5')).toBe(true)
  })

  it('matches the short suffix case-insensitively', () => {
    expect(isBluffetEventId('B117F5')).toBe(true)
  })

  it('rejects other event uuids', () => {
    expect(isBluffetEventId('aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee')).toBe(false)
  })

  it('rejects other short ids', () => {
    expect(isBluffetEventId('a1b2c3')).toBe(false)
  })

  it('rejects empty/nullish input', () => {
    expect(isBluffetEventId('')).toBe(false)
    expect(isBluffetEventId(null)).toBe(false)
    expect(isBluffetEventId(undefined)).toBe(false)
    expect(isBluffetEventId('   ')).toBe(false)
  })
})
