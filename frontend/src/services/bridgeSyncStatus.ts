export type BridgeStatusResponse = {
  connected: boolean
  pending_count: number
  syncing: boolean
  last_sync_at?: string | null
}

export type LocalBridgeStatusResponse = BridgeStatusResponse & {
  mode?: 'offline' | 'syncing' | 'online_synced' | string
}

export type SyncChipState = 'offline' | 'syncing' | 'online_synced'

export type BridgeStatusSnapshot = {
  navigatorOnline: boolean
  hosted: BridgeStatusResponse | null
  local: LocalBridgeStatusResponse | null
}

export function deriveSyncChipState(snapshot: BridgeStatusSnapshot): SyncChipState {
  const { navigatorOnline, hosted, local } = snapshot
  const localMode = local?.mode

  if (!navigatorOnline) return 'offline'
  if (localMode === 'offline') return 'offline'
  if (hosted && !hosted.connected && hosted.pending_count > 0) return 'offline'

  if (hosted?.syncing || local?.syncing || localMode === 'syncing') return 'syncing'

  if (hosted?.connected && hosted.pending_count === 0) return 'online_synced'
  if (hosted?.connected && hosted.pending_count > 0) return 'syncing'
  if (localMode === 'online_synced') return 'online_synced'

  return 'offline'
}

export function syncChipLabel(state: SyncChipState): string {
  switch (state) {
    case 'offline':
      return 'Offline'
    case 'syncing':
      return 'Syncing'
    case 'online_synced':
      return 'Online · Synced'
  }
}

export function isBridgeOffline(snapshot: BridgeStatusSnapshot): boolean {
  return deriveSyncChipState(snapshot) === 'offline'
}
