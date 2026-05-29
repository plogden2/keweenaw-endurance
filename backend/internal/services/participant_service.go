package services

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/models"
	"gorm.io/gorm"
)

var (
	ErrParticipantNotFound     = errors.New("participant not found")
	ErrInvalidParticipantInput = errors.New("invalid participant input")
)

var validGenders = map[string]bool{
	"male":   true,
	"female": true,
	"other":  true,
}

var validParticipantStatuses = map[string]bool{
	"registered": true,
	"started":    true,
	"finished":   true,
	"dnf":        true,
	"dns":        true,
}

type ParticipantService struct {
	db *gorm.DB
}

func NewParticipantService(db *gorm.DB) *ParticipantService {
	return &ParticipantService{db: db}
}

func (s *ParticipantService) ListParticipants(page, limit int, raceID *uuid.UUID) ([]models.Participant, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	query := s.db.Model(&models.Participant{})
	if raceID != nil {
		query = query.Where("race_id = ?", *raceID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var participants []models.Participant
	offset := (page - 1) * limit
	if err := query.Order("bib_number ASC").Offset(offset).Limit(limit).Find(&participants).Error; err != nil {
		return nil, 0, err
	}

	return participants, total, nil
}

func (s *ParticipantService) GetParticipant(id uuid.UUID) (*models.Participant, error) {
	var participant models.Participant
	if err := s.db.First(&participant, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrParticipantNotFound
		}
		return nil, err
	}
	return &participant, nil
}

func (s *ParticipantService) CreateParticipant(input *models.Participant) (*models.Participant, error) {
	if err := validateParticipantInput(input); err != nil {
		return nil, err
	}

	var race models.Race
	if err := s.db.First(&race, "id = ?", input.RaceID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: race not found", ErrInvalidParticipantInput)
		}
		return nil, err
	}

	var existing int64
	if err := s.db.Model(&models.Participant{}).
		Where("race_id = ? AND bib_number = ?", input.RaceID, input.BibNumber).
		Count(&existing).Error; err != nil {
		return nil, err
	}
	if existing > 0 {
		return nil, fmt.Errorf("%w: bib_number must be unique within race", ErrInvalidParticipantInput)
	}

	if err := s.ensureRFIDAvailable(input.RFIDTagUID, nil); err != nil {
		return nil, err
	}

	participant := *input
	if participant.Status == "" {
		participant.Status = "registered"
	}

	if err := s.db.Create(&participant).Error; err != nil {
		return nil, err
	}

	return &participant, nil
}

func (s *ParticipantService) UpdateParticipant(id uuid.UUID, input *models.Participant) (*models.Participant, error) {
	participant, err := s.GetParticipant(id)
	if err != nil {
		return nil, err
	}

	if input.BibNumber != "" && input.BibNumber != participant.BibNumber {
		var existing int64
		if err := s.db.Model(&models.Participant{}).
			Where("race_id = ? AND bib_number = ? AND id != ?", participant.RaceID, input.BibNumber, id).
			Count(&existing).Error; err != nil {
			return nil, err
		}
		if existing > 0 {
			return nil, fmt.Errorf("%w: bib_number must be unique within race", ErrInvalidParticipantInput)
		}
		participant.BibNumber = input.BibNumber
	}
	if input.FirstName != "" {
		participant.FirstName = input.FirstName
	}
	if input.LastName != "" {
		participant.LastName = input.LastName
	}
	if input.Gender != "" {
		if !validGenders[input.Gender] {
			return nil, fmt.Errorf("%w: invalid gender", ErrInvalidParticipantInput)
		}
		participant.Gender = input.Gender
	}
	if input.Age > 0 {
		participant.Age = input.Age
	}
	if input.RFIDTagUID != "" && input.RFIDTagUID != participant.RFIDTagUID {
		if err := s.ensureRFIDAvailable(input.RFIDTagUID, &id); err != nil {
			return nil, err
		}
		participant.RFIDTagUID = input.RFIDTagUID
	}
	if input.Status != "" {
		if !validParticipantStatuses[input.Status] {
			return nil, fmt.Errorf("%w: invalid status", ErrInvalidParticipantInput)
		}
		participant.Status = input.Status
	}

	if err := validateParticipantInput(participant); err != nil {
		return nil, err
	}

	if err := s.db.Save(participant).Error; err != nil {
		return nil, err
	}

	return participant, nil
}

func (s *ParticipantService) DeleteParticipant(id uuid.UUID) error {
	result := s.db.Delete(&models.Participant{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrParticipantNotFound
	}
	return nil
}

func (s *ParticipantService) ensureRFIDAvailable(rfid string, excludeID *uuid.UUID) error {
	if rfid == "" {
		return nil
	}

	query := s.db.Model(&models.Participant{}).Where(&models.Participant{RFIDTagUID: rfid})
	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("%w: rfid_tag_uid must be unique", ErrInvalidParticipantInput)
	}
	return nil
}

func validateParticipantInput(participant *models.Participant) error {
	if participant == nil {
		return fmt.Errorf("%w: participant is required", ErrInvalidParticipantInput)
	}
	if participant.RaceID == uuid.Nil {
		return fmt.Errorf("%w: race_id is required", ErrInvalidParticipantInput)
	}
	if strings.TrimSpace(participant.BibNumber) == "" {
		return fmt.Errorf("%w: bib_number is required", ErrInvalidParticipantInput)
	}
	if strings.TrimSpace(participant.FirstName) == "" {
		return fmt.Errorf("%w: first_name is required", ErrInvalidParticipantInput)
	}
	if strings.TrimSpace(participant.LastName) == "" {
		return fmt.Errorf("%w: last_name is required", ErrInvalidParticipantInput)
	}
	if participant.Gender != "" && !validGenders[participant.Gender] {
		return fmt.Errorf("%w: invalid gender", ErrInvalidParticipantInput)
	}
	if participant.Status != "" && !validParticipantStatuses[participant.Status] {
		return fmt.Errorf("%w: invalid status", ErrInvalidParticipantInput)
	}
	return nil
}
