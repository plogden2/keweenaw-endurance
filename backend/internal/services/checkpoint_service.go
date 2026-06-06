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
	ErrCheckpointNotFound     = errors.New("checkpoint not found")
	ErrInvalidCheckpointInput = errors.New("invalid checkpoint input")
)

var validCheckpointTypes = map[string]bool{
	"start":        true,
	"finish":       true,
	"intermediate": true,
}

type CheckpointService struct {
	db *gorm.DB
}

func NewCheckpointService(db *gorm.DB) *CheckpointService {
	return &CheckpointService{db: db}
}

func (s *CheckpointService) ListCheckpointsByRace(raceID uuid.UUID, page, limit int) ([]models.TimingCheckpoint, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	query := s.db.Model(&models.TimingCheckpoint{}).Where("race_id = ?", raceID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var checkpoints []models.TimingCheckpoint
	offset := (page - 1) * limit
	if err := query.Order("distance_from_start_km ASC").Offset(offset).Limit(limit).Find(&checkpoints).Error; err != nil {
		return nil, 0, err
	}

	return checkpoints, total, nil
}

func (s *CheckpointService) GetCheckpoint(id uuid.UUID) (*models.TimingCheckpoint, error) {
	var checkpoint models.TimingCheckpoint
	if err := s.db.First(&checkpoint, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCheckpointNotFound
		}
		return nil, err
	}
	return &checkpoint, nil
}

func (s *CheckpointService) CreateCheckpoint(input *models.TimingCheckpoint) (*models.TimingCheckpoint, error) {
	if err := validateCheckpointInput(input); err != nil {
		return nil, err
	}

	if err := s.ensureRaceExists(input.RaceID.UUID()); err != nil {
		return nil, err
	}

	checkpoint := *input
	if err := s.db.Create(&checkpoint).Error; err != nil {
		return nil, err
	}

	return &checkpoint, nil
}

func (s *CheckpointService) UpdateCheckpoint(id uuid.UUID, input *models.TimingCheckpoint) (*models.TimingCheckpoint, error) {
	checkpoint, err := s.GetCheckpoint(id)
	if err != nil {
		return nil, err
	}

	if input.Name != "" {
		checkpoint.Name = input.Name
	}
	if input.CheckpointType != "" {
		if !validCheckpointTypes[input.CheckpointType] {
			return nil, fmt.Errorf("%w: invalid checkpoint_type", ErrInvalidCheckpointInput)
		}
		checkpoint.CheckpointType = input.CheckpointType
	}
	if input.DistanceFromStartKm > 0 {
		checkpoint.DistanceFromStartKm = input.DistanceFromStartKm
	}
	if input.LocationDescription != "" {
		checkpoint.LocationDescription = input.LocationDescription
	}

	if err := validateCheckpointInput(checkpoint); err != nil {
		return nil, err
	}

	if err := s.db.Save(checkpoint).Error; err != nil {
		return nil, err
	}

	return checkpoint, nil
}

func (s *CheckpointService) DeleteCheckpoint(id uuid.UUID) error {
	result := s.db.Delete(&models.TimingCheckpoint{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrCheckpointNotFound
	}
	return nil
}

func (s *CheckpointService) ensureRaceExists(raceID uuid.UUID) error {
	var race models.Race
	if err := s.db.First(&race, "id = ?", raceID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("%w: race not found", ErrInvalidCheckpointInput)
		}
		return err
	}
	return nil
}

func validateCheckpointInput(checkpoint *models.TimingCheckpoint) error {
	if checkpoint == nil {
		return fmt.Errorf("%w: checkpoint is required", ErrInvalidCheckpointInput)
	}
	if checkpoint.RaceID.IsZero() {
		return fmt.Errorf("%w: race_id is required", ErrInvalidCheckpointInput)
	}
	if strings.TrimSpace(checkpoint.Name) == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidCheckpointInput)
	}
	if checkpoint.CheckpointType == "" {
		return fmt.Errorf("%w: checkpoint_type is required", ErrInvalidCheckpointInput)
	}
	if !validCheckpointTypes[checkpoint.CheckpointType] {
		return fmt.Errorf("%w: invalid checkpoint_type", ErrInvalidCheckpointInput)
	}
	return nil
}
