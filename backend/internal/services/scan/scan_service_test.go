package scan

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/keweenaw-endurance/backend/internal/uuidutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupScanTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.Event{},
		&models.Race{},
		&models.Participant{},
		&models.TimingCheckpoint{},
		&models.TimingRecord{},
		&models.Category{},
		&models.RFIDTagAssociation{},
		&models.ReaderStation{},
	))
	// Mirror production: at most one karaoke_bonus per source_lap_id.
	require.NoError(t, db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_timing_records_one_karaoke_per_source_lap
		ON timing_records (source_lap_id)
		WHERE record_type = 'karaoke_bonus' AND source_lap_id IS NOT NULL
	`).Error)
	return db
}

type scanFixture struct {
	db          *gorm.DB
	event       *models.Event
	race        *models.Race
	participant *models.Participant
	finish      *models.TimingCheckpoint
	category    *models.Category
	tagUID      string
}

func seedActiveLapFixture(t *testing.T, raceStatus string) *scanFixture {
	t.Helper()
	db := setupScanTestDB(t)

	event := &models.Event{
		Name:      "Bluffet Test",
		EventDate: time.Now().AddDate(0, 0, 1),
		Status:    "upcoming",
	}
	require.NoError(t, db.Create(event).Error)

	race := &models.Race{
		EventID:         event.ID,
		Name:            "12 Hour",
		RaceType:        "lap_based",
		DurationMinutes: 720,
		StartTime:       time.Now().Add(-time.Hour),
		Status:          raceStatus,
	}
	require.NoError(t, db.Create(race).Error)

	category := &models.Category{
		RaceID:       race.ID,
		Name:         "Advanced Men",
		CategoryType: "custom",
		GenderFilter: "male",
	}
	require.NoError(t, db.Create(category).Error)

	catID := category.ID
	participant := &models.Participant{
		RaceID:     race.ID,
		CategoryID: &catID,
		BibNumber:  "12",
		FirstName:  "Alex",
		LastName:   "Rivera",
		Gender:     "male",
		Status:     "started",
	}
	require.NoError(t, db.Create(participant).Error)

	finish := &models.TimingCheckpoint{
		RaceID:         race.ID,
		Name:           "Finish",
		CheckpointType: "finish",
		IsActive:       true,
	}
	require.NoError(t, db.Create(finish).Error)

	tagUID := "DEMO-TAG-0001"
	require.NoError(t, db.Create(&models.RFIDTagAssociation{
		ParticipantID: participant.ID,
		TagUID:        tagUID,
		Active:        true,
	}).Error)

	station := &models.ReaderStation{
		EventID:  event.ID,
		Mode:     "finish",
		Name:     "Finish Mat A",
		DeviceID: "laptop-finish-1",
	}
	require.NoError(t, db.Create(station).Error)

	return &scanFixture{
		db:          db,
		event:       event,
		race:        race,
		participant: participant,
		finish:      finish,
		category:    category,
		tagUID:      tagUID,
	}
}

func TestProcessScan_ActiveLap(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	svc := NewScanService(fx.db, nil)

	now := time.Now().UTC().Truncate(time.Second)
	result, err := svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-finish-1", now)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, ResultLap, result.Result)
	assert.Equal(t, 1, result.LapCount)
	assert.Equal(t, 1, result.Placement)
	assert.True(t, result.KaraokeAvailable)
	require.NotNil(t, result.TimingRecordID)
	require.NotNil(t, result.Participant)
	assert.Equal(t, "Alex", result.Participant.FirstName)
	assert.Equal(t, fx.race.ID, *result.RaceID)

	var count int64
	require.NoError(t, fx.db.Model(&models.TimingRecord{}).
		Where("participant_id = ? AND record_type = ?", fx.participant.ID, "rfid_lap").
		Count(&count).Error)
	assert.Equal(t, int64(1), count)

	var rec models.TimingRecord
	require.NoError(t, fx.db.First(&rec, "id = ?", result.TimingRecordID).Error)
	assert.Equal(t, "synced", rec.SyncStatus)
}

type pendingSyncResolver struct{}

func (pendingSyncResolver) ResolveSyncStatus() string { return "pending_sync" }

func TestProcessScan_PendingSyncWhenHostedUnreachable(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	svc := NewScanService(fx.db, pendingSyncResolver{})

	now := time.Now().UTC().Truncate(time.Second)
	result, err := svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-finish-1", now)
	require.NoError(t, err)
	require.Equal(t, ResultLap, result.Result)

	var rec models.TimingRecord
	require.NoError(t, fx.db.First(&rec, "id = ?", result.TimingRecordID).Error)
	assert.Equal(t, "pending_sync", rec.SyncStatus)
}

func TestProcessScan_Cooldown(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	svc := NewScanService(fx.db, nil)

	base := time.Now().UTC().Truncate(time.Second)
	first, err := svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-finish-1", base)
	require.NoError(t, err)
	assert.Equal(t, ResultLap, first.Result)

	second, err := svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-finish-1", base.Add(30*time.Second))
	require.NoError(t, err)
	assert.Equal(t, ResultCooldown, second.Result)
	assert.Greater(t, second.RetryAfterSeconds, 0)
	assert.LessOrEqual(t, second.RetryAfterSeconds, 60)

	var count int64
	require.NoError(t, fx.db.Model(&models.TimingRecord{}).
		Where("participant_id = ? AND record_type = ?", fx.participant.ID, "rfid_lap").
		Count(&count).Error)
	assert.Equal(t, int64(1), count)

	third, err := svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-finish-1", base.Add(61*time.Second))
	require.NoError(t, err)
	assert.Equal(t, ResultLap, third.Result)
	assert.Equal(t, 2, third.LapCount)
}

func TestProcessScan_TestReadWhenScheduled(t *testing.T) {
	fx := seedActiveLapFixture(t, "scheduled")
	svc := NewScanService(fx.db, nil)

	result, err := svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-finish-1", time.Now().UTC())
	require.NoError(t, err)
	assert.Equal(t, ResultTestRead, result.Result)
	assert.Equal(t, "scheduled", result.RaceStatus)
	require.NotNil(t, result.Participant)
	assert.Equal(t, fx.participant.BibNumber, result.Participant.BibNumber)

	var count int64
	require.NoError(t, fx.db.Model(&models.TimingRecord{}).
		Where("participant_id = ? AND record_type = ?", fx.participant.ID, "rfid_lap").
		Count(&count).Error)
	assert.Equal(t, int64(0), count)
}

func TestProcessScan_UnknownTag(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	svc := NewScanService(fx.db, nil)

	result, err := svc.ProcessScan(fx.event.ID.UUID(), "UNKNOWN-TAG", "laptop-finish-1", time.Now().UTC())
	require.NoError(t, err)
	assert.Equal(t, ResultUnknownTag, result.Result)
	assert.Nil(t, result.Participant)
}

func TestProcessScan_FallbackParticipantRFIDColumn(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	legacyUID := "LEGACY-TAG-99"
	require.NoError(t, fx.db.Model(fx.participant).Update("rfid_tag_uid", legacyUID).Error)

	svc := NewScanService(fx.db, nil)
	result, err := svc.ProcessScan(fx.event.ID.UUID(), legacyUID, "laptop-finish-1", time.Now().UTC())
	require.NoError(t, err)
	assert.Equal(t, ResultLap, result.Result)
}

func TestProcessScan_RejectsParticipantOutsideEvent(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")

	otherEvent := &models.Event{
		Name:      "Other",
		EventDate: time.Now(),
		Status:    "upcoming",
	}
	require.NoError(t, fx.db.Create(otherEvent).Error)

	svc := NewScanService(fx.db, nil)
	result, err := svc.ProcessScan(otherEvent.ID.UUID(), fx.tagUID, "laptop-finish-1", time.Now().UTC())
	require.NoError(t, err)
	assert.Equal(t, ResultUnknownTag, result.Result)
}

func TestProcessScan_PlacementUsesKaraokeBonus(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	svc := NewScanService(fx.db, nil)

	catID := fx.category.ID
	rival := &models.Participant{
		RaceID:     fx.race.ID,
		CategoryID: &catID,
		BibNumber:  "99",
		FirstName:  "Rival",
		LastName:   "Racer",
		Gender:     "male",
		Status:     "started",
	}
	require.NoError(t, fx.db.Create(rival).Error)
	require.NoError(t, fx.db.Create(&models.RFIDTagAssociation{
		ParticipantID: rival.ID,
		TagUID:        "DEMO-TAG-RIVAL",
		Active:        true,
	}).Error)

	base := time.Now().UTC().Truncate(time.Second)
	_, err := svc.ProcessScan(fx.event.ID.UUID(), "DEMO-TAG-RIVAL", "laptop-finish-1", base)
	require.NoError(t, err)

	var rivalLap models.TimingRecord
	require.NoError(t, fx.db.Where("participant_id = ? AND record_type = ?", rival.ID, "rfid_lap").First(&rivalLap).Error)
	bonus := &models.TimingRecord{
		ParticipantID:  rival.ID,
		CheckpointID:   fx.finish.ID,
		Timestamp:      base.Add(time.Second),
		LocalTimestamp: base.Add(time.Second),
		RecordType:     "karaoke_bonus",
		SourceLapID:    &rivalLap.ID,
		DeviceID:       "laptop-finish-1",
	}
	require.NoError(t, fx.db.Create(bonus).Error)

	result, err := svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-finish-1", base.Add(2*time.Second))
	require.NoError(t, err)
	assert.Equal(t, ResultLap, result.Result)
	assert.Equal(t, 1, result.LapCount)
	assert.Equal(t, 2, result.Placement) // rival has rfid_lap + karaoke = 2
}

func TestProcessScan_EmptyTag(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	svc := NewScanService(fx.db, nil)
	result, err := svc.ProcessScan(fx.event.ID.UUID(), "  ", "", time.Time{})
	require.NoError(t, err)
	assert.Equal(t, ResultUnknownTag, result.Result)
}

func TestProcessScan_ZeroTimestampUsesNow(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	svc := NewScanService(fx.db, nil)
	result, err := svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-finish-1", time.Time{})
	require.NoError(t, err)
	assert.Equal(t, ResultLap, result.Result)
}

func TestProcessScan_FinishedRaceIsTestRead(t *testing.T) {
	fx := seedActiveLapFixture(t, "finished")
	svc := NewScanService(fx.db, nil)
	result, err := svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-finish-1", time.Now().UTC())
	require.NoError(t, err)
	assert.Equal(t, ResultTestRead, result.Result)
	assert.Equal(t, "finished", result.RaceStatus)
}

func TestProcessScan_CancelledRaceIsTestRead(t *testing.T) {
	fx := seedActiveLapFixture(t, "cancelled")
	svc := NewScanService(fx.db, nil)
	result, err := svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-finish-1", time.Now().UTC())
	require.NoError(t, err)
	assert.Equal(t, ResultTestRead, result.Result)
	assert.Equal(t, "cancelled", result.RaceStatus)
	assert.Nil(t, result.TimingRecordID)
}

func TestProcessScan_UsesStartCheckpointFallback(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	require.NoError(t, fx.db.Delete(fx.finish).Error)
	start := &models.TimingCheckpoint{
		RaceID:         fx.race.ID,
		Name:           "Start",
		CheckpointType: "start",
		IsActive:       true,
	}
	require.NoError(t, fx.db.Create(start).Error)

	svc := NewScanService(fx.db, nil)
	result, err := svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-finish-1", time.Now().UTC())
	require.NoError(t, err)
	assert.Equal(t, ResultLap, result.Result)
}

func TestProcessScan_NoCheckpointErrors(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	require.NoError(t, fx.db.Delete(fx.finish).Error)
	svc := NewScanService(fx.db, nil)
	_, err := svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-finish-1", time.Now().UTC())
	require.Error(t, err)
}

func TestProcessScan_UncategorizedParticipant(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	require.NoError(t, fx.db.Model(fx.participant).Update("category_id", nil).Error)
	svc := NewScanService(fx.db, nil)
	result, err := svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-finish-1", time.Now().UTC())
	require.NoError(t, err)
	assert.Equal(t, ResultLap, result.Result)
	assert.Equal(t, 0, result.PlacementCategory)
}

func TestProcessScan_OrphanAssociation(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	orphanUID := "ORPHAN-TAG"
	require.NoError(t, fx.db.Create(&models.RFIDTagAssociation{
		ParticipantID: uuidutil.NewPublicUUID(uuid.New()),
		TagUID:        orphanUID,
		Active:        true,
	}).Error)
	svc := NewScanService(fx.db, nil)
	result, err := svc.ProcessScan(fx.event.ID.UUID(), orphanUID, "laptop-finish-1", time.Now().UTC())
	require.NoError(t, err)
	assert.Equal(t, ResultUnknownTag, result.Result)
}

func TestProcessScan_LegacyOutsideEvent(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	legacyUID := "LEGACY-OUT"
	require.NoError(t, fx.db.Model(fx.participant).Update("rfid_tag_uid", legacyUID).Error)

	otherEvent := &models.Event{Name: "Other", EventDate: time.Now(), Status: "upcoming"}
	require.NoError(t, fx.db.Create(otherEvent).Error)

	svc := NewScanService(fx.db, nil)
	result, err := svc.ProcessScan(otherEvent.ID.UUID(), legacyUID, "", time.Now().UTC())
	require.NoError(t, err)
	assert.Equal(t, ResultUnknownTag, result.Result)
}

func TestProcessScan_DBErrors(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	svc := NewScanService(fx.db, nil)
	sqlDB, err := fx.db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	_, err = svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-finish-1", time.Now().UTC())
	require.Error(t, err)

	_, err = svc.ProcessScan(fx.event.ID.UUID(), "UNKNOWN", "", time.Now().UTC())
	require.Error(t, err)

	assert.Equal(t, 0, svc.cooldownRemaining(fx.participant.ID.UUID(), time.Now()))
	_, err = svc.finishCheckpoint(fx.race.ID.UUID())
	require.Error(t, err)
	_, err = svc.scoredLapCount(fx.participant.ID.UUID())
	require.Error(t, err)
	_, err = svc.scoreRace(fx.race.ID.UUID())
	require.Error(t, err)
	_, _, err = svc.placements(fx.race.ID.UUID(), fx.participant)
	require.Error(t, err)
	assert.Equal(t, "finish", svc.stationMode(fx.event.ID.UUID(), "x"))
}

func TestProcessScan_CreateRecordError(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	require.NoError(t, fx.db.Callback().Create().Before("gorm:create").Register("fail_timing", func(db *gorm.DB) {
		if db.Statement.Schema != nil && db.Statement.Schema.Table == "timing_records" {
			_ = db.AddError(assert.AnError)
		}
	}))
	svc := NewScanService(fx.db, nil)
	_, err := svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-finish-1", time.Now().UTC())
	require.Error(t, err)
}

func TestProcessScan_RaceZeroIDUnknown(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	require.NoError(t, fx.db.Exec("PRAGMA foreign_keys = OFF").Error)
	require.NoError(t, fx.db.Exec("DELETE FROM races WHERE id = ?", fx.race.ID).Error)
	svc := NewScanService(fx.db, nil)
	result, err := svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-finish-1", time.Now().UTC())
	require.NoError(t, err)
	assert.Equal(t, ResultUnknownTag, result.Result)
}

func TestPlacements_CategoryWithNoMatchingEntry(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	svc := NewScanService(fx.db, nil)
	overall, cat, err := svc.placements(fx.race.ID.UUID(), fx.participant)
	require.NoError(t, err)
	assert.Equal(t, 0, overall)
	assert.Equal(t, 0, cat)
}



func TestStationMode_EmptyModeAndMissing(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	svc := NewScanService(fx.db, nil)
	assert.Equal(t, "finish", svc.stationMode(fx.event.ID.UUID(), "laptop-finish-1"))
	assert.Equal(t, "finish", svc.stationMode(fx.event.ID.UUID(), "unknown-device"))
	require.NoError(t, fx.db.Model(&models.ReaderStation{}).Where("event_id = ?", fx.event.ID).Update("mode", "checkpoint").Error)
	assert.Equal(t, "checkpoint", svc.stationMode(fx.event.ID.UUID(), "laptop-finish-1"))
	assert.Equal(t, "checkpoint", svc.stationMode(fx.event.ID.UUID(), ""))
}

func TestCooldownRemaining_MinimumOneSecond(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	svc := NewScanService(fx.db, nil)
	base := time.Now().UTC().Truncate(time.Millisecond)
	_, err := svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-finish-1", base)
	require.NoError(t, err)

	// Elapsed almost 60s → ceil should still be >= 1
	retry := svc.cooldownRemaining(fx.participant.ID.UUID(), base.Add(CooldownDuration-time.Nanosecond))
	assert.Equal(t, 1, retry)
}

func TestScoreRace_TieBreakEarliestLastLap(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	svc := NewScanService(fx.db, nil)

	catID := fx.category.ID
	other := &models.Participant{
		RaceID: fx.race.ID, CategoryID: &catID, BibNumber: "50",
		FirstName: "Other", LastName: "Racer", Gender: "male", Status: "started",
	}
	require.NoError(t, fx.db.Create(other).Error)
	require.NoError(t, fx.db.Create(&models.RFIDTagAssociation{
		ParticipantID: other.ID, TagUID: "TIE-TAG", Active: true,
	}).Error)

	base := time.Now().UTC().Truncate(time.Second)
	_, err := svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-finish-1", base.Add(10*time.Second))
	require.NoError(t, err)
	_, err = svc.ProcessScan(fx.event.ID.UUID(), "TIE-TAG", "laptop-finish-1", base)
	require.NoError(t, err)

	result, err := svc.ProcessScan(fx.event.ID.UUID(), "TIE-TAG", "laptop-finish-1", base.Add(70*time.Second))
	require.NoError(t, err)
	// After second lap for other, placements differ; also exercises tie-break path when equal.
	assert.Equal(t, ResultLap, result.Result)
	assert.Equal(t, 1, result.Placement)
}

func TestPlacements_SkipsDifferentCategory(t *testing.T) {
	fx := seedActiveLapFixture(t, "active")
	otherCat := &models.Category{
		RaceID: fx.race.ID, Name: "Advanced Women", CategoryType: "custom", GenderFilter: "female",
	}
	require.NoError(t, fx.db.Create(otherCat).Error)
	catID := otherCat.ID
	woman := &models.Participant{
		RaceID: fx.race.ID, CategoryID: &catID, BibNumber: "20",
		FirstName: "Pat", LastName: "Lee", Gender: "female", Status: "started",
	}
	require.NoError(t, fx.db.Create(woman).Error)
	require.NoError(t, fx.db.Create(&models.RFIDTagAssociation{
		ParticipantID: woman.ID, TagUID: "WOMEN-TAG", Active: true,
	}).Error)

	svc := NewScanService(fx.db, nil)
	base := time.Now().UTC()
	_, err := svc.ProcessScan(fx.event.ID.UUID(), "WOMEN-TAG", "laptop-finish-1", base)
	require.NoError(t, err)
	result, err := svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-finish-1", base.Add(time.Second))
	require.NoError(t, err)
	assert.Equal(t, 1, result.PlacementCategory)
	assert.Equal(t, 2, result.Placement)
}

