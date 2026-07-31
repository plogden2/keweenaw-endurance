package scan

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

// seedBluffetFixture mirrors seedActiveLapFixture but pins the event's
// primary key to the well-known Bluffet event UUID, and arms the station in
// (mis-configured) checkpoint mode so ProcessScan's force-finish safety net
// can be exercised.
func seedBluffetFixture(t *testing.T) *scanFixture {
	t.Helper()
	db := setupScanTestDB(t)

	bluffetID, err := uuid.Parse(eventpolicy.BluffetEventIDFull)
	require.NoError(t, err)

	event := &models.Event{
		ID:        uuidutil.NewPublicUUID(bluffetID),
		Name:      "All You Can East Bluffet",
		EventDate: time.Now().AddDate(0, 0, 1),
		Status:    "upcoming",
	}
	require.NoError(t, db.Create(event).Error)

	race := &models.Race{
		EventID:         event.ID,
		Name:            "12 Hour",
		RaceType:        "lap_based",
		DurationMinutes: 720,
		StartTime:       time.Now().Add(-time.Hour),
		Status:          "active",
	}
	require.NoError(t, db.Create(race).Error)

	participant := &models.Participant{
		RaceID:    race.ID,
		BibNumber: "12",
		FirstName: "Alex",
		LastName:  "Rivera",
		Status:    "started",
	}
	require.NoError(t, db.Create(participant).Error)

	finish := &models.TimingCheckpoint{
		RaceID:         race.ID,
		Name:           "Lap Check",
		CheckpointType: "finish",
		IsActive:       true,
	}
	require.NoError(t, db.Create(finish).Error)

	tagUID := "BLUFFET-TAG-0001"
	associateParticipantTag(t, db, event.ID, participant, tagUID)

	// Mis-armed as checkpoint mode — ProcessScan must still force finish for Bluffet.
	station := &models.ReaderStation{
		EventID:      event.ID,
		Mode:         "checkpoint",
		CheckpointID: &finish.ID,
		Name:         "Finish Mat A",
		DeviceID:     "laptop-finish-1",
	}
	require.NoError(t, db.Create(station).Error)

	return &scanFixture{
		db:          db,
		event:       event,
		race:        race,
		participant: participant,
		finish:      finish,
		tagUID:      tagUID,
	}
}

func TestProcessScan_ForcesFinishModeForBluffetEvenWhenStationSaysCheckpoint(t *testing.T) {
	fx := seedBluffetFixture(t)
	svc := NewScanService(fx.db, nil)

	// DB still reports checkpoint mode for this station.
	require.Equal(t, "checkpoint", svc.stationMode(fx.event.ID.UUID(), "laptop-finish-1"))

	now := time.Now().UTC().Truncate(time.Second)
	result, err := svc.ProcessScan(fx.event.ID.UUID(), fx.tagUID, "laptop-finish-1", now)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, ResultLap, result.Result)
	assert.Equal(t, 1, result.LapCount)
	require.NotNil(t, result.TimingRecordID)

	var record models.TimingRecord
	require.NoError(t, fx.db.First(&record, "id = ?", result.TimingRecordID).Error)
	assert.Equal(t, "rfid_lap", record.RecordType)
	assert.Equal(t, fx.finish.ID, record.CheckpointID)
}
