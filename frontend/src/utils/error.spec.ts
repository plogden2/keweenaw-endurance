import { describe, expect, it } from 'vitest'
import { getErrorMessage } from './error'

describe('getErrorMessage', () => {
  it('prefers axios response.data.error and includes status', () => {
    const err = Object.assign(new Error('Request failed with status code 503'), {
      isAxiosError: true,
      response: {
        status: 503,
        data: { error: 'bridge unavailable' },
      },
    })

    expect(getErrorMessage(err, 'Failed to write tag')).toBe('bridge unavailable (503)')
  })

  it('uses axios status message when body is empty', () => {
    const err = Object.assign(new Error('Request failed with status code 503'), {
      isAxiosError: true,
      response: {
        status: 503,
        data: '',
      },
    })

    expect(getErrorMessage(err, 'Failed to write tag')).toMatch(/503/)
  })

  it('falls back for non-Error values', () => {
    expect(getErrorMessage(null, 'Failed to write tag')).toBe('Failed to write tag')
  })
})
