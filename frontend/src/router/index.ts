import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import Home from '@/views/Home.vue'
import Timing from '@/views/Timing.vue'
import { eventsApi, eventsLiveApi, racesApi } from '@/services/api'
import { useStationStore } from '@/stores/station'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'home',
    component: Home,
  },
  {
    path: '/pin',
    name: 'pin-unlock',
    component: () => import('@/views/PinUnlock.vue'),
  },
  {
    path: '/station',
    name: 'station-config',
    component: () => import('@/views/StationConfig.vue'),
  },
  {
    path: '/csv',
    name: 'csv-recovery',
    component: () => import('@/views/CsvRecovery.vue'),
  },
  {
    path: '/events/:eventId/live',
    name: 'event-live',
    component: () => import('@/views/EventLive.vue'),
  },
  {
    path: '/races/:raceId/racers',
    name: 'race-racers',
    component: () => import('@/views/Racers.vue'),
  },
  {
    path: '/events/:eventId/racers',
    name: 'event-racers',
    // Prefer race-scoped `/races/:raceId/racers` (e2e). Keep event path as a stable stub.
    component: {
      template:
        '<div data-testid="racers-page"><p>Open a race’s racers page at <code>/races/:raceId/racers</code>.</p></div>',
    },
  },
  {
    path: '/timing',
    name: 'timing',
    component: Timing,
  },
  {
    path: '/timing/:eventId',
    name: 'event-details',
    component: () => import('@/views/EventDetails.vue'),
  },
  {
    path: '/timing/:eventId/race/:raceId',
    name: 'race-details',
    component: () => import('@/views/RaceDetails.vue'),
  },
  {
    path: '/timing/live/:raceId',
    name: 'live-timing',
    component: () => import('@/views/LiveTiming.vue'),
  },
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
})

/**
 * When opening `/` during an active race, land on event live view (US1 default).
 * Spectators (no station) still redirect when any Bluffet/event race is active.
 */
router.beforeEach(async (to) => {
  if (to.name !== 'home' && to.path !== '/') return true

  try {
    const station = useStationStore()
    let eventId = station.eventId
    if (!eventId) {
      try {
        await station.fetchCurrent()
        eventId = station.eventId
      } catch {
        /* no station — fall through to event discovery */
      }
    }

    if (!eventId) {
      try {
        const { data } = await eventsApi.list({ limit: 100 })
        const events = data.data ?? []
        const bluffet = events.find((e) => e.name === 'All You Can East Bluffet')
        eventId = bluffet?.id ?? events.find((e) => e.status === 'active')?.id ?? null
      } catch {
        return true
      }
    }
    if (!eventId) return true

    // Prefer live payload race statuses when available
    try {
      const { data } = await eventsLiveApi.getLive(eventId)
      const active = data.races?.some((r) => {
        const s = (r.status || '').toLowerCase()
        return s === 'active' || s === 'in_progress' || s === 'started' || s === 'running'
      })
      if (active) {
        return { name: 'event-live', params: { eventId } }
      }
    } catch {
      /* fall through to races list */
    }

    const { data } = await racesApi.list({ event_id: eventId, limit: 50 })
    const races = data.data ?? []
    const hasActive = races.some((r) => {
      const s = (r.status || '').toLowerCase()
      return s === 'active' || s === 'in_progress' || s === 'started' || s === 'running'
    })
    if (hasActive) {
      return { name: 'event-live', params: { eventId } }
    }
  } catch {
    /* stay on home */
  }
  return true
})

export default router
