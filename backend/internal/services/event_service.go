package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/models"
	"gorm.io/gorm"
)

var (
	ErrEventNotFound     = errors.New("event not found")
	ErrInvalidEventInput = errors.New("invalid event input")
)

var validEventStatuses = map[string]bool{
	"upcoming":  true,
	"active":    true,
	"completed": true,
	"cancelled": true,
}

type EventService struct {
	db *gorm.DB
}

func NewEventService(db *gorm.DB) *EventService {
	return &EventService{db: db}
}

func (s *EventService) ListEvents(page, limit int) ([]models.Event, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	var events []models.Event
	var total int64

	if err := s.db.Model(&models.Event{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	if err := s.db.Order("event_date ASC").Offset(offset).Limit(limit).Find(&events).Error; err != nil {
		return nil, 0, err
	}

	return events, total, nil
}

func (s *EventService) GetEvent(id uuid.UUID) (*models.Event, error) {
	var event models.Event
	if err := s.db.Preload("Races").First(&event, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEventNotFound
		}
		return nil, err
	}
	return &event, nil
}

func (s *EventService) CreateEvent(input *models.Event) (*models.Event, error) {
	if err := validateEventInput(input); err != nil {
		return nil, err
	}

	event := *input
	if event.Status == "" {
		event.Status = "upcoming"
	}

	if err := s.db.Create(&event).Error; err != nil {
		return nil, err
	}

	return &event, nil
}

func (s *EventService) UpdateEvent(id uuid.UUID, input *models.Event) (*models.Event, error) {
	event, err := s.GetEvent(id)
	if err != nil {
		return nil, err
	}

	if input.Name != "" {
		event.Name = input.Name
	}
	if input.Description != "" {
		event.Description = input.Description
	}
	if !input.EventDate.IsZero() {
		event.EventDate = input.EventDate
	}
	if input.Location != "" {
		event.Location = input.Location
	}
	if input.WebsiteURL != "" {
		event.WebsiteURL = input.WebsiteURL
	}
	if input.Status != "" {
		if !validEventStatuses[input.Status] {
			return nil, fmt.Errorf("%w: invalid status", ErrInvalidEventInput)
		}
		event.Status = input.Status
	}

	if err := validateEventInput(event); err != nil {
		return nil, err
	}

	if err := s.db.Save(event).Error; err != nil {
		return nil, err
	}

	return event, nil
}

func (s *EventService) DeleteEvent(id uuid.UUID) error {
	result := s.db.Delete(&models.Event{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrEventNotFound
	}
	return nil
}

func validateEventInput(event *models.Event) error {
	if event == nil {
		return fmt.Errorf("%w: event is required", ErrInvalidEventInput)
	}
	if strings.TrimSpace(event.Name) == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidEventInput)
	}
	if event.EventDate.IsZero() {
		return fmt.Errorf("%w: event_date is required", ErrInvalidEventInput)
	}
	if event.Status != "" && !validEventStatuses[event.Status] {
		return fmt.Errorf("%w: invalid status", ErrInvalidEventInput)
	}
	if event.EventDate.Before(time.Now().Truncate(24 * time.Hour).AddDate(-10, 0, 0)) {
		return fmt.Errorf("%w: event_date is too far in the past", ErrInvalidEventInput)
	}
	return nil
}
