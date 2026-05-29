<template>
  <div id="app">
    <AppHeader v-if="showTimingHeader" />

    <main class="main">
      <router-view />
    </main>

    <footer class="footer">
      <div class="footer-content">
        <UnitToggle />
        <p>&copy; 2026 Keweenaw Endurance Syndicate. All rights reserved.</p>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import AppHeader from '@/components/AppHeader.vue'
import UnitToggle from '@/components/UnitToggle.vue'

const route = useRoute()
// Show Inferior Timing branding only on timing routes.
const showTimingHeader = computed(() => route.path.startsWith('/timing'))
</script>

<style scoped>
.main {
  min-height: calc(100vh - 80px);
  padding: 2rem 0;
}

.footer {
  background-color: #34495e;
  color: white;
  padding: 2rem 0;
  margin-top: auto;
}

.footer-content {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 2rem;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
}
</style>