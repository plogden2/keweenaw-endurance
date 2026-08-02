<template>
  <div class="category-filters" data-testid="leaderboard-category-filters">
    <div v-if="showSkill" class="filter-row">
      <span class="filter-label">Class</span>
      <div
        class="chip-toggle"
        data-testid="lb-filter-skill"
        role="group"
        aria-label="Class"
      >
        <button
          v-for="option in skillOptions"
          :key="option.value"
          type="button"
          :data-testid="`lb-filter-skill-${option.value}`"
          :aria-pressed="modelValue.skill === option.value"
          @click="setSkill(option.value)"
        >
          {{ option.label }}
        </button>
      </div>
    </div>

    <div v-if="showGender" class="filter-row">
      <span class="filter-label">Gender</span>
      <div
        class="chip-toggle"
        data-testid="lb-filter-gender"
        role="group"
        aria-label="Gender"
      >
        <button
          v-for="option in genderOptions"
          :key="option.value"
          type="button"
          :data-testid="`lb-filter-gender-${option.value}`"
          :aria-pressed="modelValue.gender === option.value"
          @click="setGender(option.value)"
        >
          {{ option.label }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type {
  GenderFilter,
  LeaderboardCategoryFilter,
  SkillFilter,
} from '@/utils/leaderboardCategoryFilter'

const props = defineProps<{
  modelValue: LeaderboardCategoryFilter
  showSkill: boolean
  showGender: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: LeaderboardCategoryFilter]
}>()

const skillOptions: { value: SkillFilter; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'expert', label: 'Expert' },
  { value: 'intermediate', label: 'Intermediate' },
]

const genderOptions: { value: GenderFilter; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'men', label: 'Men' },
  { value: 'women', label: 'Women' },
]

function setSkill(skill: SkillFilter) {
  emit('update:modelValue', { ...props.modelValue, skill })
}

function setGender(gender: GenderFilter) {
  emit('update:modelValue', { ...props.modelValue, gender })
}
</script>

<style scoped>
.category-filters {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem 1.25rem;
  margin: 0 0 1rem;
}

.filter-row {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
}

.filter-label {
  font-size: calc(0.85rem * var(--live-display-scale, 1));
  opacity: 0.85;
}

.chip-toggle {
  display: inline-flex;
  gap: 0;
  border: 1px solid var(--line);
  border-radius: 6px;
  overflow: hidden;
  background: var(--surface);
}

.chip-toggle button {
  border: none;
  background: transparent;
  padding: 0.4rem 0.9rem;
  cursor: pointer;
  font: inherit;
  font-size: calc(0.9rem * var(--live-display-scale, 1));
  color: var(--ink);
}

.chip-toggle button[aria-pressed='true'] {
  background: var(--accent);
  color: var(--surface);
}
</style>
