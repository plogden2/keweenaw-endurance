package services

import (
	"testing"
	"time"

	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/keweenaw-endurance/backend/internal/rfid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRFIDService_LookupByUID(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	partSvc := NewParticipantService(db)

	participant, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "42", FirstName: "Test", LastName: "Runner",
		RFIDTagUID: "E0040150ABCD1234",
	})
	require.NoError(t, err)

	svc := NewRFIDService(db, rfid.NewMockReader())

	found, err := svc.LookupParticipantByUID("E0040150ABCD1234")
	require.NoError(t, err)
	assert.Equal(t, participant.ID, found.ID)
	assert.Equal(t, "42", found.BibNumber)

	_, err = svc.LookupParticipantByUID("UNKNOWN-TAG")
	assert.ErrorIs(t, err, ErrRFIDTagNotFound)
}

func TestRFIDService_WriteTag(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	partSvc := NewParticipantService(db)

	participant, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "7", FirstName: "Write", LastName: "Test",
	})
	require.NoError(t, err)

	mock := rfid.NewMockReader()
	svc := NewRFIDService(db, mock)

	updated, err := svc.WriteTag(participant.ID.UUID(), "NEW-TAG-001")
	require.NoError(t, err)
	assert.Equal(t, "NEW-TAG-001", updated.RFIDTagUID)
	assert.Equal(t, []string{"NEW-TAG-001"}, updated.TagUIDs)
	assert.Equal(t, "NEW-TAG-001", mock.LastUID)
	assert.Equal(t, participant.ID.Short(), mock.LastData)
}

func TestRFIDService_MultiTagAssociationCRUD(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	partSvc := NewParticipantService(db)

	participant, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "12", FirstName: "Multi", LastName: "Tag",
	})
	require.NoError(t, err)

	svc := NewRFIDService(db, rfid.NewMockReader())

	a1, err := svc.AssociateTag(participant.ID.UUID(), "TAG-A")
	require.NoError(t, err)
	assert.Equal(t, "TAG-A", a1.TagUID)

	a2, err := svc.AssociateTag(participant.ID.UUID(), "TAG-B")
	require.NoError(t, err)
	assert.Equal(t, "TAG-B", a2.TagUID)

	tags, err := svc.ListParticipantTags(participant.ID.UUID())
	require.NoError(t, err)
	require.Len(t, tags, 2)
	assert.Equal(t, "TAG-A", tags[0].TagUID)
	assert.Equal(t, "TAG-B", tags[1].TagUID)

	foundA, err := svc.LookupParticipantByUID("TAG-A")
	require.NoError(t, err)
	assert.Equal(t, participant.ID, foundA.ID)
	assert.ElementsMatch(t, []string{"TAG-A", "TAG-B"}, foundA.TagUIDs)

	foundB, err := svc.LookupParticipantByUID("TAG-B")
	require.NoError(t, err)
	assert.Equal(t, participant.ID, foundB.ID)

	// Idempotent re-associate same tag
	again, err := svc.AssociateTag(participant.ID.UUID(), "TAG-A")
	require.NoError(t, err)
	assert.Equal(t, a1.ID, again.ID)

	other, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "13", FirstName: "Other", LastName: "Racer",
	})
	require.NoError(t, err)
	_, err = svc.AssociateTag(other.ID.UUID(), "TAG-A")
	assert.ErrorIs(t, err, ErrInvalidRFIDInput)

	raceID := race.ID.UUID()
	listed, _, err := partSvc.ListParticipants(1, 50, &raceID, "")
	require.NoError(t, err)
	var multi *models.Participant
	for i := range listed {
		if listed[i].ID == participant.ID {
			multi = &listed[i]
			break
		}
	}
	require.NotNil(t, multi)
	assert.ElementsMatch(t, []string{"TAG-A", "TAG-B"}, multi.TagUIDs)
}

func TestRFIDService_WriteTag_HardwareUnavailable(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	partSvc := NewParticipantService(db)

	participant, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "8", FirstName: "No", LastName: "Hardware",
	})
	require.NoError(t, err)

	mock := rfid.NewMockReader()
	mock.Available = false
	svc := NewRFIDService(db, mock)

	_, err = svc.WriteTag(participant.ID.UUID(), "TAG-002")
	assert.ErrorIs(t, err, ErrHardwareUnavailable)
}

func TestRFIDService_ManualEntry_ByBib(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	checkpoint := createCheckpoint(t, db, race.ID, "Start", "start")

	partSvc := NewParticipantService(db)
	participant, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "101", FirstName: "Manual", LastName: "Entry",
	})
	require.NoError(t, err)

	svc := NewRFIDService(db, rfid.NewMockReader())
	now := time.Now().UTC().Truncate(time.Second)

	record, err := svc.ManualEntry(&ManualEntryInput{
		RaceID:       race.ID.UUID(),
		CheckpointID: checkpoint.ID.UUID(),
		BibNumber:    "101",
		Timestamp:    now,
		DeviceID:     "station-1",
	})
	require.NoError(t, err)
	assert.Equal(t, participant.ID, record.ParticipantID)
	assert.Equal(t, "synced", record.SyncStatus)
}

func TestRFIDService_ManualEntry_ByRFID(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	checkpoint := createCheckpoint(t, db, race.ID, "Finish", "finish")

	partSvc := NewParticipantService(db)
	participant, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "202", FirstName: "RFID", LastName: "Scan",
		RFIDTagUID: "SCAN-UID-999",
	})
	require.NoError(t, err)

	svc := NewRFIDService(db, rfid.NewMockReader())
	now := time.Now().UTC().Truncate(time.Second)

	record, err := svc.ManualEntry(&ManualEntryInput{
		RaceID:       race.ID.UUID(),
		CheckpointID: checkpoint.ID.UUID(),
		RFIDTagUID:   "SCAN-UID-999",
		Timestamp:    now,
		SyncStatus:   "pending_sync",
	})
	require.NoError(t, err)
	assert.Equal(t, participant.ID, record.ParticipantID)
	assert.Equal(t, "pending_sync", record.SyncStatus)
}

func TestRFIDService_ManualEntry_Validation(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	checkpoint := createCheckpoint(t, db, race.ID, "Start", "start")
	svc := NewRFIDService(db, rfid.NewMockReader())
	now := time.Now().UTC()

	_, err := svc.ManualEntry(&ManualEntryInput{
		RaceID: race.ID.UUID(), CheckpointID: checkpoint.ID.UUID(), Timestamp: now,
	})
	assert.ErrorIs(t, err, ErrInvalidRFIDInput)

	_, err = svc.ManualEntry(&ManualEntryInput{
		RaceID: race.ID.UUID(), CheckpointID: checkpoint.ID.UUID(), BibNumber: "999", Timestamp: now,
	})
	assert.ErrorIs(t, err, ErrParticipantNotFound)
}

func TestRFIDService_GetSyncStatus(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	checkpoint := createCheckpoint(t, db, race.ID, "Start", "start")

	partSvc := NewParticipantService(db)
	participant, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "1", FirstName: "A", LastName: "B",
	})
	require.NoError(t, err)

	timing := NewTimingService(db)
	now := time.Now().UTC()
	_, err = timing.CreateRecord(&models.TimingRecord{
		ParticipantID: participant.ID, CheckpointID: checkpoint.ID,
		Timestamp: now, LocalTimestamp: now, SyncStatus: "pending_sync",
	})
	require.NoError(t, err)
	_, err = timing.CreateRecord(&models.TimingRecord{
		ParticipantID: participant.ID, CheckpointID: checkpoint.ID,
		Timestamp: now.Add(time.Minute), LocalTimestamp: now.Add(time.Minute),
		SyncStatus: "synced",
	})
	require.NoError(t, err)

	svc := NewRFIDService(db, rfid.NewMockReader())
	status, err := svc.GetSyncStatus()
	require.NoError(t, err)
	assert.Equal(t, int64(1), status.PendingCount)
	assert.Equal(t, int64(1), status.SyncedCount)
	assert.Equal(t, int64(0), status.FailedCount)
}

func TestRFIDService_SyncPending(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	checkpoint := createCheckpoint(t, db, race.ID, "Start", "start")

	partSvc := NewParticipantService(db)
	participant, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "1", FirstName: "A", LastName: "B",
	})
	require.NoError(t, err)

	timing := NewTimingService(db)
	now := time.Now().UTC()
	record, err := timing.CreateRecord(&models.TimingRecord{
		ParticipantID: participant.ID, CheckpointID: checkpoint.ID,
		Timestamp: now, LocalTimestamp: now, SyncStatus: "pending_sync",
	})
	require.NoError(t, err)

	svc := NewRFIDService(db, rfid.NewMockReader())
	synced, err := svc.SyncPending()
	require.NoError(t, err)
	assert.Equal(t, int64(1), synced)

	updated, err := timing.GetRecord(record.ID.UUID())
	require.NoError(t, err)
	assert.Equal(t, "synced", updated.SyncStatus)
}

func TestRFIDService_LookupByUID_Empty(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewRFIDService(db, rfid.NewMockReader())

	_, err := svc.LookupParticipantByUID("")
	assert.ErrorIs(t, err, ErrInvalidRFIDInput)
}

func TestRFIDService_ManualEntry_WrongRaceForRFID(t *testing.T) {
	db := setupServiceTestDB(t)
	race1 := createTestRace(t, db)
	race2 := createTestRace(t, db)
	checkpoint := createCheckpoint(t, db, race2.ID, "Start", "start")

	partSvc := NewParticipantService(db)
	_, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: race1.ID, BibNumber: "55", FirstName: "Wrong", LastName: "Race",
		RFIDTagUID: "TAG-RACE-1",
	})
	require.NoError(t, err)

	svc := NewRFIDService(db, rfid.NewMockReader())
	_, err = svc.ManualEntry(&ManualEntryInput{
		RaceID: race2.ID.UUID(), CheckpointID: checkpoint.ID.UUID(),
		RFIDTagUID: "TAG-RACE-1", Timestamp: time.Now().UTC(),
	})
	assert.ErrorIs(t, err, ErrInvalidRFIDInput)
}
