package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/keweenaw-endurance/backend/internal/rfid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func createTestRaceForEvent(t *testing.T, db *gorm.DB, event *models.Event) *models.Race {
	t.Helper()
	race, err := NewRaceService(db).CreateRace(&models.Race{
		EventID:         event.ID,
		Name:            "Test Race",
		RaceType:        "time_based",
		DistanceKm:      10,
		DurationMinutes: 60,
	})
	require.NoError(t, err)
	return race
}

func TestCSVExport_WriteLiveSnapshotContainsSections(t *testing.T) {
	db := setupServiceTestDB(t)
	dir := t.TempDir()
	svc := NewCSVExportService(db, dir)

	event := createTestEvent(t, db)
	race, err := NewRaceService(db).CreateRace(&models.Race{
		EventID:         event.ID,
		Name:            "12 Hour",
		RaceType:        "time_based",
		DistanceKm:      50,
		DurationMinutes: 720,
	})
	require.NoError(t, err)

	cat, err := NewCategoryService(db).CreateCategory(&models.Category{
		RaceID:       race.ID,
		Name:         "Open",
		CategoryType: "overall",
		DisplayOrder: 1,
	})
	require.NoError(t, err)

	catID := cat.ID
	part, err := NewParticipantService(db).CreateParticipant(&models.Participant{
		RaceID:     race.ID,
		CategoryID: &catID,
		BibNumber:  "42",
		FirstName:  "Ada",
		LastName:   "Lovelace",
		Gender:     "female",
	})
	require.NoError(t, err)

	_, err = NewRFIDService(db, rfid.NewMockReader()).AssociateTag(part.ID.UUID(), "DEMO-TAG-0001")
	require.NoError(t, err)

	cp := createCheckpoint(t, db, race.ID, "Finish", "finish")
	now := time.Now().UTC().Truncate(time.Second)
	require.NoError(t, db.Create(&models.TimingRecord{
		ParticipantID:  part.ID,
		CheckpointID:   cp.ID,
		Timestamp:      now,
		LocalTimestamp: now,
		DeviceID:       "laptop-finish-1",
		SyncStatus:     "synced",
		RecordType:     "rfid_lap",
	}).Error)

	path, err := svc.WriteLiveSnapshot(event.ID.UUID())
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "events", event.ID.String(), "live-snapshot.csv"), path)

	body, err := os.ReadFile(path)
	require.NoError(t, err)
	text := string(body)

	assert.Contains(t, text, "#SECTION,event")
	assert.Contains(t, text, "#SECTION,races")
	assert.Contains(t, text, "#SECTION,categories")
	assert.Contains(t, text, "#SECTION,participants")
	assert.Contains(t, text, "#SECTION,tags")
	assert.Contains(t, text, "#SECTION,checkpoints")
	assert.Contains(t, text, "#SECTION,timing_records")
	assert.Contains(t, text, "DEMO-TAG-0001")
	assert.Contains(t, text, "Ada")
	assert.Contains(t, text, event.ID.String())
}

func TestCSVExport_LiveSnapshotUpdatesOnRefresh(t *testing.T) {
	db := setupServiceTestDB(t)
	dir := t.TempDir()
	svc := NewCSVExportService(db, dir)

	event := createTestEvent(t, db)
	race := createTestRaceForEvent(t, db, event)
	part := createTestParticipant(t, db, race.ID, "1")

	path, err := svc.WriteLiveSnapshot(event.ID.UUID())
	require.NoError(t, err)
	info1, err := os.Stat(path)
	require.NoError(t, err)

	time.Sleep(20 * time.Millisecond)

	_, err = NewParticipantService(db).UpdateParticipant(part.ID.UUID(), &models.Participant{
		FirstName: "Updated",
	})
	require.NoError(t, err)

	_, err = svc.WriteLiveSnapshot(event.ID.UUID())
	require.NoError(t, err)
	info2, err := os.Stat(path)
	require.NoError(t, err)
	assert.True(t, !info2.ModTime().Before(info1.ModTime()))

	body, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(body), "Updated")
}

func TestCSVExport_RoundTripPreservesLapsAndTags(t *testing.T) {
	db := setupServiceTestDB(t)
	dir := t.TempDir()
	svc := NewCSVExportService(db, dir)

	event := createTestEvent(t, db)
	race := createTestRaceForEvent(t, db, event)
	cp := createCheckpoint(t, db, race.ID, "Finish", "finish")
	part := createTestParticipant(t, db, race.ID, "7")
	_, err := NewRFIDService(db, rfid.NewMockReader()).AssociateTag(part.ID.UUID(), "TAG-ROUNDTRIP")
	require.NoError(t, err)

	now := time.Now().UTC().Truncate(time.Second)
	require.NoError(t, db.Create(&models.TimingRecord{
		ParticipantID:  part.ID,
		CheckpointID:   cp.ID,
		Timestamp:      now,
		LocalTimestamp: now,
		DeviceID:       "station-a",
		SyncStatus:     "synced",
		RecordType:     "rfid_lap",
	}).Error)
	require.NoError(t, db.Create(&models.TimingRecord{
		ParticipantID:  part.ID,
		CheckpointID:   cp.ID,
		Timestamp:      now.Add(time.Minute),
		LocalTimestamp: now.Add(time.Minute),
		DeviceID:       "station-a",
		SyncStatus:     "synced",
		RecordType:     "rfid_lap",
	}).Error)

	path, err := svc.WriteLiveSnapshot(event.ID.UUID())
	require.NoError(t, err)
	csvBytes, err := os.ReadFile(path)
	require.NoError(t, err)

	summary, err := svc.ImportCSV(event.ID.UUID(), csvBytes)
	require.NoError(t, err)
	assert.Equal(t, 1, summary.Races)
	assert.Equal(t, 1, summary.Racers)
	assert.Equal(t, 1, summary.TagAssociations)
	assert.Equal(t, 2, summary.TimingRecords)
	assert.Equal(t, event.Name, summary.EventName)

	var tags []models.RFIDTagAssociation
	require.NoError(t, db.Find(&tags).Error)
	require.Len(t, tags, 1)
	assert.Equal(t, "TAG-ROUNDTRIP", tags[0].TagUID)

	var laps int64
	require.NoError(t, db.Model(&models.TimingRecord{}).
		Where("participant_id = ? AND record_type = ?", part.ID, "rfid_lap").
		Count(&laps).Error)
	assert.Equal(t, int64(2), laps)

	afterPath, err := svc.LiveSnapshotPath(event.ID.UUID())
	require.NoError(t, err)
	_, err = os.Stat(afterPath)
	require.NoError(t, err)
}

func TestCSVExport_ImportDoesNotDeleteUnrelatedEvents(t *testing.T) {
	db := setupServiceTestDB(t)
	dir := t.TempDir()
	svc := NewCSVExportService(db, dir)

	keep := createTestEvent(t, db)
	keepRace := createTestRaceForEvent(t, db, keep)
	_ = createTestParticipant(t, db, keepRace.ID, "99")

	target := createTestEvent(t, db)
	targetRace := createTestRaceForEvent(t, db, target)
	_ = createTestParticipant(t, db, targetRace.ID, "1")

	csvBytes, err := svc.BuildCSV(target.ID.UUID())
	require.NoError(t, err)

	_, err = svc.ImportCSV(target.ID.UUID(), csvBytes)
	require.NoError(t, err)

	var keepParts int64
	require.NoError(t, db.Model(&models.Participant{}).Where("race_id = ?", keepRace.ID).Count(&keepParts).Error)
	assert.Equal(t, int64(1), keepParts)
}

func TestCSVExport_ReadLiveSnapshotAndStatus(t *testing.T) {
	db := setupServiceTestDB(t)
	dir := t.TempDir()
	svc := NewCSVExportService(db, dir)

	event := createTestEvent(t, db)
	_, err := svc.WriteLiveSnapshot(event.ID.UUID())
	require.NoError(t, err)

	status, err := svc.LiveSnapshotStatus(event.ID.UUID())
	require.NoError(t, err)
	assert.True(t, status.Exists)
	assert.False(t, status.UpdatedAt.IsZero())
	assert.Greater(t, status.SizeBytes, int64(0))
	assert.True(t, strings.Contains(status.Path, "live-snapshot.csv"))

	body, updatedAt, err := svc.ReadLiveSnapshot(event.ID.UUID())
	require.NoError(t, err)
	assert.NotEmpty(t, body)
	assert.False(t, updatedAt.IsZero())
	assert.Contains(t, string(body), "#SECTION,event")
}
