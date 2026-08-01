import { beforeEach, describe, expect, it, vi } from 'vitest'
import { rfidApi } from '@/services/api'
import { playCoinSound } from './playCoinSound'

const playMock = vi.fn().mockResolvedValue(undefined)

vi.stubGlobal(
  'Audio',
  vi.fn(function AudioMock(this: { play: typeof playMock; src: string }) {
    this.play = playMock
    this.src = ''
    return this
  }),
)

vi.mock('@/services/api', async () => {
  const actual = await vi.importActual<typeof import('@/services/api')>('@/services/api')
  return {
    ...actual,
    rfidApi: {
      ...actual.rfidApi,
      playLocalBridgeBeep: vi.fn().mockResolvedValue(false),
    },
  }
})

describe('playCoinSound', () => {
  beforeEach(() => {
    playMock.mockClear()
    vi.mocked(rfidApi.playLocalBridgeBeep).mockReset()
    vi.mocked(rfidApi.playLocalBridgeBeep).mockResolvedValue(false)
  })

  it('uses bridge beep when bridgeOnline is true', () => {
    vi.mocked(rfidApi.playLocalBridgeBeep).mockResolvedValue(true)
    playCoinSound({ bridgeOnline: true })
    expect(rfidApi.playLocalBridgeBeep).toHaveBeenCalled()
    expect(playMock).not.toHaveBeenCalled()
  })

  it('uses browser audio when bridgeOnline is false', () => {
    playCoinSound({ bridgeOnline: false })
    expect(rfidApi.playLocalBridgeBeep).not.toHaveBeenCalled()
    expect(playMock).toHaveBeenCalled()
  })

  it('falls back to browser audio when bridge beep fails', async () => {
    vi.mocked(rfidApi.playLocalBridgeBeep).mockResolvedValue(false)
    playCoinSound()
    await vi.waitFor(() => {
      expect(playMock).toHaveBeenCalled()
    })
  })
})
