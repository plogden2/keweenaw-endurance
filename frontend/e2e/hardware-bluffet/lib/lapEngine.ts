import { sampleLapDelayMs } from './clock'

export type LapState = {
  nextDue: Map<string, number>
  scored: number
}

export function initLapState(activeIds: string[], t0: number): LapState {
  const nextDue = new Map<string, number>()
  for (const id of activeIds) {
    nextDue.set(id, t0 + sampleLapDelayMs())
  }
  return { nextDue, scored: 0 }
}

export function dueRacers(state: LapState, now: number): string[] {
  return [...state.nextDue.entries()]
    .filter(([, due]) => due <= now)
    .sort((a, b) => a[1] - b[1])
    .map(([id]) => id)
}

export function scheduleNext(state: LapState, id: string, now: number) {
  state.nextDue.set(id, now + sampleLapDelayMs())
  state.scored += 1
}

export function removeRacer(state: LapState, id: string) {
  state.nextDue.delete(id)
}
