<template>
  <div class="live-timing">
    <AppHeader />
    <h1 class="page-title">Live Timing</h1>
    <p class="placeholder">Live timing station — coming in US9.</p>
  </div>
</template>

<script setup lang="ts">
import AppHeader from '@/components/AppHeader.vue'
</script>

<style scoped>
.live-timing {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 2rem 2rem;
}

.page-title {
  color: #2c3e50;
}

.placeholder {
  color: #6c757d;
}
</style>
