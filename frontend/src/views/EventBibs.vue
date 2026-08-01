<template>
  <div class="event-bibs-page" data-testid="event-bibs-page">
    <div v-if="eventsStore.loading" class="status">Loading event…</div>
    <div v-else-if="eventsStore.error" class="status error">{{ eventsStore.error }}</div>

    <template v-else-if="eventsStore.currentEvent">
      <p class="meta-bar">
        <router-link :to="`/events/${eventId}/live`" class="back-link">← Live</router-link>
        <span
          v-if="pinAuth.isAuthenticated"
          class="badge online"
          data-testid="mgmt-unlocked"
        >
          Management unlocked
        </span>
        <router-link class="btn secondary" to="/pin">PIN</router-link>
      </p>

      <h1 class="page-title">{{ eventsStore.currentEvent.name }} — Bibs</h1>
      <p class="lead">
        Event bib inventory: create ranges, see tag counts, program tags. PIN required for
        changes.
      </p>

      <form
        v-if="pinAuth.isAuthenticated"
        class="panel bulk-form"
        data-testid="bibs-bulk-create"
        @submit.prevent="onBulkCreate"
      >
        <h2>Bulk create</h2>
        <div class="row">
          <label>
            From
            <input
              v-model.number="bulkFrom"
              type="number"
              min="1"
              step="1"
              data-testid="bibs-bulk-from"
              required
            />
          </label>
          <label>
            To
            <input
              v-model.number="bulkTo"
              type="number"
              min="1"
              step="1"
              data-testid="bibs-bulk-to"
              required
            />
          </label>
          <button type="submit" class="btn" data-testid="bibs-bulk-submit" :disabled="bulkSaving">
            {{ bulkSaving ? 'Creating…' : 'Create' }}
          </button>
        </div>
        <p v-if="bulkError" class="error" role="alert">{{ bulkError }}</p>
      </form>

      <div v-if="loading" class="status">Loading bibs…</div>
      <div v-else-if="loadError" class="status error">{{ loadError }}</div>
      <div v-else class="panel">
        <p
          v-if="programSuccess"
          class="success"
          role="status"
          data-testid="bib-program-success"
        >
          {{ programSuccess }}
        </p>
        <p v-if="programError" class="error" role="alert" data-testid="bib-program-error">
          {{ programError }}
        </p>
        <div class="table-scroll">
          <table class="bibs-table" data-testid="event-bibs-table">
            <thead>
              <tr>
                <th>Bib #</th>
                <th>Tags</th>
                <th>Assigned racer</th>
                <th v-if="pinAuth.isAuthenticated">Actions</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="bib in sortedBibs"
                :key="bib.id"
                :data-testid="`bib-row-${bib.id}`"
              >
                <td class="bib-num">{{ bib.bib_number }}</td>
                <td>{{ bib.tag_count }}</td>
                <td>
                  <template v-if="bib.participant_name">{{ bib.participant_name }}</template>
                  <span v-else class="muted">unassigned</span>
                </td>
                <td v-if="pinAuth.isAuthenticated">
                  <button
                    type="button"
                    class="btn"
                    data-testid="bib-program-tag"
                    :disabled="programmingId !== null"
                    @click="programTag(bib)"
                  >
                    {{ programmingId === bib.id ? 'Writing…' : 'Program' }}
                  </button>
                </td>
              </tr>
              <tr v-if="!sortedBibs.length">
                <td :colspan="pinAuth.isAuthenticated ? 4 : 3" class="empty">No bibs yet.</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { eventBibsApi, rfidApi } from '@/services/api'
import { useEventsStore } from '@/stores/events'
import { usePinAuthStore } from '@/stores/pinAuth'
import type { BibListItem } from '@/types/models'
import { getErrorMessage } from '@/utils/error'

const route = useRoute()
const router = useRouter()
const eventsStore = useEventsStore()
const pinAuth = usePinAuthStore()

const eventId = computed(() => String(route.params.eventId))

const bibs = ref<BibListItem[]>([])
const loading = ref(false)
const loadError = ref<string | null>(null)

const sortedBibs = computed(() =>
  [...bibs.value].sort(
    (a, b) => Number(a.bib_number) - Number(b.bib_number) || String(a.bib_number).localeCompare(String(b.bib_number)),
  ),
)

const bulkFrom = ref(1)
const bulkTo = ref(100)
const bulkSaving = ref(false)
const bulkError = ref<string | null>(null)

const programmingId = ref<string | null>(null)
const programError = ref<string | null>(null)
const programSuccess = ref<string | null>(null)

async function loadEvent() {
  await eventsStore.fetchEvent(eventId.value)
}

async function loadBibs() {
  loading.value = true
  loadError.value = null
  try {
    const { data } = await eventBibsApi.list(eventId.value)
    bibs.value = data.data ?? []
  } catch (err) {
    loadError.value = getErrorMessage(err, 'Failed to load bibs')
  } finally {
    loading.value = false
  }
}

async function onBulkCreate() {
  if (!pinAuth.isAuthenticated) {
    await router.push('/pin')
    return
  }
  bulkSaving.value = true
  bulkError.value = null
  try {
    await eventBibsApi.bulkCreate(eventId.value, Number(bulkFrom.value), Number(bulkTo.value))
    await loadBibs()
  } catch (err) {
    bulkError.value = getErrorMessage(err, 'Failed to create bibs')
  } finally {
    bulkSaving.value = false
  }
}

async function programTag(bib: BibListItem) {
  if (!pinAuth.isAuthenticated) {
    await router.push('/pin')
    return
  }
  if (programmingId.value !== null) return
  programmingId.value = bib.id
  programError.value = null
  programSuccess.value = null
  try {
    await rfidApi.writeTag({
      bib_id: bib.id,
      logical_uuid: bib.logical_uuid || bib.tag_uids?.[0] || undefined,
    })
    await loadBibs()
    programSuccess.value = `Wrote tag for bib ${bib.bib_number}`
  } catch (err) {
    programError.value = getErrorMessage(err, 'Failed to write tag')
  } finally {
    programmingId.value = null
  }
}

onMounted(async () => {
  await loadEvent()
  await loadBibs()
})

watch(eventId, async () => {
  await loadEvent()
  await loadBibs()
})
</script>

<style scoped>
.event-bibs-page {
  width: 100%;
  max-width: min(1100px, 100%);
  margin: 0 auto;
  padding: 0 1rem 3rem;
  box-sizing: border-box;
  min-width: 0;
  --line: var(--border);
}

.meta-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem 1.5rem;
  align-items: center;
  margin: 1rem 0 1.25rem;
  font-size: 0.9rem;
  color: var(--muted);
}

.back-link {
  color: var(--accent-link);
  text-decoration: none;
}

.badge {
  display: inline-block;
  padding: 0.15rem 0.55rem;
  border-radius: 4px;
  font-size: 0.8rem;
  font-weight: 600;
}

.badge.online {
  background: color-mix(in srgb, var(--success) 15%, var(--surface));
  color: var(--success);
}

.page-title {
  margin: 0 0 0.35rem;
  color: var(--ink);
}

.lead {
  margin: 0 0 1.25rem;
  color: var(--muted);
  max-width: 40rem;
}

.panel {
  background: var(--surface);
  border: 1px solid var(--line);
  border-radius: 6px;
  padding: 1rem 1.15rem;
  margin-bottom: 1rem;
}

.bulk-form h2 {
  margin: 0 0 0.75rem;
  font-size: 1.1rem;
  color: var(--ink);
}

.row {
  display: flex;
  flex-wrap: wrap;
  gap: 0.6rem;
  align-items: flex-end;
}

label {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  font-size: 0.9rem;
  margin: 0;
}

input {
  padding: 0.5rem 0.7rem;
  border: 1px solid var(--border);
  border-radius: 4px;
  font: inherit;
  width: 6rem;
}

.btn {
  display: inline-block;
  padding: 0.5rem 1rem;
  border: none;
  border-radius: 4px;
  background: var(--accent-link);
  color: #fff;
  font: inherit;
  cursor: pointer;
  text-decoration: none;
}

.btn.secondary {
  background: var(--mist);
  color: var(--ink);
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.bibs-table {
  width: 100%;
  min-width: 28rem;
  border-collapse: collapse;
}

.bibs-table th,
.bibs-table td {
  text-align: left;
  padding: 0.55rem 0.4rem;
  border-bottom: 1px solid var(--line);
  vertical-align: middle;
}

.bibs-table th {
  color: var(--muted);
  font-weight: 600;
}

.bib-num {
  font-weight: 600;
}

.muted {
  color: var(--muted);
}

.empty {
  color: var(--muted);
  text-align: center;
}

.status {
  color: var(--muted);
}

.status.error,
.error {
  color: var(--signal);
}

.success {
  color: var(--success);
}
</style>
