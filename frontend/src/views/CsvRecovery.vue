<template>
  <div class="csv-recovery" data-testid="csv-recovery">
    <form
      v-if="!pinAuth.isAuthenticated"
      class="panel pin-shell"
      data-testid="pin-form"
      @submit.prevent="onPinSubmit"
    >
      <h1 class="page-title">CSV Recovery</h1>
      <p class="lead">Organizer PIN required for live CSV status and import.</p>
      <label class="sr-only" for="pin-input">PIN</label>
      <input
        id="pin-input"
        v-model="pin"
        data-testid="pin-input"
        type="password"
        inputmode="numeric"
        maxlength="8"
        autocomplete="off"
        placeholder="Organizer PIN"
      />
      <p v-if="pinAuth.error || pinError" class="error" role="alert">
        {{ pinAuth.error || pinError }}
      </p>
      <button type="submit" class="btn" data-testid="pin-submit" :disabled="pinAuth.loading">
        {{ pinAuth.loading ? 'Unlocking…' : 'Unlock management' }}
      </button>
    </form>

    <template v-else>
      <p class="meta-bar">
        <span class="badge online">Management unlocked</span>
        <span
          class="badge ok"
          data-testid="live-csv-status"
          :data-updated-at="status?.updated_at || ''"
          role="status"
        >
          {{ statusLabel }}
        </span>
        <span class="muted">Disaster recovery — PIN required for import</span>
      </p>

      <h1 class="page-title">Live CSV &amp; recovery</h1>
      <p class="lead">
        A <strong>current CSV snapshot is maintained automatically</strong> on this station
        whenever race data changes (laps, racers, tags, etc.). Copy the live file to a
        replacement machine and import — no manual export step required.
      </p>

      <div class="grid-2">
        <section class="panel">
          <h2>Live CSV (always up to date)</h2>
          <p>
            <strong>Path:</strong>
            <code data-testid="live-csv-path">{{ status?.path || '…' }}</code>
          </p>
          <table>
            <tbody>
              <tr>
                <th>Last written</th>
                <td data-testid="live-csv-updated">{{ formattedUpdated }}</td>
              </tr>
              <tr>
                <th>Trigger</th>
                <td>Auto after each lap / racer / tag change</td>
              </tr>
              <tr>
                <th>Size</th>
                <td>{{ sizeLabel }}</td>
              </tr>
              <tr>
                <th>Network needed?</th>
                <td>No — local file</td>
              </tr>
            </tbody>
          </table>
          <p class="muted hint">
            Optional: download the live file for USB or cloud backup. This is a copy of the
            already-maintained snapshot — not a separate export job.
          </p>
          <div class="row">
            <button
              type="button"
              class="btn secondary"
              data-testid="live-csv-download"
              :disabled="!eventId || downloading"
              @click="onDownload"
            >
              {{ downloading ? 'Downloading…' : 'Copy live CSV' }}
            </button>
          </div>
        </section>

        <section class="panel">
          <h2>Import on replacement laptop</h2>
          <label>
            CSV file (use the live snapshot from the failed/healthy station)
            <input
              type="file"
              accept=".csv,text/csv"
              data-testid="csv-import-input"
              @change="onFileChange"
            />
          </label>
          <label class="confirm-row">
            <input
              v-model="confirmReplace"
              type="checkbox"
              name="confirm"
              data-testid="csv-import-confirm"
            />
            <span>
              I understand this replaces local data for the event in this file on
              <strong>this laptop only</strong>.
            </span>
          </label>
          <p v-if="importError" class="error" role="alert">{{ importError }}</p>
          <div class="row">
            <button
              type="button"
              class="btn danger"
              data-testid="csv-import-submit"
              :disabled="!canImport || importing"
              @click="onImport"
            >
              {{ importing ? 'Importing…' : 'Import & restore' }}
            </button>
          </div>
          <p class="muted">
            After import: arm station → continue scanning with prior lap counts. New live CSV
            writing resumes automatically.
          </p>
        </section>
      </div>

      <section v-if="summary" class="panel" data-testid="csv-import-summary">
        <h2>Last import summary</h2>
        <table>
          <tbody>
            <tr>
              <th>Event</th>
              <td>{{ summary.event_name }}</td>
            </tr>
            <tr>
              <th>Races</th>
              <td>{{ summary.races }}</td>
            </tr>
            <tr>
              <th>Racers</th>
              <td>{{ summary.racers }}</td>
            </tr>
            <tr>
              <th>Tag associations</th>
              <td>{{ summary.tag_associations }}</td>
            </tr>
            <tr>
              <th>Timing records</th>
              <td>{{ summary.timing_records }}</td>
            </tr>
            <tr>
              <th>Imported at</th>
              <td>{{ summary.imported_at }}</td>
            </tr>
          </tbody>
        </table>
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { csvApi, type CSVImportSummary, type LiveCSVStatus } from '@/services/api'
import { usePinAuthStore } from '@/stores/pinAuth'
import { useStationStore } from '@/stores/station'
import { useReaderStation } from '@/composables/useReaderStation'
import { getErrorMessage } from '@/utils/error'

const pinAuth = usePinAuthStore()
const station = useStationStore()
const route = useRoute()
const { lastScan } = useReaderStation()

const pin = ref('')
const pinError = ref<string | null>(null)
const status = ref<LiveCSVStatus | null>(null)
const confirmReplace = ref(false)
const importFile = ref<File | null>(null)
const parsedEventId = ref('')
const importing = ref(false)
const downloading = ref(false)
const importError = ref<string | null>(null)
const summary = ref<CSVImportSummary | null>(null)
let pollTimer: number | undefined

const eventId = computed(() => {
  const q = route.query.eventId
  if (typeof q === 'string' && q) return q
  if (parsedEventId.value) return parsedEventId.value
  return station.eventId || ''
})

const statusLabel = computed(() => {
  if (!status.value?.exists) return 'Live CSV pending'
  return 'Live CSV current'
})

const formattedUpdated = computed(() => {
  if (!status.value?.updated_at) return '—'
  try {
    return new Date(status.value.updated_at).toLocaleString()
  } catch {
    return status.value.updated_at
  }
})

const sizeLabel = computed(() => {
  const n = status.value?.size_bytes
  if (n == null) return '—'
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / (1024 * 1024)).toFixed(1)} MB`
})

const canImport = computed(
  () => Boolean(eventId.value && importFile.value && confirmReplace.value),
)

async function onPinSubmit() {
  pinError.value = null
  try {
    await pinAuth.loginWithPin(pin.value)
    pin.value = ''
    await refreshStatus()
  } catch {
    pinError.value = pinAuth.error || 'Invalid PIN'
  }
}

async function refreshStatus() {
  if (!pinAuth.isAuthenticated || !eventId.value) return
  try {
    const { data } = await csvApi.getLiveStatus(eventId.value)
    status.value = data
  } catch {
    /* keep last known */
  }
}

function extractEventIdFromCsv(text: string): string {
  const lines = text.split(/\r?\n/)
  let inEvent = false
  let header: string[] | null = null
  for (const line of lines) {
    const cols = line.split(',')
    if (cols[0]?.trim() === '#SECTION' && cols[1]?.trim().toLowerCase() === 'event') {
      inEvent = true
      header = null
      continue
    }
    if (!inEvent) continue
    if (!header) {
      header = cols.map((c) => c.trim().toLowerCase())
      continue
    }
    const idIdx = header.indexOf('id')
    if (idIdx >= 0 && cols[idIdx]) {
      return cols[idIdx].trim()
    }
    break
  }
  return ''
}

async function onFileChange(ev: Event) {
  const input = ev.target as HTMLInputElement
  const file = input.files?.[0] ?? null
  importFile.value = file
  importError.value = null
  parsedEventId.value = ''
  if (!file) return
  try {
    const text = await file.text()
    parsedEventId.value = extractEventIdFromCsv(text)
  } catch {
    /* ignore parse errors; user may still have query/station eventId */
  }
}

async function onDownload() {
  if (!eventId.value) return
  downloading.value = true
  try {
    const { data } = await csvApi.downloadLiveCsv(eventId.value)
    const blob = new Blob([data], { type: 'text/csv;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `live-snapshot-${eventId.value}.csv`
    a.click()
    URL.revokeObjectURL(url)
  } catch (err) {
    importError.value = getErrorMessage(err, 'Failed to download live CSV')
  } finally {
    downloading.value = false
  }
}

async function onImport() {
  if (!canImport.value || !importFile.value || !eventId.value) return
  importing.value = true
  importError.value = null
  try {
    const { data } = await csvApi.importCsv(eventId.value, importFile.value)
    summary.value = data
    await refreshStatus()
  } catch (err) {
    importError.value = getErrorMessage(err, 'Import failed')
  } finally {
    importing.value = false
  }
}

function startPolling() {
  stopPolling()
  pollTimer = window.setInterval(() => {
    void refreshStatus()
  }, 400)
}

function stopPolling() {
  if (pollTimer != null) {
    window.clearInterval(pollTimer)
    pollTimer = undefined
  }
}

watch(
  () => [pinAuth.isAuthenticated, eventId.value] as const,
  ([authed, id]) => {
    if (authed && id) {
      void refreshStatus()
      startPolling()
    } else {
      stopPolling()
    }
  },
)

watch(lastScan, () => {
  if (pinAuth.isAuthenticated && eventId.value) {
    void refreshStatus()
  }
})

onMounted(async () => {
  try {
    await station.fetchCurrent()
  } catch {
    /* unarmed */
  }
  if (pinAuth.isAuthenticated && eventId.value) {
    await refreshStatus()
    startPolling()
  }
})

onUnmounted(() => {
  stopPolling()
})
</script>

<style scoped>
.csv-recovery {
  max-width: 1100px;
  margin: 0 auto;
  padding: 0 1.5rem 2rem;
}

.page-title {
  font-size: 1.75rem;
  margin: 0 0 0.5rem;
}

.lead {
  color: #555;
  margin: 0 0 1.25rem;
  max-width: 42rem;
}

.meta-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  align-items: center;
  margin-bottom: 1rem;
}

.badge {
  display: inline-block;
  padding: 0.25rem 0.6rem;
  border-radius: 4px;
  font-size: 0.85rem;
  font-weight: 600;
}

.badge.online {
  background: #d6eaf8;
  color: #1a5276;
}

.badge.ok {
  background: #d5f5e3;
  color: #1e8449;
}

.muted {
  color: #777;
  font-size: 0.9rem;
}

.hint {
  margin-top: 0.75rem;
}

.panel {
  background: #f8f9fa;
  border: 1px solid #dee2e6;
  border-radius: 6px;
  padding: 1.25rem;
  margin-bottom: 1rem;
}

.grid-2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
}

@media (max-width: 800px) {
  .grid-2 {
    grid-template-columns: 1fr;
  }
}

.panel h2 {
  margin: 0 0 0.75rem;
  font-size: 1.15rem;
}

.panel label {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  margin-bottom: 0.75rem;
  font-size: 0.95rem;
}

.confirm-row {
  flex-direction: row !important;
  align-items: flex-start;
  gap: 0.5rem;
}

.row {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-top: 0.75rem;
}

.btn {
  background: #2c3e50;
  color: #fff;
  border: none;
  border-radius: 4px;
  padding: 0.55rem 1rem;
  cursor: pointer;
  text-decoration: none;
  font-size: 0.95rem;
}

.btn:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.btn.secondary {
  background: #7f8c8d;
}

.btn.danger {
  background: #c0392b;
}

.error {
  color: #c0392b;
  margin: 0.5rem 0;
}

table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.95rem;
}

th,
td {
  text-align: left;
  padding: 0.35rem 0.5rem;
  border-bottom: 1px solid #e9ecef;
  vertical-align: top;
}

th {
  width: 40%;
  color: #555;
  font-weight: 600;
}

code {
  font-size: 0.85rem;
  word-break: break-all;
}

.pin-shell {
  max-width: 360px;
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  border: 0;
}

input[type='password'],
input[type='file'] {
  padding: 0.45rem;
  border: 1px solid #ced4da;
  border-radius: 4px;
}
</style>
