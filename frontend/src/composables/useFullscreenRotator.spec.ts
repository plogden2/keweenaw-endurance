import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { nextTick, ref } from 'vue'
import {
  DEFAULT_ROTATOR_DWELL_MS,
  FS_ROTATOR_SETTINGS_KEY,
  loadRotatorSettings,
  pageLabel,
  useFullscreenRotator,
  type RotatorRaceSnapshot,
} from './useFullscreenRotator'

describe('useFullscreenRotator', () => {
  beforeEach(() => {
    sessionStorage.clear()
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  function setup(
    raceOverrides: Partial<Record<'12h' | '6h', RotatorRaceSnapshot>> = {},
  ) {
    const open = ref(false)
    const races = ref({
      '12h': { available: true, status: 'active', ...raceOverrides['12h'] },
      '6h': { available: true, status: 'active', ...raceOverrides['6h'] },
    })
    const api = useFullscreenRotator({ open, races })
    return { open, races, ...api }
  }

  it('defaults to 12h/6h individual+team pages at 5s dwell', () => {
    const settings = loadRotatorSettings()
    expect(settings.dwellMs).toBe(DEFAULT_ROTATOR_DWELL_MS)
    expect(settings.pages.map((p) => `${p.race}:${p.mode}`)).toEqual([
      '12h:individuals',
      '12h:teams',
      '6h:individuals',
      '6h:teams',
    ])
    expect(settings.pages.every((p) => p.enabled)).toBe(true)
  })

  it('auto-advances while open and playing', async () => {
    const { open, currentPage, pageIndex } = setup()
    open.value = true
    await nextTick()
    expect(pageLabel(currentPage.value!)).toBe('12 Hour · Individual')

    vi.advanceTimersByTime(DEFAULT_ROTATOR_DWELL_MS)
    await nextTick()
    expect(pageIndex.value).toBe(1)
    expect(pageLabel(currentPage.value!)).toBe('12 Hour · Team')
  })

  it('pause stops auto-advance; play resumes', async () => {
    const { open, togglePlay, pageIndex, playing } = setup()
    open.value = true
    await nextTick()
    togglePlay()
    expect(playing.value).toBe(false)
    vi.advanceTimersByTime(DEFAULT_ROTATOR_DWELL_MS * 2)
    expect(pageIndex.value).toBe(0)
    togglePlay()
    expect(playing.value).toBe(true)
    vi.advanceTimersByTime(DEFAULT_ROTATOR_DWELL_MS)
    expect(pageIndex.value).toBe(1)
  })

  it('arrow keys step through active pages', async () => {
    const { open, onKeydown, pageIndex } = setup()
    open.value = true
    await nextTick()
    onKeydown(new KeyboardEvent('keydown', { key: 'ArrowRight' }))
    expect(pageIndex.value).toBe(1)
    onKeydown(new KeyboardEvent('keydown', { key: 'ArrowLeft' }))
    expect(pageIndex.value).toBe(0)
  })

  it('drops finished and cancelled 6h pages from the cycle', async () => {
    const { open, races, activePages, currentPage, pageIndex } = setup()
    open.value = true
    await nextTick()
    pageIndex.value = 2
    await nextTick()
    expect(currentPage.value?.race).toBe('6h')

    races.value = {
      ...races.value,
      '6h': { available: true, status: 'finished' },
    }
    await nextTick()
    expect(activePages.value.every((p) => p.race !== '6h')).toBe(true)
    expect(activePages.value).toHaveLength(2)
    expect(currentPage.value?.race).toBe('12h')
  })

  it('persists settings to sessionStorage', async () => {
    const { open, setDwellSeconds, setPageEnabled } = setup()
    open.value = true
    await nextTick()
    setDwellSeconds(8)
    setPageEnabled('6h', 'teams', false)
    const stored = JSON.parse(sessionStorage.getItem(FS_ROTATOR_SETTINGS_KEY)!)
    expect(stored.dwellMs).toBe(8000)
    expect(stored.pages.find((p: { race: string; mode: string }) => p.race === '6h' && p.mode === 'teams').enabled).toBe(
      false,
    )
  })

  it('Escape closes settings before exiting rotator responsibility', async () => {
    const { open, openSettings, settingsOpen, onKeydown } = setup()
    open.value = true
    await nextTick()
    openSettings()
    expect(settingsOpen.value).toBe(true)
    const ev = new KeyboardEvent('keydown', { key: 'Escape', cancelable: true })
    onKeydown(ev)
    expect(settingsOpen.value).toBe(false)
    expect(ev.defaultPrevented).toBe(true)
  })

  describe('jumpToRace', () => {
    it('jumps to preferred mode for the race and resets dwell', async () => {
      const { open, jumpToRace, pageIndex, currentPage } = setup()
      open.value = true
      await nextTick()
      pageIndex.value = 0
      await nextTick()

      expect(jumpToRace('6h', 'teams')).toBe(true)
      expect(currentPage.value).toEqual({ race: '6h', mode: 'teams', enabled: true })
      expect(pageIndex.value).toBe(3)

      vi.advanceTimersByTime(DEFAULT_ROTATOR_DWELL_MS - 1)
      expect(pageIndex.value).toBe(3)
      vi.advanceTimersByTime(1)
      await nextTick()
      expect(pageIndex.value).toBe(0)
    })

    it('falls back to individuals when teams is preferred but disabled', async () => {
      const { open, jumpToRace, setPageEnabled, currentPage } = setup()
      open.value = true
      await nextTick()
      setPageEnabled('12h', 'teams', false)

      expect(jumpToRace('12h', 'teams')).toBe(true)
      expect(currentPage.value?.race).toBe('12h')
      expect(currentPage.value?.mode).toBe('individuals')
    })

    it('switches off the team page when preferred mode is individuals', async () => {
      const { open, jumpToRace, pageIndex, currentPage } = setup()
      open.value = true
      await nextTick()
      pageIndex.value = 1 // 12h teams
      await nextTick()
      expect(currentPage.value?.mode).toBe('teams')

      expect(jumpToRace('12h', 'individuals')).toBe(true)
      expect(currentPage.value).toEqual({ race: '12h', mode: 'individuals', enabled: true })
    })

    it('does not fall back to teams when individuals is preferred but disabled', async () => {
      const { open, jumpToRace, setPageEnabled, pageIndex, currentPage } = setup()
      open.value = true
      await nextTick()
      setPageEnabled('12h', 'individuals', false)
      await nextTick()
      // Active: 12h teams, 6h individuals, 6h teams
      pageIndex.value = 0
      await nextTick()
      expect(currentPage.value).toEqual({ race: '12h', mode: 'teams', enabled: true })

      expect(jumpToRace('12h', 'individuals')).toBe(false)
      expect(currentPage.value?.mode).toBe('teams')
    })

    it('does not jump when paused', async () => {
      const { open, jumpToRace, togglePlay, pageIndex, currentPage } = setup()
      open.value = true
      await nextTick()
      togglePlay()
      expect(jumpToRace('6h', 'individuals')).toBe(false)
      expect(pageIndex.value).toBe(0)
      expect(currentPage.value?.race).toBe('12h')
    })

    it('does not jump when rotator is closed', () => {
      const { jumpToRace, pageIndex } = setup()
      expect(jumpToRace('6h', 'teams')).toBe(false)
      expect(pageIndex.value).toBe(0)
    })

    it('does not jump when settings are open', async () => {
      const { open, jumpToRace, openSettings, pageIndex, currentPage } = setup()
      open.value = true
      await nextTick()
      openSettings()
      expect(jumpToRace('6h', 'teams')).toBe(false)
      expect(pageIndex.value).toBe(0)
      expect(currentPage.value?.race).toBe('12h')
    })

    it('resets dwell when already on the target page', async () => {
      const { open, jumpToRace, pageIndex } = setup()
      open.value = true
      await nextTick()
      expect(jumpToRace('12h', 'individuals')).toBe(true)
      expect(pageIndex.value).toBe(0)

      vi.advanceTimersByTime(DEFAULT_ROTATOR_DWELL_MS - 1)
      expect(jumpToRace('12h', 'individuals')).toBe(true)
      vi.advanceTimersByTime(DEFAULT_ROTATOR_DWELL_MS - 1)
      expect(pageIndex.value).toBe(0)
      vi.advanceTimersByTime(1)
      await nextTick()
      expect(pageIndex.value).toBe(1)
    })
  })
})
