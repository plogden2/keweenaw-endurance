package bridge

import "errors"

var (
	ErrNotConnected = errors.New("bridge websocket not connected")
)

// ConnectionMode is the operator-facing bridge state.
type ConnectionMode string

const (
	ModeOffline      ConnectionMode = "offline"
	ModeSyncing      ConnectionMode = "syncing"
	ModeOnlineSynced ConnectionMode = "online_synced"
)
