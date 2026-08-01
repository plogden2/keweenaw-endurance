import { describe, it, expect } from 'vitest'
import { liveEventUrl } from './liveEventUrl'

describe('liveEventUrl', () => {
  it('builds the live event path from origin and event id', () => {
    expect(liveEventUrl('https://keweenawendurance.com', 'evt-1')).toBe(
      'https://keweenawendurance.com/events/evt-1/live',
    )
  })

  it('strips a trailing slash on origin', () => {
    expect(liveEventUrl('http://localhost:5173/', 'abc')).toBe(
      'http://localhost:5173/events/abc/live',
    )
  })
})
