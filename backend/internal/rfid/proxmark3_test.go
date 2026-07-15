package rfid

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockReader_WriteLogicalUUIDAndPoll(t *testing.T) {
	m := NewMockReader()
	const logicalUUID = "550e8400-e29b-41d4-a716-446655440001"

	require.NoError(t, m.WriteLogicalUUID(logicalUUID))

	uid, err := m.Poll()
	require.NoError(t, err)
	assert.Equal(t, logicalUUID, uid)
}

func TestMockReader_WriteLogicalUUIDRejectsNonUUID(t *testing.T) {
	m := NewMockReader()
	err := m.WriteLogicalUUID("NEW-TAG-001")
	require.Error(t, err)

	uid, pollErr := m.Poll()
	require.NoError(t, pollErr)
	assert.Empty(t, uid)
}

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
	assert.Equal(t, "tag-a", uid)

	uid, err = m.Poll()
	require.NoError(t, err)
	assert.Equal(t, "tag-b", uid)

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
	assert.Equal(t, "hw-1", uid)
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
