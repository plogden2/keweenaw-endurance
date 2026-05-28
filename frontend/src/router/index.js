import { createRouter, createWebHistory } from 'vue-router'
import Home from '../views/Home.vue'
import Timing from '../views/Timing.vue'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'home',
      component: Home,
    },
    {
      path: '/timing',
      name: 'timing',
      component: Timing,
    },
    {
      path: '/timing/:eventId',
      name: 'event-details',
      component: () => import('../views/EventDetails.vue'),
    },
    {
      path: '/timing/:eventId/race/:raceId',
      name: 'race-details',
      component: () => import('../views/RaceDetails.vue'),
    },
    {
      path: '/timing/live/:raceId',
      name: 'live-timing',
      component: () => import('../views/LiveTiming.vue'),
    },
  ],
})

export default router