import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import ScanPopup from './ScanPopup.vue'
import type { ScanResult } from '@/services/api'

const playMock = vi.fn().mockResolvedValue(undefined)

vi.stubGlobal(
  'Audio',
  vi.fn(function AudioMock(this: { play: typeof playMock; src: string }) {
    this.play = playMock
    this.src = ''
    return this
  }),
)

function lapScan(overrides: Partial<ScanResult> = {}): ScanResult {
  return {
    result: 'lap',
    participant_name: 'Alex Rivera',
    race_name: '12 Hour',
    placement: 3,
    lap_count: 14,
    timing_record_id: 'tr-1',
    karaoke_available: true,
    bib_number: '12',
    category_label: 'Expert Men',
    ...overrides,
  }
}

describe('ScanPopup.vue', () => {
  beforeEach(() => {
    playMock.mockClear()
  })

  it('renders scored lap with name, placement, and lap count testids', () => {
    const wrapper = mount(ScanPopup, {
      props: { scan: lapScan() },
    })

    expect(wrapper.find('[data-testid="scan-popup"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="scan-racer-name"]').text()).toContain('Alex Rivera')
    expect(wrapper.find('[data-testid="scan-placement"]').text()).toContain('3')
    expect(wrapper.find('[data-testid="scan-lap-count"]').text()).toContain('14')
    expect(wrapper.find('[data-testid="scan-race-name"]').text()).toContain('12 Hour')
    expect(wrapper.find('[data-testid="scan-team-line"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="karaoke-bonus-button"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="scan-popup-dismiss"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="scan-sound-playing"]').exists()).toBe(false)
    expect(wrapper.text().toLowerCase()).not.toMatch(/playing sound/)
  })

  it('shows team secondary line when team fields are present', () => {
    const wrapper = mount(ScanPopup, {
      props: {
        scan: lapScan({
          team_name: 'East Bluff A',
          team_placement: 2,
          team_avg_laps: 11.25,
        }),
      },
    })

    const line = wrapper.find('[data-testid="scan-team-line"]')
    expect(line.exists()).toBe(true)
    expect(line.text()).toBe('East Bluff A · #2 · 11.3 avg')
  })

  it('plays new-lap audio on lap without showing a sound label', async () => {
    mount(ScanPopup, { props: { scan: lapScan() } })
    expect(playMock).toHaveBeenCalled()
  })

  it('shows cooldown message for cooldown results', () => {
    const wrapper = mount(ScanPopup, {
      props: {
        scan: {
          result: 'cooldown',
          participant_name: 'Jordan Lee',
          retry_after_seconds: 42,
          lap_count: 12,
        },
      },
    })

    expect(wrapper.find('[data-testid="cooldown-message"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="scan-popup"]').exists()).toBe(false)
    expect(playMock).not.toHaveBeenCalled()
  })

  it('shows test-read message and hides scored popup', () => {
    const wrapper = mount(ScanPopup, {
      props: {
        scan: {
          result: 'test_read',
          participant_name: 'Sam Ortiz',
        },
      },
    })

    expect(wrapper.find('[data-testid="test-read-message"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="scan-popup"]').exists()).toBe(false)
  })

  it('shows unassigned bib toast with message and bib number', () => {
    const wrapper = mount(ScanPopup, {
      props: {
        scan: {
          result: 'unassigned_bib',
          bib_number: '77',
          message: 'Bib has no assigned racer',
        },
      },
    })

    const toast = wrapper.find('[data-testid="unassigned-bib-message"]')
    expect(toast.exists()).toBe(true)
    expect(toast.text()).toMatch(/unassigned bib/i)
    expect(toast.text()).toContain('77')
    expect(wrapper.find('[data-testid="scan-popup"]').exists()).toBe(false)
    expect(playMock).not.toHaveBeenCalled()
  })

  it('shows karaoke button when karaoke_available and recorded state after click', async () => {
    const wrapper = mount(ScanPopup, {
      props: { scan: lapScan({ karaoke_available: true }) },
    })

    const btn = wrapper.find('[data-testid="karaoke-bonus-button"]')
    expect(btn.exists()).toBe(true)
    await btn.trigger('click')
    expect(wrapper.emitted('karaoke')).toBeTruthy()
    expect(wrapper.find('[data-testid="karaoke-bonus-button"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="karaoke-bonus-recorded"]').text()).toContain(
      'Karaoke bonus lap recorded',
    )
  })

  it('hides karaoke button when karaoke_available is false', () => {
    const wrapper = mount(ScanPopup, {
      props: { scan: lapScan({ karaoke_available: false }) },
    })
    expect(wrapper.find('[data-testid="karaoke-bonus-button"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="karaoke-bonus-recorded"]').exists()).toBe(false)
  })

  it('shows out-of-order message without karaoke', () => {
    const wrapper = mount(ScanPopup, {
      props: {
        scan: {
          result: 'out_of_order',
          participant_name: 'Alex Rivera',
          message: 'Out of sequence — not yet a completed lap',
          karaoke_available: false,
        },
      },
    })
    expect(wrapper.find('[data-testid="out-of-order-message"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="out-of-order-message"]').text()).toMatch(/out of sequence/i)
    expect(wrapper.find('[data-testid="scan-popup"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="karaoke-bonus-button"]').exists()).toBe(false)
  })

  it('shows checkpoint_pass without karaoke', () => {
    const wrapper = mount(ScanPopup, {
      props: {
        scan: {
          result: 'checkpoint_pass',
          message: 'Checkpoint recorded — continue the sequence',
          karaoke_available: false,
        },
      },
    })
    expect(wrapper.find('[data-testid="checkpoint-pass-message"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="karaoke-bonus-button"]').exists()).toBe(false)
  })

  it('emits dismiss when dismiss button is clicked', async () => {
    const wrapper = mount(ScanPopup, { props: { scan: lapScan() } })
    await wrapper.find('[data-testid="scan-popup-dismiss"]').trigger('click')
    expect(wrapper.emitted('dismiss')).toBeTruthy()
  })

  it('shows discard confirm and emits discard on confirm', async () => {
    const wrapper = mount(ScanPopup, { props: { scan: lapScan() } })
    expect(wrapper.find('[data-testid="discard-lap-button"]').exists()).toBe(true)
    await wrapper.find('[data-testid="discard-lap-button"]').trigger('click')
    expect(wrapper.find('[data-testid="discard-confirm"]').exists()).toBe(true)
    await wrapper.find('[data-testid="discard-confirm-btn"]').trigger('click')
    expect(wrapper.emitted('discard')).toBeTruthy()
  })

  it('keeps lap when discard confirm is cancelled', async () => {
    const wrapper = mount(ScanPopup, { props: { scan: lapScan() } })
    await wrapper.find('[data-testid="discard-lap-button"]').trigger('click')
    await wrapper.find('[data-testid="discard-keep"]').trigger('click')
    expect(wrapper.find('[data-testid="discard-confirm"]').exists()).toBe(false)
    expect(wrapper.emitted('discard')).toBeFalsy()
    expect(wrapper.find('[data-testid="discard-lap-button"]').exists()).toBe(true)
  })
})
