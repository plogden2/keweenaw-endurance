package rfid

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockReader_PollEmpty(t *testing.T) {
	m := NewMockReader()
	uid, err := m.Poll()
	require.NoError(t, err)
	assert.Empty(t, uid)
}

func TestMockReader_PushUIDAndPoll(t *testing.T) {
	m := NewMockReader()
	m.PushUID("TAG-A")
	m.Enqueue("TAG-B")

	uid, err := m.Poll()
	require.NoError(t, err)
	assert.Equal(t, "TAG-A", uid)

	uid, err = m.Poll()
	require.NoError(t, err)
	assert.Equal(t, "TAG-B", uid)

	uid, err = m.Poll()
	require.NoError(t, err)
	assert.Empty(t, uid)
}

func TestMockReader_PollUnavailable(t *testing.T) {
	m := NewMockReader()
	m.Available = false
	m.PushUID("TAG-X")

	_, err := m.Poll()
	assert.ErrorIs(t, err, ErrHardwareUnavailable)
}

func TestProxmark3_PollDelegates(t *testing.T) {
	mock := NewMockReader()
	mock.PushUID("HW-1")
	pm := NewProxmark3(mock)

	uid, err := pm.Poll()
	require.NoError(t, err)
	assert.Equal(t, "HW-1", uid)
	assert.True(t, pm.IsAvailable())
}

func TestProxmark3_PollUnavailable(t *testing.T) {
	pm := NewProxmark3(&NoOpReader{})
	uid, err := pm.Poll()
	assert.ErrorIs(t, err, ErrHardwareUnavailable)
	assert.Empty(t, uid)

	pm = NewProxmark3(nil)
	_, err = pm.Poll()
	assert.ErrorIs(t, err, ErrHardwareUnavailable)
}

func TestNoOpReader_Poll(t *testing.T) {
	n := &NoOpReader{}
	uid, err := n.Poll()
	assert.ErrorIs(t, err, ErrHardwareUnavailable)
	assert.Empty(t, uid)
	assert.False(t, n.IsAvailable())
}
