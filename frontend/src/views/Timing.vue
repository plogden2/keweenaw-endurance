<template>
  <div class="timing">
    <h1 class="page-title">Race Timing</h1>

    <div v-if="eventsStore.loading" class="status">Loading events…</div>
    <div v-else-if="eventsStore.error" class="status error">{{ eventsStore.error }}</div>

    <template v-else>
      <section class="timing-section">
        <h2>Upcoming Events</h2>
        <EventsTable :events="eventsStore.upcomingEvents" empty-label="No upcoming events" />
      </section>

      <section class="timing-section">
        <h2>Active Events</h2>
        <EventsTable :events="eventsStore.activeEvents" empty-label="No active events" />
      </section>

      <section class="timing-section">
        <h2>Past Events</h2>
        <EventsTable :events="eventsStore.pastEvents" empty-label="No past events" />
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import EventsTable from '@/components/EventsTable.vue'
import { useEventsStore } from '@/stores/events'

const eventsStore = useEventsStore()

onMounted(() => {
  eventsStore.fetchEvents({ limit: 100 })
})
</script>

<style scoped>
.timing {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 2rem;
}

.page-title {
  text-align: center;
  margin-bottom: 3rem;
  color: #2c3e50;
}

.timing-section {
  margin-bottom: 3rem;
}

.timing-section h2 {
  margin-bottom: 1.5rem;
  color: #2c3e50;
}

.status {
  text-align: center;
  color: #6c757d;
}

.status.error {
  color: #dc3545;
}

@media (max-width: 768px) {
  .timing {
    padding: 0 1rem;
  }
}
</style>
