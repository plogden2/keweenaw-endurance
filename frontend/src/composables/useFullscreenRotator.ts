import { computed, getCurrentInstance, onUnmounted, ref, watch, type Ref } from 'vue'

export type RotatorRaceKey = '12h' | '6h'
export type RotatorMode = 'individuals' | 'teams'

export interface RotatorPageConfig {
  race: RotatorRaceKey
  mode: RotatorMode
  enabled: boolean
}

export interface RotatorSettings {
  dwellMs: number
  pages: RotatorPageConfig[]
}

export interface RotatorRaceSnapshot {
  status?: string
  available: boolean
}

type RaceSnapshots = Record<RotatorRaceKey, RotatorRaceSnapshot>

export const FS_ROTATOR_SETTINGS_KEY = 'event-live-fs-rotator-settings'
export const DEFAULT_ROTATOR_DWELL_MS = 5000

export const DEFAULT_ROTATOR_PAGES: RotatorPageConfig[] = [
  { race: '12h', mode: 'individuals', enabled: true },
  { race: '12h', mode: 'teams', enabled: true },
  { race: '6h', mode: 'individuals', enabled: true },
  { race: '6h', mode: 'teams', enabled: true },
]

function cloneDefaultSettings(): RotatorSettings {
  return {
    dwellMs: DEFAULT_ROTATOR_DWELL_MS,
    pages: DEFAULT_ROTATOR_PAGES.map((p) => ({ ...p })),
  }
}

function isTerminalStatus(status: string | undefined): boolean {
  return status === 'finished' || status === 'cancelled'
}

function normalizeSettings(raw: unknown): RotatorSettings {
  const defaults = cloneDefaultSettings()
  if (!raw || typeof raw !== 'object') return defaults
  const obj = raw as Partial<RotatorSettings>
  const dwellMs =
    typeof obj.dwellMs === 'number' && Number.isFinite(obj.dwellMs) && obj.dwellMs >= 1000
      ? Math.min(120_000, Math.round(obj.dwellMs))
      : defaults.dwellMs

  const byKey = new Map<string, RotatorPageConfig>()
  if (Array.isArray(obj.pages)) {
    for (const page of obj.pages) {
      if (!page || typeof page !== 'object') continue
      if (page.race !== '12h' && page.race !== '6h') continue
      if (page.mode !== 'individuals' && page.mode !== 'teams') continue
      byKey.set(`${page.race}:${page.mode}`, {
        race: page.race,
        mode: page.mode,
        enabled: page.enabled !== false,
      })
    }
  }

  const pages = defaults.pages.map((fallback) => {
    const key = `${fallback.race}:${fallback.mode}`
    return byKey.get(key) ?? { ...fallback }
  })

  // Preserve custom order when all four keys are present; otherwise keep default order.
  if (Array.isArray(obj.pages) && obj.pages.length === 4) {
    const ordered: RotatorPageConfig[] = []
    const seen = new Set<string>()
    for (const page of obj.pages) {
      if (!page || typeof page !== 'object') continue
      const key = `${page.race}:${page.mode}`
      const normalized = byKey.get(key)
      if (!normalized || seen.has(key)) continue
      ordered.push(normalized)
      seen.add(key)
    }
    if (ordered.length === 4) {
      return { dwellMs, pages: ordered }
    }
  }

  return { dwellMs, pages }
}

export function loadRotatorSettings(): RotatorSettings {
  try {
    const raw = sessionStorage.getItem(FS_ROTATOR_SETTINGS_KEY)
    if (raw == null) return cloneDefaultSettings()
    return normalizeSettings(JSON.parse(raw))
  } catch {
    return cloneDefaultSettings()
  }
}

export function saveRotatorSettings(settings: RotatorSettings): void {
  try {
    sessionStorage.setItem(FS_ROTATOR_SETTINGS_KEY, JSON.stringify(settings))
  } catch {
    // ignore storage failures
  }
}

export function pageLabel(page: Pick<RotatorPageConfig, 'race' | 'mode'>): string {
  const race = page.race === '12h' ? '12 Hour' : '6 Hour'
  const mode = page.mode === 'individuals' ? 'Individual' : 'Team'
  return `${race} · ${mode}`
}

export function useFullscreenRotator(opts: {
  open: Ref<boolean>
  races: Ref<RaceSnapshots>
}) {
  const settings = ref<RotatorSettings>(loadRotatorSettings())
  const playing = ref(true)
  const settingsOpen = ref(false)
  const pageIndex = ref(0)
  let advanceTimer: number | undefined

  const activePages = computed(() =>
    settings.value.pages.filter((page) => {
      if (!page.enabled) return false
      const race = opts.races.value[page.race]
      if (!race?.available) return false
      if (isTerminalStatus(race.status)) return false
      return true
    }),
  )

  const activePageKey = computed(() =>
    activePages.value.map((p) => `${p.race}:${p.mode}`).join('|'),
  )

  const currentPage = computed(() => {
    const pages = activePages.value
    if (!pages.length) return null
    const idx = ((pageIndex.value % pages.length) + pages.length) % pages.length
    return pages[idx] ?? null
  })

  function clearAdvanceTimer() {
    if (advanceTimer !== undefined) {
      window.clearTimeout(advanceTimer)
      advanceTimer = undefined
    }
  }

  function clampIndex() {
    const len = activePages.value.length
    if (len === 0) {
      pageIndex.value = 0
      return
    }
    if (pageIndex.value >= len || pageIndex.value < 0) pageIndex.value = 0
  }

  function scheduleAdvance() {
    clearAdvanceTimer()
    if (!opts.open.value || !playing.value || settingsOpen.value) return
    if (activePages.value.length <= 1) return
    advanceTimer = window.setTimeout(() => {
      advanceTimer = undefined
      nextPage()
    }, settings.value.dwellMs)
  }

  function nextPage() {
    const len = activePages.value.length
    if (len === 0) return
    pageIndex.value = (pageIndex.value + 1) % len
    scheduleAdvance()
  }

  function prevPage() {
    const len = activePages.value.length
    if (len === 0) return
    pageIndex.value = (pageIndex.value - 1 + len) % len
    scheduleAdvance()
  }

  /** Jump to a race page for a live lap. No-op when closed, paused, or settings open. */
  function jumpToRace(race: RotatorRaceKey, preferredMode: RotatorMode): boolean {
    if (!opts.open.value || !playing.value || settingsOpen.value) return false
    const pages = activePages.value
    if (!pages.length) return false

    const fallback: RotatorMode = preferredMode === 'teams' ? 'individuals' : 'teams'
    let idx = pages.findIndex((p) => p.race === race && p.mode === preferredMode)
    if (idx < 0) {
      idx = pages.findIndex((p) => p.race === race && p.mode === fallback)
    }
    if (idx < 0) return false

    pageIndex.value = idx
    scheduleAdvance()
    return true
  }

  function togglePlay() {
    playing.value = !playing.value
    if (playing.value) scheduleAdvance()
    else clearAdvanceTimer()
  }

  function openSettings() {
    settingsOpen.value = true
    clearAdvanceTimer()
  }

  function closeSettings() {
    settingsOpen.value = false
    saveRotatorSettings(settings.value)
    clampIndex()
    scheduleAdvance()
  }

  function applySettings(next: RotatorSettings) {
    settings.value = normalizeSettings(next)
    saveRotatorSettings(settings.value)
    clampIndex()
    scheduleAdvance()
  }

  function setDwellSeconds(seconds: number) {
    const dwellMs = Math.min(120, Math.max(1, Math.round(seconds))) * 1000
    settings.value = { ...settings.value, dwellMs }
    saveRotatorSettings(settings.value)
    scheduleAdvance()
  }

  function setPageEnabled(race: RotatorRaceKey, mode: RotatorMode, enabled: boolean) {
    settings.value = {
      ...settings.value,
      pages: settings.value.pages.map((p) =>
        p.race === race && p.mode === mode ? { ...p, enabled } : p,
      ),
    }
    saveRotatorSettings(settings.value)
    clampIndex()
    scheduleAdvance()
  }

  function movePage(race: RotatorRaceKey, mode: RotatorMode, direction: -1 | 1) {
    const pages = settings.value.pages.map((p) => ({ ...p }))
    const idx = pages.findIndex((p) => p.race === race && p.mode === mode)
    if (idx < 0) return
    const next = idx + direction
    if (next < 0 || next >= pages.length) return
    const tmp = pages[idx]!
    pages[idx] = pages[next]!
    pages[next] = tmp
    settings.value = { ...settings.value, pages }
    saveRotatorSettings(settings.value)
    clampIndex()
    scheduleAdvance()
  }

  function onKeydown(e: KeyboardEvent) {
    if (!opts.open.value) return
    if (e.key === 'Escape') {
      if (settingsOpen.value) {
        e.preventDefault()
        closeSettings()
        return
      }
      return
    }
    if (settingsOpen.value) return
    const target = e.target as HTMLElement | null
    if (
      target &&
      (target.tagName === 'INPUT' ||
        target.tagName === 'TEXTAREA' ||
        target.tagName === 'SELECT' ||
        target.isContentEditable)
    ) {
      return
    }
    if (e.key === 'ArrowRight') {
      e.preventDefault()
      nextPage()
    } else if (e.key === 'ArrowLeft') {
      e.preventDefault()
      prevPage()
    }
  }

  watch(
    () => opts.open.value,
    (open) => {
      if (open) {
        settings.value = loadRotatorSettings()
        playing.value = true
        settingsOpen.value = false
        pageIndex.value = 0
        clampIndex()
        scheduleAdvance()
      } else {
        clearAdvanceTimer()
        settingsOpen.value = false
      }
    },
  )

  watch(activePageKey, (next, prev) => {
    if (!opts.open.value) return
    if (next === prev) return
    const prevKeys = new Set((prev || '').split('|').filter(Boolean))
    const nextKeys = new Set((next || '').split('|').filter(Boolean))
    const removed = [...prevKeys].some((k) => !nextKeys.has(k))
    clampIndex()
    // Drop finished/cancelled races immediately and restart dwell.
    if (removed) scheduleAdvance()
  })

  watch(
    () => [playing.value, settingsOpen.value, settings.value.dwellMs] as const,
    () => {
      if (!opts.open.value) return
      scheduleAdvance()
    },
  )

  if (getCurrentInstance()) {
    onUnmounted(() => {
      clearAdvanceTimer()
    })
  }

  return {
    settings,
    playing,
    settingsOpen,
    pageIndex,
    activePages,
    currentPage,
    nextPage,
    prevPage,
    jumpToRace,
    togglePlay,
    openSettings,
    closeSettings,
    applySettings,
    setDwellSeconds,
    setPageEnabled,
    movePage,
    onKeydown,
    scheduleAdvance,
    clearAdvanceTimer,
  }
}
