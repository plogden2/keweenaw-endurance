<template>
  <article class="race-card" data-testid="race-card">
    <img
      :src="imageSrc"
      :alt="imageAlt"
      loading="lazy"
      class="race-image"
    />
    <h3 class="race-name">{{ name }}</h3>
    <a
      :href="externalUrl"
      target="_blank"
      rel="noopener noreferrer nofollow"
      class="race-link"
      data-testid="race-external-link"
    >
      View Details
    </a>
  </article>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    name: string
    externalUrl: string
    imageSrc?: string
    imageAlt?: string
  }>(),
  {
    imageSrc: '/images/race-placeholder.webp',
    imageAlt: '',
  },
)
</script>

<style scoped>
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
  color: var(--ink);
}

.race-link {
  display: inline-block;
  background: var(--accent);
  color: white;
  padding: 0.75rem 1.5rem;
  border-radius: 6px;
  text-decoration: none;
  font-weight: 600;
  transition: background-color 0.3s;
}

.race-link:hover {
  background: var(--accent-link);
}
</style>
