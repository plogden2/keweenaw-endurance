<template>
  <Teleport to="body">
    <div
      class="test-mode-overlay"
      role="dialog"
      aria-modal="true"
      aria-labelledby="test-mode-title"
      data-testid="event-test-mode-dialog"
    >
      <div class="test-mode-panel">
        <header class="test-mode-header">
          <div>
            <h2 id="test-mode-title">Test mode</h2>
            <p class="banner" data-testid="test-mode-banner">
              This station only — other readers still score for real. Test laps are discarded when
              you close.
            </p>
          </div>
          <button
            type="button"
            class="btn secondary"
            data-testid="test-mode-close"
            @click="requestClose"
          >
            Close &amp; discard
          </button>
        </header>

        <div class="test-mode-body">
          <section class="entry-panel panel">
            <h3>Manual entry</h3>
            <form class="bib-form" @submit.prevent="submitBib">
              <label>
                <span class="sr-only">Bib number</span>
                <input
                  ref="bibInputRef"
                  v-model="bibInput"
                  type="text"
                  class="bib-input"
                  data-testid="test-mode-bib-input"
                  placeholder="Bib number"
                  autocomplete="off"
                  :disabled="submitting"
                />
              </label>
              <button
                type="submit"
                class="btn"
                data-testid="test-mode-bib-submit"
                :disabled="submitting || !bibInput.trim()"
              >
                Record test lap
              </button>
            </form>
            <p
              v-if="store.lastFeedback"
              class="feedback"
              :class="{ error: !store.lastFeedback.ok }"
              data-testid="test-mode-feedback"
              role="status"
            >
              <template v-if="store.lastFeedback.ok">
                Test lap #{{ store.lastFeedback.lap_count }} —
                {{ store.lastFeedback.participant_name }} (bib
                {{ store.lastFeedback.bib_number }})
              </template>
              <template v-else>{{ store.lastFeedback.message }}</template>
            </p>
            <p v-else class="muted">RFID taps and bib entries appear here. No cooldown.</p>
          </section>

          <section class="chart-panel panel">
            <h3>Race flow (all races)</h3>
            <RaceFlowChart
              race-type="lap_based"
              race-status="active"
              :race-start-time="store.sessionStartedAt || undefined"
              :duration-minutes="durationMinutes"
              :external-timing-records="store.timingRecords"
              :external-participants="store.roster"
              v-model:highlight-participant-id="highlightParticipantId"
            />
          </section>

          <section class="board-panel panel">
            <h3>Leaderboard — Combined (test)</h3>
            <table data-testid="test-mode-leaderboard">
              <thead>
                <tr>
                  <th>Place</th>
                  <th>Bib</th>
                  <th>Name</th>
                  <th>Laps</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="entry in store.leaderboard"
                  :key="entry.participant_id"
                  data-testid="test-mode-leaderboard-row"
                  :data-participant-id="entry.participant_id"
                >
                  <td class="place">{{ entry.place }}</td>
                  <td>{{ entry.bib_number }}</td>
                  <td>
                    <span
                      class="cat-dot"
                      :style="{ background: resolveCategoryColor(entry.category_key) }"
                    />
                    {{ entry.name }}
                  </td>
                  <td data-testid="test-mode-leaderboard-laps">{{ entry.laps }}</td>
                </tr>
                <tr v-if="!store.leaderboard.length">
                  <td colspan="4">No test taps yet</td>
                </tr>
              </tbody>
            </table>
          </section>
        </div>
      </div>

      <div
        v-if="confirmOpen"
        class="confirm-overlay"
        role="dialog"
        aria-modal="true"
        aria-labelledby="test-mode-confirm-title"
        data-testid="test-mode-discard-confirm"
      >
        <div class="confirm-panel panel">
          <h3 id="test-mode-confirm-title">Discard test laps?</h3>
          <p>Closing clears {{ store.taps.length }} test tap(s). Production scores are unchanged.</p>
          <div class="confirm-actions">
            <button type="button" class="btn secondary" data-testid="test-mode-discard-cancel" @click="confirmOpen = false">
              Keep open
            </button>
            <button type="button" class="btn" data-testid="test-mode-discard-confirm-btn" @click="confirmClose">
              Discard &amp; close
            </button>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import RaceFlowChart from '@/components/RaceFlowChart.vue'
import { useEventTestModeStore } from '@/stores/eventTestMode'
import { resolveCategoryColor } from '@/themes/defaultLegend'

const props = defineProps<{
  /** When set, ambiguous bibs prefer this race's participant. */
  preferredRaceId?: string
}>()

const emit = defineEmits<{ close: [] }>()

const store = useEventTestModeStore()
const bibInput = ref('')
const submitting = ref(false)
const confirmOpen = ref(false)
const highlightParticipantId = ref<string | undefined>()
const bibInputRef = ref<HTMLInputElement | null>(null)

const durationMinutes = computed(() => {
  const minutes = store.roster
    .map((p) => p.race?.duration_minutes)
    .filter((n): n is number => typeof n === 'number' && n > 0)
  return minutes.length ? Math.max(...minutes) : 720
})

function submitBib() {
  const bib = bibInput.value.trim()
  if (!bib || submitting.value) return
  submitting.value = true
  try {
    const result = store.recordBibTap(bib, undefined, props.preferredRaceId)
    if (result.ok) {
      bibInput.value = ''
    }
  } finally {
    submitting.value = false
    void nextTick(() => bibInputRef.value?.focus())
  }
}

function requestClose() {
  if (store.hasTaps) {
    confirmOpen.value = true
    return
  }
  emit('close')
}

function confirmClose() {
  confirmOpen.value = false
  emit('close')
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    if (confirmOpen.value) {
      confirmOpen.value = false
      return
    }
    requestClose()
  }
}

onMounted(() => {
  window.addEventListener('keydown', onKeydown)
  void nextTick(() => bibInputRef.value?.focus())
})

onUnmounted(() => {
  window.removeEventListener('keydown', onKeydown)
})
</script>

<style scoped>
.test-mode-overlay {
  position: fixed;
  inset: 0;
  z-index: 1200;
  background: rgba(12, 18, 16, 0.55);
  display: flex;
  align-items: stretch;
  justify-content: center;
  padding: 1rem;
}

.test-mode-panel {
  background: var(--color-bg, #f4f7f5);
  color: var(--color-text, #1a2a26);
  width: min(1200px, 100%);
  max-height: 100%;
  overflow: auto;
  border-radius: 8px;
  display: flex;
  flex-direction: column;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.25);
}

.test-mode-header {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  align-items: flex-start;
  padding: 1rem 1.25rem;
  border-bottom: 1px solid rgba(26, 42, 38, 0.12);
  position: sticky;
  top: 0;
  background: inherit;
  z-index: 1;
}

.test-mode-header h2 {
  margin: 0 0 0.35rem;
}

.banner {
  margin: 0;
  max-width: 42rem;
  font-size: 0.95rem;
  color: #5a3d2b;
  background: #f3e6d8;
  border: 1px solid #d8b89a;
  border-radius: 4px;
  padding: 0.5rem 0.75rem;
}

.test-mode-body {
  display: grid;
  gap: 1rem;
  padding: 1rem 1.25rem 1.5rem;
}

.panel {
  margin: 0;
  padding: 1rem;
  background: rgba(255, 255, 255, 0.65);
  border-radius: 6px;
}

.panel h3 {
  margin: 0 0 0.75rem;
}

.bib-form {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  align-items: center;
}

.bib-input {
  min-width: 10rem;
  padding: 0.45rem 0.6rem;
  font-size: 1rem;
}

.feedback {
  margin: 0.75rem 0 0;
  font-weight: 600;
}

.feedback.error {
  color: #8b2e2e;
}

.muted {
  margin: 0.75rem 0 0;
  opacity: 0.75;
}

table {
  width: 100%;
  border-collapse: collapse;
}

th,
td {
  text-align: left;
  padding: 0.4rem 0.5rem;
  border-bottom: 1px solid rgba(26, 42, 38, 0.1);
}

.place {
  font-variant-numeric: tabular-nums;
  width: 4rem;
}

.cat-dot {
  display: inline-block;
  width: 0.65rem;
  height: 0.65rem;
  border-radius: 50%;
  margin-right: 0.4rem;
  vertical-align: middle;
}

.confirm-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.4);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1300;
  padding: 1rem;
}

.confirm-panel {
  max-width: 24rem;
  width: 100%;
}

.confirm-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  margin-top: 1rem;
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
  appearance: none;
  border: 1px solid #1a3f3d;
  background: #1a3f3d;
  color: #fff;
  border-radius: 4px;
  padding: 0.45rem 0.85rem;
  cursor: pointer;
  font: inherit;
}

.btn.secondary {
  background: transparent;
  color: #1a3f3d;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
