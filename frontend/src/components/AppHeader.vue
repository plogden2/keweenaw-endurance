<template>
  <header class="header">
    <nav class="nav">
      <router-link to="/" class="logo">
        <img
          v-if="bluffetActive"
          :src="logoPath"
          alt=""
          class="bluffet-nav-mark"
          width="28"
          height="28"
          data-testid="bluffet-nav-mark"
        />
        <h1>Inferior Timing</h1>
      </router-link>
      <div class="nav-links">
        <router-link to="/" class="nav-link">Home</router-link>
        <router-link to="/timing" class="nav-link">Timing</router-link>
      </div>
    </nav>
  </header>
</template>

<script setup lang="ts">
import { useBluffetTheme } from '@/composables/useBluffetTheme'

const { active: bluffetActive, logoPath } = useBluffetTheme()
</script>

<style scoped>
.header {
  background-color: var(--ink);
  color: var(--surface);
  padding: 1rem 0;
}

.nav {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 2rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.logo {
  text-decoration: none;
  color: var(--surface);
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
}

.logo h1 {
  margin: 0;
  font-size: 1.5rem;
  font-weight: 600;
}

.bluffet-nav-mark {
  border: 2px solid currentColor;
  border-radius: 50%;
  object-fit: cover;
}

.nav-links {
  display: flex;
  gap: 2rem;
}

.nav-link {
  color: var(--surface);
  text-decoration: none;
  font-weight: 600;
  padding: 0.5rem 1rem;
  border-radius: 4px;
  transition: background-color 0.3s;
}

.nav-link:hover,
.nav-link.router-link-active {
  background-color: var(--sage);
  color: var(--ink);
}

@media (max-width: 768px) {
  .nav {
    flex-direction: column;
    gap: 1rem;
  }

  .nav-links {
    gap: 1rem;
  }
}
</style>
