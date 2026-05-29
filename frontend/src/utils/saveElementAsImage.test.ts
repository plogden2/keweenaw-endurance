import { describe, it, expect } from 'vitest'
import { buildImageFilename } from './saveElementAsImage'

describe('buildImageFilename', () => {
  it('slugifies the label and appends suffix', () => {
    expect(buildImageFilename('Peter Karinen', 'bib-788-results')).toBe(
      'peter-karinen-bib-788-results.png',
    )
  })

  it('falls back when label is empty', () => {
    expect(buildImageFilename('!!!', 'bib-1-results')).toBe('image-bib-1-results.png')
  })
})
