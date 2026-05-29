<template>
  <article class="result-certificate" data-testid="result-certificate">
    <header class="certificate-section certificate-header">
      <h2 class="event-title">{{ eventTitle }}</h2>
      <div class="event-logo" aria-hidden="true">
        <span class="logo-text">{{ eventName }}</span>
      </div>
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
      <p class="timing-brand">Keweenaw Endurance</p>
      <p v-if="websiteUrl" class="website-url">{{ websiteUrl }}</p>
    </footer>
  </article>
</template>

<script setup lang="ts">
import {
  formatOrdinal,
  type ParticipantRankInfo,
} from '@/utils/participantResults'

defineProps<{
  eventTitle: string
  eventName: string
  eventDate: string
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
  websiteUrl?: string
}>()
</script>

<style scoped>
.result-certificate {
  max-width: 420px;
  margin: 0 auto;
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

.event-logo {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 220px;
  min-height: 72px;
  padding: 0.75rem 1rem;
  background: #f4c430;
  border: 1px solid #222;
}

.logo-text {
  font-size: 0.75rem;
  font-weight: 700;
  letter-spacing: 0.04em;
  text-transform: uppercase;
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
  margin: 0;
  font-size: 1.1rem;
  font-style: italic;
  font-weight: 700;
  color: #1f4f82;
}

.website-url {
  margin: 0.35rem 0 0;
  font-size: 0.9rem;
  font-weight: 700;
}
</style>
