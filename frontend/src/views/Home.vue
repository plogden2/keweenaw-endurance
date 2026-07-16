<template>
  <div class="home">
    <section class="hero">
      <div class="hero-content">
        <h1 class="hero-title">Keweenaw Endurance Syndicate Race Timing</h1>
        <p class="hero-subtitle">Comprehensive race timing and indexing for endurance events</p>
        <router-link
          :to="liveTimingTarget"
          class="cta-button"
          data-testid="timing-cta"
        >
          View Live Timing
        </router-link>
      </div>
    </section>

    <section class="featured-event bluffet-theme" aria-labelledby="featured-event">
      <h2 id="featured-event">Featured Event</h2>
      <div class="featured-content">
        <picture data-testid="bluffet-poster" class="bluffet-poster">
          <source type="image/avif" :srcset="posterAvif" />
          <img
            :src="posterPng"
            alt="All You Can East Bluffet"
            class="featured-logo"
          />
        </picture>
        <p class="featured-date">Saturday, August 1, 2026 · East Bluff Bike Park, Copper Harbor, MI</p>
        <div class="featured-actions">
          <router-link
            v-if="bluffetEventId"
            :to="`/events/${bluffetEventId}/live`"
            class="featured-timing-link"
            data-testid="bluffet-timing-link"
          >
            Live race flow
          </router-link>
          <a
            href="https://www.copperharbortrails.org/bluffet"
            target="_blank"
            rel="noopener noreferrer"
            class="featured-link"
            data-testid="bluffet-link"
          >
            Register at copperharbortrails.org
          </a>
        </div>
        <p class="featured-description">
          Feast on the Copper Harbor Trails Club's newest event. A brand new endurance enduro — spin
          the wheel, shred the trails, and push your limits all day long!
        </p>
      </div>
    </section>

    <section class="upcoming-races" aria-labelledby="upcoming-races">
      <h2 id="upcoming-races">Upcoming Races</h2>
      <div class="race-grid">
        <RaceCard
          v-for="race in teaserRaces"
          :key="race.name"
          :name="race.name"
          :external-url="race.externalUrl"
          :image-src="race.imageSrc"
        />
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import RaceCard from '@/components/RaceCard.vue'
import { useEventsStore } from '@/stores/events'
import {
  BLUFFET_EVENT_NAME,
  BLUFFET_POSTER_AVIF,
  BLUFFET_POSTER_PNG,
} from '@/themes/bluffetConstants'

const eventsStore = useEventsStore()
const posterAvif = BLUFFET_POSTER_AVIF
const posterPng = BLUFFET_POSTER_PNG

const bluffetEventId = computed(() => {
  const event =
    eventsStore.events.find((e) => e.name === BLUFFET_EVENT_NAME) ??
    (eventsStore.currentEvent?.name === BLUFFET_EVENT_NAME
      ? eventsStore.currentEvent
      : undefined)
  return event?.id
})

/** Spectators land on event live view when Bluffet is known; else the events list. */
const liveTimingTarget = computed(() =>
  bluffetEventId.value ? `/events/${bluffetEventId.value}/live` : '/timing',
)

onMounted(() => {
  void eventsStore.fetchEvents({ limit: 100 })
})

interface TeaserRace {
  name: string
  externalUrl: string
  imageSrc: string
}

const teaserRaces: TeaserRace[] = [
  {
    name: 'Summer Trail Challenge',
    externalUrl: 'https://example.com/summer-trail',
    imageSrc: '/images/race-placeholder.webp',
  },
  {
    name: 'Fall Colors Marathon',
    externalUrl: 'https://example.com/fall-marathon',
    imageSrc: '/images/race-placeholder.webp',
  },
  {
    name: 'Winter Ultra Challenge',
    externalUrl: 'https://example.com/winter-ultra',
    imageSrc: '/images/race-placeholder.webp',
  },
]
</script>

<style scoped>
.home {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 2rem;
}

.hero {
  text-align: center;
  padding: 4rem 0;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  border-radius: 12px;
  margin-bottom: 3rem;
}

.hero-title {
  font-size: 3rem;
  margin-bottom: 1rem;
  font-weight: 700;
}

.hero-subtitle {
  font-size: 1.2rem;
  margin-bottom: 2rem;
  opacity: 0.9;
}

.cta-button {
  display: inline-block;
  background: white;
  color: #667eea;
  padding: 1rem 2rem;
  border-radius: 8px;
  text-decoration: none;
  font-weight: 600;
  transition: transform 0.3s, box-shadow 0.3s;
}

.cta-button:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 25px rgba(0, 0, 0, 0.15);
}

.featured-event {
  padding: 3rem;
  border-radius: 12px;
  margin-bottom: 3rem;
  text-align: center;
}

.featured-event h2 {
  margin-bottom: 2rem;
  color: #2c3e50;
}

.featured-logo {
  max-width: 480px;
  width: 100%;
  height: auto;
  margin-bottom: 1rem;
  border-radius: 8px;
}

.featured-date {
  color: #2c3e50;
  font-weight: 600;
  margin-bottom: 1rem;
}

.featured-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  justify-content: center;
  margin-bottom: 1rem;
}

.featured-timing-link {
  display: inline-block;
  font-size: 1.1rem;
  color: white;
  background: #e67e22;
  padding: 0.75rem 1.5rem;
  border-radius: 8px;
  text-decoration: none;
  font-weight: 600;
  transition: background-color 0.3s;
}

.featured-timing-link:hover {
  background: #d35400;
}

.featured-link {
  display: inline-block;
  font-size: 1.1rem;
  color: #e74c3c;
  text-decoration: none;
  font-weight: 600;
  padding: 0.75rem 1.5rem;
  border: 2px solid #e74c3c;
  border-radius: 8px;
  transition: color 0.3s, background-color 0.3s;
}

.featured-link:hover {
  color: white;
  background: #e74c3c;
}

.featured-description {
  color: #7f8c8d;
  font-size: 1.1rem;
}

.upcoming-races {
  margin-bottom: 3rem;
}

.upcoming-races h2 {
  text-align: center;
  margin-bottom: 2rem;
  color: #2c3e50;
}

.race-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 2rem;
}

@media (max-width: 768px) {
  .home {
    padding: 0 1rem;
  }

  .hero-title {
    font-size: 2rem;
  }

  .race-grid {
    grid-template-columns: 1fr;
  }
}
</style>
