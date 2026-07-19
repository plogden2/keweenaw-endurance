package services

import (
	"testing"
	"time"

	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResultsService_TimeBasedRace(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	start := createCheckpoint(t, db, race.ID, "Start", "start")
	finish := createCheckpoint(t, db, race.ID, "Finish", "finish")

	partSvc := NewParticipantService(db)
	fast, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "1", FirstName: "Fast", LastName: "Runner",
	})
	require.NoError(t, err)
	slow, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "2", FirstName: "Slow", LastName: "Runner",
	})
	require.NoError(t, err)

	timingSvc := NewTimingService(db)
	base := time.Now().UTC().Truncate(time.Second)

	_, err = timingSvc.CreateRecord(&models.TimingRecord{
		ParticipantID: fast.ID, CheckpointID: start.ID,
		Timestamp: base, LocalTimestamp: base,
	})
	require.NoError(t, err)
	_, err = timingSvc.CreateRecord(&models.TimingRecord{
		ParticipantID: fast.ID, CheckpointID: finish.ID,
		Timestamp: base.Add(30 * time.Minute), LocalTimestamp: base.Add(30 * time.Minute),
	})
	require.NoError(t, err)

	_, err = timingSvc.CreateRecord(&models.TimingRecord{
		ParticipantID: slow.ID, CheckpointID: start.ID,
		Timestamp: base, LocalTimestamp: base,
	})
	require.NoError(t, err)
	_, err = timingSvc.CreateRecord(&models.TimingRecord{
		ParticipantID: slow.ID, CheckpointID: finish.ID,
		Timestamp: base.Add(45 * time.Minute), LocalTimestamp: base.Add(45 * time.Minute),
	})
	require.NoError(t, err)

	resultsSvc := NewResultsService(db, nil)
	results, err := resultsSvc.GetRaceResults(race.ID.UUID())
	require.NoError(t, err)
	require.Len(t, results, 2)
	assert.Equal(t, 1, results[0].Position)
	assert.Equal(t, "Fast", results[0].FirstName)
	assert.Equal(t, 2, results[1].Position)
	assert.InDelta(t, 1800, results[0].TotalTimeSeconds, 1)
}

func TestResultsService_LapBasedRace(t *testing.T) {
	db := setupServiceTestDB(t)
	event := createTestEvent(t, db)
	raceSvc := NewRaceService(db)
	race, err := raceSvc.CreateRace(&models.Race{
		EventID: event.ID, Name: "Lap Race", RaceType: "lap_based", DurationMinutes: 60,
	})
	require.NoError(t, err)
	finish := createCheckpoint(t, db, race.ID, "Finish", "finish")

	partSvc := NewParticipantService(db)
	p1, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "10", FirstName: "Lap", LastName: "Leader",
	})
	require.NoError(t, err)

	timingSvc := NewTimingService(db)
	base := time.Now().UTC().Truncate(time.Second)
	for i := 0; i < 3; i++ {
		ts := base.Add(time.Duration(i*10) * time.Minute)
		_, err = timingSvc.CreateRecord(&models.TimingRecord{
			ParticipantID: p1.ID, CheckpointID: finish.ID,
			Timestamp: ts, LocalTimestamp: ts,
		})
		require.NoError(t, err)
	}

	resultsSvc := NewResultsService(db, nil)
	results, err := resultsSvc.GetRaceResults(race.ID.UUID())
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, 3, results[0].Laps)
}

func TestResultsService_Leaderboard(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	start := createCheckpoint(t, db, race.ID, "Start", "start")
	finish := createCheckpoint(t, db, race.ID, "Finish", "finish")

	partSvc := NewParticipantService(db)
	male, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "1", FirstName: "John", LastName: "Doe", Gender: "male",
	})
	require.NoError(t, err)
	female, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "2", FirstName: "Jane", LastName: "Doe", Gender: "female",
	})
	require.NoError(t, err)

	catSvc := NewCategoryService(db)
	femaleCat, err := catSvc.CreateCategory(&models.Category{
		RaceID: race.ID, Name: "Female", CategoryType: "female",
	})
	require.NoError(t, err)

	timingSvc := NewTimingService(db)
	base := time.Now().UTC().Truncate(time.Second)
	for _, p := range []*models.Participant{male, female} {
		_, err = timingSvc.CreateRecord(&models.TimingRecord{
			ParticipantID: p.ID, CheckpointID: start.ID,
			Timestamp: base, LocalTimestamp: base,
		})
		require.NoError(t, err)
		_, err = timingSvc.CreateRecord(&models.TimingRecord{
			ParticipantID: p.ID, CheckpointID: finish.ID,
			Timestamp: base.Add(40 * time.Minute), LocalTimestamp: base.Add(40 * time.Minute),
		})
		require.NoError(t, err)
	}
	_ = male

	resultsSvc := NewResultsService(db, nil)
	femaleCategoryID := femaleCat.ID.UUID()
	leaderboard, err := resultsSvc.GetLeaderboard(race.ID.UUID(), &femaleCategoryID)
	require.NoError(t, err)
	require.Len(t, leaderboard, 1)
	assert.Equal(t, "Jane", leaderboard[0].FirstName)
}

func TestResultsService_OverallLeaderboardIncludesZeroLapParticipants(t *testing.T) {
	db := setupServiceTestDB(t)
	event := createTestEvent(t, db)
	raceSvc := NewRaceService(db)
	race, err := raceSvc.CreateRace(&models.Race{
		EventID: event.ID, Name: "12 Hour", RaceType: "lap_based", DurationMinutes: 720,
		Status: "active", StartTime: time.Now().Add(-time.Hour),
	})
	require.NoError(t, err)
	finish := createCheckpoint(t, db, race.ID, "Finish", "finish")

	partSvc := NewParticipantService(db)
	scored, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "10", FirstName: "Alex", LastName: "Rivera", Status: "started",
	})
	require.NoError(t, err)
	_, err = partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "11", FirstName: "Zero", LastName: "Laps", Status: "registered",
	})
	require.NoError(t, err)

	base := time.Now().UTC().Truncate(time.Second)
	require.NoError(t, db.Create(&models.TimingRecord{
		ParticipantID: scored.ID, CheckpointID: finish.ID,
		Timestamp: base, LocalTimestamp: base,
		RecordType: "rfid_lap", SyncStatus: "synced",
	}).Error)

	resultsSvc := NewResultsService(db, nil)
	live, err := resultsSvc.GetEventLive(event.ID.UUID(), nil)
	require.NoError(t, err)
	require.Len(t, live.Races, 1)
	require.Len(t, live.Races[0].LeaderboardOverall, 2)
	assert.Equal(t, 1, live.Races[0].LeaderboardOverall[0].Laps)
	assert.Equal(t, "10", live.Races[0].LeaderboardOverall[0].BibNumber)
	assert.Equal(t, 0, live.Races[0].LeaderboardOverall[1].Laps)
	assert.Equal(t, "11", live.Races[0].LeaderboardOverall[1].BibNumber)
	assert.Equal(t, 2, live.Races[0].LeaderboardOverall[1].Place)
}

func TestResultsService_LapLeaderboardIncludesZeroLapParticipants(t *testing.T) {
	db := setupServiceTestDB(t)
	event := createTestEvent(t, db)
	raceSvc := NewRaceService(db)
	race, err := raceSvc.CreateRace(&models.Race{
		EventID: event.ID, Name: "6 Hour", RaceType: "lap_based", DurationMinutes: 360,
		Status: "active", StartTime: time.Now().Add(-time.Hour),
	})
	require.NoError(t, err)
	finish := createCheckpoint(t, db, race.ID, "Finish", "finish")

	partSvc := NewParticipantService(db)
	scored, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "20", FirstName: "Has", LastName: "Lap", Status: "started",
	})
	require.NoError(t, err)
	_, err = partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "21", FirstName: "No", LastName: "Lap", Status: "registered",
	})
	require.NoError(t, err)

	base := time.Now().UTC().Truncate(time.Second)
	require.NoError(t, db.Create(&models.TimingRecord{
		ParticipantID: scored.ID, CheckpointID: finish.ID,
		Timestamp: base, LocalTimestamp: base,
		RecordType: "rfid_lap", SyncStatus: "synced",
	}).Error)

	resultsSvc := NewResultsService(db, nil)
	board, err := resultsSvc.GetLeaderboard(race.ID.UUID(), nil)
	require.NoError(t, err)
	require.Len(t, board, 2)
	assert.Equal(t, 1, board[0].Laps)
	assert.Equal(t, "20", board[0].BibNumber)
	assert.Equal(t, 0, board[1].Laps)
	assert.Equal(t, "21", board[1].BibNumber)
}

func TestResultsService_OverallLeaderboardIncludesKaraokeBonus(t *testing.T) {
	db := setupServiceTestDB(t)
	event := createTestEvent(t, db)
	raceSvc := NewRaceService(db)
	race, err := raceSvc.CreateRace(&models.Race{
		EventID: event.ID, Name: "12 Hour", RaceType: "lap_based", DurationMinutes: 720,
		Status: "active", StartTime: time.Now().Add(-time.Hour),
	})
	require.NoError(t, err)
	finish := createCheckpoint(t, db, race.ID, "Finish", "finish")

	partSvc := NewParticipantService(db)
	p1, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "10", FirstName: "Alex", LastName: "Rivera",
	})
	require.NoError(t, err)

	base := time.Now().UTC().Truncate(time.Second)
	source := &models.TimingRecord{
		ParticipantID: p1.ID, CheckpointID: finish.ID,
		Timestamp: base, LocalTimestamp: base,
		RecordType: "rfid_lap", SyncStatus: "synced",
	}
	require.NoError(t, db.Create(source).Error)
	sourceID := source.ID
	require.NoError(t, db.Create(&models.TimingRecord{
		ParticipantID: p1.ID, CheckpointID: finish.ID,
		Timestamp: base.Add(time.Minute), LocalTimestamp: base.Add(time.Minute),
		RecordType: "karaoke_bonus", SourceLapID: &sourceID, SyncStatus: "synced",
	}).Error)

	resultsSvc := NewResultsService(db, nil)
	live, err := resultsSvc.GetEventLive(event.ID.UUID(), nil)
	require.NoError(t, err)
	require.Len(t, live.Races, 1)
	require.Equal(t, 720, live.Races[0].DurationMinutes)
	require.Len(t, live.Races[0].LeaderboardOverall, 1)
	assert.Equal(t, 2, live.Races[0].LeaderboardOverall[0].Laps)
	assert.Equal(t, 1, live.Races[0].LeaderboardOverall[0].Place)
}

func TestResultsService_GetLiveTiming(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	start := createCheckpoint(t, db, race.ID, "Start", "start")
	partSvc := NewParticipantService(db)
	participant, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "99", FirstName: "Live", LastName: "Runner",
	})
	require.NoError(t, err)

	timingSvc := NewTimingService(db)
	now := time.Now().UTC().Truncate(time.Second)
	_, err = timingSvc.CreateRecord(&models.TimingRecord{
		ParticipantID: participant.ID, CheckpointID: start.ID,
		Timestamp: now, LocalTimestamp: now,
	})
	require.NoError(t, err)

	resultsSvc := NewResultsService(db, nil)
	live, err := resultsSvc.GetLiveTiming(race.ID.UUID())
	require.NoError(t, err)
	assert.Len(t, live.Records, 1)
	assert.Equal(t, race.ID, live.RaceID)
}
