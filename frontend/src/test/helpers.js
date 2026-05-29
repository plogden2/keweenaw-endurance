import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createWebHistory } from 'vue-router'

export function setupPinia() {
  const pinia = createPinia()
  setActivePinia(pinia)
  return pinia
}

export function createTestRouter(routes = []) {
  return createRouter({
    history: createWebHistory(),
    routes: [
      { path: '/', name: 'home', component: { template: '<div />' } },
      { path: '/timing', name: 'timing', component: { template: '<div />' } },
      {
        path: '/timing/:eventId',
        name: 'event-details',
        component: { template: '<div />' },
      },
      {
        path: '/timing/:eventId/race/:raceId',
        name: 'race-details',
        component: { template: '<div />' },
      },
      ...routes,
    ],
  })
}
