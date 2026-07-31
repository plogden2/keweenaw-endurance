<template>
  <div class="event-taps">
    <div v-if="eventsStore.loading" class="status">Loading event…</div>
    <div v-else-if="eventsStore.error" class="status error">{{ eventsStore.error }}</div>

    <template v-else-if="eventsStore.currentEvent">
      <router-link :to="`/timing/${eventId}`" class="back-link">← Back to event</router-link>
      <div class="header-row">
        <h1 class="page-title">{{ eventsStore.currentEvent.name }} — Taps</h1>
      </div>

      <div v-if="pinAuth.isAuthenticated" class="inline-bib-form">
        <div class="inline-bib-row">
          <label class="inline-bib-label">
            <span class="sr-only">Bib number</span>
            <input
              ref="bibInputRef"
              v-model="bibInput"
              type="text"
              class="inline-bib-input"
              data-testid="inline-bib-input"
              placeholder="Bib number"
              autocomplete="off"
              :disabled="submitting"
              @keydown.enter.prevent="submitBib"
              @input="onBibInput"
            />
          </label>
          <button
            type="button"
            class="btn"
            data-testid="inline-bib-submit"
            :disabled="submitting || !bibInput.trim()"
            @click="submitBib"
          >
            Submit
          </button>
          <label class="karaoke-toggle">
            <input
              v-model="karaokeBonus"
              type="checkbox"
              data-testid="inline-karaoke-toggle"
              :disabled="submitting"
            />
            Karaoke
          </label>
        </div>
        <label v-if="bibMatches.length > 1" class="inline-bib-match">
          <span class="sr-only">Select participant</span>
          <select
            v-model="selectedMatchId"
            class="inline-bib-match-select"
            data-testid="inline-bib-match-select"
            :disabled="submitting"
          >
            <option disabled value="">Select race / participant…</option>
            <option
              v-for="match in bibMatches"
              :key="match.id"
              :value="match.id"
            >
              Bib {{ match.bib_number }} — {{ match.first_name }} {{ match.last_name }}
              ({{ match.race?.name ?? 'Unknown race' }})
            </option>
          </select>
        </label>
        <p
          v-if="inlineSuccess"
          class="inline-bib-success"
          data-testid="inline-bib-success"
        >
          {{ inlineSuccess }}
        </p>
        <p
          v-if="inlineError"
          class="error"
          role="alert"
          data-testid="inline-bib-error"
        >
          {{ inlineError }}
        </p>
      </div>

      <div v-if="loading" class="status">Loading taps…</div>
      <div v-else-if="loadError" class="status error">{{ loadError }}</div>
      <template v-else>
        <table class="taps-table" data-testid="event-taps-table">
          <thead>
            <tr>
              <th>Time</th>
              <th>Race</th>
              <th>Bib</th>
              <th>Name</th>
              <th>Type</th>
              <th>Sync</th>
              <th v-if="pinAuth.isAuthenticated">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="record in taps"
              :key="record.id"
              :class="{ voided: Boolean(record.voided_at) }"
              :data-testid="`tap-row-${record.id}`"
            >
              <td>{{ formatTime(record.timestamp) }}</td>
              <td>{{ record.participant?.race?.name ?? '—' }}</td>
              <td>{{ record.participant?.bib_number ?? '—' }}</td>
              <td>
                {{ record.participant?.first_name }} {{ record.participant?.last_name }}
                <span v-if="record.voided_at" class="void-badge" data-testid="voided-badge">
                  voided
                </span>
              </td>
              <td>{{ typeLabel(record.record_type) }}</td>
              <td>{{ record.sync_status }}</td>
              <td v-if="pinAuth.isAuthenticated">
                <button
                  v-if="!record.voided_at"
                  type="button"
                  class="row-action discard"
                  data-testid="void-tap-btn"
                  @click="confirmVoid(record)"
                >
                  Void
                </button>
                <button
                  v-else
                  type="button"
                  class="row-action restore"
                  data-testid="restore-tap-btn"
                  @click="confirmRestore(record)"
                >
                  Restore
                </button>
              </td>
            </tr>
            <tr v-if="!taps.length">
              <td :colspan="pinAuth.isAuthenticated ? 7 : 6" class="empty">No taps yet.</td>
            </tr>
          </tbody>
        </table>
        <p v-if="actionError" class="error" data-testid="tap-action-error">{{ actionError }}</p>

        <div v-if="totalPages > 1" class="pagination" data-testid="taps-pagination">
          <button
            type="button"
            class="btn secondary"
            data-testid="taps-prev-page"
            :disabled="page <= 1"
            @click="goToPage(page - 1)"
          >
            ← Prev
          </button>
          <span class="page-label">Page {{ page }} of {{ totalPages }} ({{ total }} taps)</span>
          <button
            type="button"
            class="btn secondary"
            data-testid="taps-next-page"
            :disabled="page >= totalPages"
            @click="goToPage(page + 1)"
          >
            Next →
          </button>
        </div>
      </template>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { eventParticipantsApi, eventTapsApi, timingRecordsApi } from '@/services/api'
import { useEventTestModeStore } from '@/stores/eventTestMode'
import { useEventsStore } from '@/stores/events'
import { usePinAuthStore } from '@/stores/pinAuth'
import type { Participant, TimingRecord } from '@/types/models'
import { formatDateTime } from '@/utils/datetime'
import { getErrorMessage } from '@/utils/error'

const PAGE_LIMIT = 50
const SUCCESS_CLEAR_MS = 2000

const TYPE_LABELS: Record<string, string> = {
  rfid_lap: 'Lap',
  karaoke_bonus: 'Karaoke',
  checkpoint_pass: 'Checkpoint',
}

const route = useRoute()
const eventsStore = useEventsStore()
const pinAuth = usePinAuthStore()
const testMode = useEventTestModeStore()

const eventId = computed(() => String(route.params.eventId))

const taps = ref<TimingRecord[]>([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const loadError = ref<string | null>(null)
const actionError = ref<string | null>(null)
const actionBusy = ref(false)

const bibInput = ref('')
const bibInputRef = ref<HTMLInputElement | null>(null)
const karaokeBonus = ref(false)
const submitting = ref(false)
const inlineError = ref<string | null>(null)
const inlineSuccess = ref<string | null>(null)
const bibMatches = ref<Participant[]>([])
const selectedMatchId = ref('')
let successTimer: ReturnType<typeof setTimeout> | null = null

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / PAGE_LIMIT)))

function typeLabel(recordType: TimingRecord['record_type']): string {
  return (recordType && TYPE_LABELS[recordType]) || recordType || 'Lap'
}

function formatTime(iso: string): string {
  return formatDateTime(iso)
}

function clearSuccessTimer(): void {
  if (successTimer) {
    clearTimeout(successTimer)
    successTimer = null
  }
}

function clearBibMatches(): void {
  bibMatches.value = []
  selectedMatchId.value = ''
}

function onBibInput(): void {
  inlineSuccess.value = null
  inlineError.value = null
  clearBibMatches()
  clearSuccessTimer()
}

function selectBibInput(): void {
  void nextTick(() => {
    bibInputRef.value?.focus()
    bibInputRef.value?.select()
  })
}

function focusBibInput(): void {
  void nextTick(() => {
    bibInputRef.value?.focus()
  })
}

function setEphemeralSuccess(message: string): void {
  clearSuccessTimer()
  inlineSuccess.value = message
  successTimer = setTimeout(() => {
    inlineSuccess.value = null
    successTimer = null
  }, SUCCESS_CLEAR_MS)
}

async function createTapForParticipant(match: Participant): Promise<void> {
  const isKaraoke = karaokeBonus.value
  await eventTapsApi.create(eventId.value, {
    participant_id: match.id,
    karaoke_bonus: isKaraoke,
  })

  bibInput.value = ''
  clearBibMatches()
  const kind = isKaraoke ? 'Karaoke' : 'Lap'
  setEphemeralSuccess(`${kind} #${match.bib_number} ${match.first_name} ${match.last_name}`)
  focusBibInput()
  await loadTaps()
}

async function submitBib(): Promise<void> {
  const bib = bibInput.value.trim()
  if (!bib || submitting.value) return

  if (testMode.isActiveForEvent(eventId.value)) {
    inlineError.value = 'Use Test mode dialog'
    selectBibInput()
    return
  }

  submitting.value = true
  inlineError.value = null
  inlineSuccess.value = null
  clearSuccessTimer()

  try {
    if (bibMatches.value.length > 1) {
      const selected = bibMatches.value.find((p) => p.id === selectedMatchId.value)
      if (!selected) {
        inlineError.value = 'Select a participant'
        return
      }
      await createTapForParticipant(selected)
      return
    }

    const { data } = await eventParticipantsApi.list(eventId.value, { q: bib, limit: 20 })
    const matches = (data.data ?? []).filter((p) => String(p.bib_number) === bib)

    if (matches.length === 0) {
      clearBibMatches()
      inlineError.value = 'Bib not found'
      selectBibInput()
      return
    }
    if (matches.length > 1) {
      bibMatches.value = matches
      selectedMatchId.value = ''
      inlineError.value = null
      return
    }

    clearBibMatches()
    await createTapForParticipant(matches[0])
  } catch (err) {
    inlineError.value = getErrorMessage(err, 'Failed to record tap')
    selectBibInput()
  } finally {
    submitting.value = false
  }
}

async function loadEvent(): Promise<void> {
  await eventsStore.fetchEvent(eventId.value)
}

async function loadTaps(): Promise<void> {
  loading.value = true
  loadError.value = null
  try {
    const { data } = await eventTapsApi.list(eventId.value, {
      page: page.value,
      limit: PAGE_LIMIT,
    })
    taps.value = data.data ?? []
    total.value = data.total ?? 0
  } catch (err) {
    loadError.value = getErrorMessage(err, 'Failed to load taps')
  } finally {
    loading.value = false
  }
}

function goToPage(next: number): void {
  if (next < 1 || next > totalPages.value) return
  page.value = next
  void loadTaps()
}

async function confirmVoid(record: TimingRecord): Promise<void> {
  if (actionBusy.value) return
  const bib = record.participant?.bib_number ?? '—'
  const ok = window.confirm(
    `Void this tap for Bib ${bib}? It will be removed from the score.`,
  )
  if (!ok) return
  actionBusy.value = true
  actionError.value = null
  try {
    await timingRecordsApi.voidRecord(record.id)
    await loadTaps()
  } catch (err) {
    actionError.value = getErrorMessage(err, 'Failed to void tap')
  } finally {
    actionBusy.value = false
  }
}

async function confirmRestore(record: TimingRecord): Promise<void> {
  if (actionBusy.value) return
  const ok = window.confirm('Restore this tap to the score?')
  if (!ok) return
  actionBusy.value = true
  actionError.value = null
  try {
    await timingRecordsApi.restoreRecord(record.id)
    await loadTaps()
  } catch (err) {
    actionError.value = getErrorMessage(err, 'Failed to restore tap')
  } finally {
    actionBusy.value = false
  }
}

onMounted(async () => {
  await loadEvent()
  await loadTaps()
})

onUnmounted(() => {
  clearSuccessTimer()
})

watch(eventId, async () => {
  page.value = 1
  await loadEvent()
  await loadTaps()
})
</script>

<style scoped>
.event-taps {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 2rem 2rem;
}

.back-link {
  display: inline-block;
  margin-bottom: 1rem;
  color: var(--accent-link);
  text-decoration: none;
}

.header-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 1.25rem;
}

.page-title {
  color: var(--ink);
  margin: 0;
}

.inline-bib-form {
  margin-bottom: 1.25rem;
}

.inline-bib-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 1rem;
}

.karaoke-toggle {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  color: var(--ink);
  font-size: 0.95rem;
  cursor: pointer;
  user-select: none;
}

.inline-bib-label {
  display: block;
  flex: 1 1 12rem;
  min-width: 8rem;
  max-width: 16rem;
}

.inline-bib-input {
  width: min(280px, 100%);
  padding: 0.5rem 0.75rem;
  font: inherit;
  border: 1px solid var(--border);
  border-radius: 4px;
  background: #fff;
  color: var(--ink);
}

.inline-bib-input:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.inline-bib-match {
  display: block;
  margin-top: 0.5rem;
  max-width: 28rem;
}

.inline-bib-match-select {
  width: 100%;
  padding: 0.5rem 0.75rem;
  font: inherit;
  border: 1px solid var(--border);
  border-radius: 4px;
  background: #fff;
  color: var(--ink);
}

.inline-bib-match-select:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.inline-bib-success {
  margin: 0.4rem 0 0;
  color: #1e8449;
  font-size: 0.9rem;
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}

.btn {
  border: none;
  border-radius: 4px;
  padding: 0.5rem 1rem;
  font: inherit;
  cursor: pointer;
  background: var(--accent-link);
  color: #fff;
}

.btn.secondary {
  background: var(--mist);
  color: var(--ink);
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.taps-table {
  width: 100%;
  border-collapse: collapse;
}

.taps-table th,
.taps-table td {
  padding: 0.5rem 0.75rem;
  text-align: left;
  border-bottom: 1px solid var(--border);
}

.taps-table th {
  color: var(--muted);
  font-weight: 600;
}

.taps-table tr.voided {
  opacity: 0.65;
}

.taps-table tr.voided td {
  text-decoration: line-through;
}

.void-badge {
  display: inline-block;
  margin-left: 0.5rem;
  padding: 0.1rem 0.4rem;
  border-radius: 3px;
  background: #fadbd8;
  color: #922b21;
  font-size: 0.75rem;
  text-decoration: none;
  font-weight: 600;
}

.row-action {
  border: none;
  border-radius: 4px;
  padding: 0.35rem 0.65rem;
  font: inherit;
  cursor: pointer;
  font-size: 0.85rem;
}

.row-action.discard {
  background: #922b21;
  color: #fff;
}

.row-action.restore {
  background: #1e8449;
  color: #fff;
}

.status {
  color: var(--muted);
}

.status.error,
.error {
  color: var(--signal);
}

.empty {
  color: var(--muted);
  text-align: center;
}

.pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 1rem;
  margin-top: 1.25rem;
}

.page-label {
  color: var(--muted);
  font-size: 0.9rem;
}
</style>
