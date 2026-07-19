import { describe, it, expect, vi, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, ref } from 'vue'
import { useSpectatorIdle } from './useSpectatorIdle'

function mountComposable(
  opts: Parameters<typeof useSpectatorIdle>[0],
) {
  let exposed: ReturnType<typeof useSpectatorIdle> | undefined
  const Host = defineComponent({
    setup() {
      exposed = useSpectatorIdle(opts)
      return () => null
    },
  })
  const wrapper = mount(Host)
  return {
    wrapper,
    get exposed() {
      return exposed!
    },
  }
}

describe('useSpectatorIdle', () => {
  afterEach(() => {
    vi.useRealTimers()
  })

  it('is busy for 3s after interaction', () => {
    vi.useFakeTimers()
    vi.setSystemTime(10_000)

    const { exposed } = mountComposable({
      legendBusy: ref(false),
      pageScrolledFromTop: ref(false),
    })
    const { isBusy, noteInteraction } = exposed

    expect(isBusy.value).toBe(false)
    noteInteraction()
    expect(isBusy.value).toBe(true)
    vi.advanceTimersByTime(2999)
    expect(isBusy.value).toBe(true)
    vi.advanceTimersByTime(2)
    expect(isBusy.value).toBe(false)
  })

  it('is busy when legendBusy or pageScrolledFromTop', () => {
    const legendBusy = ref(true)
    const pageScrolledFromTop = ref(false)
    const { exposed } = mountComposable({ legendBusy, pageScrolledFromTop })
    const { isBusy } = exposed

    expect(isBusy.value).toBe(true)
    legendBusy.value = false
    expect(isBusy.value).toBe(false)
    pageScrolledFromTop.value = true
    expect(isBusy.value).toBe(true)
  })

  it('notes interaction on window pointerdown and keydown when mounted', () => {
    vi.useFakeTimers()
    vi.setSystemTime(10_000)

    const { exposed } = mountComposable({
      legendBusy: ref(false),
      pageScrolledFromTop: ref(false),
    })
    const { isBusy } = exposed

    expect(isBusy.value).toBe(false)
    window.dispatchEvent(new Event('pointerdown'))
    expect(isBusy.value).toBe(true)
    vi.advanceTimersByTime(3001)
    expect(isBusy.value).toBe(false)

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'a' }))
    expect(isBusy.value).toBe(true)
  })
})
