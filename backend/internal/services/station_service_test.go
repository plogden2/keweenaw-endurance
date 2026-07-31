package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/eventpolicy"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/keweenaw-endurance/backend/internal/uuidutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStationService_PutCurrent_CreatesFinishStation(t *testing.T) {
	db := setupServiceTestDB(t)
	event := createTestEvent(t, db)
	svc := NewStationService(db)

	station, err := svc.PutCurrent(&StationConfigInput{
		EventID:  event.ID.UUID(),
		Mode:     "finish",
		DeviceID: "station-1",
		Name:     "Finish Mat",
	})
	require.NoError(t, err)
	assert.Equal(t, "finish", station.Mode)
	assert.Nil(t, station.CheckpointID)
}

func TestStationService_PutCurrent_RejectsCheckpointModeForBluffet(t *testing.T) {
	db := setupServiceTestDB(t)
	bluffetID, err := uuid.Parse(eventpolicy.BluffetEventIDFull)
	require.NoError(t, err)
	event := &models.Event{
		ID:        uuidutil.NewPublicUUID(bluffetID),
		Name:      "All You Can East Bluffet",
		EventDate: time.Now().AddDate(0, 1, 0),
		Status:    "upcoming",
	}
	require.NoError(t, db.Create(event).Error)

	svc := NewStationService(db)

	cp := uuid.New()
	_, err = svc.PutCurrent(&StationConfigInput{
		EventID:      bluffetID,
		Mode:         "checkpoint",
		CheckpointID: &cp,
		DeviceID:     "station-1",
		Name:         "Mid Checkpoint",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidStationInput)
	assert.Contains(t, err.Error(), "finish station only")

	// Finish mode remains allowed for Bluffet.
	station, err := svc.PutCurrent(&StationConfigInput{
		EventID:  bluffetID,
		Mode:     "finish",
		DeviceID: "station-1",
		Name:     "Finish Mat",
	})
	require.NoError(t, err)
	assert.Equal(t, "finish", station.Mode)
}

func TestStationService_PutCurrent_AllowsCheckpointModeForNonBluffet(t *testing.T) {
	db := setupServiceTestDB(t)
	event := createTestEvent(t, db)
	svc := NewStationService(db)

	cpID := uuid.New()
	station, err := svc.PutCurrent(&StationConfigInput{
		EventID:      event.ID.UUID(),
		Mode:         "checkpoint",
		CheckpointID: &cpID,
		DeviceID:     "station-1",
		Name:         "Mid Checkpoint",
	})
	require.NoError(t, err)
	assert.Equal(t, "checkpoint", station.Mode)
}
