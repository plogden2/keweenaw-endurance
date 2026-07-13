package scan

import (
	"testing"
	"time"

	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedCheckpointCourseFixture(t *testing.T) (*scanFixture, *models.TimingCheckpoint, *models.TimingCheckpoint) {
	t.Helper()
	fx := seedActiveLapFixture(t, "active")

	start := &models.TimingCheckpoint{
		RaceID:              fx.race.ID,
		Name:                "Start Line",
		CheckpointType:      "start",
		DistanceFromStartKm: 0,
		IsActive:            true,
	}
	require.NoError(t, fx.db.Create(start).Error)

	// Mid / finish checkpoint (Lap Check style).
	mid := fx.finish
	require.NoError(t, fx.db.Model(mid).Updates(map[string]interface{}{
		"name":                   "Lap Check",
		"distance_from_start_km": 5.0,
	}).Error)

	require.NoError(t, fx.db.Model(&models.ReaderStation{}).
		Where("device_id = ?", "laptop-finish-1").
		Updates(map[string]interface{}{
			"mode":          "checkpoint",
			"checkpoint_id": mid.ID,
			"device_id":     "laptop-checkpoint-1",
			"name":          "Mid-loop CP",
		}).Error)

	return fx, start, mid
}

func TestCheckpoint_OutOfOrderDoesNotCompleteLap(t *testing.T) {
	fx, _, _ := seedCheckpointCourseFixture(t)
	svc := NewScanService(fx.db, nil)

	now := time.Now().UTC().Truncate(time.Second)
	result, err := svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-checkpoint-1", now)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, ResultOutOfOrder, result.Result)
	assert.False(t, result.KaraokeAvailable)
	assert.Contains(t, result.Message, "sequence")

	var lapCount, passCount int64
	require.NoError(t, fx.db.Model(&models.TimingRecord{}).
		Where("participant_id = ? AND record_type = ?", fx.participant.ID, "rfid_lap").
		Count(&lapCount).Error)
	require.NoError(t, fx.db.Model(&models.TimingRecord{}).
		Where("participant_id = ? AND record_type = ?", fx.participant.ID, "checkpoint_pass").
		Count(&passCount).Error)
	assert.Equal(t, int64(0), lapCount)
	assert.Equal(t, int64(0), passCount)
}
func TestCheckpoint_InOrderProgressThenLapOnSequenceComplete(t *testing.T) {
	fx, start, mid := seedCheckpointCourseFixture(t)
	svc := NewScanService(fx.db, nil)

	// Bind station to start first.
	require.NoError(t, fx.db.Model(&models.ReaderStation{}).
		Where("device_id = ?", "laptop-checkpoint-1").
		Update("checkpoint_id", start.ID).Error)

	now := time.Now().UTC().Truncate(time.Second)
	first, err := svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-checkpoint-1", now)
	require.NoError(t, err)
	assert.Equal(t, ResultCheckpointPass, first.Result)
	assert.False(t, first.KaraokeAvailable)

	var lapCount int64
	require.NoError(t, fx.db.Model(&models.TimingRecord{}).
		Where("participant_id = ? AND record_type = ?", fx.participant.ID, "rfid_lap").
		Count(&lapCount).Error)
	assert.Equal(t, int64(0), lapCount)

	// Advance station to finish / Lap Check and complete sequence.
	require.NoError(t, fx.db.Model(&models.ReaderStation{}).
		Where("device_id = ?", "laptop-checkpoint-1").
		Update("checkpoint_id", mid.ID).Error)

	second, err := svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-checkpoint-1", now.Add(time.Second))
	require.NoError(t, err)
	assert.Equal(t, ResultLap, second.Result)
	assert.True(t, second.KaraokeAvailable)
	assert.Equal(t, 1, second.LapCount)
	require.NotNil(t, second.TimingRecordID)

	require.NoError(t, fx.db.Model(&models.TimingRecord{}).
		Where("participant_id = ? AND record_type = ?", fx.participant.ID, "rfid_lap").
		Count(&lapCount).Error)
	assert.Equal(t, int64(1), lapCount)

	var passCount int64
	require.NoError(t, fx.db.Model(&models.TimingRecord{}).
		Where("participant_id = ? AND record_type = ?", fx.participant.ID, "checkpoint_pass").
		Count(&passCount).Error)
	assert.Equal(t, int64(2), passCount)
}

func TestCheckpoint_CompletingSequenceCreatesRFIDLapOnlyOncePerCycle(t *testing.T) {
	fx, start, mid := seedCheckpointCourseFixture(t)
	svc := NewScanService(fx.db, nil)

	now := time.Now().UTC().Truncate(time.Second)

	require.NoError(t, fx.db.Model(&models.ReaderStation{}).
		Where("device_id = ?", "laptop-checkpoint-1").
		Update("checkpoint_id", start.ID).Error)
	_, err := svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-checkpoint-1", now)
	require.NoError(t, err)

	require.NoError(t, fx.db.Model(&models.ReaderStation{}).
		Where("device_id = ?", "laptop-checkpoint-1").
		Update("checkpoint_id", mid.ID).Error)
	lap1, err := svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-checkpoint-1", now.Add(time.Second))
	require.NoError(t, err)
	assert.Equal(t, ResultLap, lap1.Result)

	// After a completed lap, mid again is out of order (start expected next).
	ooo, err := svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-checkpoint-1", now.Add(2*time.Second))
	require.NoError(t, err)
	assert.Equal(t, ResultOutOfOrder, ooo.Result)

	var lapCount int64
	require.NoError(t, fx.db.Model(&models.TimingRecord{}).
		Where("participant_id = ? AND record_type = ?", fx.participant.ID, "rfid_lap").
		Count(&lapCount).Error)
	assert.Equal(t, int64(1), lapCount)
}
