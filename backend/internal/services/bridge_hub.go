package services

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	DefaultBridgeDeviceID     = "laptop-finish-1"
	defaultBridgeWriteTimeout = 20 * time.Second
)

var (
	ErrBridgeUnavailable  = errors.New("bridge unavailable")
	ErrBridgeWriteTimeout = errors.New("bridge write timed out")
)

// BridgeConn is the subset of websocket.Conn used by BridgeHub.
type BridgeConn interface {
	WriteJSON(v any) error
	ReadJSON(v any) error
	Close() error
}

// BridgeMessage is JSON traffic on /api/rfid/bridge.
type BridgeMessage struct {
	Type         string  `json:"type"`
	RequestID    string  `json:"request_id,omitempty"`
	LogicalUUID  string  `json:"logical_uuid,omitempty"`
	SourceLapID  string  `json:"source_lap_id,omitempty"`
	OK           *bool   `json:"ok,omitempty"`
	Error        string  `json:"error,omitempty"`
	TS           string  `json:"ts,omitempty"`
	PendingCount *int    `json:"pending_count,omitempty"`
	Syncing      *bool   `json:"syncing,omitempty"`
	LastSyncAt   *string `json:"last_sync_at,omitempty"`
}

// BridgeStatus is returned by GET /api/rfid/bridge/status.
type BridgeStatus struct {
	Connected    bool       `json:"connected"`
	PendingCount int        `json:"pending_count"`
	Syncing      bool       `json:"syncing"`
	LastSyncAt   *time.Time `json:"last_sync_at,omitempty"`
}

type writeAckResult struct {
	ok     bool
	errMsg string
}

type pendingWrite struct {
	ch chan writeAckResult
}

type deviceState struct {
	conn          BridgeConn
	writeMu       sync.Mutex
	pendingWrites map[string]*pendingWrite
	status        BridgeStatus
}

// BridgeHub tracks one WebSocket per device_id and dispatches write commands.
type BridgeHub struct {
	mu      sync.RWMutex
	devices map[string]*deviceState
}

func NewBridgeHub() *BridgeHub {
	return &BridgeHub{
		devices: make(map[string]*deviceState),
	}
}

func (h *BridgeHub) Register(deviceID string, conn BridgeConn) {
	deviceID = normalizeDeviceID(deviceID)
	if deviceID == "" || conn == nil {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if old, ok := h.devices[deviceID]; ok {
		h.failPendingWrites(old, "bridge reconnected")
		if old.conn != nil {
			_ = old.conn.Close()
		}
	}

	h.devices[deviceID] = &deviceState{
		conn:          conn,
		pendingWrites: make(map[string]*pendingWrite),
		status: BridgeStatus{
			Connected: true,
		},
	}
}

func (h *BridgeHub) Unregister(deviceID string, conn BridgeConn) {
	deviceID = normalizeDeviceID(deviceID)
	if deviceID == "" {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	state, ok := h.devices[deviceID]
	if !ok || state.conn != conn {
		return
	}

	h.failPendingWrites(state, "bridge disconnected")

	state.status.Connected = false
	state.conn = nil
	delete(h.devices, deviceID)
}

func (h *BridgeHub) ConnectedDeviceIDs() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	out := make([]string, 0, len(h.devices))
	for deviceID, state := range h.devices {
		if state != nil && state.conn != nil && state.status.Connected {
			out = append(out, deviceID)
		}
	}
	return out
}

func (h *BridgeHub) IsConnected(deviceID string) bool {
	status := h.Status(deviceID)
	return status.Connected
}

func (h *BridgeHub) TargetDeviceID(configured string) string {
	connected := h.ConnectedDeviceIDs()
	if len(connected) == 1 {
		return connected[0]
	}
	if configured != "" {
		return configured
	}
	return DefaultBridgeDeviceID
}

func (h *BridgeHub) Status(deviceID string) BridgeStatus {
	deviceID = normalizeDeviceID(deviceID)
	h.mu.RLock()
	defer h.mu.RUnlock()

	state, ok := h.devices[deviceID]
	if !ok || state == nil || state.conn == nil {
		return BridgeStatus{Connected: false}
	}
	return state.status
}

func (h *BridgeHub) PendingWriteCount(deviceID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	state, ok := h.devices[deviceID]
	if !ok || state == nil {
		return 0
	}
	return len(state.pendingWrites)
}

func (h *BridgeHub) RequestWrite(deviceID, logicalUUID string, timeout time.Duration) error {
	deviceID = normalizeDeviceID(deviceID)
	if deviceID == "" {
		return ErrBridgeUnavailable
	}
	if timeout <= 0 {
		timeout = defaultBridgeWriteTimeout
	}

	requestID := uuid.New().String()
	ch := make(chan writeAckResult, 1)

	h.mu.Lock()
	state, ok := h.devices[deviceID]
	if !ok || state == nil || state.conn == nil || !state.status.Connected {
		h.mu.Unlock()
		return ErrBridgeUnavailable
	}
	state.pendingWrites[requestID] = &pendingWrite{ch: ch}
	conn := state.conn
	writeMu := &state.writeMu
	h.mu.Unlock()

	msg := BridgeMessage{
		Type:        "write",
		RequestID:   requestID,
		LogicalUUID: logicalUUID,
	}
	writeMu.Lock()
	err := conn.WriteJSON(msg)
	writeMu.Unlock()
	if err != nil {
		h.cancelPending(deviceID, requestID)
		return fmt.Errorf("%w: %v", ErrBridgeUnavailable, err)
	}

	select {
	case res := <-ch:
		if res.ok {
			return nil
		}
		if res.errMsg != "" {
			return fmt.Errorf("bridge write failed: %s", res.errMsg)
		}
		return fmt.Errorf("bridge write failed")
	case <-time.After(timeout):
		h.cancelPending(deviceID, requestID)
		return ErrBridgeWriteTimeout
	}
}

func (h *BridgeHub) HandleMessage(deviceID string, msg *BridgeMessage) error {
	if msg == nil {
		return nil
	}

	switch msg.Type {
	case "write_ack":
		return h.completeWrite(deviceID, msg)
	case "status":
		h.applyStatus(deviceID, msg)
		return nil
	default:
		return nil
	}
}

func (h *BridgeHub) completeWrite(deviceID string, msg *BridgeMessage) error {
	deviceID = normalizeDeviceID(deviceID)
	if msg.RequestID == "" {
		return nil
	}

	ok := false
	if msg.OK != nil {
		ok = *msg.OK
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	state, exists := h.devices[deviceID]
	if !exists || state == nil {
		return nil
	}
	pending, exists := state.pendingWrites[msg.RequestID]
	if !exists {
		return nil
	}
	delete(state.pendingWrites, msg.RequestID)

	select {
	case pending.ch <- writeAckResult{ok: ok, errMsg: msg.Error}:
	default:
	}
	return nil
}

func (h *BridgeHub) applyStatus(deviceID string, msg *BridgeMessage) {
	deviceID = normalizeDeviceID(deviceID)

	h.mu.Lock()
	defer h.mu.Unlock()

	state, ok := h.devices[deviceID]
	if !ok || state == nil {
		return
	}

	if msg.PendingCount != nil {
		state.status.PendingCount = *msg.PendingCount
	}
	if msg.Syncing != nil {
		state.status.Syncing = *msg.Syncing
	}
	if msg.LastSyncAt != nil && *msg.LastSyncAt != "" {
		if ts, err := time.Parse(time.RFC3339, *msg.LastSyncAt); err == nil {
			ts = ts.UTC()
			state.status.LastSyncAt = &ts
		}
	}
}

func (h *BridgeHub) cancelPending(deviceID, requestID string) {
	deviceID = normalizeDeviceID(deviceID)

	h.mu.Lock()
	defer h.mu.Unlock()

	state, ok := h.devices[deviceID]
	if !ok || state == nil {
		return
	}
	delete(state.pendingWrites, requestID)
}

func (h *BridgeHub) failPendingWrites(state *deviceState, reason string) {
	if state == nil || len(state.pendingWrites) == 0 {
		return
	}
	for requestID, pending := range state.pendingWrites {
		select {
		case pending.ch <- writeAckResult{ok: false, errMsg: reason}:
		default:
		}
		delete(state.pendingWrites, requestID)
	}
}

func normalizeDeviceID(deviceID string) string {
	return strings.TrimSpace(deviceID)
}
