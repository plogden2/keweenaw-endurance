import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import ManualTimingForm from './ManualTimingForm.vue'
import type { Checkpoint } from '@/types/models'

const checkpoints: Checkpoint[] = [
  {
    id: 'cp-1',
    race_id: 'race-1',
    name: 'Start',
    checkpoint_type: 'start',
    is_active: true,
  },
  {
    id: 'cp-2',
    race_id: 'race-1',
    name: 'Finish',
    checkpoint_type: 'finish',
    is_active: true,
  },
]

describe('ManualTimingForm.vue', () => {
  it('renders checkpoint options', () => {
    const wrapper = mount(ManualTimingForm, {
      props: { raceId: 'race-1', checkpoints },
    })

    const options = wrapper.findAll('[data-testid="checkpoint-option"]')
    expect(options).toHaveLength(2)
    expect(wrapper.text()).toContain('Start')
    expect(wrapper.text()).toContain('Finish')
  })

  it('emits submit payload with bib number', async () => {
    const wrapper = mount(ManualTimingForm, {
      props: { raceId: 'race-1', checkpoints },
    })

    const select = wrapper.find('[data-testid="checkpoint-select"]')
    await select.setValue('cp-2')
    await wrapper.find('[data-testid="bib-input"]').setValue('42')
    await wrapper.find('form').trigger('submit')

    const events = wrapper.emitted('submit')
    expect(events).toBeTruthy()
    const payload = events![0]![0] as Record<string, string>
    expect(payload.race_id).toBe('race-1')
    expect(payload.checkpoint_id).toBe('cp-2')
    expect(payload.bib_number).toBe('42')
    expect(payload.timestamp).toBeTruthy()
  })

  it('emits submit payload with RFID tag when bib is empty', async () => {
    const wrapper = mount(ManualTimingForm, {
      props: { raceId: 'race-1', checkpoints },
    })

    await wrapper.find('[data-testid="checkpoint-select"]').setValue('cp-1')
    await wrapper.find('[data-testid="rfid-input"]').setValue('TAG-99')
    await wrapper.find('form').trigger('submit')

    const payload = wrapper.emitted('submit')![0]![0] as Record<string, string>
    expect(payload.rfid_tag_uid).toBe('TAG-99')
    expect(payload.bib_number).toBeUndefined()
  })

  it('shows validation error when checkpoint and identifiers are missing', async () => {
    const wrapper = mount(ManualTimingForm, {
      props: { raceId: 'race-1', checkpoints },
    })

    await wrapper.find('form').trigger('submit')

    expect(wrapper.text()).toMatch(/checkpoint/i)
    expect(wrapper.emitted('submit')).toBeFalsy()
  })
})
