<template>
  <div class="racers-page" data-testid="racers-page">
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

    <h1 class="page-title" data-testid="event-racers-title">
      {{ eventName }} — All racers
    </h1>
    <p class="lead">
      Search and manage racers across every race in this event. PIN required for changes. Teams stay
      on each race’s racers page.
    </p>

    <div class="panel">
      <div class="row toolbar">
        <label class="search-label" for="event-racers-search">
          Search by name or bib
          <input
            id="event-racers-search"
            v-model="searchInput"
            type="search"
            data-testid="racers-search"
            placeholder="Type to filter…"
            aria-label="Search racers by name or bib"
            autocomplete="off"
          />
        </label>
        <label class="race-filter-label" for="event-racers-race-filter">
          Race
          <select
            id="event-racers-race-filter"
            v-model="raceFilter"
            data-testid="race-filter"
            aria-label="Filter by race"
          >
            <option value="">All races</option>
            <option v-for="r in races" :key="r.id" :value="r.id">{{ r.name }}</option>
          </select>
        </label>
        <button
          v-if="pinAuth.isAuthenticated"
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

    <div v-if="pinAuth.isAuthenticated && showAdd" class="panel" data-testid="add-racer-form">
      <h2>Add racer</h2>
      <form @submit.prevent="onAddRacer">
        <div class="grid-2">
          <label>
            Race
            <select
              v-model="addForm.race_id"
              data-testid="add-race-select"
              required
            >
              <option disabled value="">Select race…</option>
              <option v-for="r in races" :key="r.id" :value="r.id">{{ r.name }}</option>
            </select>
          </label>
          <label>
            Category
            <select
              v-model="addForm.category_id"
              data-testid="racer-category"
              required
              :disabled="!addForm.race_id"
            >
              <option disabled value="">Select category…</option>
              <option v-for="cat in addCategories" :key="cat.id" :value="cat.id">
                {{ cat.name }}
              </option>
            </select>
          </label>
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
            Bib number (optional — leave blank if unassigned)
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
      <div class="table-scroll">
        <table data-testid="racers-list">
          <thead>
            <tr>
              <th>Bib</th>
              <th>Name</th>
              <th>Race</th>
              <th>Category</th>
              <th>Team</th>
              <th>Tags</th>
              <th v-if="pinAuth.isAuthenticated">Actions</th>
            </tr>
          </thead>
          <tbody>
            <template v-for="racer in filteredRacers" :key="racer.id">
              <tr
                data-testid="racer-row"
                :class="{ programming: programmingId === racer.id }"
              >
                <td class="bib-cell">
                  <template v-if="pinAuth.isAuthenticated && !hasBib(racer)">
                    <input
                      v-model="unassignedBibDrafts[racer.id]"
                      type="text"
                      inputmode="numeric"
                      class="bib-assign-input"
                      data-testid="bib-assign-input"
                      placeholder="Bib #"
                      :aria-label="`Assign bib for ${racer.first_name} ${racer.last_name}`"
                      @keydown.enter.prevent="assignBib(racer)"
                      @keydown.escape="unassignedBibDrafts[racer.id] = ''"
                    />
                  </template>
                  <template v-else-if="pinAuth.isAuthenticated && editingBibId === racer.id">
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
                        Save
                      </button>
                    </span>
                  </template>
                  <button
                    v-else-if="pinAuth.isAuthenticated"
                    type="button"
                    class="bib-display"
                    data-testid="bib-edit"
                    :aria-label="`Edit bib for ${racer.first_name} ${racer.last_name}`"
                    @click="startBibEdit(racer)"
                  >
                    {{ racer.bib_number }}
                  </button>
                  <span v-else>{{ racer.bib_number || '—' }}</span>
                </td>
                <td>
                  {{ racer.first_name }} {{ racer.last_name }}
                  <span
                    v-if="racer.status === 'dns'"
                    class="dns-badge"
                    data-testid="racer-dns-badge"
                  >
                    DNS
                  </span>
                </td>
                <td>{{ raceLabel(racer) }}</td>
                <td>{{ categoryLabel(racer) }}</td>
                <td>{{ teamLabel(racer) }}</td>
                <td class="tag-count">
                  {{ (racer.tag_uids?.length || 0) }}
                  {{ (racer.tag_uids?.length || 0) === 1 ? 'tag' : 'tags' }}
                </td>
                <td v-if="pinAuth.isAuthenticated" class="actions-cell">
                  <button
                    type="button"
                    class="btn secondary"
                    data-testid="racer-edit"
                    @click="toggleEdit(racer)"
                  >
                    Edit
                  </button>
                  <button
                    type="button"
                    class="btn"
                    data-testid="program-tag"
                    @click="toggleProgram(racer)"
                  >
                    Program tag
                  </button>
                  <router-link
                    class="btn secondary"
                    :to="`/races/${racer.race_id}/racers`"
                    data-testid="racer-open-race"
                  >
                    Race page
                  </router-link>
                </td>
              </tr>
              <tr v-if="pinAuth.isAuthenticated && editingId === racer.id" class="program-row">
                <td :colspan="pinAuth.isAuthenticated ? 7 : 6">
                  <div class="program-inline" data-testid="racer-edit-panel">
                    <div class="grid-2">
                      <label>
                        First name
                        <input
                          v-model="editForm.first_name"
                          data-testid="racer-edit-first"
                          required
                        />
                      </label>
                      <label>
                        Last name
                        <input
                          v-model="editForm.last_name"
                          data-testid="racer-edit-last"
                          required
                        />
                      </label>
                      <label>
                        Race
                        <select
                          v-model="editForm.race_id"
                          data-testid="racer-edit-race"
                          required
                          @change="onEditRaceChange"
                        >
                          <option
                            v-for="race in races"
                            :key="race.id"
                            :value="race.id"
                          >
                            {{ race.name }}
                          </option>
                        </select>
                      </label>
                      <label>
                        Category
                        <select
                          v-model="editForm.category_id"
                          data-testid="racer-edit-category"
                          required
                        >
                          <option disabled value="">Select category…</option>
                          <option
                            v-for="cat in categoriesFor(editForm.race_id)"
                            :key="cat.id"
                            :value="cat.id"
                          >
                            {{ cat.name }}
                          </option>
                        </select>
                      </label>
                      <label>
                        Team
                        <select v-model="editForm.team_id" data-testid="racer-edit-team">
                          <option value="">No team</option>
                          <option
                            v-for="team in teamsFor(editForm.race_id)"
                            :key="team.id"
                            :value="team.id"
                          >
                            {{ team.name }}
                          </option>
                        </select>
                      </label>
                    </div>
                    <p v-if="editError" class="error" role="alert">{{ editError }}</p>
                    <div class="row">
                      <button
                        type="button"
                        class="btn"
                        data-testid="racer-edit-save"
                        :disabled="editSaving"
                        @click="saveEdit(racer)"
                      >
                        {{ editSaving ? 'Saving…' : 'Save' }}
                      </button>
                      <button
                        type="button"
                        class="btn secondary"
                        data-testid="racer-edit-cancel"
                        @click="closeEdit"
                      >
                        Cancel
                      </button>
                      <button
                        type="button"
                        class="btn danger"
                        data-testid="racer-delete"
                        :disabled="editSaving"
                        @click="deleteRacer(racer)"
                      >
                        Delete
                      </button>
                    </div>
                  </div>
                </td>
              </tr>
              <tr v-if="pinAuth.isAuthenticated && programmingId === racer.id" class="program-row">
                <td :colspan="pinAuth.isAuthenticated ? 7 : 6">
                  <div class="program-inline" data-testid="program-tag-panel">
                    <p class="muted">
                      Place a tag on the Proxmark3, then write. Programs this racer’s event bib UUID
                      onto the chip.
                    </p>
                    <p v-if="!hasBib(racer)" class="error" role="alert">
                      Assign a bib number first, then write the tag.
                    </p>
                    <div class="row">
                      <button
                        type="button"
                        class="btn ok"
                        data-testid="program-tag-write"
                        :disabled="programming || !hasBib(racer)"
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
                    <p v-if="programSuccess" class="success" role="status">{{ programSuccess }}</p>
                    <p v-if="programError" class="error" role="alert">{{ programError }}</p>
                  </div>
                </td>
              </tr>
            </template>
            <tr v-if="!filteredRacers.length">
              <td :colspan="pinAuth.isAuthenticated ? 7 : 6" class="muted">No racers match.</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  eventParticipantsApi,
  eventsApi,
  raceParticipantsApi,
  racesApi,
  raceTeamsApi,
  rfidApi,
} from '@/services/api'
import { usePinAuthStore } from '@/stores/pinAuth'
import type { Category, Participant, Race, Team } from '@/types/models'
import { getErrorMessage } from '@/utils/error'

const SEARCH_DEBOUNCE_MS = 200

const route = useRoute()
const router = useRouter()
const pinAuth = usePinAuthStore()

const eventId = computed(() => String(route.params.eventId || ''))
const eventName = ref('Event')
const races = ref<Race[]>([])
const racers = ref<Participant[]>([])
const categoriesByRace = reactive<Record<string, Category[]>>({})
const teamsByRace = reactive<Record<string, Team[]>>({})
const raceFilter = ref('')
const searchInput = ref('')
const searchQuery = ref('')
const showAdd = ref(false)
const saving = ref(false)
const formError = ref<string | null>(null)
const loadError = ref<string | null>(null)

const editingBibId = ref<string | null>(null)
const bibOriginal = ref('')
const bibDraft = ref('')
const bibDirty = computed(() => bibDraft.value.trim() !== bibOriginal.value)
const bibNoTagsWarn = ref<string | null>(null)
const unassignedBibDrafts = reactive<Record<string, string>>({})

const programmingId = ref<string | null>(null)
const programming = ref(false)
const programError = ref<string | null>(null)
const programSuccess = ref<string | null>(null)

const editingId = ref<string | null>(null)
const editSaving = ref(false)
const editError = ref<string | null>(null)
const editForm = reactive({
  first_name: '',
  last_name: '',
  race_id: '',
  category_id: '',
  team_id: '',
})

const addForm = reactive({
  race_id: '',
  first_name: '',
  last_name: '',
  gender: 'male',
  category_id: '',
  bib_number: '',
})

let searchTimer: ReturnType<typeof setTimeout> | undefined

const addCategories = computed(() => categoriesByRace[addForm.race_id] ?? [])

const nextBibHint = computed(() => {
  let max = 0
  for (const r of racers.value) {
    const n = Number.parseInt(String(r.bib_number), 10)
    if (!Number.isNaN(n) && n > max) max = n
  }
  return `Suggested: ${max + 1}`
})

function hasBib(racer: Participant): boolean {
  return Boolean(String(racer.bib_number ?? '').trim())
}

function raceLabel(racer: Participant): string {
  if (racer.race?.name) return racer.race.name
  return races.value.find((r) => r.id === racer.race_id)?.name ?? '—'
}

function categoryLabel(racer: Participant): string {
  if (racer.category?.name) return racer.category.name
  const cat = categoriesFor(racer.race_id).find((c) => c.id === racer.category_id)
  return cat?.name ?? '—'
}

function teamLabel(racer: Participant): string {
  return racer.team?.name || racer.team_name || '—'
}

function categoriesFor(raceId: string): Category[] {
  return categoriesByRace[raceId] ?? []
}

function teamsFor(raceId: string): Team[] {
  return teamsByRace[raceId] ?? []
}

const filteredRacers = computed(() => {
  let list = racers.value
  if (raceFilter.value) {
    list = list.filter((r) => r.race_id === raceFilter.value)
  }
  const q = searchQuery.value.trim().toLowerCase()
  if (!q) return list
  return list.filter((r) => {
    const hay = [
      r.bib_number,
      r.first_name,
      r.last_name,
      `${r.first_name} ${r.last_name}`,
      raceLabel(r),
      categoryLabel(r),
    ]
      .join(' ')
      .toLowerCase()
    return hay.includes(q)
  })
})

watch(searchInput, (value) => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    searchQuery.value = value
  }, SEARCH_DEBOUNCE_MS)
})

async function ensureCategories(raceId: string) {
  if (!raceId || categoriesByRace[raceId]) return
  const { data } = await raceParticipantsApi.listCategories(raceId)
  categoriesByRace[raceId] = data.data ?? []
}

async function ensureTeams(raceId: string) {
  if (!raceId || teamsByRace[raceId]) return
  const { data } = await raceTeamsApi.list(raceId)
  teamsByRace[raceId] = Array.isArray(data) ? data : (data.data ?? [])
}

watch(
  () => addForm.race_id,
  async (raceId) => {
    if (!raceId) {
      addForm.category_id = ''
      return
    }
    addForm.category_id = ''
    try {
      await ensureCategories(raceId)
      addForm.category_id = addCategories.value[0]?.id ?? ''
    } catch (err) {
      formError.value = getErrorMessage(err, 'Failed to load categories')
    }
  },
)

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

async function assignBib(racer: Participant) {
  if (!pinAuth.isAuthenticated) {
    await router.push('/pin')
    return
  }
  const next = String(unassignedBibDrafts[racer.id] ?? '').trim()
  if (!next) return
  try {
    const { data } = await raceParticipantsApi.update(racer.id, { bib_number: next })
    const idx = racers.value.findIndex((r) => r.id === racer.id)
    if (idx >= 0) {
      racers.value[idx] = { ...racers.value[idx], ...data, bib_number: data.bib_number ?? next }
    }
    delete unassignedBibDrafts[racer.id]
    if (!(data.tag_uids?.length)) {
      bibNoTagsWarn.value = `Bib ${data.bib_number ?? next} has no programmed tags yet. Program a tag from this row when ready.`
    } else {
      bibNoTagsWarn.value = null
    }
    loadError.value = null
  } catch (err) {
    loadError.value = getErrorMessage(err, 'Failed to assign bib')
  }
}

async function saveBib(racer: Participant) {
  if (!bibDirty.value) return
  if (!pinAuth.isAuthenticated) {
    await router.push('/pin')
    return
  }
  const next = bibDraft.value.trim()
  const unassigning = next === ''
  const hasTags = Boolean(racer.tag_uids?.length)
  const raceStatus =
    racer.race?.status ?? races.value.find((r) => r.id === racer.race_id)?.status
  const raceActive = raceStatus === 'active'
  if (unassigning || hasTags || raceActive) {
    const ok = window.confirm(
      unassigning
        ? `Unassign bib ${bibOriginal.value}? Tags stay with the bib.`
        : hasTags
          ? `This racer already has programmed tags. Tags stay with the bib number. Change bib from ${bibOriginal.value} to ${next}?`
          : `This race is active. Change bib from ${bibOriginal.value} to ${next}?`,
    )
    if (!ok) return
  }
  try {
    const { data } = await raceParticipantsApi.update(racer.id, { bib_number: next })
    const idx = racers.value.findIndex((r) => r.id === racer.id)
    if (idx >= 0) {
      racers.value[idx] = {
        ...racers.value[idx],
        ...data,
        bib_number: data.bib_number ?? next,
      }
    }
    if (unassigning) {
      bibNoTagsWarn.value = null
    } else if (!(data.tag_uids?.length)) {
      bibNoTagsWarn.value = `Bib ${data.bib_number ?? next} has no programmed tags yet. Program a tag from this row when ready.`
    } else {
      bibNoTagsWarn.value = null
    }
    cancelBibEdit()
  } catch (err) {
    loadError.value = getErrorMessage(err, 'Failed to save bib')
  }
}

async function toggleEdit(racer: Participant) {
  if (editingId.value === racer.id) {
    closeEdit()
    return
  }
  programmingId.value = null
  editingId.value = racer.id
  editForm.first_name = racer.first_name
  editForm.last_name = racer.last_name
  editForm.race_id = racer.race_id
  editForm.category_id = racer.category_id || ''
  editForm.team_id = racer.team_id || ''
  editError.value = null
  try {
    await Promise.all([ensureCategories(racer.race_id), ensureTeams(racer.race_id)])
  } catch (err) {
    editError.value = getErrorMessage(err, 'Failed to load race options')
  }
}

async function onEditRaceChange() {
  const raceId = editForm.race_id
  editForm.category_id = ''
  editForm.team_id = ''
  editError.value = null
  if (!raceId) return
  try {
    await Promise.all([ensureCategories(raceId), ensureTeams(raceId)])
    editForm.category_id = categoriesFor(raceId)[0]?.id ?? ''
  } catch (err) {
    editError.value = getErrorMessage(err, 'Failed to load race options')
  }
}

function closeEdit() {
  editingId.value = null
  editError.value = null
  editSaving.value = false
}

async function saveEdit(racer: Participant) {
  if (!pinAuth.isAuthenticated) {
    await router.push('/pin')
    return
  }
  editSaving.value = true
  editError.value = null
  try {
    const nextRaceId = editForm.race_id || racer.race_id
    const { data } = await raceParticipantsApi.update(racer.id, {
      first_name: editForm.first_name.trim(),
      last_name: editForm.last_name.trim(),
      race_id: nextRaceId,
      category_id: editForm.category_id || undefined,
      team_id: editForm.team_id || null,
    })
    const idx = racers.value.findIndex((r) => r.id === racer.id)
    if (idx >= 0) {
      const raceMeta = races.value.find((r) => r.id === (data.race_id || nextRaceId))
      const cat = categoriesFor(data.race_id || nextRaceId).find(
        (c) => c.id === (data.category_id || editForm.category_id),
      )
      const team = editForm.team_id
        ? teamsFor(data.race_id || nextRaceId).find((t) => t.id === editForm.team_id)
        : undefined
      racers.value[idx] = {
        ...racers.value[idx],
        ...data,
        category: cat ?? (data.category_id ? racers.value[idx].category : undefined),
        team: team ?? (data.team_id ? racers.value[idx].team : undefined),
        team_name: team?.name ?? (data.team_id ? racers.value[idx].team_name : undefined),
        race: raceMeta
          ? {
              ...(racers.value[idx].race ?? {
                id: raceMeta.id,
                event_id: raceMeta.event_id,
                name: raceMeta.name,
                race_type: raceMeta.race_type,
                status: raceMeta.status,
              }),
              id: raceMeta.id,
              name: raceMeta.name,
              event_id: raceMeta.event_id,
              race_type: raceMeta.race_type,
              status: raceMeta.status,
            }
          : racers.value[idx].race,
      }
    }
    closeEdit()
  } catch (err) {
    editError.value = getErrorMessage(err, 'Failed to update racer')
  } finally {
    editSaving.value = false
  }
}

async function deleteRacer(racer: Participant) {
  if (!pinAuth.isAuthenticated) {
    await router.push('/pin')
    return
  }
  const ok = window.confirm(
    `Remove ${racer.first_name} ${racer.last_name}? Racers with recorded laps become DNS instead of being deleted.`,
  )
  if (!ok) return
  editSaving.value = true
  editError.value = null
  try {
    const { data } = await raceParticipantsApi.remove(racer.id)
    if (data.action === 'dns' && data.participant) {
      const idx = racers.value.findIndex((r) => r.id === racer.id)
      if (idx >= 0) {
        racers.value[idx] = {
          ...racers.value[idx],
          ...data.participant,
          race: racers.value[idx].race,
        }
      }
      closeEdit()
    } else {
      racers.value = racers.value.filter((r) => r.id !== racer.id)
      closeEdit()
    }
  } catch (err) {
    editError.value = getErrorMessage(err, 'Failed to remove racer')
  } finally {
    editSaving.value = false
  }
}

function toggleProgram(racer: Participant) {
  editingId.value = null
  programmingId.value = programmingId.value === racer.id ? null : racer.id
  programError.value = null
  programSuccess.value = null
}

async function writeTag(racer: Participant) {
  if (!pinAuth.isAuthenticated) {
    await router.push('/pin')
    return
  }
  if (!hasBib(racer)) {
    programError.value = 'Assign a bib number first, then write the tag.'
    return
  }
  programming.value = true
  programError.value = null
  programSuccess.value = null
  const who = `Bib ${racer.bib_number} · ${racer.first_name} ${racer.last_name}`.trim()
  try {
    const { data: writeData } = await rfidApi.writeTag({
      participant_id: racer.id,
      race_id: racer.race_id,
      logical_uuid: racer.tag_uids?.[0] ?? racer.rfid_tag_uid ?? undefined,
    })
    const { data } = await raceParticipantsApi.listTags(racer.race_id, racer.id)
    const tags = (data.data ?? []).map((t) => t.tag_uid)
    const idx = racers.value.findIndex((r) => r.id === racer.id)
    if (idx >= 0) {
      racers.value[idx] = {
        ...racers.value[idx],
        tag_uids: tags,
        rfid_tag_uid: tags[tags.length - 1] ?? racers.value[idx].rfid_tag_uid,
      }
    }
    const uid =
      (writeData && typeof writeData === 'object' && 'logical_uuid' in writeData
        ? writeData.logical_uuid
        : undefined) ||
      tags[tags.length - 1] ||
      ''
    programSuccess.value = uid
      ? `WRITE OK — ${who} — verified read (${uid})`
      : `WRITE OK — ${who} — verified read`
  } catch (err) {
    programError.value = `WRITE FAILED — ${who} — ${getErrorMessage(err, 'write failed')}`
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
  if (!addForm.race_id) {
    formError.value = 'Select a race'
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
    const { data } = await raceParticipantsApi.create(addForm.race_id, payload)
    const race = races.value.find((r) => r.id === addForm.race_id)
    racers.value = [...racers.value, { ...data, race: data.race ?? race }].sort((a, b) =>
      String(a.bib_number).localeCompare(String(b.bib_number), undefined, { numeric: true }),
    )
    showAdd.value = false
    addForm.first_name = ''
    addForm.last_name = ''
    addForm.bib_number = ''
    addForm.category_id = addCategories.value[0]?.id ?? ''
  } catch (err) {
    formError.value = getErrorMessage(err, 'Failed to add racer')
  } finally {
    saving.value = false
  }
}

async function load() {
  loadError.value = null
  try {
    const [eventRes, racesRes, listRes] = await Promise.all([
      eventsApi.get(eventId.value),
      racesApi.list({ event_id: eventId.value, limit: 100 }),
      eventParticipantsApi.list(eventId.value, { limit: 500 }),
    ])
    eventName.value = eventRes.data.name
    races.value = racesRes.data.data ?? []
    racers.value = listRes.data.data ?? []
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
  width: 100%;
  max-width: min(1200px, 100%);
  margin: 0 auto;
  padding: 0 1rem 3rem;
  box-sizing: border-box;
  min-width: 0;
  --line: var(--border);
}

.page-title {
  margin: 0 0 0.35rem;
  color: var(--ink);
}

.lead {
  margin: 0 0 1.25rem;
  color: var(--muted);
  max-width: 42rem;
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
  font-weight: 600;
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

.search-label,
.race-filter-label {
  margin: 0;
}

.search-label {
  flex: 1;
  min-width: 200px;
}

.race-filter-label {
  min-width: 10rem;
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

.btn.danger {
  background: var(--signal);
  color: #fff;
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.table-scroll {
  overflow-x: auto;
}

table {
  width: 100%;
  min-width: 42rem;
  border-collapse: collapse;
}

th,
td {
  text-align: left;
  padding: 0.55rem 0.4rem;
  border-bottom: 1px solid var(--line);
  vertical-align: middle;
}

.bib-assign-input {
  width: 4.5rem;
  margin: 0;
  padding: 0.3rem 0.4rem;
}

.actions-cell {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
}

.dns-badge {
  display: inline-block;
  margin-left: 0.35rem;
  padding: 0.1rem 0.4rem;
  border-radius: 3px;
  font-size: 0.7rem;
  font-weight: 700;
  letter-spacing: 0.04em;
  color: var(--muted);
  background: color-mix(in srgb, var(--muted) 18%, var(--surface));
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
  padding: 0.25rem 0.5rem;
  border: 1px solid var(--line);
  border-radius: 4px;
  background: var(--surface);
  cursor: pointer;
  color: var(--success);
  font: inherit;
  font-size: 0.85rem;
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

.success {
  color: var(--success);
  font-weight: 600;
}

.error {
  color: var(--signal);
  font-weight: 600;
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
