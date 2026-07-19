import { computed, onMounted, onUnmounted, ref, type Ref } from 'vue'

export function useSpectatorIdle(opts: {
  legendBusy: Ref<boolean>
  pageScrolledFromTop: Ref<boolean>
  interactionWindowMs?: number
}) {
  const windowMs = opts.interactionWindowMs ?? 3000
  const lastInteractionAt = ref(0)
  const now = ref(Date.now())
  let tick: number | undefined

  function noteInteraction() {
    lastInteractionAt.value = Date.now()
    now.value = Date.now()
  }

  const isBusy = computed(() => {
    if (opts.legendBusy.value) return true
    if (opts.pageScrolledFromTop.value) return true
    return now.value - lastInteractionAt.value < windowMs
  })

  onMounted(() => {
    const bump = () => noteInteraction()
    window.addEventListener('pointerdown', bump, { passive: true })
    window.addEventListener('keydown', bump)
    tick = window.setInterval(() => {
      now.value = Date.now()
    }, 250)
    onUnmounted(() => {
      window.removeEventListener('pointerdown', bump)
      window.removeEventListener('keydown', bump)
      if (tick !== undefined) clearInterval(tick)
    })
  })

  return { isBusy, noteInteraction }
}
