package services

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/models"
	"gorm.io/gorm"
)

var (
	ErrTimingRecordNotFound = errors.New("timing record not found")
	ErrInvalidTimingInput   = errors.New("invalid timing input")
)

var validSyncStatuses = map[string]bool{
	"synced":        true,
	"pending_sync":  true,
	"failed_sync":   true,
}

type TimingService struct {
	db *gorm.DB
}

func NewTimingService(db *gorm.DB) *TimingService {
	return &TimingService{db: db}
}

func (s *TimingService) GetRecord(id uuid.UUID) (*models.TimingRecord, error) {
	var record models.TimingRecord
	if err := s.db.Preload("Participant").Preload("Checkpoint").First(&record, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTimingRecordNotFound
		}
		return nil, err
	}
	return &record, nil
}

func (s *TimingService) CreateRecord(input *models.TimingRecord) (*models.TimingRecord, error) {
	if err := validateTimingInput(input); err != nil {
		return nil, err
	}

	var participant models.Participant
	if err := s.db.First(&participant, "id = ?", input.ParticipantID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: participant not found", ErrInvalidTimingInput)
		}
		return nil, err
	}

	var checkpoint models.TimingCheckpoint
	if err := s.db.First(&checkpoint, "id = ?", input.CheckpointID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: checkpoint not found", ErrInvalidTimingInput)
		}
		return nil, err
	}

	if participant.RaceID != checkpoint.RaceID {
		return nil, fmt.Errorf("%w: participant and checkpoint must belong to the same race", ErrInvalidTimingInput)
	}

	record := *input
	if record.LocalTimestamp.IsZero() {
		record.LocalTimestamp = record.Timestamp
	}
	if record.SyncStatus == "" {
		record.SyncStatus = "synced"
	}

	if err := s.db.Create(&record).Error; err != nil {
		return nil, err
	}

	return &record, nil
}

func (s *TimingService) UpdateRecord(id uuid.UUID, input *models.TimingRecord) (*models.TimingRecord, error) {
	record, err := s.GetRecord(id)
	if err != nil {
		return nil, err
	}

	if !input.Timestamp.IsZero() {
		record.Timestamp = input.Timestamp
	}
	if !input.LocalTimestamp.IsZero() {
		record.LocalTimestamp = input.LocalTimestamp
	}
	if input.SyncStatus != "" {
		if !validSyncStatuses[input.SyncStatus] {
			return nil, fmt.Errorf("%w: invalid sync_status", ErrInvalidTimingInput)
		}
		record.SyncStatus = input.SyncStatus
	}
	if input.DeviceID != "" {
		record.DeviceID = input.DeviceID
	}

	if err := s.db.Save(record).Error; err != nil {
		return nil, err
	}

	return record, nil
}

func (s *TimingService) ListRecordsByRace(raceID uuid.UUID) ([]models.TimingRecord, error) {
	var records []models.TimingRecord
	err := s.db.
		Joins("JOIN participants ON participants.id = timing_records.participant_id").
		Where("participants.race_id = ?", raceID).
		Preload("Participant").
		Preload("Checkpoint").
		Order("timing_records.timestamp ASC").
		Find(&records).Error
	if err != nil {
		return nil, err
	}
	return records, nil
}

func (s *TimingService) CountPendingSync() (int64, error) {
	var count int64
	err := s.db.Model(&models.TimingRecord{}).Where("sync_status = ?", "pending_sync").Count(&count).Error
	return count, err
}

func validateTimingInput(record *models.TimingRecord) error {
	if record == nil {
		return fmt.Errorf("%w: timing record is required", ErrInvalidTimingInput)
	}
	if record.ParticipantID == uuid.Nil {
		return fmt.Errorf("%w: participant_id is required", ErrInvalidTimingInput)
	}
	if record.CheckpointID == uuid.Nil {
		return fmt.Errorf("%w: checkpoint_id is required", ErrInvalidTimingInput)
	}
	if record.Timestamp.IsZero() {
		return fmt.Errorf("%w: timestamp is required", ErrInvalidTimingInput)
	}
	if record.SyncStatus != "" && !validSyncStatuses[record.SyncStatus] {
		return fmt.Errorf("%w: invalid sync_status", ErrInvalidTimingInput)
	}
	return nil
}
