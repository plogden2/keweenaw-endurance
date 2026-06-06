package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/keweenaw-endurance/backend/internal/uuidutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func createCheckpoint(t *testing.T, db *gorm.DB, raceID uuidutil.PublicUUID, name, cpType string) *models.TimingCheckpoint {
	t.Helper()
	svc := NewCheckpointService(db)
	cp, err := svc.CreateCheckpoint(&models.TimingCheckpoint{
		RaceID:         raceID,
		Name:           name,
		CheckpointType: cpType,
	})
	require.NoError(t, err)
	return cp
}

func TestTimingService_CreateAndGet(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	start := createCheckpoint(t, db, race.ID, "Start", "start")

	partSvc := NewParticipantService(db)
	participant, err := partSvc.CreateParticipant(&models.Participant{
		RaceID:    race.ID,
		BibNumber: "101",
		FirstName: "Ada",
		LastName:  "Lovelace",
	})
	require.NoError(t, err)

	svc := NewTimingService(db)
	now := time.Now().UTC().Truncate(time.Second)
	record, err := svc.CreateRecord(&models.TimingRecord{
		ParticipantID:  participant.ID,
		CheckpointID:   start.ID,
		Timestamp:      now,
		LocalTimestamp: now,
		DeviceID:       "station-1",
	})
	require.NoError(t, err)
	assert.Equal(t, "synced", record.SyncStatus)

	fetched, err := svc.GetRecord(record.ID.UUID())
	require.NoError(t, err)
	assert.Equal(t, participant.ID, fetched.ParticipantID)
}

func TestTimingService_CreateValidation(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	svc := NewTimingService(db)
	now := time.Now()

	_, err := svc.CreateRecord(&models.TimingRecord{
		ParticipantID: uuidutil.NewPublicUUID(uuid.New()),
		CheckpointID:  uuidutil.NewPublicUUID(uuid.New()),
		Timestamp:     now,
	})
	assert.ErrorIs(t, err, ErrInvalidTimingInput)

	partSvc := NewParticipantService(db)
	participant, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "1", FirstName: "A", LastName: "B",
	})
	require.NoError(t, err)

	otherRace := createTestRace(t, db)
	otherCP := createCheckpoint(t, db, otherRace.ID, "Start", "start")

	_, err = svc.CreateRecord(&models.TimingRecord{
		ParticipantID:  participant.ID,
		CheckpointID:   otherCP.ID,
		Timestamp:      now,
		LocalTimestamp: now,
	})
	assert.ErrorIs(t, err, ErrInvalidTimingInput)
}

func TestTimingService_UpdateRecord(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	start := createCheckpoint(t, db, race.ID, "Start", "start")
	partSvc := NewParticipantService(db)
	participant, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "2", FirstName: "Grace", LastName: "Hopper",
	})
	require.NoError(t, err)

	svc := NewTimingService(db)
	now := time.Now().UTC().Truncate(time.Second)
	record, err := svc.CreateRecord(&models.TimingRecord{
		ParticipantID: participant.ID, CheckpointID: start.ID,
		Timestamp: now, LocalTimestamp: now,
	})
	require.NoError(t, err)

	updatedTime := now.Add(time.Minute)
	updated, err := svc.UpdateRecord(record.ID.UUID(), &models.TimingRecord{
		Timestamp: updatedTime, LocalTimestamp: updatedTime,
		SyncStatus: "pending_sync",
	})
	require.NoError(t, err)
	assert.Equal(t, "pending_sync", updated.SyncStatus)
}

func TestTimingService_ListByRace(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	start := createCheckpoint(t, db, race.ID, "Start", "start")
	partSvc := NewParticipantService(db)
	participant, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "3", FirstName: "K", LastName: "P",
	})
	require.NoError(t, err)

	svc := NewTimingService(db)
	now := time.Now()
	_, err = svc.CreateRecord(&models.TimingRecord{
		ParticipantID: participant.ID, CheckpointID: start.ID,
		Timestamp: now, LocalTimestamp: now,
	})
	require.NoError(t, err)

	records, err := svc.ListRecordsByRace(race.ID.UUID())
	require.NoError(t, err)
	assert.Len(t, records, 1)
}
