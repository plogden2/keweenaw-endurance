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

	fetched, err := svc.GetRace(race.ID.UUID())
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

	eventID := event.ID.UUID()
	races, total, err := svc.ListRaces(1, 10, &eventID)
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

func TestRaceService_DeleteCancelsAndHidesFromList(t *testing.T) {
	db := setupServiceTestDB(t)
	event := createTestEvent(t, db)
	svc := NewRaceService(db)

	race, err := svc.CreateRace(&models.Race{
		EventID:         event.ID,
		Name:            "To Cancel",
		RaceType:        "lap_based",
		DurationMinutes: 60,
	})
	require.NoError(t, err)

	require.NoError(t, svc.DeleteRace(race.ID.UUID()))

	fetched, err := svc.GetRace(race.ID.UUID())
	require.NoError(t, err)
	assert.Equal(t, "cancelled", fetched.Status)

	eventID := event.ID.UUID()
	races, total, err := svc.ListRaces(1, 10, &eventID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, races)

	err = svc.DeleteRace(race.ID.UUID())
	assert.ErrorIs(t, err, ErrRaceNotFound)
}

func TestRaceService_AutoStartDueRaces(t *testing.T) {
	db := setupServiceTestDB(t)
	event := createTestEvent(t, db)
	svc := NewRaceService(db)

	pastStart := time.Now().Add(-2 * time.Minute)
	futureStart := time.Now().Add(2 * time.Hour)

	due, err := svc.CreateRace(&models.Race{
		EventID:         event.ID,
		Name:            "Due Race",
		RaceType:        "lap_based",
		DurationMinutes: 60,
		StartTime:       pastStart,
		Status:          "scheduled",
	})
	require.NoError(t, err)

	notDue, err := svc.CreateRace(&models.Race{
		EventID:         event.ID,
		Name:            "Future Race",
		RaceType:        "lap_based",
		DurationMinutes: 60,
		StartTime:       futureStart,
		Status:          "scheduled",
	})
	require.NoError(t, err)

	n, err := svc.AutoStartDueRaces(time.Now())
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	started, err := svc.GetRace(due.ID.UUID())
	require.NoError(t, err)
	assert.Equal(t, "active", started.Status)

	stillScheduled, err := svc.GetRace(notDue.ID.UUID())
	require.NoError(t, err)
	assert.Equal(t, "scheduled", stillScheduled.Status)

	n, err = svc.AutoStartDueRaces(time.Now())
	require.NoError(t, err)
	assert.Equal(t, 0, n)
}

func TestRaceService_StartAndFinishRace(t *testing.T) {
	db := setupServiceTestDB(t)
	event := createTestEvent(t, db)
	svc := NewRaceService(db)

	race, err := svc.CreateRace(&models.Race{
		EventID:         event.ID,
		Name:            "Manual Start Race",
		RaceType:        "lap_based",
		DurationMinutes: 90,
		StartTime:       time.Now().Add(time.Hour),
		Status:          "scheduled",
	})
	require.NoError(t, err)

	started, err := svc.StartRace(race.ID.UUID())
	require.NoError(t, err)
	assert.Equal(t, "active", started.Status)

	_, err = svc.StartRace(race.ID.UUID())
	assert.ErrorIs(t, err, ErrInvalidRaceTransition)

	finished, err := svc.FinishRace(race.ID.UUID())
	require.NoError(t, err)
	assert.Equal(t, "finished", finished.Status)

	_, err = svc.FinishRace(race.ID.UUID())
	assert.ErrorIs(t, err, ErrInvalidRaceTransition)

	_, err = svc.StartRace(uuid.New())
	assert.ErrorIs(t, err, ErrRaceNotFound)

	_, err = svc.FinishRace(uuid.New())
	assert.ErrorIs(t, err, ErrRaceNotFound)
}
