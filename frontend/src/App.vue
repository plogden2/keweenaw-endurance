<template>
  <div id="app">
    <AppHeader v-if="showTimingHeader" />

    <main class="main">
      <router-view />
    </main>

    <ScanPopup :scan="lastScan" @dismiss="clearLastScan" @karaoke="onKaraoke" />

    <footer class="footer">
      <div class="footer-content">
        <UnitToggle />
        <p>&copy; 2026 Keweenaw Endurance Syndicate. All rights reserved.</p>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import AppHeader from '@/components/AppHeader.vue'
import UnitToggle from '@/components/UnitToggle.vue'
import ScanPopup from '@/components/ScanPopup.vue'
import { useReaderStation } from '@/composables/useReaderStation'
import { timingRecordsApi } from '@/services/api'
import { useStationStore } from '@/stores/station'

const route = useRoute()
const station = useStationStore()
const { lastScan, clearLastScan, start, stop } = useReaderStation()

// Show Inferior Timing branding only on timing routes.
const showTimingHeader = computed(() => route.path.startsWith('/timing'))

async function onKaraoke() {
  const scan = lastScan.value
  if (!scan?.timing_record_id) return
  try {
    const { data } = await timingRecordsApi.karaokeBonus(scan.timing_record_id)
    lastScan.value = {
      ...scan,
      lap_count: data.lap_count,
      placement: data.placement,
      placement_category: data.placement_category,
      karaoke_available: false,
    }
  } catch {
    /* ScanPopup already shows recorded; duplicate POSTs return 409 */
  }
}

onMounted(() => {
  void station.fetchCurrent().catch(() => {
    /* station may be unarmed */
  })
  start()
})

onUnmounted(() => {
  stop()
})
</script>

<style scoped>
.main {
  min-height: calc(100vh - 80px);
  padding: 2rem 0;
}

.footer {
  background-color: #34495e;
  color: white;
  padding: 2rem 0;
  margin-top: auto;
}

.footer-content {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 2rem;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
}
</style>
