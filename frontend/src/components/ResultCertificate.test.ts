import { describe, it, expect, vi, beforeEach } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { mount, flushPromises } from '@vue/test-utils'
import ResultCertificate from './ResultCertificate.vue'
import { createTestRouter } from '@/test/helpers'
import { saveElementAsImage } from '@/utils/saveElementAsImage'

vi.mock('@/utils/saveElementAsImage', async () => {
  const actual = await vi.importActual<typeof import('@/utils/saveElementAsImage')>(
    '@/utils/saveElementAsImage',
  )

  return {
    ...actual,
    saveElementAsImage: vi.fn().mockResolvedValue(undefined),
  }
})

const router = createTestRouter()

const baseProps = {
  eventTitle: 'Copper Harbor Trails Fest - Long XC',
  eventName: 'Copper Harbor Trails Fest',
  eventDate: '2025-08-30T00:00:00Z',
  participantName: 'Peter Karinen',
  location: 'Tucson AZ',
  bibNumber: '788',
  raceName: 'Long XC',
  categoryLabel: 'M 25-29',
  finishTime: '02:10:29',
  mph: '13.4',
  overallRank: { position: 1, total: 140 },
  genderRank: { position: 1, total: 120 },
  categoryRank: { position: 1, total: 15 },
  genderRankLabel: 'Male Rank',
  categoryRankLabel: 'Category Rank',
  leaderboardTo: { name: 'race-details', params: { eventId: 'evt-1', raceId: 'race-1' } },
}

describe('ResultCertificate.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('does not hardcode legacy certificate chrome hex', () => {
    const src = readFileSync(join(process.cwd(), 'src/components/ResultCertificate.vue'), 'utf8')
    expect(src).not.toMatch(/#2c3e50|#152536|#1f2d3a/i)
    const style = src.split('<style')[1] ?? ''
    expect(style).toMatch(/var\(--ink\)/)
    expect(style).toMatch(/var\(--ink-deep\)/)
  })

  it('renders save image buttons', () => {
    const wrapper = mount(ResultCertificate, {
      props: baseProps,
      global: { plugins: [router] },
    })

    expect(wrapper.find('[data-testid="save-certificate-image"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="save-social-square-image"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('Save square image')
  })

  it('formats ISO event dates for display', () => {
    const wrapper = mount(ResultCertificate, {
      props: baseProps,
      global: { plugins: [router] },
    })

    expect(wrapper.find('.certificate-date').text()).toBe('08/30/2025')
    expect(wrapper.find('.social-square-date').text()).toBe('August 30, 2025')
  })

  it('renders prominent rank placement on the social square', () => {
    const wrapper = mount(ResultCertificate, {
      props: baseProps,
      global: { plugins: [router] },
    })

    const values = wrapper.findAll('.social-square-rank-value')
    const labels = wrapper.findAll('.social-square-rank-label')

    expect(values).toHaveLength(2)
    expect(labels).toHaveLength(2)
    expect(values[0].text()).toBe('1st')
    expect(labels[0].text()).toBe('Overall')
    expect(values[1].text()).toBe('1st')
    expect(labels[1].text()).toBe('M 25-29')
  })

  it('uses a split hero layout on the social square', () => {
    const wrapper = mount(ResultCertificate, {
      props: baseProps,
      global: { plugins: [router] },
    })

    expect(wrapper.find('.social-square-header').exists()).toBe(true)
    expect(wrapper.find('.social-square-main').exists()).toBe(true)
    expect(wrapper.find('.social-square-hero').exists()).toBe(true)
    expect(wrapper.findAll('.social-square-rank-card')).toHaveLength(2)
  })

  it('captures the certificate with an event-first filename', async () => {
    const wrapper = mount(ResultCertificate, {
      props: baseProps,
      global: { plugins: [router] },
    })

    await wrapper.find('[data-testid="save-certificate-image"]').trigger('click')
    await flushPromises()

    expect(saveElementAsImage).toHaveBeenCalledTimes(1)
    expect(saveElementAsImage).toHaveBeenCalledWith(
      wrapper.find('[data-testid="result-certificate"]').element,
      'copper-harbor-trails-fest-peter-karinen-bib-788-results.png',
    )
  })

  it('captures the social square card at 1080x1080', async () => {
    const wrapper = mount(ResultCertificate, {
      props: baseProps,
      global: { plugins: [router] },
    })

    await wrapper.find('[data-testid="save-social-square-image"]').trigger('click')
    await flushPromises()

    expect(saveElementAsImage).toHaveBeenCalledTimes(1)
    expect(saveElementAsImage).toHaveBeenCalledWith(
      wrapper.find('[data-testid="social-square-card"]').element,
      'copper-harbor-trails-fest-peter-karinen-bib-788-social.png',
      {
        backgroundColor: '#203429',
        scale: 1,
        width: 1080,
        height: 1080,
      },
    )
  })
})
