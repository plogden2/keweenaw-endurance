import { describe, expect, it } from 'vitest'
import {
  clampZoomWindow,
  createFullZoomWindow,
  isZoomAtFull,
  MAX_VISIBLE_DATASETS,
  prefersCoarsePointer,
  resolveChartDevicePixelRatio,
  selectFlowsForRender,
  zoomInX,
  zoomOutX,
  zoomToLastMinutes,
} from './raceFlowZoom'

describe('raceFlowZoom', () => {
  it('creates a full window from 0 to fullMax', () => {
    expect(createFullZoomWindow(720)).toEqual({ min: 0, max: 720 })
  })

  it('zooms in around the window center and clamps to domain', () => {
    const next = zoomInX({ min: 0, max: 100 }, 100)
    expect(next.max - next.min).toBeLessThan(100)
    expect(next.min).toBeGreaterThanOrEqual(0)
    expect(next.max).toBeLessThanOrEqual(100)
  })

  it('zooms out until the full domain is restored', () => {
    let window = { min: 40, max: 60 }
    for (let i = 0; i < 8; i += 1) {
      window = zoomOutX(window, 100)
    }
    expect(isZoomAtFull(window, 100)).toBe(true)
    expect(clampZoomWindow(window, 100)).toEqual({ min: 0, max: 100 })
  })

  it('zooms to the last N minutes', () => {
    expect(zoomToLastMinutes(720, 60)).toEqual({ min: 660, max: 720 })
  })

  it('caps visible flows at MAX_VISIBLE_DATASETS and keeps sticky', () => {
    const flows = Array.from({ length: 100 }, (_, index) => ({
      participantId: `p${index}`,
      points: [{ value: index }],
    }))

    const { rendered, capped, totalVisible } = selectFlowsForRender(flows, {
      stickyParticipantId: 'p1',
    })

    expect(totalVisible).toBe(100)
    expect(capped).toBe(true)
    expect(rendered).toHaveLength(MAX_VISIBLE_DATASETS)
    expect(rendered.some((flow) => flow.participantId === 'p1')).toBe(true)
    // Highest progress racers preferred among non-sticky.
    expect(rendered.some((flow) => flow.participantId === 'p99')).toBe(true)
  })

  it('does not cap when under the limit', () => {
    const flows = [
      { participantId: 'a', points: [{ value: 1 }] },
      { participantId: 'b', points: [{ value: 2 }] },
    ]
    const result = selectFlowsForRender(flows)
    expect(result.capped).toBe(false)
    expect(result.rendered).toHaveLength(2)
  })

  it('caps devicePixelRatio at 2', () => {
    expect(resolveChartDevicePixelRatio(3)).toBe(2)
    expect(resolveChartDevicePixelRatio(1.5)).toBe(1.5)
  })

  it('detects coarse pointer via matchMedia', () => {
    expect(
      prefersCoarsePointer((query) => ({
        matches: query.includes('pointer: coarse'),
      })),
    ).toBe(true)
    expect(
      prefersCoarsePointer(() => ({ matches: false })),
    ).toBe(false)
  })
})
