<template>
  <div class="events-table">
    <div class="table-header">
      <span>Event Name</span>
      <span>Date</span>
      <span>Participants</span>
      <span>Status</span>
    </div>
    <div v-if="events.length" class="table-body">
      <router-link
        v-for="event in events"
        :key="event.id"
        :to="`/timing/${event.id}`"
        class="table-row"
      >
        <span>{{ event.name }}</span>
        <span>{{ formatDate(event.event_date) }}</span>
        <span>—</span>
        <span :class="`status-${event.status}`">{{ event.status }}</span>
      </router-link>
    </div>
    <p v-else class="empty">{{ emptyLabel }}</p>
  </div>
</template>

<script setup lang="ts">
import type { Event } from '@/types/models'

withDefaults(
  defineProps<{
    events: Event[]
    emptyLabel?: string
  }>(),
  {
    emptyLabel: 'No events',
  },
)

function formatDate(value: string | undefined): string {
  if (!value) return ''
  return new Date(value).toLocaleDateString()
}
</script>

<style scoped>
.events-table {
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
  overflow: hidden;
}

.table-header {
  display: grid;
  grid-template-columns: 2fr 1fr 1fr 1fr;
  gap: 1rem;
  padding: 1rem;
  background: #f8f9fa;
  font-weight: 600;
  color: #2c3e50;
}

.table-row {
  display: grid;
  grid-template-columns: 2fr 1fr 1fr 1fr;
  gap: 1rem;
  padding: 1rem;
  border-bottom: 1px solid #e9ecef;
  text-decoration: none;
  color: inherit;
  transition: background-color 0.3s;
}

.table-row:hover {
  background: #f8f9fa;
}

.table-row:last-child {
  border-bottom: none;
}

.status-active {
  color: #28a745;
  font-weight: 500;
  text-transform: capitalize;
}

.status-completed {
  color: #6c757d;
  font-weight: 500;
  text-transform: capitalize;
}

.empty {
  padding: 1rem;
  color: #6c757d;
  margin: 0;
}

@media (max-width: 768px) {
  .table-header,
  .table-row {
    grid-template-columns: 1fr;
    gap: 0.5rem;
  }
}
</style>
