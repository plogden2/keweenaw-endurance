import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import ResultCertificate from './ResultCertificate.vue'
import { createTestRouter } from '@/test/helpers'
import { saveElementAsImage } from '@/utils/saveElementAsImage'

vi.mock('@/utils/saveElementAsImage', () => ({
  buildImageFilename: vi.fn(
    (label: string, suffix: string) => `${label}-${suffix}.png`,
  ),
  saveElementAsImage: vi.fn().mockResolvedValue(undefined),
}))

const router = createTestRouter()

const baseProps = {
  eventTitle: 'Copper Harbor Trails Fest - Long XC',
  eventName: 'Copper Harbor Trails Fest',
  eventDate: 'August 30, 2025',
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

  it('renders save image button', () => {
    const wrapper = mount(ResultCertificate, {
      props: baseProps,
      global: { plugins: [router] },
    })

    expect(wrapper.find('[data-testid="save-certificate-image"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="save-certificate-image"]').text()).toBe('Save image')
  })

  it('captures the certificate and downloads a PNG', async () => {
    const wrapper = mount(ResultCertificate, {
      props: baseProps,
      global: { plugins: [router] },
    })

    await wrapper.find('[data-testid="save-certificate-image"]').trigger('click')
    await flushPromises()

    expect(saveElementAsImage).toHaveBeenCalledTimes(1)
    expect(saveElementAsImage).toHaveBeenCalledWith(
      wrapper.find('[data-testid="result-certificate"]').element,
      'Peter Karinen-bib-788-results.png',
    )
  })
})
