<template>
  <div
    class="dialog-overlay"
    role="dialog"
    aria-modal="true"
    aria-labelledby="add-tap-title"
    data-testid="add-tap-dialog"
    @click.self="onCancel"
  >
    <div class="dialog-panel">
      <h2 id="add-tap-title">Add tap</h2>

      <label class="field">
        <span>Racer</span>
        <input
          v-model="searchInput"
          type="text"
          data-testid="tap-participant-search"
          placeholder="Search by bib or name…"
          autocomplete="off"
          @focus="showResults = true"
        />
      </label>

      <ul
        v-if="showResults && (searching || results.length || searchInput.trim())"
        class="results-list"
        data-testid="tap-participant-results"
      >
        <li v-if="searching" class="hint">Searching…</li>
        <li v-else-if="!results.length" class="hint">No matching racers.</li>
        <li v-for="p in results" :key="p.id">
          <button
            type="button"
            class="result-option"
            data-testid="tap-participant-option"
            @click="selectParticipant(p)"
          >
            {{ optionLabel(p) }}
          </button>
        </li>
      </ul>

      <p v-if="selectedParticipant" class="selected" data-testid="tap-participant-selected">
        Selected: <strong>{{ optionLabel(selectedParticipant) }}</strong>
      </p>

      <label class="toggle-field">
        <input
          v-model="karaokeBonus"
          type="checkbox"
          data-testid="karaoke-toggle"
        />
        <span>Karaoke bonus (standalone bonus lap, no scored lap)</span>
      </label>

      <p v-if="formError" class="error" role="alert" data-testid="add-tap-error">
        {{ formError }}
      </p>

      <div class="row">
        <button
          type="button"
          class="btn secondary"
          data-testid="add-tap-cancel"
          @click="onCancel"
        >
          Cancel
        </button>
        <button
          type="button"
          class="btn"
          data-testid="add-tap-submit"
          :disabled="submitting"
          @click="onSubmit"
        >
          {{ submitting ? 'Adding…' : 'Add tap' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onUnmounted, ref, watch } from 'vue'
import { eventParticipantsApi, eventTapsApi } from '@/services/api'
import type { Participant } from '@/types/models'
import { getErrorMessage } from '@/utils/error'

const SEARCH_DEBOUNCE_MS = 250

const props = defineProps<{
  eventId: string
}>()

const emit = defineEmits<{
  close: []
  refresh: []
}>()

const searchInput = ref('')
const results = ref<Participant[]>([])
const searching = ref(false)
const showResults = ref(false)
const selectedParticipant = ref<Participant | null>(null)
const karaokeBonus = ref(false)
const submitting = ref(false)
const formError = ref<string | null>(null)

let searchTimer: ReturnType<typeof setTimeout> | undefined

function optionLabel(p: Participant): string {
  const raceName = p.race?.name
  const name = `${p.first_name} ${p.last_name}`.trim()
  return raceName ? `#${p.bib_number} ${name} (${raceName})` : `#${p.bib_number} ${name}`
}

async function runSearch(query: string): Promise<void> {
  searching.value = true
  try {
    const { data } = await eventParticipantsApi.list(props.eventId, {
      q: query,
      limit: 20,
    })
    results.value = data.data ?? []
  } catch (err) {
    formError.value = getErrorMessage(err, 'Failed to search racers')
    results.value = []
  } finally {
    searching.value = false
  }
}

function selectParticipant(p: Participant): void {
  selectedParticipant.value = p
  searchInput.value = optionLabel(p)
  showResults.value = false
  results.value = []
}

function onCancel(): void {
  emit('close')
}

async function onSubmit(): Promise<void> {
  formError.value = null
  if (!selectedParticipant.value) {
    formError.value = 'Search for and select a racer first.'
    return
  }
  submitting.value = true
  try {
    await eventTapsApi.create(props.eventId, {
      participant_id: selectedParticipant.value.id,
      karaoke_bonus: karaokeBonus.value,
    })
    emit('refresh')
    emit('close')
  } catch (err) {
    formError.value = getErrorMessage(err, 'Failed to add tap')
  } finally {
    submitting.value = false
  }
}

watch(searchInput, (value) => {
  if (selectedParticipant.value && value === optionLabel(selectedParticipant.value)) {
    return
  }
  selectedParticipant.value = null
  showResults.value = true
  if (searchTimer) clearTimeout(searchTimer)
  const query = value.trim()
  if (!query) {
    results.value = []
    searching.value = false
    return
  }
  searchTimer = setTimeout(() => {
    void runSearch(query)
  }, SEARCH_DEBOUNCE_MS)
})

onUnmounted(() => {
  if (searchTimer) clearTimeout(searchTimer)
})
</script>

<style scoped>
.dialog-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.35);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1100;
  padding: 1rem;
}

.dialog-panel {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 1.5rem;
  max-width: 28rem;
  width: 100%;
}

.dialog-panel h2 {
  margin: 0 0 1rem;
  color: var(--ink);
}

.field {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  margin-bottom: 0.75rem;
  font-size: 0.9rem;
  color: var(--muted);
}

.field input {
  padding: 0.55rem 0.75rem;
  border: 1px solid var(--border);
  border-radius: 4px;
  font: inherit;
}

.results-list {
  list-style: none;
  margin: 0 0 0.75rem;
  padding: 0;
  max-height: 12rem;
  overflow-y: auto;
  border: 1px solid var(--border);
  border-radius: 4px;
}

.results-list .hint {
  padding: 0.5rem 0.75rem;
  color: var(--muted);
  font-size: 0.85rem;
}

.result-option {
  display: block;
  width: 100%;
  text-align: left;
  padding: 0.5rem 0.75rem;
  border: none;
  background: var(--surface);
  cursor: pointer;
  font: inherit;
  color: var(--ink);
}

.result-option:hover {
  background: var(--mist);
}

.selected {
  margin: 0 0 0.75rem;
  color: var(--ink);
  font-size: 0.9rem;
}

.toggle-field {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 1rem;
  font-size: 0.9rem;
  color: var(--ink);
}

.error {
  color: var(--signal);
  margin: 0 0 0.75rem;
}

.row {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
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

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn.secondary {
  background: var(--mist);
  color: var(--ink);
}
</style>
