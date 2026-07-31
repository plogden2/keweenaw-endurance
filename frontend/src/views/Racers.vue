<template>
  <div class="racers-page" data-testid="racers-page">
    <p class="meta-bar">
      <span v-if="race">Race: <strong>{{ race.name }}</strong></span>
      <span
        v-if="pinAuth.isAuthenticated"
        class="badge online"
        data-testid="mgmt-unlocked"
      >
        Management unlocked
      </span>
      <router-link class="btn secondary" to="/pin">PIN</router-link>
    </p>

    <h1 class="page-title">Racers</h1>
    <p class="lead">
      Search, add, click bib to edit, program tags from the row. PIN required for
      changes.
    </p>

    <div class="panel">
      <div class="row toolbar">
        <label class="search-label" for="racers-search">
          Search by name or bib
          <input
            id="racers-search"
            v-model="searchInput"
            type="search"
            data-testid="racers-search"
            placeholder="Type to filter…"
            aria-label="Search racers by name or bib"
            autocomplete="off"
          />
        </label>
        <button
          type="button"
          class="btn secondary"
          data-testid="add-racer"
          @click="showAdd = true"
        >
          Add racer
        </button>
      </div>
      <p class="muted hint">Results update as you type (debounced). No search button.</p>
    </div>

    <div v-if="showAdd" class="panel" data-testid="add-racer-form">
      <h2>Add racer</h2>
      <form @submit.prevent="onAddRacer">
        <div class="grid-2">
          <label>
            First name
            <input v-model="addForm.first_name" data-testid="racer-first-name" required />
          </label>
          <label>
            Last name
            <input v-model="addForm.last_name" data-testid="racer-last-name" required />
          </label>
          <label>
            Gender
            <select v-model="addForm.gender" data-testid="racer-gender">
              <option value="male">Men</option>
              <option value="female">Women</option>
            </select>
          </label>
          <label>
            Category
            <select v-model="addForm.category_id" data-testid="racer-category" required>
              <option disabled value="">Select category…</option>
              <option v-for="cat in categories" :key="cat.id" :value="cat.id">
                {{ cat.name }}
              </option>
            </select>
          </label>
          <label>
            Bib number (optional — default next sequential)
            <input
              v-model="addForm.bib_number"
              type="text"
              inputmode="numeric"
              data-testid="racer-bib"
              :placeholder="nextBibHint"
            />
          </label>
        </div>
        <p v-if="formError" class="error" role="alert">{{ formError }}</p>
        <div class="row">
          <button type="submit" class="btn" data-testid="racer-save" :disabled="saving">
            {{ saving ? 'Saving…' : 'Save racer' }}
          </button>
          <button type="button" class="btn secondary" @click="showAdd = false">Cancel</button>
        </div>
      </form>
    </div>

    <div class="panel" data-testid="teams-section">
      <h2>
        Teams
        <span class="muted">({{ teams.length }})</span>
      </h2>
      <form class="row team-create" @submit.prevent="onCreateTeam">
        <label class="grow">
          New team name
          <input
            v-model="newTeamName"
            type="text"
            data-testid="team-name-input"
            placeholder="East Bluff A"
            required
          />
        </label>
        <button
          type="submit"
          class="btn"
          data-testid="team-create"
          :disabled="creatingTeam"
        >
          {{ creatingTeam ? 'Creating…' : 'Create team' }}
        </button>
      </form>
      <p v-if="teamError" class="error" role="alert">{{ teamError }}</p>
      <table data-testid="teams-list">
        <thead>
          <tr>
            <th>Name</th>
            <th>Members</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="team in teams" :key="team.id" data-testid="team-row">
            <td>{{ team.name }}</td>
            <td>{{ memberCountForTeam(team.id) }}</td>
            <td>
              <button
                type="button"
                class="btn secondary"
                data-testid="team-delete"
                @click="onDeleteTeam(team)"
              >
                Delete
              </button>
            </td>
          </tr>
          <tr v-if="!teams.length">
            <td colspan="3" class="muted">No teams yet. Create one, then assign racers below.</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div class="panel">
      <h2>
        Racer list
        <span class="muted">({{ filteredRacers.length }} shown)</span>
      </h2>
      <p v-if="loadError" class="error" role="alert">{{ loadError }}</p>
      <p
        v-if="bibNoTagsWarn"
        class="warn"
        role="status"
        data-testid="bib-no-tags-warn"
      >
        {{ bibNoTagsWarn }}
      </p>
      <table data-testid="racers-list">
        <thead>
          <tr>
            <th>Bib</th>
            <th>Name</th>
            <th>Category</th>
            <th>Team</th>
            <th>Tags</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="racer in filteredRacers" :key="racer.id">
            <tr
              data-testid="racer-row"
              :class="{ programming: programmingId === racer.id }"
            >
              <td class="bib-cell">
                <template v-if="editingBibId === racer.id">
                  <span class="bib-edit-wrap">
                    <input
                      v-model="bibDraft"
                      type="text"
                      inputmode="numeric"
                      data-testid="bib-edit-input"
                      aria-label="Edit bib number"
                      @keydown.enter="saveBib(racer)"
                      @keydown.escape="cancelBibEdit"
                    />
                    <button
                      v-if="bibDirty"
                      type="button"
                      class="icon-btn"
                      data-testid="bib-save"
                      title="Save bib"
                      aria-label="Save bib"
                      @click="saveBib(racer)"
                    >
                      <svg
                        width="16"
                        height="16"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="2"
                        aria-hidden="true"
                      >
                        <path
                          d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"
                        />
                        <polyline points="17 21 17 13 7 13 7 21" />
                        <polyline points="7 3 7 8 15 8" />
                      </svg>
                    </button>
                  </span>
                </template>
                <button
                  v-else
                  type="button"
                  class="bib-display"
                  data-testid="bib-edit"
                  :aria-label="`Edit bib for ${racer.first_name} ${racer.last_name}`"
                  @click="startBibEdit(racer)"
                >
                  {{ racer.bib_number }}
                </button>
              </td>
              <td>{{ racer.first_name }} {{ racer.last_name }}</td>
              <td>{{ categoryLabel(racer) }}</td>
              <td>
                <select
                  class="team-select"
                  data-testid="racer-team-select"
                  :aria-label="`Team for ${racer.first_name} ${racer.last_name}`"
                  :value="racer.team_id ?? ''"
                  @change="onAssignTeam(racer, ($event.target as HTMLSelectElement).value)"
                >
                  <option value="">No team</option>
                  <option v-for="team in teams" :key="team.id" :value="team.id">
                    {{ team.name }}
                  </option>
                </select>
              </td>
              <td class="tag-count">
                {{ (racer.tag_uids?.length || 0) }}
                {{ (racer.tag_uids?.length || 0) === 1 ? 'tag' : 'tags' }}
              </td>
              <td>
                <button
                  type="button"
                  class="btn"
                  data-testid="program-tag"
                  @click="toggleProgram(racer.id)"
                >
                  Program tag
                </button>
              </td>
            </tr>
            <tr v-if="programmingId === racer.id" class="program-row">
              <td colspan="6">
                <div class="program-inline" data-testid="program-tag-panel">
                  <p class="muted">
                    Place a tag on the Proxmark3, then write. This programs this racer’s event bib
                    UUID onto the chip. Replacement tags get the same bib UUID.
                  </p>
                  <div class="row">
                    <button
                      type="button"
                      class="btn ok"
                      data-testid="program-tag-write"
                      :disabled="programming"
                      @click="writeTag(racer)"
                    >
                      {{ programming ? 'Writing…' : 'Write tag' }}
                    </button>
                    <button
                      type="button"
                      class="btn secondary"
                      data-testid="program-tag-done"
                      @click="programmingId = null"
                    >
                      Done
                    </button>
                  </div>
                  <p class="muted" data-testid="program-tag-list">
                    <template v-if="racer.tag_uids?.length">
                      Associated:
                      <span
                        v-for="uid in racer.tag_uids"
                        :key="uid"
                        class="tag-chip"
                      >
                        {{ uid }}
                      </span>
                    </template>
                    <template v-else>No tags yet</template>
                  </p>
                  <p v-if="programError" class="error" role="alert">{{ programError }}</p>
                </div>
              </td>
            </tr>
          </template>
          <tr v-if="!filteredRacers.length">
            <td colspan="6" class="muted">No racers match.</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { raceParticipantsApi, racesApi, raceTeamsApi, rfidApi } from '@/services/api'
import { usePinAuthStore } from '@/stores/pinAuth'
import type { Category, Participant, Race, Team } from '@/types/models'
import { getErrorMessage } from '@/utils/error'

const SEARCH_DEBOUNCE_MS = 200

const route = useRoute()
const router = useRouter()
const pinAuth = usePinAuthStore()

const raceId = computed(() => String(route.params.raceId || ''))

const race = ref<Race | null>(null)
const categories = ref<Category[]>([])
const racers = ref<Participant[]>([])
const teams = ref<Team[]>([])
const searchInput = ref('')
const searchQuery = ref('')
const showAdd = ref(false)
const saving = ref(false)
const formError = ref<string | null>(null)
const loadError = ref<string | null>(null)
const teamError = ref<string | null>(null)
const newTeamName = ref('')
const creatingTeam = ref(false)

const editingBibId = ref<string | null>(null)
const bibOriginal = ref('')
const bibDraft = ref('')
const bibDirty = computed(() => bibDraft.value.trim() !== bibOriginal.value)
const bibNoTagsWarn = ref<string | null>(null)

const programmingId = ref<string | null>(null)
const programming = ref(false)
const programError = ref<string | null>(null)

const addForm = reactive({
  first_name: '',
  last_name: '',
  gender: 'male',
  category_id: '',
  bib_number: '',
})

let searchTimer: ReturnType<typeof setTimeout> | undefined

const nextBibHint = computed(() => {
  let max = 0
  for (const r of racers.value) {
    const n = Number.parseInt(String(r.bib_number), 10)
    if (!Number.isNaN(n) && n > max) max = n
  }
  return `Auto: ${max + 1}`
})

const filteredRacers = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return racers.value
  return racers.value.filter((r) => {
    const hay = [
      r.bib_number,
      r.first_name,
      r.last_name,
      `${r.first_name} ${r.last_name}`,
      categoryLabel(r),
    ]
      .join(' ')
      .toLowerCase()
    return hay.includes(q)
  })
})

function categoryLabel(racer: Participant): string {
  if (racer.category?.name) return racer.category.name
  const cat = categories.value.find((c) => c.id === racer.category_id)
  return cat?.name ?? '—'
}

function memberCountForTeam(teamId: string): number {
  return racers.value.filter((r) => r.team_id === teamId).length
}

function normalizeTeamsResponse(data: unknown): Team[] {
  if (Array.isArray(data)) {
    return data as Team[]
  }
  if (data && typeof data === 'object' && Array.isArray((data as { data?: Team[] }).data)) {
    return (data as { data: Team[] }).data
  }
  return []
}

watch(searchInput, (value) => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    searchQuery.value = value
  }, SEARCH_DEBOUNCE_MS)
})

function startBibEdit(racer: Participant) {
  editingBibId.value = racer.id
  bibOriginal.value = racer.bib_number
  bibDraft.value = racer.bib_number
}

function cancelBibEdit() {
  editingBibId.value = null
  bibDraft.value = ''
  bibOriginal.value = ''
}

async function saveBib(racer: Participant) {
  if (!bibDirty.value) return
  if (!pinAuth.isAuthenticated) {
    await router.push('/pin')
    return
  }
  const next = bibDraft.value.trim() || bibOriginal.value
  const hasTags = Boolean(racer.tag_uids?.length)
  const raceActive = race.value?.status === 'active'
  if (hasTags || raceActive) {
    const reason = hasTags
      ? 'This racer already has programmed tags. Tags stay with the bib number.'
      : 'This race is active.'
    const ok = window.confirm(
      `${reason} Change bib from ${bibOriginal.value} to ${next}?`,
    )
    if (!ok) return
  }
  try {
    const { data } = await raceParticipantsApi.update(racer.id, { bib_number: next })
    const idx = racers.value.findIndex((r) => r.id === racer.id)
    if (idx >= 0) {
      racers.value[idx] = { ...racers.value[idx], ...data, bib_number: data.bib_number ?? next }
    }
    if (!(data.tag_uids?.length)) {
      bibNoTagsWarn.value = `Bib ${data.bib_number ?? next} has no programmed tags yet. Program a tag from this row when ready.`
    } else {
      bibNoTagsWarn.value = null
    }
    cancelBibEdit()
  } catch (err) {
    loadError.value = getErrorMessage(err, 'Failed to save bib')
  }
}

function toggleProgram(id: string) {
  programmingId.value = programmingId.value === id ? null : id
  programError.value = null
}

async function writeTag(racer: Participant) {
  if (!pinAuth.isAuthenticated) {
    await router.push('/pin')
    return
  }
  programming.value = true
  programError.value = null
  try {
    await rfidApi.writeTag({
      participant_id: racer.id,
      race_id: raceId.value,
      logical_uuid: racer.tag_uids?.[0] ?? racer.rfid_tag_uid ?? undefined,
    })
    const { data } = await raceParticipantsApi.listTags(raceId.value, racer.id)
    const tags = (data.data ?? []).map((t) => t.tag_uid)
    const idx = racers.value.findIndex((r) => r.id === racer.id)
    if (idx >= 0) {
      racers.value[idx] = {
        ...racers.value[idx],
        tag_uids: tags,
        rfid_tag_uid: tags[tags.length - 1] ?? racers.value[idx].rfid_tag_uid,
      }
    }
  } catch (err) {
    programError.value = getErrorMessage(err, 'Failed to write tag')
  } finally {
    programming.value = false
  }
}

async function onAddRacer() {
  formError.value = null
  if (!pinAuth.isAuthenticated) {
    await router.push('/pin')
    return
  }
  saving.value = true
  try {
    const payload: Parameters<typeof raceParticipantsApi.create>[1] = {
      first_name: addForm.first_name.trim(),
      last_name: addForm.last_name.trim(),
      gender: addForm.gender,
      category_id: addForm.category_id,
      status: 'registered',
    }
    if (addForm.bib_number.trim()) {
      payload.bib_number = addForm.bib_number.trim()
    }
    const { data } = await raceParticipantsApi.create(raceId.value, payload)
    racers.value = [...racers.value, data].sort((a, b) =>
      String(a.bib_number).localeCompare(String(b.bib_number), undefined, { numeric: true }),
    )
    showAdd.value = false
    addForm.first_name = ''
    addForm.last_name = ''
    addForm.bib_number = ''
    addForm.category_id = categories.value[0]?.id ?? ''
  } catch (err) {
    formError.value = getErrorMessage(err, 'Failed to add racer')
  } finally {
    saving.value = false
  }
}

async function onCreateTeam() {
  teamError.value = null
  if (!pinAuth.isAuthenticated) {
    await router.push('/pin')
    return
  }
  const name = newTeamName.value.trim()
  if (!name) return
  creatingTeam.value = true
  try {
    const { data } = await raceTeamsApi.create(raceId.value, { name })
    teams.value = [...teams.value, data].sort((a, b) =>
      a.name.localeCompare(b.name, undefined, { sensitivity: 'base' }),
    )
    newTeamName.value = ''
  } catch (err) {
    teamError.value = getErrorMessage(err, 'Failed to create team')
  } finally {
    creatingTeam.value = false
  }
}

async function onDeleteTeam(team: Team) {
  teamError.value = null
  if (!pinAuth.isAuthenticated) {
    await router.push('/pin')
    return
  }
  try {
    await raceTeamsApi.remove(team.id)
    teams.value = teams.value.filter((t) => t.id !== team.id)
    racers.value = racers.value.map((r) =>
      r.team_id === team.id ? { ...r, team_id: null, team: undefined } : r,
    )
  } catch (err) {
    teamError.value = getErrorMessage(err, 'Failed to delete team')
  }
}

async function onAssignTeam(racer: Participant, teamIdValue: string) {
  teamError.value = null
  if (!pinAuth.isAuthenticated) {
    await router.push('/pin')
    return
  }
  const nextTeamId = teamIdValue || null
  const previous = racer.team_id ?? null
  if (previous === nextTeamId) return

  const idx = racers.value.findIndex((r) => r.id === racer.id)
  if (idx >= 0) {
    const team = nextTeamId ? teams.value.find((t) => t.id === nextTeamId) : undefined
    racers.value[idx] = {
      ...racers.value[idx],
      team_id: nextTeamId,
      team: team ?? undefined,
    }
  }

  try {
    const { data } = await raceParticipantsApi.update(racer.id, { team_id: nextTeamId })
    if (idx >= 0) {
      racers.value[idx] = {
        ...racers.value[idx],
        ...data,
        team_id: data.team_id ?? nextTeamId,
      }
    }
  } catch (err) {
    if (idx >= 0) {
      const team = previous ? teams.value.find((t) => t.id === previous) : undefined
      racers.value[idx] = {
        ...racers.value[idx],
        team_id: previous,
        team: team ?? undefined,
      }
    }
    teamError.value = getErrorMessage(err, 'Failed to assign team')
  }
}

async function load() {
  loadError.value = null
  teamError.value = null
  try {
    const [raceRes, catsRes, listRes, teamsRes] = await Promise.all([
      racesApi.get(raceId.value),
      raceParticipantsApi.listCategories(raceId.value),
      raceParticipantsApi.list(raceId.value, { limit: 500 }),
      raceTeamsApi.list(raceId.value),
    ])
    race.value = raceRes.data
    categories.value = catsRes.data.data ?? []
    racers.value = listRes.data.data ?? []
    teams.value = normalizeTeamsResponse(teamsRes.data)
    if (!addForm.category_id && categories.value.length) {
      addForm.category_id = categories.value[0].id
    }
  } catch (err) {
    loadError.value = getErrorMessage(err, 'Failed to load racers')
  }
}

onMounted(() => {
  void load()
})

onUnmounted(() => {
  if (searchTimer) clearTimeout(searchTimer)
})
</script>

<style scoped>
.racers-page {
  max-width: 1100px;
  margin: 0 auto;
  padding: 0 1.5rem 3rem;
  --line: var(--border);
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

.meta-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem 1.5rem;
  align-items: center;
  margin: 1rem 0 1.25rem;
  font-size: 0.9rem;
  color: var(--muted);
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

.panel {
  background: var(--surface);
  border: 1px solid var(--line);
  border-radius: 6px;
  padding: 1rem 1.15rem;
  margin-bottom: 1rem;
}

.toolbar {
  justify-content: space-between;
  width: 100%;
}

.team-create {
  margin-bottom: 0.75rem;
  align-items: flex-end;
}

.team-select {
  min-width: 9rem;
  margin: 0;
}

.search-label {
  flex: 1;
  min-width: 200px;
  margin: 0;
}

.hint {
  margin: 0.5rem 0 0;
}

.muted {
  color: var(--muted);
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

label {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  font-size: 0.9rem;
  margin-bottom: 0.75rem;
}

label.grow {
  flex: 1;
  margin: 0;
}

input,
select {
  padding: 0.5rem 0.7rem;
  border: 1px solid var(--border);
  border-radius: 4px;
  font: inherit;
}

.row {
  display: flex;
  flex-wrap: wrap;
  gap: 0.6rem;
  align-items: flex-end;
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

.btn.ok {
  background: var(--success);
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

table {
  width: 100%;
  border-collapse: collapse;
}

th,
td {
  text-align: left;
  padding: 0.55rem 0.4rem;
  border-bottom: 1px solid var(--line);
  vertical-align: middle;
}

.bib-display {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.25rem 0.4rem;
  border-radius: 4px;
  border: 1px solid transparent;
  background: transparent;
  font: inherit;
  font-weight: 600;
  cursor: pointer;
  color: var(--ink);
}

.bib-display:hover {
  border-color: var(--line);
  background: var(--mist);
}

.bib-edit-wrap {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
}

.bib-edit-wrap input {
  width: 4rem;
  margin: 0;
  padding: 0.3rem 0.4rem;
}

.icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 2rem;
  height: 2rem;
  padding: 0;
  border: 1px solid var(--line);
  border-radius: 4px;
  background: var(--surface);
  cursor: pointer;
  color: var(--success);
}

.icon-btn:hover {
  background: color-mix(in srgb, var(--success) 12%, var(--surface));
}

tr.programming {
  background: color-mix(in srgb, var(--accent-link) 8%, var(--surface));
}

.program-inline {
  margin-top: 0.25rem;
  padding-top: 0.75rem;
}

.tag-chip {
  display: inline-block;
  margin-right: 0.5rem;
  font-family: ui-monospace, monospace;
}

.error {
  color: var(--signal);
}

.warn {
  color: var(--ink);
  background: color-mix(in srgb, var(--signal) 12%, var(--surface));
  border: 1px solid color-mix(in srgb, var(--signal) 35%, var(--line));
  border-radius: 4px;
  padding: 0.55rem 0.75rem;
  margin: 0 0 0.75rem;
}
</style>
