import { describe, it, expect, vi, beforeEach, type Mock } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { usePinAuthStore } from './pinAuth'
import { useStationStore } from './station'
import { authApi, stationsApi } from '@/services/api'

vi.mock('@/services/api', () => ({
  authApi: {
    loginWithPin: vi.fn(),
  },
  stationsApi: {
    getCurrent: vi.fn(),
    putCurrent: vi.fn(),
  },
  setAuthToken: vi.fn(),
}))

describe('usePinAuthStore', () => {
  beforeEach(() => {
    sessionStorage.clear()
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('logs in with PIN and reports authenticated', async () => {
    const expiresAt = Math.floor(Date.now() / 1000) + 3600
    ;(authApi.loginWithPin as Mock).mockResolvedValue({
      data: { token: 'tok', role: 'admin', expires_at: expiresAt },
    })

    const store = usePinAuthStore()
    expect(store.isAuthenticated).toBe(false)

    await store.loginWithPin('1738')

    expect(authApi.loginWithPin).toHaveBeenCalledWith('1738')
    expect(store.token).toBe('tok')
    expect(store.role).toBe('admin')
    expect(store.expiresAt).toBe(expiresAt)
    expect(store.isAuthenticated).toBe(true)
  })

  it('persists PIN session across store recreation', async () => {
    const expiresAt = Math.floor(Date.now() / 1000) + 3600
    ;(authApi.loginWithPin as Mock).mockResolvedValue({
      data: { token: 'tok', role: 'admin', expires_at: expiresAt },
    })

    const store = usePinAuthStore()
    await store.loginWithPin('1738')
    expect(sessionStorage.getItem('keweenaw-pin-auth')).toBeTruthy()

    setActivePinia(createPinia())
    const restored = usePinAuthStore()
    expect(restored.token).toBe('tok')
    expect(restored.isAuthenticated).toBe(true)
  })

  it('clears state on logout and login failure', async () => {
    const store = usePinAuthStore()
    store.token = 'tok'
    store.role = 'admin'
    store.expiresAt = Math.floor(Date.now() / 1000) + 60
    store.logout()
    expect(store.token).toBeNull()
    expect(store.isAuthenticated).toBe(false)

    ;(authApi.loginWithPin as Mock).mockRejectedValue(new Error('bad pin'))
    await expect(store.loginWithPin('0000')).rejects.toThrow('bad pin')
    expect(store.error).toBe('bad pin')
    expect(store.token).toBeNull()
  })

  it('treats expired or incomplete tokens as unauthenticated', () => {
    const store = usePinAuthStore()
    store.token = 'tok'
    store.expiresAt = Math.floor(Date.now() / 1000) - 10
    expect(store.isAuthenticated).toBe(false)

    store.expiresAt = null
    expect(store.isAuthenticated).toBe(false)

    store.token = null
    store.expiresAt = Math.floor(Date.now() / 1000) + 60
    expect(store.isAuthenticated).toBe(false)
  })

  it('uses fallback message for non-Error login failures', async () => {
    const store = usePinAuthStore()
    ;(authApi.loginWithPin as Mock).mockRejectedValue('denied')
    await expect(store.loginWithPin('0000')).rejects.toBe('denied')
    expect(store.error).toBe('Invalid PIN')
  })
})

describe('useStationStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('fetches and saves current station config', async () => {
    ;(stationsApi.getCurrent as Mock).mockResolvedValue({
      data: {
        event_id: 'evt-1',
        mode: 'finish',
        device_id: 'laptop-1',
        name: 'Finish A',
        checkpoint_id: null,
      },
    })
    ;(stationsApi.putCurrent as Mock).mockResolvedValue({
      data: {
        event_id: 'evt-1',
        mode: 'checkpoint',
        device_id: 'laptop-1',
        name: 'CP1',
        checkpoint_id: 'cp-1',
      },
    })

    const store = useStationStore()
    expect(store.isConfigured).toBe(false)

    await store.fetchCurrent()
    expect(store.eventId).toBe('evt-1')
    expect(store.isConfigured).toBe(true)
    expect(store.currentConfig.device_id).toBe('laptop-1')

    await store.saveCurrent({ mode: 'checkpoint', name: 'CP1', checkpoint_id: 'cp-1' })
    expect(stationsApi.putCurrent).toHaveBeenCalled()
    expect(store.mode).toBe('checkpoint')
    expect(store.name).toBe('CP1')
    expect(store.checkpointId).toBe('cp-1')
  })

  it('applies defaults when station fields are missing', async () => {
    ;(stationsApi.getCurrent as Mock).mockResolvedValue({
      data: { event_id: null },
    })
    ;(stationsApi.putCurrent as Mock).mockResolvedValue({
      data: { event_id: 'evt-2' },
    })

    const store = useStationStore()
    await store.fetchCurrent()
    expect(store.mode).toBe('finish')
    expect(store.deviceId).toBe('')
    expect(store.name).toBe('')
    expect(store.checkpointId).toBeNull()

    await store.saveCurrent()
    expect(store.eventId).toBe('evt-2')
    expect(store.mode).toBe('finish')
    expect(store.deviceId).toBe('')
    expect(store.name).toBe('')
  })

  it('records errors from fetch/save failures', async () => {
    const store = useStationStore()
    ;(stationsApi.getCurrent as Mock).mockRejectedValue(new Error('offline'))
    await expect(store.fetchCurrent()).rejects.toThrow('offline')
    expect(store.error).toBe('offline')

    ;(stationsApi.putCurrent as Mock).mockRejectedValue('nope')
    await expect(store.saveCurrent()).rejects.toBe('nope')
    expect(store.error).toBe('Failed to save station config')
  })
})
