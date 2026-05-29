import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { registerSW } from 'virtual:pwa-register'
import router from './router'
import App from './App.vue'
import { initOfflineQueue } from '@/services/offlineQueue'
import './assets/main.css'

initOfflineQueue()

if (import.meta.env.PROD) {
  registerSW({
    immediate: true,
    onOfflineReady() {
      console.info('App ready for offline use')
    },
  })
}

const app = createApp(App)

app.use(createPinia())
app.use(router)

app.mount('#app')
