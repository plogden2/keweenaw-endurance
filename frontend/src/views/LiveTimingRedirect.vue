<template>
  <div class="redirect-status" data-testid="live-timing-redirect">
    <p v-if="error" class="error" role="alert">{{ error }}</p>
    <p v-else>Redirecting to event taps…</p>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { racesApi } from '@/services/api'
import { getErrorMessage } from '@/utils/error'

// Manual entry for a single race moved to the event-scoped taps editor.
// This view only exists to translate old `/timing/live/:raceId` links.
const route = useRoute()
const router = useRouter()
const error = ref<string | null>(null)

onMounted(async () => {
  const raceId = String(route.params.raceId || '')
  try {
    const { data: race } = await racesApi.get(raceId)
    await router.replace(`/events/${race.event_id}/taps`)
  } catch (err) {
    error.value = getErrorMessage(err, 'Failed to find that race’s event')
  }
})
</script>

<style scoped>
.redirect-status {
  max-width: 640px;
  margin: 4rem auto;
  padding: 0 2rem;
  text-align: center;
  color: var(--muted);
}

.error {
  color: var(--signal);
}
</style>
