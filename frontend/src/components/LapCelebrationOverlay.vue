<template>
  <div
    v-if="visible"
    class="lap-celebration"
    data-testid="lap-celebration"
    aria-live="polite"
  >
    <div class="sparkles" aria-hidden="true">★ ♥ ✦</div>
    <p class="eyebrow">New lap</p>
    <p class="name">{{ name }}</p>
    <span class="plus-one">+1</span>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  name: string
  visible: boolean
}>()
</script>

<style scoped>
.lap-celebration {
  position: absolute;
  top: 1rem;
  right: 1rem;
  pointer-events: none;
  z-index: 100;
  padding: 0.85rem 1.1rem;
  background: var(--bluffet-panel, #f7f3ea);
  color: var(--bluffet-ink, #1a1a1a);
  border: 2px solid var(--bluffet-ink, #1a1a1a);
  box-shadow: 4px 4px 0 var(--bluffet-ink, #1a1a1a);
  transform: rotate(-2deg);
  animation: lap-celebration-enter 300ms ease-out;
}

.sparkles {
  font-size: 0.75rem;
  color: var(--bluffet-red, #c8102e);
  letter-spacing: 0.15em;
  margin-bottom: 0.25rem;
}

.eyebrow {
  margin: 0;
  font-size: 0.7rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--bluffet-red, #c8102e);
}

.name {
  margin: 0.15rem 0 0.35rem;
  font-size: 1.1rem;
  font-weight: 700;
  line-height: 1.2;
}

.plus-one {
  display: inline-block;
  padding: 0.15rem 0.45rem;
  font-size: 0.85rem;
  font-weight: 800;
  background: var(--bluffet-red, #c8102e);
  color: var(--bluffet-panel, #f7f3ea);
  border-radius: 4px;
}

@keyframes lap-celebration-enter {
  from {
    opacity: 0;
    transform: rotate(-2deg) scale(0.85);
  }
  to {
    opacity: 1;
    transform: rotate(-2deg) scale(1);
  }
}
</style>
