package bridge

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/keweenaw-endurance/backend/internal/services"
)

// WSReadSender sends bridge read messages over an established WebSocket.
type WSReadSender struct {
	conn *websocket.Conn
	mu   *sync.Mutex
}

// NewWSReadSender creates a sender that serializes writes on mu.
func NewWSReadSender(conn *websocket.Conn, mu *sync.Mutex) *WSReadSender {
	return &WSReadSender{conn: conn, mu: mu}
}

// SendRead emits a hosted bridge read message for a pending lap.
func (s *WSReadSender) SendRead(lap PendingLap) error {
	if s == nil || s.conn == nil {
		return ErrNotConnected
	}
	msg := services.BridgeMessage{
		Type:        "read",
		LogicalUUID: lap.LogicalUUID,
		SourceLapID: lap.ID,
		TS:          lap.TS.UTC().Format(time.RFC3339),
	}
	if s.mu != nil {
		s.mu.Lock()
		defer s.mu.Unlock()
	}
	return s.conn.WriteJSON(msg)
}

// SendStatus reports queue depth and sync state to the hosted hub.
func SendStatus(conn *websocket.Conn, mu *sync.Mutex, pendingCount int, syncing bool, lastSyncAt *time.Time) error {
	if conn == nil {
		return ErrNotConnected
	}
	syncingVal := syncing
	pendingVal := pendingCount
	msg := services.BridgeMessage{
		Type:         "status",
		PendingCount: &pendingVal,
		Syncing:      &syncingVal,
	}
	if lastSyncAt != nil && !lastSyncAt.IsZero() {
		formatted := lastSyncAt.UTC().Format(time.RFC3339)
		msg.LastSyncAt = &formatted
	}
	if mu != nil {
		mu.Lock()
		defer mu.Unlock()
	}
	return conn.WriteJSON(msg)
}

// SendWriteAck acknowledges a hosted write command.
func SendWriteAck(conn *websocket.Conn, mu *sync.Mutex, requestID string, ok bool, errMsg string) error {
	if conn == nil {
		return ErrNotConnected
	}
	msg := services.BridgeMessage{
		Type:      "write_ack",
		RequestID: requestID,
		OK:        &ok,
		Error:     errMsg,
	}
	if mu != nil {
		mu.Lock()
		defer mu.Unlock()
	}
	return conn.WriteJSON(msg)
}
