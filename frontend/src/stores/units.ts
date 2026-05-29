import { defineStore } from 'pinia'
import type { UnitSystem } from '@/utils/units'

const STORAGE_KEY = 'keweenaw-endurance-unit-system'

function loadUnitSystem(): UnitSystem {
  if (typeof localStorage === 'undefined') {
    return 'imperial'
  }

  const stored = localStorage.getItem(STORAGE_KEY)
  return stored === 'metric' ? 'metric' : 'imperial'
}

interface UnitsState {
  unitSystem: UnitSystem
}

export const useUnitsStore = defineStore('units', {
  state: (): UnitsState => ({
    unitSystem: loadUnitSystem(),
  }),

  actions: {
    setUnitSystem(unitSystem: UnitSystem): void {
      this.unitSystem = unitSystem
      if (typeof localStorage !== 'undefined') {
        localStorage.setItem(STORAGE_KEY, unitSystem)
      }
    },

    toggleUnitSystem(): void {
      this.setUnitSystem(this.unitSystem === 'imperial' ? 'metric' : 'imperial')
    },
  },
})
