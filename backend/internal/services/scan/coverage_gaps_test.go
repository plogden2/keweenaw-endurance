package scan

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/keweenaw-endurance/backend/internal/uuidutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestSetOnEventChange_NotifyHook(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	svc := NewScanService(fx.db, nil)

	var seen uuid.UUID
	svc.SetOnEventChange(func(eventID uuid.UUID) {
		seen = eventID
	})

	_, err := svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-finish-1", time.Now().UTC())
	require.NoError(t, err)
	assert.Equal(t, fx.event.ID.UUID(), seen)

	// Nil event ID must not invoke the hook.
	called := false
	svc.SetOnEventChange(func(uuid.UUID) { called = true })
	svc.notifyChange(uuid.Nil)
	assert.False(t, called)
}

func TestWithScanDisplay_NilAndUncategorized(t *testing.T) {
	assert.Nil(t, withScanDisplay(nil))

	r := withScanDisplay(&ScanResult{Result: ResultUnknownTag})
	require.NotNil(t, r)
	assert.Equal(t, ResultUnknownTag, r.Result)

	p := &models.Participant{FirstName: "Sam", LastName: "Lee", BibNumber: "7"}
	out := withScanDisplay(&ScanResult{Participant: p})
	assert.Equal(t, "Sam Lee", out.ParticipantName)
	assert.Equal(t, "7", out.BibNumber)
	assert.Empty(t, out.CategoryLabel)
	assert.Empty(t, out.RaceName)
}

func TestIsUniqueViolation(t *testing.T) {
	assert.False(t, isUniqueViolation(nil))
	assert.False(t, isUniqueViolation(errors.New("other")))
	assert.True(t, isUniqueViolation(errors.New("UNIQUE constraint failed")))
	assert.True(t, isUniqueViolation(errors.New("pq: unique violation")))
	assert.True(t, isUniqueViolation(errors.New("constraint failed: foo")))
}

func TestAddKaraokeBonus_NotFoundAndSyncStatus(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	svc := NewScanService(fx.db, pendingSyncResolver{})

	_, err := svc.AddKaraokeBonus(uuid.New())
	assert.ErrorIs(t, err, ErrSourceLapNotFound)

	scanResult, err := svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-finish-1", time.Now().UTC())
	require.NoError(t, err)

	bonus, err := svc.AddKaraokeBonus(scanResult.TimingRecordID.UUID())
	require.NoError(t, err)
	assert.Equal(t, "pending_sync", bonus.Record.SyncStatus)
}

func TestAddKaraokeBonus_CreateUniqueViolation(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	svc := NewScanService(fx.db, nil)

	scanResult, err := svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-finish-1", time.Now().UTC())
	require.NoError(t, err)
	sourceID := scanResult.TimingRecordID.UUID()

	require.NoError(t, fx.db.Callback().Create().Before("gorm:create").Register("fail_karaoke_unique", func(tx *gorm.DB) {
		rec, ok := tx.Statement.Dest.(*models.TimingRecord)
		if !ok || rec.RecordType != "karaoke_bonus" {
			return
		}
		_ = tx.AddError(errors.New("UNIQUE constraint failed: timing_records.source_lap_id"))
	}))
	t.Cleanup(func() {
		_ = fx.db.Callback().Create().Remove("fail_karaoke_unique")
	})

	_, err = svc.AddKaraokeBonus(sourceID)
	assert.ErrorIs(t, err, ErrAlreadyExists)
}

func TestAddKaraokeBonus_ClosedDB(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	svc := NewScanService(fx.db, nil)

	scanResult, err := svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-finish-1", time.Now().UTC())
	require.NoError(t, err)

	sqlDB, err := fx.db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	_, err = svc.AddKaraokeBonus(scanResult.TimingRecordID.UUID())
	assert.Error(t, err)
	assert.False(t, errors.Is(err, ErrSourceLapNotFound))
}

func TestAddKaraokeBonus_ParticipantMissingAfterCreate(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	svc := NewScanService(fx.db, nil)

	scanResult, err := svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-finish-1", time.Now().UTC())
	require.NoError(t, err)

	require.NoError(t, fx.db.Exec("DELETE FROM participants WHERE id = ?", fx.participant.ID).Error)

	_, err = svc.AddKaraokeBonus(scanResult.TimingRecordID.UUID())
	assert.Error(t, err)
}

func TestPtrStationID_Nil(t *testing.T) {
	assert.Nil(t, ptrStationID(nil))
	assert.Nil(t, ptrStationID(&models.ReaderStation{}))
}

func TestCheckpointSequencePriority_Default(t *testing.T) {
	assert.Equal(t, 0, checkpointSequencePriority(models.TimingCheckpoint{CheckpointType: "start"}))
	assert.Equal(t, 1, checkpointSequencePriority(models.TimingCheckpoint{CheckpointType: "intermediate"}))
	assert.Equal(t, 2, checkpointSequencePriority(models.TimingCheckpoint{CheckpointType: "finish"}))
	assert.Equal(t, 9, checkpointSequencePriority(models.TimingCheckpoint{CheckpointType: "aid"}))
}

func TestProcessCheckpoint_UnboundStation(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	svc := NewScanService(fx.db, nil)

	station := &models.ReaderStation{Mode: "checkpoint", Name: "Loose"}
	result, err := svc.processCheckpointMode(station, fx.participant, fx.race, "dev", time.Now().UTC())
	require.NoError(t, err)
	assert.Equal(t, ResultOutOfOrder, result.Result)
	assert.Contains(t, result.Message, "not bound")
}

func TestProcessCheckpoint_NoCheckpointsConfigured(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	svc := NewScanService(fx.db, nil)

	require.NoError(t, fx.db.Where("race_id = ?", fx.race.ID).Delete(&models.TimingCheckpoint{}).Error)

	cpID := uuidutil.NewPublicUUID(uuid.New())
	station := &models.ReaderStation{
		EventID:      fx.event.ID,
		Mode:         "checkpoint",
		CheckpointID: &cpID,
		DeviceID:     "cp-empty",
	}
	require.NoError(t, fx.db.Create(station).Error)

	_, err := svc.processCheckpointMode(station, fx.participant, fx.race, "cp-empty", time.Now().UTC())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no checkpoints")
}

func TestProcessCheckpoint_CooldownOnSequenceComplete(t *testing.T) {
	fx, start, mid := seedCheckpointCourseFixture(t)
	svc := NewScanService(fx.db, pendingSyncResolver{})

	now := time.Now().UTC().Truncate(time.Second)

	// Existing RFID lap puts cooldown in effect.
	require.NoError(t, fx.db.Create(&models.TimingRecord{
		ParticipantID:  fx.participant.ID,
		CheckpointID:   mid.ID,
		Timestamp:      now.Add(-30 * time.Second),
		LocalTimestamp: now.Add(-30 * time.Second),
		SyncStatus:     "synced",
		RecordType:     "rfid_lap",
	}).Error)

	require.NoError(t, fx.db.Model(&models.ReaderStation{}).
		Where("device_id = ?", "laptop-checkpoint-1").
		Update("checkpoint_id", start.ID).Error)
	_, err := svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-checkpoint-1", now)
	require.NoError(t, err)

	require.NoError(t, fx.db.Model(&models.ReaderStation{}).
		Where("device_id = ?", "laptop-checkpoint-1").
		Update("checkpoint_id", mid.ID).Error)
	result, err := svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-checkpoint-1", now.Add(time.Second))
	require.NoError(t, err)
	assert.Equal(t, ResultCooldown, result.Result)
	assert.Greater(t, result.RetryAfterSeconds, 0)
}

func TestOrderedCheckpoints_IncludesIntermediateAndSort(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	svc := NewScanService(fx.db, nil)

	midFar := &models.TimingCheckpoint{
		RaceID:              fx.race.ID,
		Name:                "Mid Far",
		CheckpointType:      "intermediate",
		DistanceFromStartKm: 8,
		IsActive:            true,
	}
	midNear := &models.TimingCheckpoint{
		RaceID:              fx.race.ID,
		Name:                "Mid Near",
		CheckpointType:      "intermediate",
		DistanceFromStartKm: 3,
		IsActive:            true,
	}
	start := &models.TimingCheckpoint{
		RaceID:              fx.race.ID,
		Name:                "Start",
		CheckpointType:      "start",
		DistanceFromStartKm: 0,
		IsActive:            true,
	}
	require.NoError(t, fx.db.Create(midFar).Error)
	require.NoError(t, fx.db.Create(midNear).Error)
	require.NoError(t, fx.db.Create(start).Error)

	seq, err := svc.orderedCheckpoints(fx.race.ID.UUID())
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(seq), 3)
	assert.Equal(t, "start", seq[0].CheckpointType)
	// Intermediates sorted by distance before finish.
	var intermediates []models.TimingCheckpoint
	for _, cp := range seq {
		if cp.CheckpointType == "intermediate" {
			intermediates = append(intermediates, cp)
		}
	}
	require.Len(t, intermediates, 2)
	assert.Less(t, intermediates[0].DistanceFromStartKm, intermediates[1].DistanceFromStartKm)
}

func TestOrderedCheckpoints_DBError(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	svc := NewScanService(fx.db, nil)
	sqlDB, err := fx.db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	_, err = svc.orderedCheckpoints(fx.race.ID.UUID())
	assert.Error(t, err)
}

func TestExpectedCheckpoint_RestartWhenAllPassed(t *testing.T) {
	fx, start, mid := seedCheckpointCourseFixture(t)
	svc := NewScanService(fx.db, nil)
	now := time.Now().UTC()

	require.NoError(t, fx.db.Create(&models.TimingRecord{
		ParticipantID:  fx.participant.ID,
		CheckpointID:   start.ID,
		Timestamp:      now,
		LocalTimestamp: now,
		SyncStatus:     "synced",
		RecordType:     "checkpoint_pass",
	}).Error)
	require.NoError(t, fx.db.Create(&models.TimingRecord{
		ParticipantID:  fx.participant.ID,
		CheckpointID:   mid.ID,
		Timestamp:      now.Add(time.Second),
		LocalTimestamp: now.Add(time.Second),
		SyncStatus:     "synced",
		RecordType:     "checkpoint_pass",
	}).Error)

	seq, err := svc.orderedCheckpoints(fx.race.ID.UUID())
	require.NoError(t, err)
	expected, err := svc.expectedCheckpoint(fx.participant.ID.UUID(), seq)
	require.NoError(t, err)
	require.NotNil(t, expected)
	assert.Equal(t, seq[0].ID, expected.ID)
}

func TestExpectedCheckpoint_EmptySequenceAndDBError(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	svc := NewScanService(fx.db, nil)

	expected, err := svc.expectedCheckpoint(fx.participant.ID.UUID(), nil)
	require.NoError(t, err)
	assert.Nil(t, expected)

	sqlDB, err := fx.db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	_, err = svc.expectedCheckpoint(fx.participant.ID.UUID(), []models.TimingCheckpoint{{
		ID: uuidutil.NewPublicUUID(uuid.New()),
	}})
	assert.Error(t, err)
}

func TestProcessScan_CheckpointModeNilStationFallback(t *testing.T) {
	// Defensive branch: processCheckpointMode with a synthetic unbound station.
	fx := seedActiveLapFixture(t, "active")
	svc := NewScanService(fx.db, nil)
	result, err := svc.processCheckpointMode(
		&models.ReaderStation{Mode: "checkpoint"},
		fx.participant,
		fx.race,
		"",
		time.Now().UTC(),
	)
	require.NoError(t, err)
	assert.Equal(t, ResultOutOfOrder, result.Result)
}

func TestAddKaraokeBonus_ExistingLookupDBError(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	svc := NewScanService(fx.db, nil)

	scanResult, err := svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-finish-1", time.Now().UTC())
	require.NoError(t, err)

	queries := 0
	require.NoError(t, fx.db.Callback().Query().Before("gorm:query").Register("fail_karaoke_existing", func(tx *gorm.DB) {
		if tx.Statement.Table != "timing_records" {
			return
		}
		queries++
		// 1 = load source lap, 2 = existing karaoke lookup
		if queries == 2 {
			_ = tx.AddError(errors.New("connection reset"))
		}
	}))
	t.Cleanup(func() {
		_ = fx.db.Callback().Query().Remove("fail_karaoke_existing")
	})

	_, err = svc.AddKaraokeBonus(scanResult.TimingRecordID.UUID())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection reset")
}

func TestAddKaraokeBonus_CreateNonUniqueError(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	svc := NewScanService(fx.db, nil)

	scanResult, err := svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-finish-1", time.Now().UTC())
	require.NoError(t, err)

	require.NoError(t, fx.db.Callback().Create().Before("gorm:create").Register("fail_karaoke_disk", func(tx *gorm.DB) {
		rec, ok := tx.Statement.Dest.(*models.TimingRecord)
		if !ok || rec.RecordType != "karaoke_bonus" {
			return
		}
		_ = tx.AddError(errors.New("disk full"))
	}))
	t.Cleanup(func() {
		_ = fx.db.Callback().Create().Remove("fail_karaoke_disk")
	})

	_, err = svc.AddKaraokeBonus(scanResult.TimingRecordID.UUID())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "disk full")
	assert.False(t, errors.Is(err, ErrAlreadyExists))
}

func TestProcessCheckpoint_ExpectedCheckpointError(t *testing.T) {
	fx, start, _ := seedCheckpointCourseFixture(t)
	svc := NewScanService(fx.db, nil)

	require.NoError(t, fx.db.Model(&models.ReaderStation{}).
		Where("device_id = ?", "laptop-checkpoint-1").
		Update("checkpoint_id", start.ID).Error)

	queries := 0
	require.NoError(t, fx.db.Callback().Query().Before("gorm:query").Register("fail_expected_cp", func(tx *gorm.DB) {
		if tx.Statement.Table != "timing_records" {
			return
		}
		queries++
		if queries == 1 {
			_ = tx.AddError(errors.New("expected checkpoint query failed"))
		}
	}))
	t.Cleanup(func() {
		_ = fx.db.Callback().Query().Remove("fail_expected_cp")
	})

	station := &models.ReaderStation{
		Mode:         "checkpoint",
		CheckpointID: &start.ID,
		DeviceID:     "laptop-checkpoint-1",
	}
	_, err := svc.processCheckpointMode(station, fx.participant, fx.race, "laptop-checkpoint-1", time.Now().UTC())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected checkpoint query failed")
}

func TestProcessCheckpoint_CreatePassAndLapErrors(t *testing.T) {
	fx, start, mid := seedCheckpointCourseFixture(t)
	svc := NewScanService(fx.db, nil)
	now := time.Now().UTC().Truncate(time.Second)

	require.NoError(t, fx.db.Model(&models.ReaderStation{}).
		Where("device_id = ?", "laptop-checkpoint-1").
		Update("checkpoint_id", start.ID).Error)

	creates := 0
	require.NoError(t, fx.db.Callback().Create().Before("gorm:create").Register("fail_cp_create", func(tx *gorm.DB) {
		rec, ok := tx.Statement.Dest.(*models.TimingRecord)
		if !ok {
			return
		}
		if rec.RecordType == "checkpoint_pass" {
			creates++
			if creates == 1 {
				_ = tx.AddError(errors.New("pass insert failed"))
			}
		}
	}))

	station := &models.ReaderStation{
		Mode:         "checkpoint",
		CheckpointID: &start.ID,
		DeviceID:     "laptop-checkpoint-1",
	}
	_, err := svc.processCheckpointMode(station, fx.participant, fx.race, "laptop-checkpoint-1", now)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pass insert failed")
	_ = fx.db.Callback().Create().Remove("fail_cp_create")

	// Happy path through start, then fail lap create on finish.
	require.NoError(t, fx.db.Callback().Create().Before("gorm:create").Register("fail_lap_create", func(tx *gorm.DB) {
		rec, ok := tx.Statement.Dest.(*models.TimingRecord)
		if !ok || rec.RecordType != "rfid_lap" {
			return
		}
		_ = tx.AddError(errors.New("lap insert failed"))
	}))
	t.Cleanup(func() {
		_ = fx.db.Callback().Create().Remove("fail_lap_create")
	})

	_, err = svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-checkpoint-1", now)
	require.NoError(t, err)

	require.NoError(t, fx.db.Model(&models.ReaderStation{}).
		Where("device_id = ?", "laptop-checkpoint-1").
		Update("checkpoint_id", mid.ID).Error)
	_, err = svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-checkpoint-1", now.Add(time.Second))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "lap insert failed")
}

func TestExpectedCheckpoint_LastLapDBError(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	svc := NewScanService(fx.db, nil)

	seq := []models.TimingCheckpoint{*fx.finish}
	require.NoError(t, fx.db.Callback().Query().Before("gorm:query").Register("fail_last_lap", func(tx *gorm.DB) {
		if tx.Statement.Table == "timing_records" {
			_ = tx.AddError(errors.New("last lap query failed"))
		}
	}))
	t.Cleanup(func() {
		_ = fx.db.Callback().Query().Remove("fail_last_lap")
	})

	_, err := svc.expectedCheckpoint(fx.participant.ID.UUID(), seq)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "last lap query failed")
}

func TestExpectedCheckpoint_PassesFindDBError(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	svc := NewScanService(fx.db, nil)

	seq := []models.TimingCheckpoint{*fx.finish}
	queries := 0
	require.NoError(t, fx.db.Callback().Query().Before("gorm:query").Register("fail_passes_find", func(tx *gorm.DB) {
		if tx.Statement.Table != "timing_records" {
			return
		}
		queries++
		// 1 = last rfid_lap First, 2 = checkpoint_pass Find
		if queries == 2 {
			_ = tx.AddError(errors.New("passes query failed"))
		}
	}))
	t.Cleanup(func() {
		_ = fx.db.Callback().Query().Remove("fail_passes_find")
	})

	_, err := svc.expectedCheckpoint(fx.participant.ID.UUID(), seq)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "passes query failed")
}

func TestProcessCheckpoint_OrderedCheckpointsError(t *testing.T) {
	fx, start, _ := seedCheckpointCourseFixture(t)
	svc := NewScanService(fx.db, nil)

	require.NoError(t, fx.db.Callback().Query().Before("gorm:query").Register("fail_ordered_cps", func(tx *gorm.DB) {
		if tx.Statement.Table == "timing_checkpoints" {
			_ = tx.AddError(errors.New("ordered checkpoints query failed"))
		}
	}))
	t.Cleanup(func() {
		_ = fx.db.Callback().Query().Remove("fail_ordered_cps")
	})

	station := &models.ReaderStation{
		Mode:         "checkpoint",
		CheckpointID: &start.ID,
		DeviceID:     "laptop-checkpoint-1",
	}
	_, err := svc.processCheckpointMode(station, fx.participant, fx.race, "laptop-checkpoint-1", time.Now().UTC())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ordered checkpoints query failed")
}
