package bridge

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingSender struct {
	reads []PendingLap
	err   error
}

func (r *recordingSender) SendRead(lap PendingLap) error {
	if r.err != nil {
		return r.err
	}
	r.reads = append(r.reads, lap)
	return nil
}

func TestSyncer_FlushSendsReadMessages(t *testing.T) {
	dir := t.TempDir()
	eventID := uuid.New().String()
	store, err := NewLocalStore(dir, eventID)
	require.NoError(t, err)

	ts := time.Date(2026, 7, 16, 12, 1, 0, 0, time.UTC)
	lap := PendingLap{
		ID:          uuid.New().String(),
		LogicalUUID: "9fe78eeb-a21c-594a-acc2-7e1efe378201",
		TS:          ts,
		DeviceID:    "laptop-finish-1",
	}
	require.NoError(t, store.EnqueueLap(lap))

	sender := &recordingSender{}
	syncer := NewSyncer(store)
	n, err := syncer.Flush(sender)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	require.Len(t, sender.reads, 1)
	assert.Equal(t, lap.LogicalUUID, sender.reads[0].LogicalUUID)
	assert.Equal(t, lap.ID, sender.reads[0].ID)
	assert.Equal(t, 0, store.PendingCount())
}

func TestSyncer_FlushIdempotent(t *testing.T) {
	dir := t.TempDir()
	eventID := uuid.New().String()
	store, err := NewLocalStore(dir, eventID)
	require.NoError(t, err)

	lap := PendingLap{
		ID:          uuid.New().String(),
		LogicalUUID: "9fe78eeb-a21c-594a-acc2-7e1efe378201",
		TS:          time.Now().UTC(),
		DeviceID:    "laptop-finish-1",
	}
	require.NoError(t, store.EnqueueLap(lap))

	sender := &recordingSender{}
	syncer := NewSyncer(store)

	n1, err := syncer.Flush(sender)
	require.NoError(t, err)
	assert.Equal(t, 1, n1)

	n2, err := syncer.Flush(sender)
	require.NoError(t, err)
	assert.Equal(t, 0, n2)
	assert.Len(t, sender.reads, 1)
}

func TestSyncer_FlushPartialFailureKeepsRemaining(t *testing.T) {
	dir := t.TempDir()
	eventID := uuid.New().String()
	store, err := NewLocalStore(dir, eventID)
	require.NoError(t, err)

	lap1 := PendingLap{ID: uuid.New().String(), LogicalUUID: "aaaa", TS: time.Now().UTC(), DeviceID: "d1"}
	lap2 := PendingLap{ID: uuid.New().String(), LogicalUUID: "bbbb", TS: time.Now().UTC(), DeviceID: "d1"}
	require.NoError(t, store.EnqueueLap(lap1))
	require.NoError(t, store.EnqueueLap(lap2))

	failAfter := &failAfterSender{n: 1}
	syncer := NewSyncer(store)

	n, err := syncer.Flush(failAfter)
	require.Error(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, 1, store.PendingCount())

	pending, err := store.ListPending()
	require.NoError(t, err)
	require.Len(t, pending, 1)
	assert.Equal(t, lap2.ID, pending[0].ID)
}

type failAfterSender struct {
	n     int
	sent  int
	reads []PendingLap
}

func (f *failAfterSender) SendRead(lap PendingLap) error {
	f.reads = append(f.reads, lap)
	f.sent++
	if f.sent > f.n {
		return assert.AnError
	}
	return nil
}
