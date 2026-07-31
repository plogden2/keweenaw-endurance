package services

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/keweenaw-endurance/backend/internal/uuidutil"
	"gorm.io/gorm"
)

// ensureBibForParticipant finds or creates the event-scoped Bib for a participant's bib number.
func ensureBibForParticipant(db *gorm.DB, participant *models.Participant) (*models.Bib, error) {
	if participant == nil {
		return nil, fmt.Errorf("participant is required")
	}
	bibNumber := strings.TrimSpace(participant.BibNumber)
	if bibNumber == "" {
		return nil, fmt.Errorf("bib_number is required")
	}
	var race models.Race
	if err := db.Select("id", "event_id").First(&race, "id = ?", participant.RaceID).Error; err != nil {
		return nil, err
	}
	return ensureBib(db, race.EventID, bibNumber)
}

func ensureBib(db *gorm.DB, eventID uuidutil.PublicUUID, bibNumber string) (*models.Bib, error) {
	return NewBibService(db).EnsureBib(eventID.UUID(), bibNumber)
}

func participantForBib(db *gorm.DB, bib *models.Bib) (*models.Participant, error) {
	var p models.Participant
	err := db.Joins("JOIN races ON races.id = participants.race_id").
		Where("races.event_id = ? AND participants.bib_number = ?", bib.EventID, bib.BibNumber).
		First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func bibIDsForParticipant(db *gorm.DB, participantID uuid.UUID) ([]uuidutil.PublicUUID, error) {
	var participant models.Participant
	if err := db.First(&participant, "id = ?", participantID).Error; err != nil {
		return nil, err
	}
	var race models.Race
	if err := db.Select("id", "event_id").First(&race, "id = ?", participant.RaceID).Error; err != nil {
		return nil, err
	}
	var ids []uuidutil.PublicUUID
	err := db.Model(&models.Bib{}).
		Where("event_id = ? AND bib_number = ?", race.EventID, participant.BibNumber).
		Pluck("id", &ids).Error
	return ids, err
}
