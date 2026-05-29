<template>
  <form class="manual-form" @submit.prevent="onSubmit">
    <h2 class="section-title">Manual Timing Entry</h2>

    <label class="field">
      <span>Checkpoint</span>
      <select
        v-model="checkpointId"
        data-testid="checkpoint-select"
      >
        <option value="" disabled>Select checkpoint</option>
        <option
          v-for="cp in checkpoints"
          :key="cp.id"
          :value="cp.id"
          data-testid="checkpoint-option"
        >
          {{ cp.name }} ({{ cp.checkpoint_type }})
        </option>
      </select>
    </label>

    <label class="field">
      <span>Bib number</span>
      <input
        v-model="bibNumber"
        type="text"
        data-testid="bib-input"
        placeholder="e.g. 42"
      />
    </label>

    <label class="field">
      <span>RFID tag UID</span>
      <input
        v-model="rfidTagUid"
        type="text"
        data-testid="rfid-input"
        placeholder="Scan or enter tag"
      />
    </label>

    <p v-if="validationError" class="error" data-testid="form-error">
      {{ validationError }}
    </p>

    <button
      type="submit"
      class="submit-btn"
      data-testid="manual-submit"
      :disabled="submitting"
    >
      {{ submitting ? 'Recording…' : 'Record time' }}
    </button>
  </form>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { Checkpoint, ManualTimingEntryPayload } from '@/types/models'

const props = defineProps<{
  raceId: string
  checkpoints: Checkpoint[]
  submitting?: boolean
}>()

const emit = defineEmits<{
  submit: [payload: ManualTimingEntryPayload]
}>()

const checkpointId = ref('')
const bibNumber = ref('')
const rfidTagUid = ref('')
const validationError = ref<string | null>(null)

function onSubmit(): void {
  validationError.value = null

  if (!checkpointId.value) {
    validationError.value = 'Select a checkpoint before recording.'
    return
  }

  const bib = bibNumber.value.trim()
  const uid = rfidTagUid.value.trim()
  if (!bib && !uid) {
    validationError.value = 'Enter a bib number or RFID tag UID.'
    return
  }

  const payload: ManualTimingEntryPayload = {
    race_id: props.raceId,
    checkpoint_id: checkpointId.value,
    timestamp: new Date().toISOString(),
  }
  if (bib) {
    payload.bib_number = bib
  }
  if (uid) {
    payload.rfid_tag_uid = uid
  }

  emit('submit', payload)
}
</script>

<style scoped>
.manual-form {
  background: #fff;
  border: 1px solid #dee2e6;
  border-radius: 8px;
  padding: 1.25rem;
}

.section-title {
  margin: 0 0 1rem;
  font-size: 1.1rem;
  color: #2c3e50;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  margin-bottom: 1rem;
}

.field span {
  font-size: 0.9rem;
  color: #495057;
}

.field input,
.field select {
  padding: 0.5rem 0.75rem;
  border: 1px solid #ced4da;
  border-radius: 4px;
  font-size: 1rem;
}

.submit-btn {
  padding: 0.6rem 1.25rem;
  background: #27ae60;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 1rem;
}

.submit-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.error {
  color: #c0392b;
  font-size: 0.9rem;
  margin: 0 0 0.75rem;
}
</style>
