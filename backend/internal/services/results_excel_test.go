package services

import (
	"bytes"
	"testing"
	"time"

	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func TestResultsExcelBuildEventResultsWorkbook(t *testing.T) {
	db := setupServiceTestDB(t)
	event := &models.Event{
		Name:      "Bluffet 2026",
		EventDate: time.Date(2026, time.July, 18, 0, 0, 0, 0, time.UTC),
		Status:    "active",
	}
	require.NoError(t, db.Create(event).Error)

	race, err := NewRaceService(db).CreateRace(&models.Race{
		EventID: event.ID, Name: "12 Hour", RaceType: "lap_based", DurationMinutes: 720,
		Status: "active", StartTime: time.Now().UTC().Add(-time.Hour),
	})
	require.NoError(t, err)
	require.NoError(t, db.Create(&models.Race{
		EventID: event.ID, Name: "Cancelled Race", RaceType: "lap_based", DurationMinutes: 60,
		Status: "cancelled",
	}).Error)
	finish := createCheckpoint(t, db, race.ID, "Finish", "finish")

	men, err := NewCategoryService(db).CreateCategory(&models.Category{
		RaceID: race.ID, Name: "Men", CategoryType: "male",
	})
	require.NoError(t, err)
	emptyCategory, err := NewCategoryService(db).CreateCategory(&models.Category{
		RaceID: race.ID, Name: "Empty", CategoryType: "female",
	})
	require.NoError(t, err)
	_ = emptyCategory

	team, err := NewTeamService(db).CreateTeam(&models.Team{RaceID: race.ID, Name: "Bluff Crew"})
	require.NoError(t, err)
	partSvc := NewParticipantService(db)
	leader, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, CategoryID: &men.ID, TeamID: &team.ID, BibNumber: "10",
		FirstName: "Alex", LastName: "Rivera", Age: 32, Gender: "male",
	})
	require.NoError(t, err)
	zero, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, CategoryID: &men.ID, TeamID: &team.ID, BibNumber: "11",
		FirstName: "Zero", LastName: "Laps", Age: 31, Gender: "male",
	})
	require.NoError(t, err)
	_ = zero

	now := time.Now().UTC().Truncate(time.Second)
	voidedAt := now.Add(time.Minute)
	require.NoError(t, db.Create(&models.TimingRecord{
		ParticipantID: leader.ID, CheckpointID: finish.ID, Timestamp: now, LocalTimestamp: now,
		RecordType: "rfid_lap", SyncStatus: "synced",
	}).Error)
	require.NoError(t, db.Create(&models.TimingRecord{
		ParticipantID: leader.ID, CheckpointID: finish.ID, Timestamp: now.Add(time.Minute), LocalTimestamp: now.Add(time.Minute),
		RecordType: "karaoke_bonus", SyncStatus: "synced",
	}).Error)
	require.NoError(t, db.Create(&models.TimingRecord{
		ParticipantID: leader.ID, CheckpointID: finish.ID, Timestamp: now.Add(2 * time.Minute), LocalTimestamp: now.Add(2 * time.Minute),
		RecordType: "rfid_lap", VoidedAt: &voidedAt, SyncStatus: "synced",
	}).Error)

	data, filename, err := NewResultsService(db, nil).BuildEventResultsWorkbook(event.ID.UUID())
	require.NoError(t, err)
	assert.Equal(t, "bluffet-2026-results-20260718.xlsx", filename)

	workbook, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer func() { require.NoError(t, workbook.Close()) }()

	assert.Equal(t, []string{"12 hour individual overall", "12 hour Men", "12 hour team overall"}, workbook.GetSheetList())
	for _, sheet := range workbook.GetSheetList() {
		assert.LessOrEqual(t, len([]rune(sheet)), 31)
		tables, tableErr := workbook.GetTables(sheet)
		require.NoError(t, tableErr)
		assert.Len(t, tables, 1)
	}

	rows, err := workbook.GetRows("12 hour individual overall")
	require.NoError(t, err)
	require.Len(t, rows, 3)
	assert.Equal(t, []string{"Place", "Racer name", "Bib", "Laps", "Age", "Gender", "Team name"}, rows[0])
	assert.Equal(t, []string{"1", "Alex Rivera", "10", "2", "32", "male", "Bluff Crew"}, rows[1])
	assert.Equal(t, []string{"2", "Zero Laps", "11", "0", "31", "male", "Bluff Crew"}, rows[2])

	teamRows, err := workbook.GetRows("12 hour team overall")
	require.NoError(t, err)
	require.Len(t, teamRows, 2)
	assert.Equal(t, []string{"Place", "Team", "Avg laps", "Members"}, teamRows[0])
	assert.Equal(t, []string{"1", "Bluff Crew", "1", "2"}, teamRows[1])
}

func TestResultsExcelOmitsTeamSheetWithoutEligibleTeams(t *testing.T) {
	db := setupServiceTestDB(t)
	event := createTestEvent(t, db)
	race, err := NewRaceService(db).CreateRace(&models.Race{
		EventID: event.ID, Name: "6 Hour", RaceType: "lap_based", DurationMinutes: 360,
		Status: "active",
	})
	require.NoError(t, err)
	participant, err := NewParticipantService(db).CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "20", FirstName: "Solo", LastName: "Rider",
	})
	require.NoError(t, err)
	_ = participant

	data, _, err := NewResultsService(db, nil).BuildEventResultsWorkbook(event.ID.UUID())
	require.NoError(t, err)
	workbook, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer func() { require.NoError(t, workbook.Close()) }()

	assert.Equal(t, []string{"6 hour individual overall"}, workbook.GetSheetList())
}

func TestResultsExcelShortensLongSheetNames(t *testing.T) {
	db := setupServiceTestDB(t)
	event := createTestEvent(t, db)
	race, err := NewRaceService(db).CreateRace(&models.Race{
		EventID: event.ID, Name: "90-Minute Kids Extravaganza With A Very Long Name", RaceType: "lap_based",
		DurationMinutes: 90, Status: "active",
	})
	require.NoError(t, err)
	_, err = NewParticipantService(db).CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "30", FirstName: "Kid", LastName: "Rider",
	})
	require.NoError(t, err)

	data, _, err := NewResultsService(db, nil).BuildEventResultsWorkbook(event.ID.UUID())
	require.NoError(t, err)
	workbook, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)
	defer func() { require.NoError(t, workbook.Close()) }()

	require.Len(t, workbook.GetSheetList(), 1)
	assert.LessOrEqual(t, len([]rune(workbook.GetSheetList()[0])), 31)
}
