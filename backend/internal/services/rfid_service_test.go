package services

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/config"
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
	svc.ConfigureBridge(&config.Config{RFID: config.RFIDConfig{Hardware: true}}, nil)

	updated, err := svc.WriteTag(participant.ID.UUID())
	require.NoError(t, err)
	require.NotEmpty(t, updated.RFIDTagUID)
	_, parseErr := uuid.Parse(updated.RFIDTagUID)
	require.NoError(t, parseErr)
	assert.Equal(t, []string{updated.RFIDTagUID}, updated.TagUIDs)

	uid, err := mock.Poll()
	require.NoError(t, err)
	assert.Equal(t, strings.ToLower(updated.RFIDTagUID), uid)
}

func TestWriteTag_ProgramsLogicalUUIDWithoutSilicon(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	partSvc := NewParticipantService(db)
	p, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "99", FirstName: "Logical", LastName: "Write",
	})
	require.NoError(t, err)

	mock := rfid.NewMockReader()
	svc := NewRFIDService(db, mock)
	svc.ConfigureBridge(&config.Config{RFID: config.RFIDConfig{Hardware: true}}, nil)
	logical := uuid.New().String()
	_, err = svc.AssociateTag(p.ID.UUID(), logical)
	require.NoError(t, err)

	_, err = svc.WriteTag(p.ID.UUID())
	require.NoError(t, err)

	got, err := mock.Poll()
	require.NoError(t, err)
	require.Equal(t, strings.ToLower(logical), got)

	found, err := svc.LookupParticipantByUID(got)
	require.NoError(t, err)
	require.Equal(t, p.ID, found.ID)
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

func TestRFIDService_WriteTag_ReusesLegacyRFIDTagUID(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	partSvc := NewParticipantService(db)

	const legacyUID = "550e8400-e29b-41d4-a716-446655440099"
	participant, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "19", FirstName: "Legacy", LastName: "Column",
		RFIDTagUID: legacyUID,
	})
	require.NoError(t, err)

	mock := rfid.NewMockReader()
	svc := NewRFIDService(db, mock)
	svc.ConfigureBridge(&config.Config{RFID: config.RFIDConfig{Hardware: true}}, nil)

	updated, err := svc.WriteTag(participant.ID.UUID())
	require.NoError(t, err)
	assert.Equal(t, legacyUID, updated.RFIDTagUID)
	assert.Equal(t, []string{legacyUID}, updated.TagUIDs)

	uid, err := mock.Poll()
	require.NoError(t, err)
	assert.Equal(t, strings.ToLower(legacyUID), uid)
}

func TestRFIDService_WriteTag_RetriesSameLogicalAfterWriteFailure(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	partSvc := NewParticipantService(db)

	participant, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "20", FirstName: "Retry", LastName: "Write",
	})
	require.NoError(t, err)

	mock := rfid.NewMockReader()
	svc := NewRFIDService(db, mock)
	svc.ConfigureBridge(&config.Config{RFID: config.RFIDConfig{Hardware: true}}, nil)

	mock.WriteErr = errors.New("hardware write failed")
	_, err = svc.WriteTag(participant.ID.UUID())
	require.Error(t, err)

	afterFail, err := partSvc.GetParticipant(participant.ID.UUID())
	require.NoError(t, err)
	require.NotEmpty(t, afterFail.RFIDTagUID)
	firstLogical := afterFail.RFIDTagUID

	mock.WriteErr = nil
	written, err := svc.WriteTag(participant.ID.UUID())
	require.NoError(t, err)
	assert.Equal(t, firstLogical, written.RFIDTagUID)

	uid, err := mock.Poll()
	require.NoError(t, err)
	assert.Equal(t, strings.ToLower(firstLogical), uid)
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
	svc.ConfigureBridge(&config.Config{RFID: config.RFIDConfig{Hardware: true}}, nil)

	_, err = svc.WriteTag(participant.ID.UUID())
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

func TestRFIDService_WriteTag_ViaBridge(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	partSvc := NewParticipantService(db)

	participant, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "77", FirstName: "Bridge", LastName: "Write",
	})
	require.NoError(t, err)

	hub := NewBridgeHub()
	conn := newFakeBridgeConn(4)
	hub.Register("laptop-finish-1", conn)

	svc := NewRFIDService(db, rfid.NewMockReader())
	svc.ConfigureBridge(&config.Config{RFID: config.RFIDConfig{Hardware: false, BridgeDeviceID: "laptop-finish-1"}}, hub)

	done := make(chan struct{})
	go func() {
		msg, err := conn.awaitOutbound(time.Second)
		require.NoError(t, err)
		ok := true
		require.NoError(t, hub.HandleMessage("laptop-finish-1", &BridgeMessage{
			Type:      "write_ack",
			RequestID: msg.RequestID,
			OK:        &ok,
		}))
		close(done)
	}()

	updated, err := svc.WriteTag(participant.ID.UUID())
	require.NoError(t, err)
	require.NotEmpty(t, updated.RFIDTagUID)
	<-done
}

func TestRFIDService_WriteTag_BridgeUnavailable(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	partSvc := NewParticipantService(db)

	participant, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "78", FirstName: "No", LastName: "Bridge",
	})
	require.NoError(t, err)

	svc := NewRFIDService(db, rfid.NewMockReader())
	svc.ConfigureBridge(&config.Config{RFID: config.RFIDConfig{Hardware: false}}, NewBridgeHub())

	_, err = svc.WriteTag(participant.ID.UUID())
	assert.ErrorIs(t, err, ErrBridgeUnavailable)
}
