package scan

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/keweenaw-endurance/backend/internal/uuidutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddKaraokeBonus_CreatesOneBonusPerSourceLap(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	svc := NewScanService(fx.db, nil)

	scanResult, err := svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-finish-1", time.Now().UTC())
	require.NoError(t, err)
	require.Equal(t, ResultLap, scanResult.Result)
	require.NotNil(t, scanResult.TimingRecordID)
	require.Equal(t, 1, scanResult.LapCount)

	bonus, err := svc.AddKaraokeBonus(scanResult.TimingRecordID.UUID())
	require.NoError(t, err)
	require.NotNil(t, bonus)
	assert.Equal(t, "karaoke_bonus", bonus.Record.RecordType)
	assert.Equal(t, scanResult.TimingRecordID.UUID(), bonus.Record.SourceLapID.UUID())
	assert.Equal(t, 2, bonus.LapCount)
	assert.Equal(t, fx.participant.ID, bonus.Record.ParticipantID)

	var count int64
	require.NoError(t, fx.db.Model(&models.TimingRecord{}).
		Where("record_type = ? AND source_lap_id = ?", "karaoke_bonus", scanResult.TimingRecordID).
		Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestAddKaraokeBonus_SecondCallReturnsAlreadyExists(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	svc := NewScanService(fx.db, nil)

	scanResult, err := svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-finish-1", time.Now().UTC())
	require.NoError(t, err)
	require.NotNil(t, scanResult.TimingRecordID)

	_, err = svc.AddKaraokeBonus(scanResult.TimingRecordID.UUID())
	require.NoError(t, err)

	_, err = svc.AddKaraokeBonus(scanResult.TimingRecordID.UUID())
	assert.ErrorIs(t, err, ErrAlreadyExists)

	var count int64
	require.NoError(t, fx.db.Model(&models.TimingRecord{}).
		Where("record_type = ? AND source_lap_id = ?", "karaoke_bonus", scanResult.TimingRecordID).
		Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestAddKaraokeBonus_RejectsNonRFIDLap(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	svc := NewScanService(fx.db, nil)

	sourceID := uuidutil.NewPublicUUID(uuid.New())
	bonusOnly := &models.TimingRecord{
		ParticipantID:  fx.participant.ID,
		CheckpointID:   fx.finish.ID,
		Timestamp:      time.Now().UTC(),
		LocalTimestamp: time.Now().UTC(),
		SyncStatus:     "synced",
		RecordType:     "karaoke_bonus",
		SourceLapID:    &sourceID,
	}
	require.NoError(t, fx.db.Create(bonusOnly).Error)

	_, err := svc.AddKaraokeBonus(bonusOnly.ID.UUID())
	assert.ErrorIs(t, err, ErrInvalidSourceLap)
}

func TestAddKaraokeBonus_RejectsCheckpointPass(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	svc := NewScanService(fx.db, nil)

	pass := &models.TimingRecord{
		ParticipantID:  fx.participant.ID,
		CheckpointID:   fx.finish.ID,
		Timestamp:      time.Now().UTC(),
		LocalTimestamp: time.Now().UTC(),
		SyncStatus:     "synced",
		RecordType:     "checkpoint_pass",
	}
	require.NoError(t, fx.db.Create(pass).Error)

	_, err := svc.AddKaraokeBonus(pass.ID.UUID())
	assert.ErrorIs(t, err, ErrInvalidSourceLap)
}

func TestAddKaraokeBonus_CountsTowardPlacement(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	svc := NewScanService(fx.db, nil)

	scanResult, err := svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-finish-1", time.Now().UTC())
	require.NoError(t, err)

	bonus, err := svc.AddKaraokeBonus(scanResult.TimingRecordID.UUID())
	require.NoError(t, err)
	assert.Equal(t, 2, bonus.LapCount)
	assert.Equal(t, 1, bonus.Placement)
}
