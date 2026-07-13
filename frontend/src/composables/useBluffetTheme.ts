import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useEventsStore } from '@/stores/events'
import { useStationStore } from '@/stores/station'
import {
  BLUFFET_EVENT_ID,
  BLUFFET_EVENT_NAME,
  BLUFFET_LOGO_PATH,
  BLUFFET_POSTER_AVIF,
  BLUFFET_POSTER_PNG,
  BLUFFET_THEME_CLASS,
} from '@/themes/bluffetConstants'

function isBluffetId(id: string | null | undefined): boolean {
  return Boolean(id && id === BLUFFET_EVENT_ID)
}

function isBluffetName(name: string | null | undefined): boolean {
  return Boolean(name && name === BLUFFET_EVENT_NAME)
}

export function useBluffetTheme() {
  const route = useRoute()
  const events = useEventsStore()
  const station = useStationStore()

  const active = computed(() => {
    const routeEventId = typeof route.params.eventId === 'string' ? route.params.eventId : null
    if (isBluffetId(routeEventId)) return true
    if (isBluffetId(events.currentEvent?.id) || isBluffetName(events.currentEvent?.name)) return true
    if (isBluffetId(station.eventId)) return true
    return false
  })

  return {
    active,
    themeClass: computed(() => BLUFFET_THEME_CLASS),
    posterAvif: computed(() => BLUFFET_POSTER_AVIF),
    posterPng: computed(() => BLUFFET_POSTER_PNG),
    logoPath: computed(() => BLUFFET_LOGO_PATH),
  }
}
