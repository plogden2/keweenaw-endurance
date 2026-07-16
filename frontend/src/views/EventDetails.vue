<template>
  <div class="event-details">
    <div v-if="eventsStore.loading" class="status">Loading event…</div>
    <div v-else-if="eventsStore.error" class="status error">{{ eventsStore.error }}</div>

    <template v-else-if="eventsStore.currentEvent">
      <header class="event-header">
        <EventLogo
          :logo-url="eventsStore.currentEvent.logo_url"
          :alt="`${eventsStore.currentEvent.name} logo`"
          class="event-logo-large"
        />
        <div class="event-heading">
          <h1 class="page-title">{{ eventsStore.currentEvent.name }}</h1>
          <p class="meta">
            {{ formatDate(eventsStore.currentEvent.event_date) }}
            <span v-if="eventsStore.currentEvent.location">
              · {{ eventsStore.currentEvent.location }}
            </span>
          </p>
        </div>
      </header>

      <section class="races-section">
        <h2>Races</h2>
        <p class="live-link-wrap">
          <router-link
            :to="`/events/${eventId}/live`"
            class="live-link"
            data-testid="event-live-link"
          >
            Open live race flow
          </router-link>
        </p>
        <div v-if="racesStore.loading" class="status">Loading races…</div>
        <ul v-else class="race-list">
          <li v-for="race in racesStore.races" :key="race.id">
            <router-link
              :to="`/timing/${eventId}/race/${race.id}`"
              class="race-link"
            >
              <span class="race-name">{{ race.name }}</span>
              <span v-if="showRaceDistance(race)" class="race-distance">
                {{ formatRaceDistance(race.distance_km!) }}
              </span>
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
import EventLogo from '@/components/EventLogo.vue'
import { useEventsStore } from '@/stores/events'
import { useRacesStore } from '@/stores/races'
import { useUnitsStore } from '@/stores/units'
import { formatEventDate } from '@/utils/participantResults'
import { formatDistance } from '@/utils/units'
import type { Race } from '@/types/models'

const route = useRoute()
const eventsStore = useEventsStore()
const racesStore = useRacesStore()
const unitsStore = useUnitsStore()

const eventId = computed(() => String(route.params.eventId))

function formatDate(value: string | undefined): string {
  return formatEventDate(value)
}

function showRaceDistance(race: Race): boolean {
  return race.race_type !== 'lap_based' && race.distance_km != null && race.distance_km > 0
}

function formatRaceDistance(distanceKm: number): string {
  return formatDistance(distanceKm, unitsStore.unitSystem)
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
  color: var(--ink);
}

.event-header {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
  margin-bottom: 2rem;
  text-align: center;
}

.event-heading {
  width: 100%;
}

.meta {
  color: var(--muted);
  margin-bottom: 0;
}

.races-section h2 {
  margin-bottom: 1rem;
  color: var(--ink);
}

.live-link-wrap {
  margin: 0 0 1rem;
}

.live-link {
  display: inline-block;
  font-weight: 600;
  color: var(--accent-link);
  text-decoration: none;
}

.live-link:hover {
  text-decoration: underline;
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
  gap: 1rem;
  padding: 1rem;
  background: var(--surface);
  border-radius: 8px;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.08);
  margin-bottom: 0.75rem;
  text-decoration: none;
  color: var(--ink);
}

.race-link:hover {
  background: var(--mist);
}

.race-name {
  flex: 1;
}

.race-distance {
  font-size: 0.875rem;
  color: var(--muted);
  white-space: nowrap;
}

.race-status {
  font-size: 0.875rem;
  font-weight: 600;
  text-transform: capitalize;
}

.status-active {
  color: var(--success);
}

.status-finished,
.status-completed {
  color: var(--muted);
}

.status {
  color: var(--muted);
}

.status.error {
  color: var(--signal);
}

.empty {
  color: var(--muted);
}
</style>
