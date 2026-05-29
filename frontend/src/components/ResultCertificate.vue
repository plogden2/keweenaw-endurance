<template>
  <div class="result-certificate-container">
    <article ref="certificateRef" class="result-certificate" data-testid="result-certificate">
    <header class="certificate-section certificate-header">
      <h2 class="event-title">{{ eventTitle }}</h2>
      <EventLogo
        :logo-url="logoUrl"
        :alt="`${eventName} logo`"
      />
    </header>

    <section class="certificate-section certificate-date">
      <p>{{ eventDate }}</p>
    </section>

    <section class="certificate-section certificate-participant">
      <h3 class="participant-name">{{ participantName }}</h3>
      <p v-if="location" class="participant-location">{{ location }}</p>
    </section>

    <section class="certificate-section certificate-details">
      <dl class="detail-rows">
        <div class="detail-row">
          <dt>Bib</dt>
          <dd>{{ bibNumber }}</dd>
        </div>
        <div class="detail-row">
          <dt>Event</dt>
          <dd>{{ raceName }}</dd>
        </div>
        <div class="detail-row">
          <dt>Category</dt>
          <dd>{{ categoryLabel }}</dd>
        </div>
      </dl>
    </section>

    <section class="certificate-section certificate-results">
      <h4 class="results-heading">Preliminary Results:</h4>
      <dl class="detail-rows">
        <div class="detail-row">
          <dt>Finish Time</dt>
          <dd><strong>{{ finishTime }}</strong></dd>
        </div>
        <div v-if="mph" class="detail-row">
          <dt>MPH</dt>
          <dd><strong>{{ mph }}</strong></dd>
        </div>
        <div class="detail-row">
          <dt>Overall Rank</dt>
          <dd>
            <strong>{{ formatOrdinal(overallRank.position) }}</strong>
            out of {{ overallRank.total }}
          </dd>
        </div>
        <div v-if="genderRank" class="detail-row">
          <dt>{{ genderRankLabel }}</dt>
          <dd>
            <strong>{{ formatOrdinal(genderRank.position) }}</strong>
            out of {{ genderRank.total }}
          </dd>
        </div>
        <div v-if="categoryRank" class="detail-row">
          <dt>{{ categoryRankLabel }}</dt>
          <dd>
            <strong>{{ formatOrdinal(categoryRank.position) }}</strong>
            out of {{ categoryRank.total }}
          </dd>
        </div>
      </dl>
    </section>

    <footer class="certificate-section certificate-footer">
      <p class="footer-note">Official Results available at</p>
      <RouterLink
        :to="leaderboardTo"
        class="timing-brand"
        data-testid="inferior-timing-link"
        @click="$emit('viewLeaderboard')"
      >
        Inferior Timing
      </RouterLink>
    </footer>
    </article>

    <div class="certificate-toolbar">
      <button
        type="button"
        class="save-image-btn"
        data-testid="save-certificate-image"
        :disabled="saving"
        @click="saveImage"
      >
        {{ saving ? 'Saving…' : 'Save image' }}
      </button>
      <p v-if="saveError" class="save-error">{{ saveError }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { RouteLocationRaw } from 'vue-router'
import EventLogo from '@/components/EventLogo.vue'
import {
  formatOrdinal,
  type ParticipantRankInfo,
} from '@/utils/participantResults'
import { buildImageFilename, saveElementAsImage } from '@/utils/saveElementAsImage'

const props = defineProps<{
  eventTitle: string
  eventName: string
  eventDate: string
  logoUrl?: string
  participantName: string
  location?: string
  bibNumber: string
  raceName: string
  categoryLabel: string
  finishTime: string
  mph?: string | null
  overallRank: ParticipantRankInfo
  genderRank?: ParticipantRankInfo | null
  categoryRank?: ParticipantRankInfo | null
  genderRankLabel?: string
  categoryRankLabel?: string
  leaderboardTo: RouteLocationRaw
}>()

defineEmits<{
  viewLeaderboard: []
}>()

const certificateRef = ref<HTMLElement | null>(null)
const saving = ref(false)
const saveError = ref<string | null>(null)

async function saveImage(): Promise<void> {
  if (!certificateRef.value || saving.value) {
    return
  }

  saving.value = true
  saveError.value = null

  try {
    const filename = buildImageFilename(
      props.participantName,
      `bib-${props.bibNumber}-results`,
    )
    await saveElementAsImage(certificateRef.value, filename)
  } catch {
    saveError.value = 'Could not save image. Try again.'
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.result-certificate-container {
  max-width: 420px;
  margin: 0 auto;
}

.certificate-toolbar {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
  margin-top: 0.75rem;
}

.save-image-btn {
  padding: 0.55rem 1rem;
  border: 1px solid #2c3e50;
  border-radius: 4px;
  background: #2c3e50;
  color: white;
  cursor: pointer;
  font: inherit;
  font-weight: 600;
}

.save-image-btn:hover:not(:disabled) {
  background: #1f2d3a;
}

.save-image-btn:disabled {
  opacity: 0.7;
  cursor: wait;
}

.save-error {
  margin: 0;
  color: #dc3545;
  font-size: 0.85rem;
}

.result-certificate {
  background: white;
  border: 1px solid #222;
  font-family: Arial, Helvetica, sans-serif;
  color: #111;
}

.certificate-section {
  padding: 1rem 1.25rem;
  border-bottom: 1px solid #222;
  text-align: center;
}

.certificate-section:last-child {
  border-bottom: none;
}

.certificate-header {
  padding-top: 1.25rem;
}

.event-title {
  margin: 0 0 0.75rem;
  font-size: 1rem;
  font-weight: 700;
  line-height: 1.35;
}

.certificate-date p {
  margin: 0;
  font-size: 1.75rem;
  font-weight: 700;
}

.participant-name {
  margin: 0;
  font-size: 1.5rem;
  font-weight: 700;
}

.participant-location {
  margin: 0.35rem 0 0;
  font-size: 0.95rem;
}

.certificate-details,
.certificate-results {
  text-align: left;
}

.results-heading {
  margin: 0 0 0.75rem;
  font-size: 1.1rem;
  font-weight: 700;
}

.detail-rows {
  margin: 0;
}

.detail-row {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.35rem 0;
  font-size: 0.95rem;
}

.detail-row dt {
  margin: 0;
  font-weight: 400;
}

.detail-row dd {
  margin: 0;
  text-align: right;
}

.certificate-footer {
  padding-bottom: 1.25rem;
}

.footer-note {
  margin: 0 0 0.5rem;
  font-size: 0.8rem;
}

.timing-brand {
  display: inline-block;
  margin: 0;
  font-size: 1.1rem;
  font-style: italic;
  font-weight: 700;
  color: #1f4f82;
  text-decoration: none;
}

.timing-brand:hover {
  text-decoration: underline;
}
</style>
