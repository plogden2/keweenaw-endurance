<template>
  <div class="scan-overlay" data-testid="scan-feedback" aria-live="polite">
    <!-- Cooldown toast -->
    <div
      v-if="scan?.result === 'cooldown'"
      class="toast cooldown"
      role="status"
      aria-label="Cooldown notice"
      data-testid="cooldown-message"
    >
      <strong>Cooldown:</strong>
      {{ scan.participant_name || 'Racer' }}
      <template v-if="scan.retry_after_seconds != null">
        — try again in {{ formatRetry(scan.retry_after_seconds) }}
      </template>
    </div>

    <!-- Out-of-order checkpoint -->
    <div
      v-else-if="scan?.result === 'out_of_order'"
      class="toast out-of-order"
      role="status"
      aria-label="Out of sequence checkpoint"
      data-testid="out-of-order-message"
    >
      <strong>Out of sequence:</strong>
      {{ scan.message || 'This checkpoint is not next — no lap completed yet' }}
    </div>

    <!-- Intermediate checkpoint pass (not a completed lap) -->
    <div
      v-else-if="scan?.result === 'checkpoint_pass'"
      class="toast checkpoint-pass"
      role="status"
      aria-label="Checkpoint progress"
      data-testid="checkpoint-pass-message"
    >
      <strong>Checkpoint:</strong>
      {{ scan.message || 'Progress recorded — continue the sequence to complete a lap' }}
    </div>

    <!-- Pre-start test read -->
    <div
      v-else-if="scan?.result === 'test_read'"
      class="toast test-read"
      role="status"
      aria-label="Pre-start test read"
      data-testid="test-read-message"
    >
      <strong>Test read (pre-start):</strong>
      {{ scan.participant_name || 'Racer' }} identified — race not started, no lap counted
    </div>

    <!-- Unknown tag -->
    <div
      v-else-if="scan?.result === 'unknown_tag'"
      class="toast unknown"
      role="status"
      aria-label="Unknown RFID tag"
      data-testid="unknown-tag-message"
    >
      <strong>Unknown tag:</strong> No racer associated with this chip
    </div>

    <!-- Scored lap modal — karaoke only when completed lap + karaoke_available -->
    <div
      v-else-if="scan?.result === 'lap'"
      class="backdrop"
      data-testid="scan-popup"
      role="dialog"
      aria-modal="true"
      aria-label="Lap recorded"
      :aria-labelledby="nameId"
    >
      <div class="modal">
        <p class="meta">
          Lap recorded
          <template v-if="scan.race_name">
            · <span data-testid="scan-race-name">{{ scan.race_name }}</span>
          </template>
          <template v-if="scan.category_label"> · {{ scan.category_label }}</template>
        </p>
        <p :id="nameId" class="name" data-testid="scan-racer-name">
          {{ scan.participant_name || 'Unknown' }}
        </p>
        <p v-if="scan.bib_number" class="bib">Bib #{{ scan.bib_number }}</p>
        <div class="stats">
          <div class="stat">
            <span>Placement</span>
            <strong data-testid="scan-placement">{{ scan.placement ?? '—' }}</strong>
          </div>
          <div class="stat">
            <span>Laps</span>
            <strong data-testid="scan-lap-count">{{ scan.lap_count ?? '—' }}</strong>
          </div>
        </div>
        <div class="actions">
          <button
            v-if="showKaraokeButton"
            type="button"
            class="btn ok"
            data-testid="karaoke-bonus-button"
            aria-label="Add karaoke bonus lap"
            @click="onKaraoke"
          >
            + Karaoke bonus lap
          </button>
          <span
            v-else-if="karaokeRecorded"
            class="karaoke-done"
            role="status"
            aria-live="polite"
            data-testid="karaoke-bonus-recorded"
          >
            Karaoke bonus lap recorded
          </span>
          <button
            type="button"
            class="btn secondary"
            data-testid="scan-popup-dismiss"
            aria-label="Dismiss lap confirmation"
            @click="$emit('dismiss')"
          >
            Dismiss
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { ScanResult } from '@/services/api'
import newLapUrl from '@/assets/audio/new-lap.mp3'

const props = defineProps<{
  scan: ScanResult | null
}>()

const emit = defineEmits<{
  dismiss: []
  karaoke: []
}>()

const nameId = 'scan-racer-name-label'
const karaokeRecorded = ref(false)

const showKaraokeButton = computed(
  () =>
    // Karaoke only after a completed RFID lap — never checkpoint_pass / out_of_order.
    props.scan?.result === 'lap' &&
    Boolean(props.scan.karaoke_available) &&
    !karaokeRecorded.value,
)

function onKaraoke() {
  karaokeRecorded.value = true
  emit('karaoke')
}

function formatRetry(seconds: number): string {
  const m = Math.floor(seconds / 60)
  const s = seconds % 60
  return `${m}:${String(s).padStart(2, '0')}`
}

watch(
  () => props.scan?.timing_record_id,
  () => {
    karaokeRecorded.value = false
  },
)

watch(
  () => props.scan,
  (scan) => {
    if (scan?.result === 'lap') {
      try {
        const audio = new Audio(newLapUrl)
        void audio.play().catch(() => {
          /* autoplay may be blocked in some browsers; no UI label */
        })
      } catch {
        /* ignore missing audio */
      }
    }
  },
  { immediate: true },
)
</script>

<style scoped>
.scan-overlay {
  pointer-events: none;
}

.backdrop {
  pointer-events: auto;
  position: fixed;
  inset: 0;
  z-index: 2000;
  background: rgba(44, 62, 80, 0.45);
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding: 4rem 1rem 1rem;
}

.modal {
  max-width: 28rem;
  width: 100%;
  background: #fff;
  border-radius: 8px;
  padding: 1.5rem 1.5rem 1.25rem;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.18);
  text-align: center;
}

.meta {
  color: #6c757d;
  margin: 0;
  font-size: 0.95rem;
}

.name {
  font-size: 1.6rem;
  font-weight: 700;
  margin: 0.25rem 0;
  color: #2c3e50;
}

.bib {
  color: #6c757d;
  margin: 0;
}

.stats {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
  margin: 1.25rem 0;
  text-align: left;
}

.stat {
  background: #f4f6f8;
  border-radius: 6px;
  padding: 0.75rem;
}

.stat span {
  display: block;
  color: #6c757d;
  font-size: 0.8rem;
}

.stat strong {
  font-size: 1.5rem;
  color: #2c3e50;
}

.actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  justify-content: center;
}

.btn {
  border: none;
  border-radius: 4px;
  padding: 0.55rem 1rem;
  font: inherit;
  cursor: pointer;
  background: #1a5276;
  color: #fff;
}

.btn.secondary {
  background: #ecf0f1;
  color: #2c3e50;
}

.btn.ok {
  background: #1e8449;
}

.karaoke-done {
  display: inline-flex;
  align-items: center;
  padding: 0.55rem 1rem;
  color: #1e8449;
  font-weight: 600;
}

.toast {
  pointer-events: auto;
  position: fixed;
  left: 50%;
  transform: translateX(-50%);
  bottom: 1.5rem;
  z-index: 2000;
  max-width: 28rem;
  width: calc(100% - 2rem);
  padding: 0.85rem 1rem;
  border-radius: 6px;
  text-align: left;
}

.toast.cooldown {
  background: #fdebd0;
  color: #7d6608;
}

.toast.out-of-order {
  background: #fadbd8;
  color: #922b21;
}

.toast.checkpoint-pass {
  background: #e8f6f3;
  color: #0e6655;
}

.toast.test-read {
  background: #d6eaf8;
  color: #1a5276;
}

.toast.unknown {
  background: #fadbd8;
  color: #922b21;
}
</style>
