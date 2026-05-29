<template>
  <div class="unit-toggle" data-testid="unit-toggle">
    <span class="unit-toggle-label">Units</span>
    <button
      type="button"
      class="unit-toggle-btn"
      :class="{ active: unitsStore.unitSystem === 'imperial' }"
      data-testid="unit-toggle-imperial"
      @click="unitsStore.setUnitSystem('imperial')"
    >
      Miles
    </button>
    <button
      type="button"
      class="unit-toggle-btn"
      :class="{ active: unitsStore.unitSystem === 'metric' }"
      data-testid="unit-toggle-metric"
      @click="unitsStore.setUnitSystem('metric')"
    >
      Metric
    </button>
  </div>
</template>

<script setup lang="ts">
import { useUnitsStore } from '@/stores/units'

const unitsStore = useUnitsStore()
</script>

<style scoped>
.unit-toggle {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
}

.unit-toggle-label {
  font-size: 0.875rem;
  opacity: 0.85;
}

.unit-toggle-btn {
  padding: 0.35rem 0.75rem;
  border: 1px solid rgba(255, 255, 255, 0.35);
  border-radius: 4px;
  background: transparent;
  color: inherit;
  cursor: pointer;
  font-size: 0.875rem;
  transition: background-color 0.2s, border-color 0.2s;
}

.unit-toggle-btn:hover {
  background: rgba(255, 255, 255, 0.1);
}

.unit-toggle-btn.active {
  background: rgba(255, 255, 255, 0.2);
  border-color: rgba(255, 255, 255, 0.6);
  font-weight: 600;
}
</style>
