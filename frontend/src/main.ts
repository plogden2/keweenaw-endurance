import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { registerSW } from 'virtual:pwa-register'
import router from './router'
import App from './App.vue'
import { initOfflineQueue } from '@/services/offlineQueue'
import '@fontsource/outfit/400.css'
import '@fontsource/outfit/600.css'
import '@fontsource/outfit/700.css'
import '@fontsource/ibm-plex-sans/400.css'
import '@fontsource/ibm-plex-sans/600.css'
import '@fontsource/ibm-plex-sans/700.css'
import '@fontsource/yuji-mai/400.css'
import './assets/main.css'
import './themes/bluffet.css'

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

app.mount('#app-host')
