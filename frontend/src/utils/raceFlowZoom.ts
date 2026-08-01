/** X-axis zoom helpers for race-flow charts (toolbar zoom; no freehand pan). */

export interface ZoomWindow {
  min: number
  max: number
}

/** Soft cap on painted datasets to protect mobile GPU/CPU. */
export const MAX_VISIBLE_DATASETS = 80

/** Minimum visible X span in minutes. */
export const MIN_ZOOM_SPAN_MINUTES = 1

export function createFullZoomWindow(fullMax: number, fullMin = 0): ZoomWindow {
  const min = fullMin
  const max = Math.max(fullMax, min + MIN_ZOOM_SPAN_MINUTES)
  return { min, max }
}

export function clampZoomWindow(
  window: ZoomWindow,
  fullMax: number,
  fullMin = 0,
): ZoomWindow {
  const domainMin = fullMin
  const domainMax = Math.max(fullMax, domainMin + MIN_ZOOM_SPAN_MINUTES)
  let { min, max } = window
  const span = Math.max(max - min, MIN_ZOOM_SPAN_MINUTES)

  if (span >= domainMax - domainMin - 1e-9) {
    return createFullZoomWindow(domainMax, domainMin)
  }

  const center = (min + max) / 2
  min = center - span / 2
  max = center + span / 2

  if (min < domainMin) {
    max += domainMin - min
    min = domainMin
  }
  if (max > domainMax) {
    min -= max - domainMax
    max = domainMax
  }
  if (min < domainMin) {
    min = domainMin
  }

  return { min, max: Math.max(max, min + MIN_ZOOM_SPAN_MINUTES) }
}

export function zoomInX(
  window: ZoomWindow,
  fullMax: number,
  factor = 0.7,
  centerX?: number,
  fullMin = 0,
): ZoomWindow {
  const span = Math.max(window.max - window.min, MIN_ZOOM_SPAN_MINUTES)
  const newSpan = Math.max(span * factor, MIN_ZOOM_SPAN_MINUTES)
  const center =
    centerX != null && Number.isFinite(centerX)
      ? centerX
      : (window.min + window.max) / 2
  return clampZoomWindow(
    { min: center - newSpan / 2, max: center + newSpan / 2 },
    fullMax,
    fullMin,
  )
}

export function zoomOutX(
  window: ZoomWindow,
  fullMax: number,
  factor = 1 / 0.7,
  centerX?: number,
  fullMin = 0,
): ZoomWindow {
  const span = Math.max(window.max - window.min, MIN_ZOOM_SPAN_MINUTES)
  const newSpan = span * factor
  const center =
    centerX != null && Number.isFinite(centerX)
      ? centerX
      : (window.min + window.max) / 2
  return clampZoomWindow(
    { min: center - newSpan / 2, max: center + newSpan / 2 },
    fullMax,
    fullMin,
  )
}

export function zoomToLastMinutes(
  fullMax: number,
  minutes: number,
  fullMin = 0,
): ZoomWindow {
  const domainMax = Math.max(fullMax, fullMin + MIN_ZOOM_SPAN_MINUTES)
  const domainSpan = domainMax - fullMin
  const span = Math.min(Math.max(minutes, MIN_ZOOM_SPAN_MINUTES), domainSpan)
  return clampZoomWindow(
    { min: domainMax - span, max: domainMax },
    domainMax,
    fullMin,
  )
}

export function isZoomAtFull(
  window: ZoomWindow,
  fullMax: number,
  fullMin = 0,
): boolean {
  const full = createFullZoomWindow(fullMax, fullMin)
  return (
    Math.abs(window.min - full.min) < 1e-6 && Math.abs(window.max - full.max) < 1e-6
  )
}

/**
 * Cap painted flows for performance. Sticky highlight is always kept;
 * remaining slots prefer higher progress (last point value).
 */
export function selectFlowsForRender<T extends { participantId: string; points: Array<{ value: number }> }>(
  flows: readonly T[],
  options: {
    maxDatasets?: number
    stickyParticipantId?: string | null
  } = {},
): { rendered: T[]; capped: boolean; totalVisible: number } {
  const maxDatasets = options.maxDatasets ?? MAX_VISIBLE_DATASETS
  const totalVisible = flows.length

  if (flows.length <= maxDatasets) {
    return { rendered: [...flows], capped: false, totalVisible }
  }

  const stickyId = options.stickyParticipantId ?? undefined
  const sticky = stickyId
    ? flows.find((flow) => flow.participantId === stickyId)
    : undefined

  const ranked = [...flows]
    .filter((flow) => flow.participantId !== stickyId)
    .sort((a, b) => {
      const aVal = a.points.at(-1)?.value ?? 0
      const bVal = b.points.at(-1)?.value ?? 0
      return bVal - aVal
    })

  const slots = sticky ? maxDatasets - 1 : maxDatasets
  const rendered = sticky
    ? [sticky, ...ranked.slice(0, slots)]
    : ranked.slice(0, slots)

  // Preserve original relative order among selected ids for stable dataset indexes.
  const keep = new Set(rendered.map((flow) => flow.participantId))
  return {
    rendered: flows.filter((flow) => keep.has(flow.participantId)),
    capped: true,
    totalVisible,
  }
}

/** Cap devicePixelRatio to reduce iOS Safari canvas memory pressure. */
export function resolveChartDevicePixelRatio(
  devicePixelRatio: number | undefined = typeof window !== 'undefined'
    ? window.devicePixelRatio
    : 1,
): number {
  const dpr = devicePixelRatio != null && Number.isFinite(devicePixelRatio) ? devicePixelRatio : 1
  return Math.min(Math.max(dpr, 1), 2)
}

export function prefersCoarsePointer(
  matchMediaFn?: ((query: string) => { matches: boolean }) | null,
): boolean {
  const matchMedia =
    matchMediaFn ??
    (typeof window !== 'undefined' && typeof window.matchMedia === 'function'
      ? (query: string) => window.matchMedia(query)
      : null)
  if (!matchMedia) {
    return false
  }
  try {
    return matchMedia('(pointer: coarse)').matches || matchMedia('(hover: none)').matches
  } catch {
    return false
  }
}
