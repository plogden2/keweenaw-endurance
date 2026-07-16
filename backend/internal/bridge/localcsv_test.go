package bridge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalStore_EnqueueAppendsCSVAndPending(t *testing.T) {
	dir := t.TempDir()
	eventID := uuid.New().String()
	store, err := NewLocalStore(dir, eventID)
	require.NoError(t, err)

	ts := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	lap := PendingLap{
		ID:          uuid.New().String(),
		LogicalUUID: "9fe78eeb-a21c-594a-acc2-7e1efe378201",
		TS:          ts,
		DeviceID:    "laptop-finish-1",
	}
	require.NoError(t, store.EnqueueLap(lap))

	assert.Equal(t, 1, store.PendingCount())

	pending, err := store.ListPending()
	require.NoError(t, err)
	require.Len(t, pending, 1)
	assert.Equal(t, lap.ID, pending[0].ID)
	assert.Equal(t, lap.LogicalUUID, pending[0].LogicalUUID)

	csvPath := filepath.Join(dir, "events", eventID, "live-snapshot.csv")
	data, err := os.ReadFile(csvPath)
	require.NoError(t, err)
	body := string(data)
	assert.True(t, strings.Contains(body, "#SECTION,timing_records"))
	assert.True(t, strings.Contains(body, lap.ID))
	assert.True(t, strings.Contains(body, lap.LogicalUUID))
	assert.True(t, strings.Contains(body, "pending_sync"))
}

func TestLocalStore_EnqueueAppendsToExistingCSV(t *testing.T) {
	dir := t.TempDir()
	eventID := uuid.New().String()
	eventDir := filepath.Join(dir, "events", eventID)
	require.NoError(t, os.MkdirAll(eventDir, 0o755))
	existing := "#SECTION,event\nid,name\n" + eventID + ",Test\n#SECTION,timing_records\nid,participant_id,checkpoint_id,timestamp,local_timestamp,device_id,sync_status,record_type,source_lap_id,station_id\n"
	require.NoError(t, os.WriteFile(filepath.Join(eventDir, "live-snapshot.csv"), []byte(existing), 0o644))

	store, err := NewLocalStore(dir, eventID)
	require.NoError(t, err)

	lap := PendingLap{
		ID:          uuid.New().String(),
		LogicalUUID: "20131e69-ae96-53a7-893b-eb9ef607a13e",
		TS:          time.Now().UTC(),
		DeviceID:    "laptop-finish-1",
	}
	require.NoError(t, store.EnqueueLap(lap))

	data, err := os.ReadFile(filepath.Join(eventDir, "live-snapshot.csv"))
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	assert.GreaterOrEqual(t, len(lines), 5)
	assert.True(t, strings.Contains(lines[len(lines)-1], lap.ID))
}

func TestLocalStore_ClearPending(t *testing.T) {
	dir := t.TempDir()
	eventID := uuid.New().String()
	store, err := NewLocalStore(dir, eventID)
	require.NoError(t, err)

	require.NoError(t, store.EnqueueLap(PendingLap{
		ID:          uuid.New().String(),
		LogicalUUID: "9fe78eeb-a21c-594a-acc2-7e1efe378201",
		TS:          time.Now().UTC(),
		DeviceID:    "laptop-finish-1",
	}))
	require.Equal(t, 1, store.PendingCount())

	require.NoError(t, store.ClearPending())
	assert.Equal(t, 0, store.PendingCount())
	pending, err := store.ListPending()
	require.NoError(t, err)
	assert.Empty(t, pending)
}

func TestLocalStore_EnqueueRollsBackCSVWhenPendingFails(t *testing.T) {
	dir := t.TempDir()
	eventID := uuid.New().String()
	store, err := NewLocalStore(dir, eventID)
	require.NoError(t, err)

	// Block pending writes so only CSV append succeeds first.
	require.NoError(t, os.Mkdir(store.PendingPath(), 0o755))

	lap := PendingLap{
		ID:          uuid.New().String(),
		LogicalUUID: "9fe78eeb-a21c-594a-acc2-7e1efe378201",
		TS:          time.Now().UTC(),
		DeviceID:    "laptop-finish-1",
	}
	err = store.EnqueueLap(lap)
	require.Error(t, err)
	assert.Equal(t, 0, store.PendingCount())

	data, readErr := os.ReadFile(store.CSVPath())
	if readErr == nil {
		assert.NotContains(t, string(data), lap.ID)
	}
}
