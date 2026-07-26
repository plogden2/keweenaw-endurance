package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/keweenaw-endurance/backend/internal/uuidutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTeamService_CreateListSetMembers(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	partSvc := NewParticipantService(db)
	p1, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "1", FirstName: "A", LastName: "One", Gender: "male",
	})
	require.NoError(t, err)
	p2, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "2", FirstName: "B", LastName: "Two", Gender: "female",
	})
	require.NoError(t, err)
	p3, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "3", FirstName: "C", LastName: "Three", Gender: "other",
	})
	require.NoError(t, err)

	svc := NewTeamService(db)
	team, err := svc.CreateTeam(&models.Team{RaceID: race.ID, Name: "East Bluff A", DisplayOrder: 1})
	require.NoError(t, err)
	assert.False(t, team.ID.IsZero())

	_, err = svc.SetMembers(team.ID.UUID(), []uuid.UUID{p1.ID.UUID()})
	assert.ErrorIs(t, err, ErrInvalidTeamInput)

	updated, err := svc.SetMembers(team.ID.UUID(), []uuid.UUID{p1.ID.UUID(), p2.ID.UUID()})
	require.NoError(t, err)
	require.Len(t, updated.Participants, 2)

	teams, err := svc.ListTeamsByRace(race.ID.UUID())
	require.NoError(t, err)
	require.Len(t, teams, 1)
	assert.Equal(t, "East Bluff A", teams[0].Name)

	cleared, err := svc.SetMembers(team.ID.UUID(), nil)
	require.NoError(t, err)
	assert.Len(t, cleared.Participants, 0)

	_, err = svc.SetMembers(team.ID.UUID(), []uuid.UUID{p1.ID.UUID(), p2.ID.UUID(), p3.ID.UUID()})
	require.NoError(t, err)
}

func TestTeamService_RejectsCrossRaceMember(t *testing.T) {
	db := setupServiceTestDB(t)
	raceA := createTestRace(t, db)
	raceB := createTestRace(t, db)
	partSvc := NewParticipantService(db)
	a1, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: raceA.ID, BibNumber: "1", FirstName: "A", LastName: "One", Gender: "male",
	})
	require.NoError(t, err)
	b1, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: raceB.ID, BibNumber: "1", FirstName: "B", LastName: "One", Gender: "male",
	})
	require.NoError(t, err)

	svc := NewTeamService(db)
	team, err := svc.CreateTeam(&models.Team{RaceID: raceA.ID, Name: "Mixed"})
	require.NoError(t, err)

	_, err = svc.SetMembers(team.ID.UUID(), []uuid.UUID{a1.ID.UUID(), b1.ID.UUID()})
	assert.ErrorIs(t, err, ErrInvalidTeamInput)
}

func TestTeamService_DeleteClearsMembership(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	partSvc := NewParticipantService(db)
	p1, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "1", FirstName: "A", LastName: "One", Gender: "male",
	})
	require.NoError(t, err)
	p2, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "2", FirstName: "B", LastName: "Two", Gender: "female",
	})
	require.NoError(t, err)

	svc := NewTeamService(db)
	team, err := svc.CreateTeam(&models.Team{RaceID: race.ID, Name: "Temp"})
	require.NoError(t, err)
	_, err = svc.SetMembers(team.ID.UUID(), []uuid.UUID{p1.ID.UUID(), p2.ID.UUID()})
	require.NoError(t, err)

	require.NoError(t, svc.DeleteTeam(team.ID.UUID()))

	got, err := partSvc.GetParticipant(p1.ID.UUID())
	require.NoError(t, err)
	assert.Nil(t, got.TeamID)

	_, err = svc.GetTeam(team.ID.UUID())
	assert.ErrorIs(t, err, ErrTeamNotFound)
}

func TestTeamService_ValidateTeamForRace(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	other := createTestRace(t, db)
	svc := NewTeamService(db)
	team, err := svc.CreateTeam(&models.Team{RaceID: race.ID, Name: "A"})
	require.NoError(t, err)

	require.NoError(t, svc.ValidateTeamForRace(team.ID.UUID(), race.ID.UUID()))
	assert.ErrorIs(t, svc.ValidateTeamForRace(team.ID.UUID(), other.ID.UUID()), ErrInvalidParticipantInput)
	assert.ErrorIs(t, svc.ValidateTeamForRace(uuid.New(), race.ID.UUID()), ErrInvalidParticipantInput)

	_ = uuidutil.NewPublicUUID(team.ID.UUID())
}
