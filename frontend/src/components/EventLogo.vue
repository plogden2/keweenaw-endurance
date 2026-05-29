<template>
  <div v-if="logoUrl" class="event-logo" data-testid="event-logo">
    <img :src="logoUrl" :alt="alt" class="event-logo-image" crossorigin="anonymous" />
  </div>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    logoUrl?: string
    alt?: string
  }>(),
  {
    alt: 'Event logo',
  },
)
</script>

<style scoped>
.event-logo {
  display: flex;
  align-items: center;
  justify-content: center;
}

.event-logo-image {
  max-width: 220px;
  max-height: 220px;
  width: auto;
  height: auto;
}

.event-logo-large .event-logo-image {
  max-width: 280px;
  max-height: 280px;
}
</style>
