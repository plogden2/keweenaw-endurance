package services

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupServiceTestDB(t *testing.T) *gorm.DB {
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
	))
	return db
}

func TestEventService_CreateAndGet(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewEventService(db)

	event, err := svc.CreateEvent(&models.Event{
		Name:      "Copper Harbor Classic",
		EventDate: time.Now().AddDate(0, 1, 0),
		Location:  "Copper Harbor, MI",
		Status:    "upcoming",
	})
	require.NoError(t, err)
	assert.False(t, event.ID.IsZero())

	fetched, err := svc.GetEvent(event.ID)
	require.NoError(t, err)
	assert.Equal(t, "Copper Harbor Classic", fetched.Name)
}

func TestEventService_CreateValidation(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewEventService(db)

	_, err := svc.CreateEvent(&models.Event{
		EventDate: time.Now().AddDate(0, 1, 0),
	})
	assert.ErrorIs(t, err, ErrInvalidEventInput)

	_, err = svc.CreateEvent(&models.Event{
		Name:   "No Date Event",
		Status: "invalid",
	})
	assert.ErrorIs(t, err, ErrInvalidEventInput)
}

func TestEventService_ListEvents(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewEventService(db)

	for i := 0; i < 3; i++ {
		_, err := svc.CreateEvent(&models.Event{
			Name:      fmt.Sprintf("Event %d", i),
			EventDate: time.Now().AddDate(0, i+1, 0),
		})
		require.NoError(t, err)
	}

	events, total, err := svc.ListEvents(1, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, events, 2)
}

func TestEventService_UpdateAndDelete(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewEventService(db)

	event, err := svc.CreateEvent(&models.Event{
		Name:      "Original Name",
		EventDate: time.Now().AddDate(0, 1, 0),
	})
	require.NoError(t, err)

	updated, err := svc.UpdateEvent(event.ID, &models.Event{Name: "Updated Name"})
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", updated.Name)

	require.NoError(t, svc.DeleteEvent(event.ID))

	_, err = svc.GetEvent(event.ID)
	assert.ErrorIs(t, err, ErrEventNotFound)
}

func TestEventService_GetNotFound(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewEventService(db)

	_, err := svc.GetEvent(uuid.New())
	assert.ErrorIs(t, err, ErrEventNotFound)
}
