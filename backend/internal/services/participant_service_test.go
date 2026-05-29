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

	fetched, err := svc.GetParticipant(participant.ID)
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

	_, err = svc.UpdateParticipant(first.ID, &models.Participant{RFIDTagUID: "RFID-B"})
	assert.ErrorIs(t, err, ErrInvalidParticipantInput)

	updated, err := svc.UpdateParticipant(first.ID, &models.Participant{RFIDTagUID: "RFID-A"})
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

	participants, total, err := svc.ListParticipants(1, 10, &race.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, participants, 3)
}

func TestParticipantService_DeleteNotFound(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewParticipantService(db)
	err := svc.DeleteParticipant(uuid.New())
	assert.ErrorIs(t, err, ErrParticipantNotFound)
}
