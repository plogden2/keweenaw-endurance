import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import Home from '@/views/Home.vue'
import Timing from '@/views/Timing.vue'

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

export default router
