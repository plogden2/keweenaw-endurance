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

func TestTimingService_ListRecordsByEvent(t *testing.T) {
	db := setupServiceTestDB(t)
	event := createTestEvent(t, db)
	raceSvc := NewRaceService(db)
	raceOne, err := raceSvc.CreateRace(&models.Race{
		EventID: event.ID, Name: "Morning", RaceType: "time_based", DistanceKm: 5,
	})
	require.NoError(t, err)
	raceTwo, err := raceSvc.CreateRace(&models.Race{
		EventID: event.ID, Name: "Afternoon", RaceType: "time_based", DistanceKm: 10,
	})
	require.NoError(t, err)
	finishOne := createCheckpoint(t, db, raceOne.ID, "Finish", "finish")
	finishTwo := createCheckpoint(t, db, raceTwo.ID, "Finish", "finish")

	participants := NewParticipantService(db)
	ada, err := participants.CreateParticipant(&models.Participant{
		RaceID: raceOne.ID, BibNumber: "101", FirstName: "Ada", LastName: "Lovelace",
	})
	require.NoError(t, err)
	grace, err := participants.CreateParticipant(&models.Participant{
		RaceID: raceTwo.ID, BibNumber: "202", FirstName: "Grace", LastName: "Hopper",
	})
	require.NoError(t, err)

	svc := NewTimingService(db)
	now := time.Now().UTC().Truncate(time.Second)
	oldest, err := svc.CreateRecord(&models.TimingRecord{
		ParticipantID: ada.ID, CheckpointID: finishOne.ID, Timestamp: now.Add(-2 * time.Minute), LocalTimestamp: now.Add(-2 * time.Minute),
	})
	require.NoError(t, err)
	_, err = svc.CreateRecord(&models.TimingRecord{
		ParticipantID: grace.ID, CheckpointID: finishTwo.ID, Timestamp: now, LocalTimestamp: now,
	})
	require.NoError(t, err)
	_, _, err = svc.VoidRecord(oldest.ID.UUID())
	require.NoError(t, err)

	pageOne, total, err := svc.ListRecordsByEvent(event.ID.UUID(), 1, 1, nil, "")
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	require.Len(t, pageOne, 1)
	assert.Equal(t, grace.ID, pageOne[0].ParticipantID)
	assert.Equal(t, raceTwo.ID, pageOne[0].Participant.RaceID)
	assert.Equal(t, raceTwo.Name, pageOne[0].Participant.Race.Name)

	pageTwo, total, err := svc.ListRecordsByEvent(event.ID.UUID(), 2, 1, nil, "")
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	require.Len(t, pageTwo, 1)
	assert.Equal(t, oldest.ID, pageTwo[0].ID)
	assert.NotNil(t, pageTwo[0].VoidedAt)

	filtered, total, err := svc.ListRecordsByEvent(event.ID.UUID(), 1, 50, ptrUUID(raceOne.ID.UUID()), "ADA")
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, filtered, 1)
	assert.Equal(t, ada.ID, filtered[0].ParticipantID)

	byBib, total, err := svc.ListRecordsByEvent(event.ID.UUID(), 1, 50, nil, "202")
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, byBib, 1)
	assert.Equal(t, grace.ID, byBib[0].ParticipantID)
}

func TestTimingService_ListRecordsByEvent_UsesIDForEqualTimestamps(t *testing.T) {
	db := setupServiceTestDB(t)
	event := createTestEvent(t, db)
	raceSvc := NewRaceService(db)
	race, err := raceSvc.CreateRace(&models.Race{
		EventID: event.ID, Name: "Stable Order", RaceType: "time_based", DistanceKm: 5,
	})
	require.NoError(t, err)
	finish := createCheckpoint(t, db, race.ID, "Finish", "finish")
	participant, err := NewParticipantService(db).CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "11", FirstName: "Stable", LastName: "Sort",
	})
	require.NoError(t, err)

	svc := NewTimingService(db)
	timestamp := time.Date(2026, time.July, 30, 18, 0, 0, 0, time.UTC)
	firstID := uuidutil.NewPublicUUID(uuid.MustParse("00000000-0000-0000-0000-000000000001"))
	secondID := uuidutil.NewPublicUUID(uuid.MustParse("00000000-0000-0000-0000-000000000002"))
	_, err = svc.CreateRecord(&models.TimingRecord{
		ID: firstID, ParticipantID: participant.ID, CheckpointID: finish.ID, Timestamp: timestamp, LocalTimestamp: timestamp,
	})
	require.NoError(t, err)
	_, err = svc.CreateRecord(&models.TimingRecord{
		ID: secondID, ParticipantID: participant.ID, CheckpointID: finish.ID, Timestamp: timestamp, LocalTimestamp: timestamp,
	})
	require.NoError(t, err)

	pageOne, total, err := svc.ListRecordsByEvent(event.ID.UUID(), 1, 1, nil, "")
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	require.Len(t, pageOne, 1)
	assert.Equal(t, secondID, pageOne[0].ID)

	pageTwo, total, err := svc.ListRecordsByEvent(event.ID.UUID(), 2, 1, nil, "")
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	require.Len(t, pageTwo, 1)
	assert.Equal(t, firstID, pageTwo[0].ID)
}

func TestTimingService_CreateEventTap(t *testing.T) {
	db := setupServiceTestDB(t)
	event := createTestEvent(t, db)
	raceSvc := NewRaceService(db)
	race, err := raceSvc.CreateRace(&models.Race{
		EventID: event.ID, Name: "Tap Race", RaceType: "time_based", DistanceKm: 5,
	})
	require.NoError(t, err)
	finish := createCheckpoint(t, db, race.ID, "Finish", "finish")
	participant, err := NewParticipantService(db).CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "7", FirstName: "Manual", LastName: "Tap",
	})
	require.NoError(t, err)

	svc := NewTimingService(db)
	timestamp := time.Date(2026, time.July, 30, 18, 0, 0, 0, time.UTC)
	lap, err := svc.CreateEventTap(CreateEventTapInput{
		EventID: event.ID.UUID(), ParticipantID: participant.ID.UUID(), Timestamp: &timestamp,
	})
	require.NoError(t, err)
	assert.Equal(t, "rfid_lap", lap.RecordType)
	assert.Equal(t, finish.ID, lap.CheckpointID)
	assert.Nil(t, lap.SourceLapID)
	assert.Equal(t, timestamp, lap.Timestamp)
	assert.Equal(t, "manual-event-taps", lap.DeviceID)
	assert.Equal(t, "synced", lap.SyncStatus)

	bonus, err := svc.CreateEventTap(CreateEventTapInput{
		EventID: event.ID.UUID(), ParticipantID: participant.ID.UUID(), KaraokeBonus: true, DeviceID: "tablet-1",
	})
	require.NoError(t, err)
	assert.Equal(t, "karaoke_bonus", bonus.RecordType)
	assert.Equal(t, finish.ID, bonus.CheckpointID)
	assert.Nil(t, bonus.SourceLapID)
	assert.Equal(t, "tablet-1", bonus.DeviceID)
}

func TestTimingService_CreateEventTap_RejectsInvalidEventParticipantAndMissingFinish(t *testing.T) {
	db := setupServiceTestDB(t)
	firstEvent := createTestEvent(t, db)
	secondEvent := createTestEvent(t, db)
	raceSvc := NewRaceService(db)
	otherRace, err := raceSvc.CreateRace(&models.Race{
		EventID: secondEvent.ID, Name: "Other Race", RaceType: "time_based", DistanceKm: 5,
	})
	require.NoError(t, err)
	otherParticipant, err := NewParticipantService(db).CreateParticipant(&models.Participant{
		RaceID: otherRace.ID, BibNumber: "8", FirstName: "Other", LastName: "Event",
	})
	require.NoError(t, err)

	svc := NewTimingService(db)
	_, err = svc.CreateEventTap(CreateEventTapInput{
		EventID: firstEvent.ID.UUID(), ParticipantID: otherParticipant.ID.UUID(),
	})
	assert.Error(t, err)

	raceWithoutFinish, err := raceSvc.CreateRace(&models.Race{
		EventID: firstEvent.ID, Name: "No Finish", RaceType: "time_based", DistanceKm: 5,
	})
	require.NoError(t, err)
	participant, err := NewParticipantService(db).CreateParticipant(&models.Participant{
		RaceID: raceWithoutFinish.ID, BibNumber: "9", FirstName: "No", LastName: "Finish",
	})
	require.NoError(t, err)
	_, err = svc.CreateEventTap(CreateEventTapInput{
		EventID: firstEvent.ID.UUID(), ParticipantID: participant.ID.UUID(),
	})
	assert.Error(t, err)
}

func ptrUUID(id uuid.UUID) *uuid.UUID {
	return &id
}

func TestTimingService_VoidRecord_CascadesKaraoke(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	finish := createCheckpoint(t, db, race.ID, "Finish", "finish")
	partSvc := NewParticipantService(db)
	participant, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "44", FirstName: "Void", LastName: "Me",
	})
	require.NoError(t, err)

	svc := NewTimingService(db)
	now := time.Now().UTC().Truncate(time.Second)
	lap, err := svc.CreateRecord(&models.TimingRecord{
		ParticipantID: participant.ID, CheckpointID: finish.ID,
		Timestamp: now, LocalTimestamp: now, RecordType: "rfid_lap",
	})
	require.NoError(t, err)

	sourceID := lap.ID
	bonus, err := svc.CreateRecord(&models.TimingRecord{
		ParticipantID: participant.ID, CheckpointID: finish.ID,
		Timestamp: now.Add(time.Second), LocalTimestamp: now.Add(time.Second),
		RecordType: "karaoke_bonus", SourceLapID: &sourceID,
	})
	require.NoError(t, err)

	voided, cascaded, err := svc.VoidRecord(lap.ID.UUID())
	require.NoError(t, err)
	require.NotNil(t, voided.VoidedAt)
	require.Len(t, cascaded, 1)
	assert.Equal(t, bonus.ID.UUID(), cascaded[0])

	refetched, err := svc.GetRecord(bonus.ID.UUID())
	require.NoError(t, err)
	require.NotNil(t, refetched.VoidedAt)

	// Idempotent
	again, cascaded2, err := svc.VoidRecord(lap.ID.UUID())
	require.NoError(t, err)
	require.NotNil(t, again.VoidedAt)
	assert.Empty(t, cascaded2)
}

func TestTimingService_VoidKaraokeAlone(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	finish := createCheckpoint(t, db, race.ID, "Finish", "finish")
	partSvc := NewParticipantService(db)
	participant, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "45", FirstName: "Keep", LastName: "Lap",
	})
	require.NoError(t, err)

	svc := NewTimingService(db)
	now := time.Now().UTC().Truncate(time.Second)
	lap, err := svc.CreateRecord(&models.TimingRecord{
		ParticipantID: participant.ID, CheckpointID: finish.ID,
		Timestamp: now, LocalTimestamp: now, RecordType: "rfid_lap",
	})
	require.NoError(t, err)
	sourceID := lap.ID
	bonus, err := svc.CreateRecord(&models.TimingRecord{
		ParticipantID: participant.ID, CheckpointID: finish.ID,
		Timestamp: now.Add(time.Second), LocalTimestamp: now.Add(time.Second),
		RecordType: "karaoke_bonus", SourceLapID: &sourceID,
	})
	require.NoError(t, err)

	_, cascaded, err := svc.VoidRecord(bonus.ID.UUID())
	require.NoError(t, err)
	assert.Empty(t, cascaded)

	stillActive, err := svc.GetRecord(lap.ID.UUID())
	require.NoError(t, err)
	assert.Nil(t, stillActive.VoidedAt)
}

func TestTimingService_RestoreRules(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	finish := createCheckpoint(t, db, race.ID, "Finish", "finish")
	partSvc := NewParticipantService(db)
	participant, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "46", FirstName: "Restore", LastName: "Me",
	})
	require.NoError(t, err)

	svc := NewTimingService(db)
	now := time.Now().UTC().Truncate(time.Second)
	lap, err := svc.CreateRecord(&models.TimingRecord{
		ParticipantID: participant.ID, CheckpointID: finish.ID,
		Timestamp: now, LocalTimestamp: now, RecordType: "rfid_lap",
	})
	require.NoError(t, err)
	sourceID := lap.ID
	bonus, err := svc.CreateRecord(&models.TimingRecord{
		ParticipantID: participant.ID, CheckpointID: finish.ID,
		Timestamp: now.Add(time.Second), LocalTimestamp: now.Add(time.Second),
		RecordType: "karaoke_bonus", SourceLapID: &sourceID,
	})
	require.NoError(t, err)

	_, _, err = svc.VoidRecord(lap.ID.UUID())
	require.NoError(t, err)

	_, _, err = svc.RestoreRecord(bonus.ID.UUID())
	assert.ErrorIs(t, err, ErrKaraokeSourceStillVoided)

	restored, _, err := svc.RestoreRecord(lap.ID.UUID())
	require.NoError(t, err)
	assert.Nil(t, restored.VoidedAt)

	restoredBonus, _, err := svc.RestoreRecord(bonus.ID.UUID())
	require.NoError(t, err)
	assert.Nil(t, restoredBonus.VoidedAt)

	// Idempotent restore
	again, _, err := svc.RestoreRecord(lap.ID.UUID())
	require.NoError(t, err)
	assert.Nil(t, again.VoidedAt)
}
