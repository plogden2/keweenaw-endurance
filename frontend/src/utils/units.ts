export type UnitSystem = 'imperial' | 'metric'

export const KM_TO_MILES = 0.621371

export function convertDistanceFromKm(km: number, unitSystem: UnitSystem): number {
  return unitSystem === 'imperial' ? km * KM_TO_MILES : km
}

export function getDistanceUnitAbbreviation(unitSystem: UnitSystem): string {
  return unitSystem === 'imperial' ? 'mi' : 'km'
}

export function getDistanceAxisLabel(unitSystem: UnitSystem): string {
  return unitSystem === 'imperial' ? 'Distance (mi)' : 'Distance (km)'
}

export function formatDistance(km: number | undefined | null, unitSystem: UnitSystem): string {
  if (km == null) {
    return '—'
  }

  const value = convertDistanceFromKm(km, unitSystem)
  const unit = getDistanceUnitAbbreviation(unitSystem)
  const formatted = value >= 10 ? value.toFixed(1) : value.toFixed(2)

  return `${formatted} ${unit}`
}
