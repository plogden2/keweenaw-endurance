import { describe, it, expect } from 'vitest'
import {
  deriveSyncChipState,
  syncChipLabel,
  type BridgeStatusSnapshot,
} from './bridgeSyncStatus'

describe('bridgeSyncStatus', () => {
  const base: BridgeStatusSnapshot = {
    navigatorOnline: true,
    hosted: { connected: true, pending_count: 0, syncing: false },
    local: { connected: true, pending_count: 0, syncing: false, mode: 'online_synced' },
  }

  it('maps navigator offline to Offline chip', () => {
    expect(
      syncChipLabel(
        deriveSyncChipState({ ...base, navigatorOnline: false }),
      ),
    ).toBe('Offline')
  })

  it('maps local bridge offline mode to Offline chip', () => {
    expect(
      syncChipLabel(
        deriveSyncChipState({
          ...base,
          local: { connected: false, pending_count: 2, syncing: false, mode: 'offline' },
        }),
      ),
    ).toBe('Offline')
  })

  it('prefers local offline over hosted still connected (partition lag)', () => {
    expect(
      deriveSyncChipState({
        navigatorOnline: true,
        hosted: { connected: true, pending_count: 0, syncing: false },
        local: { connected: false, pending_count: 4, syncing: false, mode: 'offline' },
      }),
    ).toBe('offline')
  })

  it('maps hosted disconnected with pending to Offline chip', () => {
    expect(
      syncChipLabel(
        deriveSyncChipState({
          ...base,
          hosted: { connected: false, pending_count: 3, syncing: false },
        }),
      ),
    ).toBe('Offline')
  })

  it('maps hosted syncing to Syncing chip', () => {
    expect(
      syncChipLabel(
        deriveSyncChipState({
          ...base,
          hosted: { connected: true, pending_count: 2, syncing: true },
        }),
      ),
    ).toBe('Syncing')
  })

  it('maps local syncing mode to Syncing chip', () => {
    expect(
      syncChipLabel(
        deriveSyncChipState({
          ...base,
          hosted: null,
          local: { connected: false, pending_count: 1, syncing: true, mode: 'syncing' },
        }),
      ),
    ).toBe('Syncing')
  })

  it('maps connected with zero pending to Online · Synced chip', () => {
    expect(
      syncChipLabel(
        deriveSyncChipState({
          ...base,
          hosted: { connected: true, pending_count: 0, syncing: false },
        }),
      ),
    ).toBe('Online · Synced')
  })

  it('maps hosted connected with pending to Syncing chip', () => {
    expect(
      syncChipLabel(
        deriveSyncChipState({
          ...base,
          hosted: { connected: true, pending_count: 3, syncing: false },
        }),
      ),
    ).toBe('Syncing')
  })
})
