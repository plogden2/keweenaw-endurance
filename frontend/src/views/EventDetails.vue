<template>
  <div class="event-details">
    <AppHeader />

    <div v-if="eventsStore.loading" class="status">Loading event…</div>
    <div v-else-if="eventsStore.error" class="status error">{{ eventsStore.error }}</div>

    <template v-else-if="eventsStore.currentEvent">
      <h1 class="page-title">{{ eventsStore.currentEvent.name }}</h1>
      <p class="meta">
        {{ formatDate(eventsStore.currentEvent.event_date) }}
        <span v-if="eventsStore.currentEvent.location">
          · {{ eventsStore.currentEvent.location }}
        </span>
      </p>

      <section class="races-section">
        <h2>Races</h2>
        <div v-if="racesStore.loading" class="status">Loading races…</div>
        <ul v-else class="race-list">
          <li v-for="race in racesStore.races" :key="race.id">
            <router-link
              :to="`/timing/${eventId}/race/${race.id}`"
              class="race-link"
            >
              <span class="race-name">{{ race.name }}</span>
              <span class="race-status" :class="`status-${race.status}`">
                {{ race.status }}
              </span>
            </router-link>
          </li>
        </ul>
        <p v-if="!racesStore.loading && racesStore.races.length === 0" class="empty">
          No races configured for this event.
        </p>
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import AppHeader from '@/components/AppHeader.vue'
import { useEventsStore } from '@/stores/events'
import { useRacesStore } from '@/stores/races'

const route = useRoute()
const eventsStore = useEventsStore()
const racesStore = useRacesStore()

const eventId = computed(() => String(route.params.eventId))

function formatDate(value: string | undefined): string {
  if (!value) return ''
  return new Date(value).toLocaleDateString()
}

async function loadEvent(): Promise<void> {
  await eventsStore.fetchEvent(eventId.value)
  await racesStore.fetchRaces({ event_id: eventId.value })
}

onMounted(loadEvent)
watch(eventId, loadEvent)
</script>

<style scoped>
.event-details {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 2rem 2rem;
}

.page-title {
  margin-bottom: 0.5rem;
  color: #2c3e50;
}

.meta {
  color: #6c757d;
  margin-bottom: 2rem;
}

.races-section h2 {
  margin-bottom: 1rem;
  color: #2c3e50;
}

.race-list {
  list-style: none;
  padding: 0;
  margin: 0;
}

.race-link {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem;
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.08);
  margin-bottom: 0.75rem;
  text-decoration: none;
  color: #2c3e50;
}

.race-link:hover {
  background: #f8f9fa;
}

.race-status {
  font-size: 0.875rem;
  font-weight: 500;
  text-transform: capitalize;
}

.status-active {
  color: #28a745;
}

.status-finished,
.status-completed {
  color: #6c757d;
}

.status {
  color: #6c757d;
}

.status.error {
  color: #dc3545;
}

.empty {
  color: #6c757d;
}
</style>
