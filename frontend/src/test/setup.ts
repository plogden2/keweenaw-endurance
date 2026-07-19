import 'fake-indexeddb/auto'
import { vi } from 'vitest'

vi.stubEnv('VITE_API_URL', '')

if (typeof PointerEvent === 'undefined') {
  class PointerEventPolyfill extends MouseEvent {
    readonly pointerId: number

    constructor(type: string, params: PointerEventInit = {}) {
      super(type, params)
      this.pointerId = params.pointerId ?? 0
    }
  }
  globalThis.PointerEvent = PointerEventPolyfill as typeof PointerEvent
}
