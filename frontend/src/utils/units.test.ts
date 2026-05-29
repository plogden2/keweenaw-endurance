import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import {
  convertDistanceFromKm,
  formatDistance,
  getDistanceAxisLabel,
  getDistanceUnitAbbreviation,
  KM_TO_MILES,
} from '@/utils/units'
import { useUnitsStore } from '@/stores/units'

describe('units utils', () => {
  it('converts kilometers to miles', () => {
    expect(convertDistanceFromKm(10, 'imperial')).toBeCloseTo(10 * KM_TO_MILES, 5)
    expect(convertDistanceFromKm(10, 'metric')).toBe(10)
  })

  it('formats distances with the selected unit', () => {
    expect(formatDistance(21.1, 'metric')).toBe('21.1 km')
    expect(formatDistance(21.1, 'imperial')).toBe('13.1 mi')
    expect(formatDistance(null, 'imperial')).toBe('—')
  })

  it('returns axis and abbreviation labels', () => {
    expect(getDistanceAxisLabel('imperial')).toBe('Distance (mi)')
    expect(getDistanceAxisLabel('metric')).toBe('Distance (km)')
    expect(getDistanceUnitAbbreviation('imperial')).toBe('mi')
    expect(getDistanceUnitAbbreviation('metric')).toBe('km')
  })
})

describe('units store', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
  })

  it('defaults to imperial units', () => {
    const store = useUnitsStore()
    expect(store.unitSystem).toBe('imperial')
  })

  it('persists unit preference to localStorage', () => {
    const store = useUnitsStore()

    store.setUnitSystem('metric')
    expect(store.unitSystem).toBe('metric')
    expect(localStorage.getItem('keweenaw-endurance-unit-system')).toBe('metric')

    store.toggleUnitSystem()
    expect(store.unitSystem).toBe('imperial')
  })
})
