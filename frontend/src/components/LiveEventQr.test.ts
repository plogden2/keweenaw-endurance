import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import LiveEventQr from './LiveEventQr.vue'

vi.mock('qrcode', () => ({
  default: {
    toCanvas: vi.fn(async (_canvas: HTMLCanvasElement, _text: string) => undefined),
  },
}))

import QRCode from 'qrcode'

describe('LiveEventQr', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders caption and generates QR for the given url', async () => {
    const wrapper = mount(LiveEventQr, {
      props: { url: 'https://keweenawendurance.com/events/evt-1/live' },
    })
    await flushPromises()
    expect(wrapper.text()).toContain('view results at keweenawendurance.com')
    expect(wrapper.find('[data-testid="rotator-live-qr"]').exists()).toBe(true)
    expect(QRCode.toCanvas).toHaveBeenCalled()
    const [, text] = (QRCode.toCanvas as ReturnType<typeof vi.fn>).mock.calls[0]!
    expect(text).toBe('https://keweenawendurance.com/events/evt-1/live')
  })
})
