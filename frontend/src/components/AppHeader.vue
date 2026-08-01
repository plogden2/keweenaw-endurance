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
        <router-link
          v-if="pinAuth.isAuthenticated"
          to="/pin"
          class="nav-link"
          data-testid="nav-management"
        >
          Management
        </router-link>
      </div>
    </nav>
  </header>
</template>

<script setup lang="ts">
import { useBluffetTheme } from '@/composables/useBluffetTheme'
import { usePinAuthStore } from '@/stores/pinAuth'

const { active: bluffetActive, logoPath } = useBluffetTheme()
const pinAuth = usePinAuthStore()
</script>

<style scoped>
.header {
  background-color: var(--ink);
  color: var(--surface);
  padding: 1rem 0;
}

.nav {
  width: 100%;
  max-width: min(1200px, 100%);
  margin: 0 auto;
  padding: 0 1rem;
  box-sizing: border-box;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 0.75rem;
  min-width: 0;
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
  font-size: clamp(1.1rem, 4.5vw, 1.5rem);
  font-weight: 600;
  white-space: nowrap;
}

.bluffet-nav-mark {
  border: 2px solid currentColor;
  border-radius: 50%;
  object-fit: cover;
}

.nav-links {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem 1.25rem;
  justify-content: flex-end;
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

@media (min-width: 769px) {
  .nav {
    padding: 0 2rem;
  }
}

@media (max-width: 768px) {
  .nav {
    flex-direction: column;
    gap: 0.75rem;
  }

  .nav-links {
    gap: 0.5rem;
    justify-content: center;
    width: 100%;
  }

  .nav-link {
    padding: 0.4rem 0.75rem;
  }
}
</style>
