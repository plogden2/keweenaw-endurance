package services

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeBridgeConn struct {
	mu       sync.Mutex
	outbound chan json.RawMessage
	inbound  chan json.RawMessage
	closed   bool
}

func newFakeBridgeConn(buffer int) *fakeBridgeConn {
	return &fakeBridgeConn{
		outbound: make(chan json.RawMessage, buffer),
		inbound:  make(chan json.RawMessage, buffer),
	}
}

func (f *fakeBridgeConn) WriteJSON(v any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return assert.AnError
	}
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	f.outbound <- data
	return nil
}

func (f *fakeBridgeConn) ReadJSON(v any) error {
	f.mu.Lock()
	closed := f.closed
	f.mu.Unlock()
	if closed {
		return assert.AnError
	}
	select {
	case data := <-f.inbound:
		return json.Unmarshal(data, v)
	case <-time.After(2 * time.Second):
		return assert.AnError
	}
}

func (f *fakeBridgeConn) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	return nil
}

func (f *fakeBridgeConn) pushInbound(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	f.inbound <- data
	return nil
}

func (f *fakeBridgeConn) awaitOutbound(timeout time.Duration) (BridgeMessage, error) {
	select {
	case raw := <-f.outbound:
		var msg BridgeMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			return BridgeMessage{}, err
		}
		return msg, nil
	case <-time.After(timeout):
		return BridgeMessage{}, assert.AnError
	}
}

func TestBridgeHub_RequestWriteRoundTrip(t *testing.T) {
	hub := NewBridgeHub()
	conn := newFakeBridgeConn(4)
	hub.Register("laptop-finish-1", conn)

	done := make(chan error, 1)
	go func() {
		msg, err := conn.awaitOutbound(time.Second)
		require.NoError(t, err)
		assert.Equal(t, "write", msg.Type)
		assert.NotEmpty(t, msg.RequestID)
		assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", msg.LogicalUUID)
		ok := true
		require.NoError(t, hub.HandleMessage("laptop-finish-1", &BridgeMessage{
			Type:      "write_ack",
			RequestID: msg.RequestID,
			OK:        &ok,
		}))
	}()

	go func() {
		done <- hub.RequestWrite("laptop-finish-1", "550e8400-e29b-41d4-a716-446655440000", time.Second)
	}()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("RequestWrite timed out")
	}
}

func TestBridgeHub_RequestWriteNotConnected(t *testing.T) {
	hub := NewBridgeHub()
	err := hub.RequestWrite("laptop-finish-1", "uuid", time.Second)
	assert.ErrorIs(t, err, ErrBridgeUnavailable)
}

func TestBridgeHub_RequestWriteAckFailure(t *testing.T) {
	hub := NewBridgeHub()
	conn := newFakeBridgeConn(4)
	hub.Register("laptop-finish-1", conn)

	go func() {
		msg, err := conn.awaitOutbound(time.Second)
		require.NoError(t, err)
		ok := false
		require.NoError(t, hub.HandleMessage("laptop-finish-1", &BridgeMessage{
			Type:      "write_ack",
			RequestID: msg.RequestID,
			OK:        &ok,
			Error:     "tag not present",
		}))
	}()

	err := hub.RequestWrite("laptop-finish-1", "uuid", time.Second)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tag not present")
}

func TestBridgeHub_Status(t *testing.T) {
	hub := NewBridgeHub()
	conn := newFakeBridgeConn(2)
	hub.Register("laptop-finish-1", conn)

	status := hub.Status("laptop-finish-1")
	assert.True(t, status.Connected)
	assert.Equal(t, 0, status.PendingCount)
	assert.False(t, status.Syncing)

	pending := 3
	syncing := true
	lastSync := "2026-07-16T12:00:00Z"
	require.NoError(t, hub.HandleMessage("laptop-finish-1", &BridgeMessage{
		Type:         "status",
		PendingCount: &pending,
		Syncing:      &syncing,
		LastSyncAt:   &lastSync,
	}))

	status = hub.Status("laptop-finish-1")
	assert.Equal(t, 3, status.PendingCount)
	assert.True(t, status.Syncing)
	require.NotNil(t, status.LastSyncAt)
	assert.Equal(t, "2026-07-16T12:00:00Z", status.LastSyncAt.UTC().Format(time.RFC3339))
}

func TestBridgeHub_TargetDeviceID(t *testing.T) {
	hub := NewBridgeHub()
	assert.Equal(t, DefaultBridgeDeviceID, hub.TargetDeviceID(""))

	conn := newFakeBridgeConn(1)
	hub.Register("only-device", conn)
	assert.Equal(t, "only-device", hub.TargetDeviceID("laptop-finish-1"))
}

func TestBridgeHub_UnregisterCancelsPending(t *testing.T) {
	hub := NewBridgeHub()
	conn := newFakeBridgeConn(4)
	hub.Register("laptop-finish-1", conn)

	writeStarted := make(chan struct{})
	go func() {
		close(writeStarted)
		_, _ = conn.awaitOutbound(time.Second)
	}()

	<-writeStarted

	done := make(chan error, 1)
	go func() {
		done <- hub.RequestWrite("laptop-finish-1", "uuid", 5*time.Second)
	}()

	require.Eventually(t, func() bool {
		return hub.PendingWriteCount("laptop-finish-1") == 1
	}, time.Second, 10*time.Millisecond)

	hub.Unregister("laptop-finish-1", conn)

	select {
	case err := <-done:
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bridge disconnected")
	case <-time.After(time.Second):
		t.Fatal("expected pending write to fail fast on unregister")
	}
}

func TestBridgeHub_RegisterReconnectCancelsPending(t *testing.T) {
	hub := NewBridgeHub()
	conn1 := newFakeBridgeConn(4)
	hub.Register("laptop-finish-1", conn1)

	writeStarted := make(chan struct{})
	go func() {
		close(writeStarted)
		_, _ = conn1.awaitOutbound(time.Second)
	}()

	<-writeStarted

	done := make(chan error, 1)
	go func() {
		done <- hub.RequestWrite("laptop-finish-1", "uuid", 5*time.Second)
	}()

	require.Eventually(t, func() bool {
		return hub.PendingWriteCount("laptop-finish-1") == 1
	}, time.Second, 10*time.Millisecond)

	conn2 := newFakeBridgeConn(4)
	hub.Register("laptop-finish-1", conn2)

	select {
	case err := <-done:
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bridge reconnected")
	case <-time.After(time.Second):
		t.Fatal("expected pending write to fail fast on reconnect")
	}
}

func TestBridgeHub_RequestWriteTimeout(t *testing.T) {
	hub := NewBridgeHub()
	conn := newFakeBridgeConn(1)
	hub.Register("laptop-finish-1", conn)

	go func() {
		_, _ = conn.awaitOutbound(time.Second)
	}()

	err := hub.RequestWrite("laptop-finish-1", "uuid", 50*time.Millisecond)
	assert.ErrorIs(t, err, ErrBridgeWriteTimeout)
}
