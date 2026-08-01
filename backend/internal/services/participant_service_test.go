package services

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func createTestRace(t *testing.T, db *gorm.DB) *models.Race {
	t.Helper()
	event := createTestEvent(t, db)
	svc := NewRaceService(db)
	race, err := svc.CreateRace(&models.Race{
		EventID:    event.ID,
		Name:       "Test Race",
		RaceType:   "time_based",
		DistanceKm: 42.195,
	})
	require.NoError(t, err)
	return race
}

func TestParticipantService_CreateAndGet(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	svc := NewParticipantService(db)

	participant, err := svc.CreateParticipant(&models.Participant{
		RaceID:    race.ID,
		BibNumber: "101",
		FirstName: "Jane",
		LastName:  "Runner",
		Gender:    "female",
		Age:       32,
	})
	require.NoError(t, err)
	assert.Equal(t, "registered", participant.Status)

	fetched, err := svc.GetParticipant(participant.ID.UUID())
	require.NoError(t, err)
	assert.Equal(t, "Jane", fetched.FirstName)
}

func TestParticipantService_DuplicateBibNumber(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	svc := NewParticipantService(db)

	_, err := svc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "101", FirstName: "A", LastName: "One",
	})
	require.NoError(t, err)

	_, err = svc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "101", FirstName: "B", LastName: "Two",
	})
	assert.ErrorIs(t, err, ErrInvalidParticipantInput)
	assert.ErrorContains(t, err, "bib_number must be unique within event")
}

func TestParticipantService_DuplicateBibAcrossRacesSameEvent(t *testing.T) {
	db := setupServiceTestDB(t)
	event := createTestEvent(t, db)
	raceSvc := NewRaceService(db)
	raceA, err := raceSvc.CreateRace(&models.Race{
		EventID: event.ID, Name: "Race A", RaceType: "time_based", DistanceKm: 5,
	})
	require.NoError(t, err)
	raceB, err := raceSvc.CreateRace(&models.Race{
		EventID: event.ID, Name: "Race B", RaceType: "time_based", DistanceKm: 10,
	})
	require.NoError(t, err)
	svc := NewParticipantService(db)

	_, err = svc.CreateParticipant(&models.Participant{
		RaceID: raceA.ID, BibNumber: "7", FirstName: "A", LastName: "One",
	})
	require.NoError(t, err)

	_, err = svc.CreateParticipant(&models.Participant{
		RaceID: raceB.ID, BibNumber: "7", FirstName: "B", LastName: "Two",
	})
	assert.ErrorIs(t, err, ErrInvalidParticipantInput)
	assert.ErrorContains(t, err, "bib_number must be unique within event")
}

func TestParticipantService_DuplicateBibOnUpdateAcrossEvent(t *testing.T) {
	db := setupServiceTestDB(t)
	event := createTestEvent(t, db)
	raceSvc := NewRaceService(db)
	raceA, err := raceSvc.CreateRace(&models.Race{
		EventID: event.ID, Name: "Race A", RaceType: "time_based", DistanceKm: 5,
	})
	require.NoError(t, err)
	raceB, err := raceSvc.CreateRace(&models.Race{
		EventID: event.ID, Name: "Race B", RaceType: "time_based", DistanceKm: 10,
	})
	require.NoError(t, err)
	svc := NewParticipantService(db)

	_, err = svc.CreateParticipant(&models.Participant{
		RaceID: raceA.ID, BibNumber: "7", FirstName: "A", LastName: "One",
	})
	require.NoError(t, err)

	other, err := svc.CreateParticipant(&models.Participant{
		RaceID: raceB.ID, BibNumber: "8", FirstName: "B", LastName: "Two",
	})
	require.NoError(t, err)

	_, err = svc.UpdateParticipant(other.ID.UUID(), &models.Participant{BibNumber: "7"})
	assert.ErrorIs(t, err, ErrInvalidParticipantInput)
	assert.ErrorContains(t, err, "bib_number must be unique within event")
}

func TestParticipantService_SameBibAllowedAcrossEvents(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewParticipantService(db)
	raceA := createTestRace(t, db)
	raceB := createTestRace(t, db)

	_, err := svc.CreateParticipant(&models.Participant{
		RaceID: raceA.ID, BibNumber: "7", FirstName: "A", LastName: "One",
	})
	require.NoError(t, err)

	_, err = svc.CreateParticipant(&models.Participant{
		RaceID: raceB.ID, BibNumber: "7", FirstName: "B", LastName: "Two",
	})
	require.NoError(t, err)
}

func TestParticipantService_UpdateRaceSameEventClearsStaleCategoryAndTeam(t *testing.T) {
	db := setupServiceTestDB(t)
	event := createTestEvent(t, db)
	raceSvc := NewRaceService(db)
	first, err := raceSvc.CreateRace(&models.Race{
		EventID: event.ID, Name: "12H", RaceType: "time_based", DistanceKm: 42, DurationMinutes: 720,
	})
	require.NoError(t, err)
	second, err := raceSvc.CreateRace(&models.Race{
		EventID: event.ID, Name: "6H", RaceType: "time_based", DistanceKm: 21, DurationMinutes: 360,
	})
	require.NoError(t, err)

	catSvc := NewCategoryService(db)
	cat, err := catSvc.CreateCategory(&models.Category{
		RaceID: first.ID, Name: "Open", CategoryType: "custom",
	})
	require.NoError(t, err)
	team, err := NewTeamService(db).CreateTeam(&models.Team{RaceID: first.ID, Name: "Alpha"})
	require.NoError(t, err)

	svc := NewParticipantService(db)
	created, err := svc.CreateParticipant(&models.Participant{
		RaceID: first.ID, BibNumber: "5", FirstName: "Alex", LastName: "Rivera",
		CategoryID: &cat.ID, TeamID: &team.ID,
	})
	require.NoError(t, err)

	moved, err := svc.UpdateParticipant(created.ID.UUID(), &models.Participant{RaceID: second.ID})
	require.NoError(t, err)
	assert.Equal(t, second.ID, moved.RaceID)
	assert.Nil(t, moved.CategoryID)
	assert.Nil(t, moved.TeamID)
	assert.Equal(t, "5", moved.BibNumber)

	otherEventRace := createTestRace(t, db)
	_, err = svc.UpdateParticipant(created.ID.UUID(), &models.Participant{RaceID: otherEventRace.ID})
	assert.ErrorIs(t, err, ErrInvalidParticipantInput)
	assert.ErrorContains(t, err, "same event")
}

func TestParticipantService_EnsureBibOnCreateAndUpdate(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	svc := NewParticipantService(db)

	created, err := svc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "42", FirstName: "Jane", LastName: "Runner",
	})
	require.NoError(t, err)
	assert.Empty(t, created.TagUIDs)

	var bib42 models.Bib
	err = db.Where("event_id = ? AND bib_number = ?", race.EventID, "42").First(&bib42).Error
	require.NoError(t, err)
	assert.Equal(t, "42", bib42.BibNumber)

	updated, err := svc.UpdateParticipant(created.ID.UUID(), &models.Participant{BibNumber: "99"})
	require.NoError(t, err)
	assert.Empty(t, updated.TagUIDs)

	var bib99 models.Bib
	err = db.Where("event_id = ? AND bib_number = ?", race.EventID, "99").First(&bib99).Error
	require.NoError(t, err)
	assert.Equal(t, "99", bib99.BibNumber)
}

func TestParticipantService_ClearBibKeepsBibAndTags(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	svc := NewParticipantService(db)

	created, err := svc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "12", FirstName: "Tag", LastName: "Owner",
	})
	require.NoError(t, err)

	bib, err := NewBibService(db).EnsureBib(race.EventID.UUID(), "12")
	require.NoError(t, err)
	_, err = NewRFIDService(db, nil).AssociateTagToBib(bib.ID.UUID(), "TAG-STAYS-ON-BIB")
	require.NoError(t, err)

	before, err := svc.GetParticipant(created.ID.UUID())
	require.NoError(t, err)
	assert.Equal(t, []string{"TAG-STAYS-ON-BIB"}, before.TagUIDs)

	cleared, err := svc.UpdateParticipant(created.ID.UUID(), &models.Participant{ClearBibNumber: true})
	require.NoError(t, err)
	assert.Equal(t, "", cleared.BibNumber)
	assert.Empty(t, cleared.TagUIDs)

	// Bib inventory row and tag association remain for the number.
	var still models.Bib
	require.NoError(t, db.Where("event_id = ? AND bib_number = ?", race.EventID, "12").First(&still).Error)
	var assocs []models.RFIDTagAssociation
	require.NoError(t, db.Where("bib_id = ? AND active = ?", still.ID, true).Find(&assocs).Error)
	require.Len(t, assocs, 1)
	assert.Equal(t, "TAG-STAYS-ON-BIB", assocs[0].TagUID)

	// Re-assign then name-only update must keep bib when ClearBibNumber is false.
	_, err = svc.UpdateParticipant(created.ID.UUID(), &models.Participant{BibNumber: "12"})
	require.NoError(t, err)
	renamed, err := svc.UpdateParticipant(created.ID.UUID(), &models.Participant{FirstName: "Pat"})
	require.NoError(t, err)
	assert.Equal(t, "12", renamed.BibNumber)
	assert.Equal(t, "Pat", renamed.FirstName)
}

func TestParticipantService_AttachTagUIDsViaBib(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	svc := NewParticipantService(db)

	participant, err := svc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "12", FirstName: "Tag", LastName: "Owner",
	})
	require.NoError(t, err)

	bib, err := NewBibService(db).EnsureBib(race.EventID.UUID(), "12")
	require.NoError(t, err)

	_, err = NewRFIDService(db, nil).AssociateTagToBib(bib.ID.UUID(), "TAG-VIA-BIB-001")
	require.NoError(t, err)

	fetched, err := svc.GetParticipant(participant.ID.UUID())
	require.NoError(t, err)
	assert.Equal(t, []string{"TAG-VIA-BIB-001"}, fetched.TagUIDs)

	raceID := race.ID.UUID()
	listed, _, err := svc.ListParticipants(1, 10, &raceID, "")
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Equal(t, []string{"TAG-VIA-BIB-001"}, listed[0].TagUIDs)
}

func TestParticipantService_DuplicateRFID(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	svc := NewParticipantService(db)

	_, err := svc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "101", FirstName: "A", LastName: "One",
		RFIDTagUID: "RFID-001",
	})
	require.NoError(t, err)

	_, err = svc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "102", FirstName: "B", LastName: "Two",
		RFIDTagUID: "RFID-001",
	})
	assert.ErrorIs(t, err, ErrInvalidParticipantInput)
}

func TestParticipantService_DuplicateRFIDOnUpdate(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	svc := NewParticipantService(db)

	first, err := svc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "201", FirstName: "A", LastName: "One",
		RFIDTagUID: "RFID-A",
	})
	require.NoError(t, err)

	_, err = svc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "202", FirstName: "B", LastName: "Two",
		RFIDTagUID: "RFID-B",
	})
	require.NoError(t, err)

	_, err = svc.UpdateParticipant(first.ID.UUID(), &models.Participant{RFIDTagUID: "RFID-B"})
	assert.ErrorIs(t, err, ErrInvalidParticipantInput)

	updated, err := svc.UpdateParticipant(first.ID.UUID(), &models.Participant{RFIDTagUID: "RFID-A"})
	require.NoError(t, err)
	assert.Equal(t, "RFID-A", updated.RFIDTagUID)
}

func TestParticipantService_ListByRace(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	svc := NewParticipantService(db)

	for i := 1; i <= 3; i++ {
		_, err := svc.CreateParticipant(&models.Participant{
			RaceID: race.ID, BibNumber: fmt.Sprintf("%03d", i),
			FirstName: "Runner", LastName: fmt.Sprintf("%d", i),
		})
		require.NoError(t, err)
	}

	raceID := race.ID.UUID()
	participants, total, err := svc.ListParticipants(1, 10, &raceID, "")
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, participants, 3)
}

func TestParticipantService_ListParticipantsByEvent(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewParticipantService(db)
	event := createTestEvent(t, db)
	raceSvc := NewRaceService(db)
	firstRace, err := raceSvc.CreateRace(&models.Race{
		EventID: event.ID, Name: "Morning Race", RaceType: "time_based", DistanceKm: 5,
	})
	require.NoError(t, err)
	secondRace, err := raceSvc.CreateRace(&models.Race{
		EventID: event.ID, Name: "Afternoon Race", RaceType: "time_based", DistanceKm: 10,
	})
	require.NoError(t, err)
	otherRace := createTestRace(t, db)

	for _, participant := range []models.Participant{
		{RaceID: firstRace.ID, BibNumber: "101", FirstName: "Alex", LastName: "Rivera"},
		{RaceID: secondRace.ID, BibNumber: "202", FirstName: "Jamie", LastName: "Stone"},
		{RaceID: otherRace.ID, BibNumber: "303", FirstName: "Alex", LastName: "Other"},
	} {
		_, err := svc.CreateParticipant(&participant)
		require.NoError(t, err)
	}

	participants, total, err := svc.ListParticipantsByEvent(event.ID.UUID(), 2, 1, "")
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, participants, 1)
	assert.Equal(t, "202", participants[0].BibNumber)
	assert.Equal(t, secondRace.ID, participants[0].Race.ID)
	assert.Equal(t, "Afternoon Race", participants[0].Race.Name)

	participants, total, err = svc.ListParticipantsByEvent(event.ID.UUID(), 1, 10, "101")
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, participants, 1)
	assert.Equal(t, "Alex", participants[0].FirstName)

	participants, total, err = svc.ListParticipantsByEvent(event.ID.UUID(), 1, 10, "jamie")
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, participants, 1)
	assert.Equal(t, "202", participants[0].BibNumber)

	participants, total, err = svc.ListParticipantsByEvent(event.ID.UUID(), 1, 10, "rivera")
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, participants, 1)
	assert.Equal(t, "101", participants[0].BibNumber)
}

func TestParticipantService_SequentialBibDefault(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	svc := NewParticipantService(db)

	p1, err := svc.CreateParticipant(&models.Participant{
		RaceID: race.ID, FirstName: "Auto", LastName: "One",
	})
	require.NoError(t, err)
	assert.Equal(t, "1", p1.BibNumber)

	_, err = svc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "10", FirstName: "Ten", LastName: "Runner",
	})
	require.NoError(t, err)

	p3, err := svc.CreateParticipant(&models.Participant{
		RaceID: race.ID, FirstName: "Auto", LastName: "Eleven",
	})
	require.NoError(t, err)
	assert.Equal(t, "11", p3.BibNumber)
}

func TestParticipantService_NextSequentialBibEventScoped(t *testing.T) {
	db := setupServiceTestDB(t)
	event := createTestEvent(t, db)
	raceSvc := NewRaceService(db)
	raceA, err := raceSvc.CreateRace(&models.Race{
		EventID: event.ID, Name: "Race A", RaceType: "time_based", DistanceKm: 5,
	})
	require.NoError(t, err)
	raceB, err := raceSvc.CreateRace(&models.Race{
		EventID: event.ID, Name: "Race B", RaceType: "time_based", DistanceKm: 10,
	})
	require.NoError(t, err)
	svc := NewParticipantService(db)

	_, err = svc.CreateParticipant(&models.Participant{
		RaceID: raceA.ID, BibNumber: "40", FirstName: "A", LastName: "One",
	})
	require.NoError(t, err)

	// Empty race B must continue from event max, not restart at 1 (would 400).
	p2, err := svc.CreateParticipant(&models.Participant{
		RaceID: raceB.ID, FirstName: "B", LastName: "Two",
	})
	require.NoError(t, err)
	assert.Equal(t, "41", p2.BibNumber)

	next, err := svc.NextSequentialBib(event.ID.UUID())
	require.NoError(t, err)
	assert.Equal(t, "42", next)
}

func TestParticipantService_SearchByQuery(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	svc := NewParticipantService(db)

	_, err := svc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "12", FirstName: "Alex", LastName: "Rivera",
	})
	require.NoError(t, err)
	_, err = svc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "18", FirstName: "Jordan", LastName: "Lee",
	})
	require.NoError(t, err)

	raceID := race.ID.UUID()
	found, total, err := svc.ListParticipants(1, 50, &raceID, "rivera")
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, "Alex", found[0].FirstName)

	found, total, err = svc.ListParticipants(1, 50, &raceID, "18")
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Equal(t, "Jordan", found[0].FirstName)
}

func TestParticipantService_DeleteNotFound(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewParticipantService(db)
	_, err := svc.DeleteParticipant(uuid.New())
	assert.ErrorIs(t, err, ErrParticipantNotFound)
}

func TestParticipantService_CreateAllowsEmptyBib(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	svc := NewParticipantService(db)

	p1, err := svc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "", FirstName: "No", LastName: "Bib",
	})
	require.NoError(t, err)
	assert.Equal(t, "", p1.BibNumber)

	p2, err := svc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "   ", FirstName: "Also", LastName: "Empty",
	})
	require.NoError(t, err)
	assert.Equal(t, "", p2.BibNumber)

	updated, err := svc.UpdateParticipant(p1.ID.UUID(), &models.Participant{BibNumber: "55"})
	require.NoError(t, err)
	assert.Equal(t, "55", updated.BibNumber)
}

func TestParticipantService_DeleteHardWhenNoTaps(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	svc := NewParticipantService(db)
	p, err := svc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "9", FirstName: "Gone", LastName: "Soon",
	})
	require.NoError(t, err)

	result, err := svc.DeleteParticipant(p.ID.UUID())
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "deleted", result.Action)

	_, err = svc.GetParticipant(p.ID.UUID())
	assert.ErrorIs(t, err, ErrParticipantNotFound)
}

func TestParticipantService_DeleteDNSWhenHasTaps(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	svc := NewParticipantService(db)
	p, err := svc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "8", FirstName: "Has", LastName: "Laps",
	})
	require.NoError(t, err)

	cp := &models.TimingCheckpoint{
		RaceID: race.ID, Name: "Finish", CheckpointType: "finish", IsActive: true,
	}
	require.NoError(t, db.Create(cp).Error)
	now := db.NowFunc()
	require.NoError(t, db.Create(&models.TimingRecord{
		ParticipantID:  p.ID,
		CheckpointID:   cp.ID,
		Timestamp:      now,
		LocalTimestamp: now,
		RecordType:     "rfid_lap",
		SyncStatus:     "synced",
	}).Error)

	result, err := svc.DeleteParticipant(p.ID.UUID())
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "dns", result.Action)
	require.NotNil(t, result.Participant)
	assert.Equal(t, "dns", result.Participant.Status)

	fetched, err := svc.GetParticipant(p.ID.UUID())
	require.NoError(t, err)
	assert.Equal(t, "dns", fetched.Status)
}
