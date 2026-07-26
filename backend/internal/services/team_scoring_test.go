package services

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResultsService_BuildTeamLeaderboardAverage(t *testing.T) {
	db := setupServiceTestDB(t)
	event := createTestEvent(t, db)
	race, err := NewRaceService(db).CreateRace(&models.Race{
		EventID:         event.ID,
		Name:            "12 Hour",
		RaceType:        "lap_based",
		DurationMinutes: 30,
		StartTime:       time.Now().UTC().Add(-time.Hour),
		Status:          "active",
	})
	require.NoError(t, err)

	finish := &models.TimingCheckpoint{
		RaceID: race.ID, Name: "Finish", CheckpointType: "finish",
	}
	require.NoError(t, db.Create(finish).Error)

	partSvc := NewParticipantService(db)
	teamSvc := NewTeamService(db)
	teamA, err := teamSvc.CreateTeam(&models.Team{RaceID: race.ID, Name: "East Bluff A"})
	require.NoError(t, err)
	teamB, err := teamSvc.CreateTeam(&models.Team{RaceID: race.ID, Name: "East Bluff B"})
	require.NoError(t, err)

	var membersA, membersB []*models.Participant
	for i := 0; i < 4; i++ {
		p, err := partSvc.CreateParticipant(&models.Participant{
			RaceID: race.ID, BibNumber: fmt.Sprintf("A%d", i+1), FirstName: "A", LastName: "M", Gender: "male",
		})
		require.NoError(t, err)
		membersA = append(membersA, p)
	}
	for i := 0; i < 4; i++ {
		p, err := partSvc.CreateParticipant(&models.Participant{
			RaceID: race.ID, BibNumber: fmt.Sprintf("B%d", i+1), FirstName: "B", LastName: "M", Gender: "female",
		})
		require.NoError(t, err)
		membersB = append(membersB, p)
	}

	idAList := make([]uuid.UUID, 0, 4)
	for _, p := range membersA {
		idAList = append(idAList, p.ID.UUID())
	}
	idBList := make([]uuid.UUID, 0, 4)
	for _, p := range membersB {
		idBList = append(idBList, p.ID.UUID())
	}
	_, err = teamSvc.SetMembers(teamA.ID.UUID(), idAList)
	require.NoError(t, err)
	_, err = teamSvc.SetMembers(teamB.ID.UUID(), idBList)
	require.NoError(t, err)

	base := time.Now().UTC()
	// Team A: 3+3+3+3 = 12 → avg 3.0
	for _, p := range membersA {
		for n := 0; n < 3; n++ {
			require.NoError(t, db.Create(&models.TimingRecord{
				ParticipantID:  p.ID,
				CheckpointID:   finish.ID,
				Timestamp:      base.Add(time.Duration(n) * time.Minute),
				LocalTimestamp: base,
				RecordType:     "rfid_lap",
				SyncStatus:     "synced",
			}).Error)
		}
	}
	// Team B: 4+4+0+0 = 8 → avg 2.0 (zeros stay in denominator)
	for _, p := range membersB[:2] {
		for n := 0; n < 4; n++ {
			require.NoError(t, db.Create(&models.TimingRecord{
				ParticipantID:  p.ID,
				CheckpointID:   finish.ID,
				Timestamp:      base.Add(time.Duration(n) * time.Minute),
				LocalTimestamp: base,
				RecordType:     "rfid_lap",
				SyncStatus:     "synced",
			}).Error)
		}
	}

	board, err := NewResultsService(db, nil).BuildTeamLeaderboard(race.ID.UUID())
	require.NoError(t, err)
	require.Len(t, board, 2)
	assert.Equal(t, "East Bluff A", board[0].Name)
	assert.Equal(t, 1, board[0].Place)
	assert.InDelta(t, 3.0, board[0].AvgLaps, 0.01)
	assert.Equal(t, 4, board[0].MemberCount)
	assert.Equal(t, "East Bluff B", board[1].Name)
	assert.InDelta(t, 2.0, board[1].AvgLaps, 0.01)
}
