<template>
  <div class="home">
    <section class="hero">
      <div class="hero-content">
        <h1 class="hero-title">Keweenaw Endurance Syndicate Race Timing</h1>
        <p class="hero-subtitle">Comprehensive race timing and indexing for endurance events</p>
        <router-link to="/timing" class="cta-button" data-testid="timing-cta">
          View Live Timing
        </router-link>
      </div>
    </section>

    <section class="featured-event" aria-labelledby="featured-event">
      <h2 id="featured-event">Featured Event</h2>
      <div class="featured-content">
        <a
          href="https://www.copperharbortrails.org/bluffet"
          target="_blank"
          rel="noopener noreferrer"
          class="featured-link"
          data-testid="bluffet-link"
        >
          All You Can East Bluffet
        </a>
        <p class="featured-description">
          Experience the ultimate trail running adventure in the beautiful Keweenaw Peninsula.
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
import RaceCard from '@/components/RaceCard.vue'

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
  background: #f8f9fa;
  padding: 3rem;
  border-radius: 12px;
  margin-bottom: 3rem;
  text-align: center;
}

.featured-event h2 {
  margin-bottom: 2rem;
  color: #2c3e50;
}

.featured-link {
  display: inline-block;
  font-size: 1.5rem;
  color: #e74c3c;
  text-decoration: none;
  font-weight: 600;
  margin-bottom: 1rem;
  transition: color 0.3s;
}

.featured-link:hover {
  color: #c0392b;
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
