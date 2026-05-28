<template>
  <div class="home">
    <section class="hero">
      <div class="hero-content">
        <h1 class="hero-title">Keweenaw Endurance Race Timing</h1>
        <p class="hero-subtitle">Comprehensive race timing and indexing for endurance events</p>
        <router-link to="/timing" class="cta-button">View Live Timing</router-link>
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
        <article class="race-card" data-testid="race-card">
          <img 
            src="/images/race-placeholder.webp" 
            alt="" 
            loading="lazy"
            class="race-image"
          />
          <h3 class="race-name">Summer Trail Challenge</h3>
          <a 
            href="https://example.com/summer-trail" 
            target="_blank" 
            rel="noopener noreferrer nofollow"
            class="race-link"
          >
            View Details
          </a>
        </article>

        <article class="race-card" data-testid="race-card">
          <img 
            src="/images/race-placeholder.webp" 
            alt="" 
            loading="lazy"
            class="race-image"
          />
          <h3 class="race-name">Fall Colors Marathon</h3>
          <a 
            href="https://example.com/fall-marathon" 
            target="_blank" 
            rel="noopener noreferrer nofollow"
            class="race-link"
          >
            View Details
          </a>
        </article>

        <article class="race-card" data-testid="race-card">
          <img 
            src="/images/race-placeholder.webp" 
            alt="" 
            loading="lazy"
            class="race-image"
          />
          <h3 class="race-name">Winter Ultra Challenge</h3>
          <a 
            href="https://example.com/winter-ultra" 
            target="_blank" 
            rel="noopener noreferrer nofollow"
            class="race-link"
          >
            View Details
          </a>
        </article>
      </div>
    </section>
  </div>
</template>

<script setup>
// Home component - minimal teaser approach as specified
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

.race-card {
  background: white;
  border-radius: 12px;
  padding: 1.5rem;
  box-shadow: 0 4px 15px rgba(0, 0, 0, 0.1);
  transition: transform 0.3s, box-shadow 0.3s;
}

.race-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 30px rgba(0, 0, 0, 0.15);
}

.race-image {
  width: 100%;
  height: 200px;
  object-fit: cover;
  border-radius: 8px;
  margin-bottom: 1rem;
}

.race-name {
  font-size: 1.3rem;
  margin-bottom: 1rem;
  color: #2c3e50;
}

.race-link {
  display: inline-block;
  background: #3498db;
  color: white;
  padding: 0.75rem 1.5rem;
  border-radius: 6px;
  text-decoration: none;
  font-weight: 500;
  transition: background-color 0.3s;
}

.race-link:hover {
  background: #2980b9;
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