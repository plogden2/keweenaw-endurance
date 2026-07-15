export function sampleLapDelayMs(rng = Math.random): number {
  const u = rng()
  const ms = 30_000 + u * u * 150_000
  return Math.min(180_000, Math.max(30_000, Math.round(ms)))
}

export function sleep(ms: number) {
  return new Promise((r) => setTimeout(r, ms))
}
