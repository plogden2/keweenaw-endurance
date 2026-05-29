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
	ErrRaceNotFound     = errors.New("race not found")
	ErrInvalidRaceInput = errors.New("invalid race input")
)

var validRaceTypes = map[string]bool{
	"time_based": true,
	"lap_based":  true,
}

var validRaceStatuses = map[string]bool{
	"scheduled": true,
	"active":    true,
	"finished":  true,
	"cancelled": true,
}

type RaceService struct {
	db *gorm.DB
}

func NewRaceService(db *gorm.DB) *RaceService {
	return &RaceService{db: db}
}

func (s *RaceService) ListRaces(page, limit int, eventID *uuid.UUID) ([]models.Race, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	query := s.db.Model(&models.Race{})
	if eventID != nil {
		query = query.Where("event_id = ?", *eventID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var races []models.Race
	offset := (page - 1) * limit
	if err := query.Order("start_time ASC").Offset(offset).Limit(limit).Find(&races).Error; err != nil {
		return nil, 0, err
	}

	return races, total, nil
}

func (s *RaceService) GetRace(id uuid.UUID) (*models.Race, error) {
	var race models.Race
	if err := s.db.Preload("Participants").Preload("Checkpoints").First(&race, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRaceNotFound
		}
		return nil, err
	}
	return &race, nil
}

func (s *RaceService) CreateRace(input *models.Race) (*models.Race, error) {
	if err := validateRaceInput(input); err != nil {
		return nil, err
	}

	var event models.Event
	if err := s.db.First(&event, "id = ?", input.EventID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: event not found", ErrInvalidRaceInput)
		}
		return nil, err
	}

	race := *input
	if race.Status == "" {
		race.Status = "scheduled"
	}

	if err := s.db.Create(&race).Error; err != nil {
		return nil, err
	}

	return &race, nil
}

func (s *RaceService) UpdateRace(id uuid.UUID, input *models.Race) (*models.Race, error) {
	race, err := s.GetRace(id)
	if err != nil {
		return nil, err
	}

	if input.Name != "" {
		race.Name = input.Name
	}
	if input.RaceType != "" {
		if !validRaceTypes[input.RaceType] {
			return nil, fmt.Errorf("%w: invalid race_type", ErrInvalidRaceInput)
		}
		race.RaceType = input.RaceType
	}
	if input.DistanceKm > 0 {
		race.DistanceKm = input.DistanceKm
	}
	if input.DurationMinutes > 0 {
		race.DurationMinutes = input.DurationMinutes
	}
	if !input.StartTime.IsZero() {
		race.StartTime = input.StartTime
	}
	if input.Status != "" {
		if !validRaceStatuses[input.Status] {
			return nil, fmt.Errorf("%w: invalid status", ErrInvalidRaceInput)
		}
		race.Status = input.Status
	}

	if err := validateRaceInput(race); err != nil {
		return nil, err
	}

	if err := s.db.Save(race).Error; err != nil {
		return nil, err
	}

	return race, nil
}

func (s *RaceService) DeleteRace(id uuid.UUID) error {
	result := s.db.Delete(&models.Race{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrRaceNotFound
	}
	return nil
}

func validateRaceInput(race *models.Race) error {
	if race == nil {
		return fmt.Errorf("%w: race is required", ErrInvalidRaceInput)
	}
	if race.EventID.IsZero() {
		return fmt.Errorf("%w: event_id is required", ErrInvalidRaceInput)
	}
	if strings.TrimSpace(race.Name) == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidRaceInput)
	}
	if race.RaceType == "" {
		return fmt.Errorf("%w: race_type is required", ErrInvalidRaceInput)
	}
	if !validRaceTypes[race.RaceType] {
		return fmt.Errorf("%w: invalid race_type", ErrInvalidRaceInput)
	}
	if race.RaceType == "time_based" && race.DistanceKm <= 0 {
		return fmt.Errorf("%w: distance_km is required for time_based races", ErrInvalidRaceInput)
	}
	if race.RaceType == "lap_based" && race.DurationMinutes <= 0 {
		return fmt.Errorf("%w: duration_minutes is required for lap_based races", ErrInvalidRaceInput)
	}
	if race.Status != "" && !validRaceStatuses[race.Status] {
		return fmt.Errorf("%w: invalid status", ErrInvalidRaceInput)
	}
	return nil
}
