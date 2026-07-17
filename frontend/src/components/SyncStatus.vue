<template>
  <section class="sync-status" data-testid="sync-status">
    <h2 class="section-title">Sync Status</h2>

    <div v-if="loading" class="status">Loading sync status…</div>
    <div v-else-if="error" class="status error">{{ error }}</div>

    <p v-if="isOffline" class="offline-banner" data-testid="offline-banner">
      Offline — {{ localPendingCount }} record(s) queued locally
    </p>

    <ul v-if="localPendingCount > 0" class="counts">
      <li data-testid="local-pending-count">
        <span class="label">Local queue</span>
        <span class="value">{{ localPendingCount }}</span>
      </li>
    </ul>

    <template v-if="status">
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
    </template>

    <button
      v-if="!loading"
      type="button"
      class="sync-btn"
      data-testid="sync-pending-btn"
      :disabled="syncing || (localPendingCount === 0 && (!status || status.pending_count === 0))"
      @click="syncPending"
    >
      {{ syncing ? 'Syncing…' : 'Sync pending records' }}
    </button>
    <p v-if="syncMessage" class="sync-message">{{ syncMessage }}</p>
  </section>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { rfidApi } from '@/services/api'
import {
  getLocalPendingCount,
  onOnline,
  syncAll as syncOfflineQueue,
} from '@/services/offlineQueue'
import type { SyncStatusResponse } from '@/types/models'
import { getErrorMessage } from '@/utils/error'

const emit = defineEmits<{
  synced: []
}>()

const status = ref<SyncStatusResponse | null>(null)
const localPendingCount = ref(0)
const loading = ref(false)
const error = ref<string | null>(null)
const syncing = ref(false)
const syncMessage = ref<string | null>(null)
const isOffline = ref(typeof navigator !== 'undefined' ? !navigator.onLine : false)

let removeOnlineListener: (() => void) | undefined

async function loadLocalPendingCount(): Promise<void> {
  localPendingCount.value = await getLocalPendingCount()
}

async function loadStatus(): Promise<void> {
  loading.value = true
  error.value = null
  isOffline.value = !navigator.onLine
  await loadLocalPendingCount()

  if (isOffline.value) {
    status.value = null
    loading.value = false
    return
  }

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
    const localResult = await syncOfflineQueue()
    let serverSynced = 0

    if (navigator.onLine) {
      const { data } = await rfidApi.syncPending()
      serverSynced = data.synced_count
    }

    const totalSynced = localResult.synced + serverSynced
    syncMessage.value =
      localResult.failed > 0
        ? `Synced ${totalSynced} record(s); ${localResult.failed} failed locally`
        : `Synced ${totalSynced} record(s)`
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
  removeOnlineListener = onOnline(() => {
    void loadStatus()
  })
})

onUnmounted(() => {
  removeOnlineListener?.()
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
  color: var(--ink);
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
  color: var(--ink);
}

.sync-btn {
  padding: 0.5rem 1rem;
  background: var(--accent);
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
  color: var(--success);
  font-size: 0.9rem;
}

.status {
  color: #6c757d;
}

.status.error {
  color: var(--signal);
}

.offline-banner {
  margin: 0 0 1rem;
  padding: 0.75rem;
  background: #fff3cd;
  border-radius: 4px;
  color: #856404;
  font-size: 0.9rem;
}
</style>
