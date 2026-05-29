<template>
  <section class="sync-status" data-testid="sync-status">
    <h2 class="section-title">Sync Status</h2>

    <div v-if="loading" class="status">Loading sync status…</div>
    <div v-else-if="error" class="status error">{{ error }}</div>

    <template v-else-if="status">
      <ul class="counts">
        <li data-testid="pending-count">
          <span class="label">Pending</span>
          <span class="value">{{ status.pending_count }}</span>
        </li>
        <li data-testid="failed-count">
          <span class="label">Failed</span>
          <span class="value">{{ status.failed_count }}</span>
        </li>
        <li data-testid="synced-count">
          <span class="label">Synced</span>
          <span class="value">{{ status.synced_count }}</span>
        </li>
      </ul>

      <button
        type="button"
        class="sync-btn"
        data-testid="sync-pending-btn"
        :disabled="syncing || status.pending_count === 0"
        @click="syncPending"
      >
        {{ syncing ? 'Syncing…' : 'Sync pending records' }}
      </button>
      <p v-if="syncMessage" class="sync-message">{{ syncMessage }}</p>
    </template>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { rfidApi } from '@/services/api'
import type { SyncStatusResponse } from '@/types/models'
import { getErrorMessage } from '@/utils/error'

const emit = defineEmits<{
  synced: []
}>()

const status = ref<SyncStatusResponse | null>(null)
const loading = ref(false)
const error = ref<string | null>(null)
const syncing = ref(false)
const syncMessage = ref<string | null>(null)

async function loadStatus(): Promise<void> {
  loading.value = true
  error.value = null
  try {
    const { data } = await rfidApi.getSyncStatus()
    status.value = data
  } catch (err) {
    error.value = getErrorMessage(err, 'Failed to load sync status')
  } finally {
    loading.value = false
  }
}

async function syncPending(): Promise<void> {
  syncing.value = true
  syncMessage.value = null
  try {
    const { data } = await rfidApi.syncPending()
    syncMessage.value = `Synced ${data.synced_count} record(s)`
    await loadStatus()
    emit('synced')
  } catch (err) {
    syncMessage.value = getErrorMessage(err, 'Sync failed')
  } finally {
    syncing.value = false
  }
}

onMounted(() => {
  void loadStatus()
})

defineExpose({ loadStatus, syncPending })
</script>

<style scoped>
.sync-status {
  background: #f8f9fa;
  border-radius: 8px;
  padding: 1.25rem;
}

.section-title {
  margin: 0 0 1rem;
  font-size: 1.1rem;
  color: #2c3e50;
}

.counts {
  list-style: none;
  margin: 0 0 1rem;
  padding: 0;
  display: flex;
  gap: 1.5rem;
}

.label {
  display: block;
  font-size: 0.85rem;
  color: #6c757d;
}

.value {
  font-size: 1.5rem;
  font-weight: 600;
  color: #2c3e50;
}

.sync-btn {
  padding: 0.5rem 1rem;
  background: #3498db;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}

.sync-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.sync-message {
  margin: 0.75rem 0 0;
  color: #27ae60;
  font-size: 0.9rem;
}

.status {
  color: #6c757d;
}

.status.error {
  color: #c0392b;
}
</style>
