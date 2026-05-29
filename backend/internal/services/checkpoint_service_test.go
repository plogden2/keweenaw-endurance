package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckpointService_CreateAndGet(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	svc := NewCheckpointService(db)

	checkpoint, err := svc.CreateCheckpoint(&models.TimingCheckpoint{
		RaceID:                race.ID,
		Name:                  "Start Line",
		CheckpointType:        "start",
		DistanceFromStartKm:   0,
		LocationDescription:   "Main staging area",
	})
	require.NoError(t, err)
	assert.False(t, checkpoint.ID.IsZero())
	assert.True(t, checkpoint.IsActive)

	fetched, err := svc.GetCheckpoint(checkpoint.ID)
	require.NoError(t, err)
	assert.Equal(t, "Start Line", fetched.Name)
	assert.Equal(t, race.ID, fetched.RaceID)
}

func TestCheckpointService_CreateValidation(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	svc := NewCheckpointService(db)

	_, err := svc.CreateCheckpoint(&models.TimingCheckpoint{
		RaceID: race.ID,
	})
	assert.ErrorIs(t, err, ErrInvalidCheckpointInput)

	_, err = svc.CreateCheckpoint(&models.TimingCheckpoint{
		RaceID:         race.ID,
		Name:           "Bad Type",
		CheckpointType: "invalid",
	})
	assert.ErrorIs(t, err, ErrInvalidCheckpointInput)

	_, err = svc.CreateCheckpoint(&models.TimingCheckpoint{
		RaceID:         uuid.New(),
		Name:           "Orphan",
		CheckpointType: "finish",
	})
	assert.ErrorIs(t, err, ErrInvalidCheckpointInput)
}

func TestCheckpointService_ListByRace(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	otherRace := createTestRace(t, db)
	svc := NewCheckpointService(db)

	for _, name := range []string{"Start", "Finish"} {
		_, err := svc.CreateCheckpoint(&models.TimingCheckpoint{
			RaceID:         race.ID,
			Name:           name,
			CheckpointType: "intermediate",
		})
		require.NoError(t, err)
	}
	_, err := svc.CreateCheckpoint(&models.TimingCheckpoint{
		RaceID:         otherRace.ID,
		Name:           "Other",
		CheckpointType: "start",
	})
	require.NoError(t, err)

	checkpoints, total, err := svc.ListCheckpointsByRace(race.ID, 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	assert.Len(t, checkpoints, 2)
}

func TestCheckpointService_UpdateAndDelete(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	svc := NewCheckpointService(db)

	checkpoint, err := svc.CreateCheckpoint(&models.TimingCheckpoint{
		RaceID:         race.ID,
		Name:           "CP1",
		CheckpointType: "intermediate",
	})
	require.NoError(t, err)

	updated, err := svc.UpdateCheckpoint(checkpoint.ID, &models.TimingCheckpoint{
		Name:              "Updated CP",
		DistanceFromStartKm: 5.5,
	})
	require.NoError(t, err)
	assert.Equal(t, "Updated CP", updated.Name)
	assert.Equal(t, 5.5, updated.DistanceFromStartKm)

	require.NoError(t, svc.DeleteCheckpoint(checkpoint.ID))

	_, err = svc.GetCheckpoint(checkpoint.ID)
	assert.ErrorIs(t, err, ErrCheckpointNotFound)
}

func TestCheckpointService_GetNotFound(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewCheckpointService(db)

	_, err := svc.GetCheckpoint(uuid.New())
	assert.ErrorIs(t, err, ErrCheckpointNotFound)
}
