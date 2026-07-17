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
        <p>{{ displayEventDate }}</p>
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

    <div
      ref="socialSquareRef"
      class="social-square-card"
      data-testid="social-square-card"
      aria-hidden="true"
    >
      <div class="social-square-inner">
        <div class="social-square-main">
          <header class="social-square-header">
            <EventLogo
              :logo-url="logoUrl"
              :alt="`${eventName} logo`"
              class="social-square-logo"
            />
            <div class="social-square-header-text">
              <p class="social-square-event">{{ eventName }}</p>
              <p class="social-square-race">{{ raceName }}</p>
              <p class="social-square-date">{{ displayEventDateLong }}</p>
            </div>
          </header>

          <div class="social-square-results">
            <section class="social-square-hero">
              <div class="social-square-athlete">
                <p class="social-square-bib">Bib {{ bibNumber }}</p>
                <h3 class="social-square-name">{{ participantName }}</h3>
                <p v-if="location" class="social-square-location">{{ location }}</p>
              </div>
              <div class="social-square-result">
                <p class="social-square-time">{{ finishTime }}</p>
                <p v-if="mph" class="social-square-mph">{{ mph }} mph</p>
              </div>
            </section>

            <section
              class="social-square-ranks"
              :class="{ 'social-square-ranks--single': !categoryRank }"
            >
              <div class="social-square-rank-card">
                <span class="social-square-rank-value">{{ formatOrdinal(overallRank.position) }}</span>
                <span class="social-square-rank-label">Overall</span>
              </div>
              <div v-if="categoryRank" class="social-square-rank-card">
                <span class="social-square-rank-value">{{ formatOrdinal(categoryRank.position) }}</span>
                <span class="social-square-rank-label">{{ categoryLabel }}</span>
              </div>
            </section>
          </div>
        </div>

        <footer class="social-square-brand">Inferior Timing</footer>
      </div>
    </div>

    <div class="certificate-toolbar">
      <div class="certificate-actions">
        <button
          type="button"
          class="save-btn"
          data-testid="save-certificate-image"
          :disabled="isSaving"
          @click="saveCertificateImage"
        >
          <span class="save-btn-label">{{ savingTarget === 'certificate' ? 'Saving…' : 'Save image' }}</span>
        </button>
        <button
          type="button"
          class="save-btn"
          data-testid="save-social-square-image"
          :disabled="isSaving"
          @click="saveSocialSquareImage"
        >
          <span class="save-btn-label">{{ savingTarget === 'social' ? 'Saving…' : 'Save square image' }}</span>
        </button>
      </div>
      <p v-if="saveError" class="save-error">{{ saveError }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import type { RouteLocationRaw } from 'vue-router'
import EventLogo from '@/components/EventLogo.vue'
import {
  formatEventDate,
  formatEventDateLong,
  formatOrdinal,
  type ParticipantRankInfo,
} from '@/utils/participantResults'
import { buildCertificateFilename, saveElementAsImage } from '@/utils/saveElementAsImage'

const SOCIAL_SQUARE_SIZE = 1080
const INK_DEEP_COLOR = '#203429'

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
const socialSquareRef = ref<HTMLElement | null>(null)
const savingTarget = ref<'certificate' | 'social' | null>(null)
const saveError = ref<string | null>(null)

const isSaving = computed(() => savingTarget.value !== null)

const displayEventDate = computed(() => formatEventDate(props.eventDate))
const displayEventDateLong = computed(() => formatEventDateLong(props.eventDate))

function resultsFilename(suffix: string): string {
  return buildCertificateFilename(
    props.eventName,
    props.participantName,
    `bib-${props.bibNumber}-${suffix}`,
  )
}

async function saveCertificateImage(): Promise<void> {
  if (!certificateRef.value || isSaving.value) {
    return
  }

  savingTarget.value = 'certificate'
  saveError.value = null

  try {
    await saveElementAsImage(
      certificateRef.value,
      resultsFilename('results'),
    )
  } catch {
    saveError.value = 'Could not save certificate image. Try again.'
  } finally {
    savingTarget.value = null
  }
}

async function saveSocialSquareImage(): Promise<void> {
  if (!socialSquareRef.value || isSaving.value) {
    return
  }

  savingTarget.value = 'social'
  saveError.value = null

  try {
    await saveElementAsImage(
      socialSquareRef.value,
      resultsFilename('social'),
      {
        backgroundColor: INK_DEEP_COLOR,
        scale: 1,
        width: SOCIAL_SQUARE_SIZE,
        height: SOCIAL_SQUARE_SIZE,
      },
    )
  } catch {
    saveError.value = 'Could not save square image. Try again.'
  } finally {
    savingTarget.value = null
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
  align-items: stretch;
  gap: 0.5rem;
  margin-top: 1rem;
}

.certificate-actions {
  display: flex;
  gap: 0.75rem;
}

.save-btn {
  flex: 1;
  padding: 0.65rem 1.25rem;
  border: none;
  border-radius: 4px;
  background: var(--ink);
  color: var(--surface);
  cursor: pointer;
  font: inherit;
  font-weight: 600;
  text-align: center;
}

.save-btn:hover:not(:disabled) {
  background: var(--ink-deep);
}

.save-btn:disabled {
  opacity: 0.7;
  cursor: wait;
}

.save-btn-label {
  font-size: 0.95rem;
  line-height: 1.3;
}

.save-error {
  margin: 0;
  color: var(--signal);
  font-size: 0.85rem;
  text-align: center;
}

.social-square-card {
  position: fixed;
  left: -10000px;
  top: 0;
  width: 1080px;
  height: 1080px;
  overflow: hidden;
  pointer-events: none;
  font-family: Arial, Helvetica, sans-serif;
}

.social-square-inner {
  display: flex;
  flex-direction: column;
  justify-content: flex-start;
  width: 100%;
  height: 100%;
  padding: 52px 64px 40px;
  box-sizing: border-box;
  color: #ffffff;
  background:
    radial-gradient(circle at top right, rgba(47, 107, 90, 0.28), transparent 42%),
    linear-gradient(160deg, #1a3f3d 0%, #203429 52%, #203429 100%);
}

.social-square-main {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 80px;
  min-height: 0;
}

.social-square-results {
  display: flex;
  flex-direction: column;
  gap: 28px;
  flex-shrink: 0;
}

.social-square-hero {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: end;
  gap: 24px;
  flex-shrink: 0;
}

.social-square-header {
  display: flex;
  align-items: center;
  gap: 32px;
  flex-shrink: 0;
}

.social-square-header :deep(.event-logo-image) {
  flex-shrink: 0;
  max-width: 176px;
  max-height: 176px;
}

.social-square-header-text {
  flex: 1;
  min-width: 0;
  text-align: left;
}

.social-square-event {
  margin: 0;
  font-size: 36px;
  font-weight: 700;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  line-height: 1.15;
}

.social-square-race {
  margin: 6px 0 0;
  font-size: 28px;
  font-weight: 600;
  color: #cbd5e1;
  line-height: 1.25;
}

.social-square-date {
  margin: 8px 0 0;
  font-size: 38px;
  font-weight: 700;
  color: #f8fafc;
  letter-spacing: 0.02em;
  line-height: 1.2;
}

.social-square-athlete {
  text-align: left;
  min-width: 0;
}

.social-square-bib {
  margin: 0;
  font-size: 28px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: #5eead4;
}

.social-square-name {
  margin: 8px 0 0;
  font-size: 80px;
  font-weight: 800;
  line-height: 1.02;
  word-wrap: break-word;
}

.social-square-location {
  margin: 6px 0 0;
  font-size: 28px;
  color: #cbd5e1;
}

.social-square-result {
  text-align: right;
  flex-shrink: 0;
}

.social-square-time {
  margin: 0;
  font-size: 104px;
  font-weight: 800;
  line-height: 1;
  letter-spacing: -0.02em;
  color: #ffffff;
  white-space: nowrap;
}

.social-square-mph {
  margin: 10px 0 0;
  font-size: 34px;
  font-weight: 600;
  color: #99f6e4;
}

.social-square-ranks {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
  padding-top: 24px;
  border-top: 1px solid rgba(255, 255, 255, 0.12);
}

.social-square-ranks--single {
  grid-template-columns: 1fr;
  max-width: 440px;
  margin: 0 auto;
  width: 100%;
}

.social-square-rank-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 40px 24px;
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.14);
  border-radius: 12px;
  text-align: center;
}

.social-square-rank-value {
  font-size: 88px;
  font-weight: 800;
  color: #ffffff;
  letter-spacing: -0.02em;
  line-height: 1;
}

.social-square-rank-label {
  font-size: 28px;
  font-weight: 600;
  color: #94a3b8;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  line-height: 1.2;
}

.social-square-brand {
  flex-shrink: 0;
  margin: 32px 0 0;
  text-align: right;
  font-size: 26px;
  font-style: italic;
  font-weight: 700;
  color: #7dd3fc;
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
  color: var(--accent-link);
  text-decoration: none;
}

.timing-brand:hover {
  text-decoration: underline;
}

@media (max-width: 480px) {
  .certificate-actions {
    flex-direction: column;
  }
}
</style>
