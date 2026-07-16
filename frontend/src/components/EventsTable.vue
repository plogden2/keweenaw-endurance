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
        :to="eventPath(event)"
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
import { formatEventDate } from '@/utils/participantResults'

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
  return formatEventDate(value)
}

/** Active events → live spectator view; others → timing event details. */
function eventPath(event: Event): string {
  if (event.status === 'active') {
    return `/events/${event.id}/live`
  }
  return `/timing/${event.id}`
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
  background: var(--mist);
  font-weight: 600;
  color: var(--ink);
}

.table-row {
  display: grid;
  grid-template-columns: 2fr 1fr 1fr 1fr;
  gap: 1rem;
  padding: 1rem;
  border-bottom: 1px solid var(--border);
  text-decoration: none;
  color: inherit;
  transition: background-color 0.3s;
}

.table-row:hover {
  background: var(--mist);
}

.table-row:last-child {
  border-bottom: none;
}

.status-active {
  color: var(--success);
  font-weight: 600;
  text-transform: capitalize;
}

.status-upcoming {
  color: var(--copper);
  font-weight: 600;
  text-transform: capitalize;
}

.status-completed {
  color: var(--muted);
  font-weight: 600;
  text-transform: capitalize;
}

.empty {
  padding: 1rem;
  color: var(--muted);
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
