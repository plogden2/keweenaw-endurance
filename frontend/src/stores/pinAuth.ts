import { defineStore } from 'pinia'
import { authApi, setAuthToken } from '@/services/api'
import { getErrorMessage } from '@/utils/error'

const STORAGE_KEY = 'keweenaw-pin-auth'

interface PinAuthState {
  token: string | null
  role: string | null
  expiresAt: number | null
  loading: boolean
  error: string | null
}

function readStoredAuth(): Pick<PinAuthState, 'token' | 'role' | 'expiresAt'> {
  if (typeof sessionStorage === 'undefined') {
    return { token: null, role: null, expiresAt: null }
  }
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY)
    if (!raw) return { token: null, role: null, expiresAt: null }
    const parsed = JSON.parse(raw) as {
      token?: string
      role?: string
      expiresAt?: number
    }
    if (
      parsed.token &&
      parsed.expiresAt &&
      parsed.expiresAt * 1000 > Date.now()
    ) {
      setAuthToken(parsed.token)
      return {
        token: parsed.token,
        role: parsed.role ?? null,
        expiresAt: parsed.expiresAt,
      }
    }
  } catch {
    /* ignore corrupt storage */
  }
  sessionStorage.removeItem(STORAGE_KEY)
  return { token: null, role: null, expiresAt: null }
}

function persistAuth(token: string | null, role: string | null, expiresAt: number | null) {
  if (typeof sessionStorage === 'undefined') return
  if (token && expiresAt) {
    sessionStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({ token, role, expiresAt }),
    )
  } else {
    sessionStorage.removeItem(STORAGE_KEY)
  }
}

export const usePinAuthStore = defineStore('pinAuth', {
  state: (): PinAuthState => {
    const stored = readStoredAuth()
    return {
      token: stored.token,
      role: stored.role,
      expiresAt: stored.expiresAt,
      loading: false,
      error: null,
    }
  },

  getters: {
    isAuthenticated: (state): boolean =>
      Boolean(state.token && state.expiresAt && state.expiresAt * 1000 > Date.now()),
  },

  actions: {
    async loginWithPin(pin: string) {
      this.loading = true
      this.error = null
      try {
        const { data } = await authApi.loginWithPin(pin)
        this.token = data.token
        this.role = data.role
        this.expiresAt = data.expires_at
        setAuthToken(data.token)
        persistAuth(data.token, data.role, data.expires_at)
        this.loading = false
        return data
      } catch (err) {
        this.token = null
        this.role = null
        this.expiresAt = null
        setAuthToken(null)
        persistAuth(null, null, null)
        this.error = getErrorMessage(err, 'Invalid PIN')
        this.loading = false
        throw err
      }
    },

    logout() {
      this.token = null
      this.role = null
      this.expiresAt = null
      this.error = null
      setAuthToken(null)
      persistAuth(null, null, null)
    },
  },
})
