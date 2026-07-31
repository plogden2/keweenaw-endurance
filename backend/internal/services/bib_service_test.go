package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBibService_EnsureBib_Idempotent(t *testing.T) {
	db := setupServiceTestDB(t)
	event := createTestEvent(t, db)
	svc := NewBibService(db)

	first, err := svc.EnsureBib(event.ID.UUID(), "42")
	require.NoError(t, err)
	require.NotNil(t, first)
	assert.Equal(t, "42", first.BibNumber)
	assert.Equal(t, event.ID, first.EventID)
	assert.False(t, first.ID.IsZero())

	second, err := svc.EnsureBib(event.ID.UUID(), "42")
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID)

	var count int64
	require.NoError(t, db.Model(&models.Bib{}).Where("event_id = ? AND bib_number = ?", event.ID, "42").Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestBibService_EnsureBib_RejectsEmpty(t *testing.T) {
	db := setupServiceTestDB(t)
	event := createTestEvent(t, db)
	svc := NewBibService(db)

	_, err := svc.EnsureBib(event.ID.UUID(), "")
	assert.ErrorIs(t, err, ErrInvalidBibInput)

	_, err = svc.EnsureBib(event.ID.UUID(), "   ")
	assert.ErrorIs(t, err, ErrInvalidBibInput)
}

func TestBibService_BulkCreateBibs(t *testing.T) {
	db := setupServiceTestDB(t)
	event := createTestEvent(t, db)
	svc := NewBibService(db)

	created, err := svc.BulkCreateBibs(event.ID.UUID(), 1, 5)
	require.NoError(t, err)
	require.Len(t, created, 5)

	numbers := make([]string, len(created))
	for i, b := range created {
		numbers[i] = b.BibNumber
	}
	assert.Equal(t, []string{"1", "2", "3", "4", "5"}, numbers)

	// Pre-existing 3 should keep same id; full range still returned.
	existingID := created[2].ID
	again, err := svc.BulkCreateBibs(event.ID.UUID(), 1, 5)
	require.NoError(t, err)
	require.Len(t, again, 5)
	assert.Equal(t, existingID, again[2].ID)

	var count int64
	require.NoError(t, db.Model(&models.Bib{}).Where("event_id = ?", event.ID).Count(&count).Error)
	assert.Equal(t, int64(5), count)
}

func TestBibService_BulkCreateBibs_RejectsInvalidRange(t *testing.T) {
	db := setupServiceTestDB(t)
	event := createTestEvent(t, db)
	svc := NewBibService(db)
	eventID := event.ID.UUID()

	_, err := svc.BulkCreateBibs(eventID, 5, 1)
	assert.ErrorIs(t, err, ErrInvalidBibInput)

	_, err = svc.BulkCreateBibs(eventID, -1, 5)
	assert.ErrorIs(t, err, ErrInvalidBibInput)

	_, err = svc.BulkCreateBibs(eventID, 1, -5)
	assert.ErrorIs(t, err, ErrInvalidBibInput)

	_, err = svc.BulkCreateBibs(eventID, 1, 502) // span 502 > 500
	assert.ErrorIs(t, err, ErrInvalidBibInput)

	_, err = svc.BulkCreateBibs(eventID, 1, 500) // exactly 500 OK
	require.NoError(t, err)
}

func TestBibService_ListBibs_WithParticipantAndTags(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	eventID := race.EventID.UUID()
	svc := NewBibService(db)
	partSvc := NewParticipantService(db)

	bib, err := svc.EnsureBib(eventID, "42")
	require.NoError(t, err)

	participant, err := partSvc.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "42", FirstName: "Ada", LastName: "Lovelace",
	})
	require.NoError(t, err)

	require.NoError(t, db.Create(&models.RFIDTagAssociation{
		BibID:  bib.ID,
		TagUID: "TAG-A",
		Active: true,
	}).Error)
	require.NoError(t, db.Create(&models.RFIDTagAssociation{
		BibID:  bib.ID,
		TagUID: "TAG-B",
		Active: true,
	}).Error)

	// Unassigned bib (no participant).
	_, err = svc.EnsureBib(eventID, "99")
	require.NoError(t, err)

	items, err := svc.ListBibs(eventID)
	require.NoError(t, err)
	require.Len(t, items, 2)

	byNumber := map[string]BibListItem{}
	for _, item := range items {
		byNumber[item.BibNumber] = item
	}

	assigned := byNumber["42"]
	assert.Equal(t, bib.ID, assigned.ID)
	assert.Equal(t, 2, assigned.TagCount)
	assert.ElementsMatch(t, []string{"TAG-A", "TAG-B"}, assigned.TagUIDs)
	require.NotNil(t, assigned.ParticipantID)
	assert.Equal(t, participant.ID, *assigned.ParticipantID)
	assert.Equal(t, "Ada Lovelace", assigned.ParticipantName)
	require.NotNil(t, assigned.RaceID)
	assert.Equal(t, race.ID, *assigned.RaceID)

	unassigned := byNumber["99"]
	assert.Equal(t, 0, unassigned.TagCount)
	assert.Empty(t, unassigned.TagUIDs)
	assert.Nil(t, unassigned.ParticipantID)
	assert.Empty(t, unassigned.ParticipantName)
	assert.Nil(t, unassigned.RaceID)
}

func TestBibService_GetBibAndListBibTags(t *testing.T) {
	db := setupServiceTestDB(t)
	event := createTestEvent(t, db)
	svc := NewBibService(db)

	bib, err := svc.EnsureBib(event.ID.UUID(), "7")
	require.NoError(t, err)

	got, err := svc.GetBib(event.ID.UUID(), bib.ID.UUID())
	require.NoError(t, err)
	assert.Equal(t, bib.ID, got.ID)

	_, err = svc.GetBib(event.ID.UUID(), uuid.New())
	assert.ErrorIs(t, err, ErrBibNotFound)

	require.NoError(t, db.Create(&models.RFIDTagAssociation{
		BibID: bib.ID, TagUID: "UID-7", Active: true,
	}).Error)

	tags, err := svc.ListBibTags(bib.ID.UUID())
	require.NoError(t, err)
	require.Len(t, tags, 1)
	assert.Equal(t, "UID-7", tags[0].TagUID)
}
