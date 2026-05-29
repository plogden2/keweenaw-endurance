package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func createTestEvent(t *testing.T, db *gorm.DB) *models.Event {
	t.Helper()
	svc := NewEventService(db)
	event, err := svc.CreateEvent(&models.Event{
		Name:      "Test Event",
		EventDate: time.Now().AddDate(0, 1, 0),
		Status:    "upcoming",
	})
	require.NoError(t, err)
	return event
}

func TestRaceService_CreateAndGet(t *testing.T) {
	db := setupServiceTestDB(t)
	event := createTestEvent(t, db)
	svc := NewRaceService(db)

	race, err := svc.CreateRace(&models.Race{
		EventID:    event.ID,
		Name:       "50K Ultra",
		RaceType:   "time_based",
		DistanceKm: 50,
	})
	require.NoError(t, err)
	assert.Equal(t, "scheduled", race.Status)

	fetched, err := svc.GetRace(race.ID)
	require.NoError(t, err)
	assert.Equal(t, "50K Ultra", fetched.Name)
}

func TestRaceService_LapBasedValidation(t *testing.T) {
	db := setupServiceTestDB(t)
	event := createTestEvent(t, db)
	svc := NewRaceService(db)

	_, err := svc.CreateRace(&models.Race{
		EventID:  event.ID,
		Name:     "Lap Race",
		RaceType: "lap_based",
	})
	assert.ErrorIs(t, err, ErrInvalidRaceInput)

	race, err := svc.CreateRace(&models.Race{
		EventID:         event.ID,
		Name:            "Lap Race",
		RaceType:        "lap_based",
		DurationMinutes: 60,
	})
	require.NoError(t, err)
	assert.Equal(t, "lap_based", race.RaceType)
}

func TestRaceService_ListByEvent(t *testing.T) {
	db := setupServiceTestDB(t)
	event := createTestEvent(t, db)
	svc := NewRaceService(db)

	_, err := svc.CreateRace(&models.Race{
		EventID: event.ID, Name: "Race A", RaceType: "time_based", DistanceKm: 10,
	})
	require.NoError(t, err)
	_, err = svc.CreateRace(&models.Race{
		EventID: event.ID, Name: "Race B", RaceType: "time_based", DistanceKm: 20,
	})
	require.NoError(t, err)

	races, total, err := svc.ListRaces(1, 10, &event.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, races, 2)
}

func TestRaceService_DeleteNotFound(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewRaceService(db)
	err := svc.DeleteRace(uuid.New())
	assert.ErrorIs(t, err, ErrRaceNotFound)
}
