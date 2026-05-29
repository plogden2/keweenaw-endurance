import { describe, it, expect } from 'vitest'
import {
  buildCertificateFilename,
  buildImageFilename,
  slugifyFilenamePart,
} from './saveElementAsImage'

describe('slugifyFilenamePart', () => {
  it('slugifies text for filenames', () => {
    expect(slugifyFilenamePart('Peter Karinen')).toBe('peter-karinen')
  })
})

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

describe('buildCertificateFilename', () => {
  it('starts with the event name before the participant name', () => {
    expect(
      buildCertificateFilename(
        'Copper Harbor Trails Fest',
        'Peter Karinen',
        'bib-788-results',
      ),
    ).toBe('copper-harbor-trails-fest-peter-karinen-bib-788-results.png')
  })
})
