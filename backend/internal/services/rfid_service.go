package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/keweenaw-endurance/backend/internal/rfid"
	"gorm.io/gorm"
)

var (
	ErrRFIDTagNotFound    = errors.New("rfid tag not found")
	ErrInvalidRFIDInput   = errors.New("invalid rfid input")
	ErrHardwareUnavailable = rfid.ErrHardwareUnavailable
)

type ManualEntryInput struct {
	RaceID       uuid.UUID
	CheckpointID uuid.UUID
	BibNumber    string
	RFIDTagUID   string
	Timestamp    time.Time
	DeviceID     string
	SyncStatus   string
}

type SyncStatusResponse struct {
	PendingCount int64 `json:"pending_count"`
	FailedCount  int64 `json:"failed_count"`
	SyncedCount  int64 `json:"synced_count"`
}

type RFIDService struct {
	db     *gorm.DB
	timing *TimingService
	reader rfid.Reader
}

func NewRFIDService(db *gorm.DB, reader rfid.Reader) *RFIDService {
	if reader == nil {
		reader = rfid.DefaultReader()
	}
	return &RFIDService{
		db:     db,
		timing: NewTimingService(db),
		reader: reader,
	}
}

func (s *RFIDService) LookupParticipantByUID(uid string) (*models.Participant, error) {
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return nil, fmt.Errorf("%w: uid is required", ErrInvalidRFIDInput)
	}

	var participant models.Participant
	if err := s.db.Where(&models.Participant{RFIDTagUID: uid}).First(&participant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRFIDTagNotFound
		}
		return nil, err
	}
	return &participant, nil
}

func (s *RFIDService) WriteTag(participantID uuid.UUID, tagUID string) (*models.Participant, error) {
	tagUID = strings.TrimSpace(tagUID)
	if tagUID == "" {
		return nil, fmt.Errorf("%w: tag_uid is required", ErrInvalidRFIDInput)
	}
	if participantID == uuid.Nil {
		return nil, fmt.Errorf("%w: participant_id is required", ErrInvalidRFIDInput)
	}

	partSvc := NewParticipantService(s.db)
	participant, err := partSvc.GetParticipant(participantID)
	if err != nil {
		return nil, err
	}

	device := rfid.NewProxmark3(s.reader)
	if err := device.WriteTag(tagUID, participant.ID.String()); err != nil {
		return nil, err
	}

	updated, err := partSvc.UpdateParticipant(participantID, &models.Participant{RFIDTagUID: tagUID})
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *RFIDService) ManualEntry(input *ManualEntryInput) (*models.TimingRecord, error) {
	if input == nil {
		return nil, fmt.Errorf("%w: manual entry input is required", ErrInvalidRFIDInput)
	}
	if input.RaceID == uuid.Nil {
		return nil, fmt.Errorf("%w: race_id is required", ErrInvalidRFIDInput)
	}
	if input.CheckpointID == uuid.Nil {
		return nil, fmt.Errorf("%w: checkpoint_id is required", ErrInvalidRFIDInput)
	}
	if input.Timestamp.IsZero() {
		return nil, fmt.Errorf("%w: timestamp is required", ErrInvalidRFIDInput)
	}

	bib := strings.TrimSpace(input.BibNumber)
	uid := strings.TrimSpace(input.RFIDTagUID)
	if bib == "" && uid == "" {
		return nil, fmt.Errorf("%w: bib_number or rfid_tag_uid is required", ErrInvalidRFIDInput)
	}

	var participant models.Participant
	switch {
	case bib != "":
		if err := s.db.First(&participant, "race_id = ? AND bib_number = ?", input.RaceID, bib).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrParticipantNotFound
			}
			return nil, err
		}
	default:
		if err := s.db.Where(&models.Participant{RFIDTagUID: uid}).First(&participant).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrRFIDTagNotFound
			}
			return nil, err
		}
		if participant.RaceID != input.RaceID {
			return nil, fmt.Errorf("%w: participant is not registered for this race", ErrInvalidRFIDInput)
		}
	}

	record := &models.TimingRecord{
		ParticipantID:  participant.ID,
		CheckpointID:   input.CheckpointID,
		Timestamp:      input.Timestamp,
		LocalTimestamp: input.Timestamp,
		DeviceID:       input.DeviceID,
		SyncStatus:     input.SyncStatus,
	}
	return s.timing.CreateRecord(record)
}

func (s *RFIDService) GetSyncStatus() (*SyncStatusResponse, error) {
	var pending, failed, synced int64

	if err := s.db.Model(&models.TimingRecord{}).Where("sync_status = ?", "pending_sync").Count(&pending).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&models.TimingRecord{}).Where("sync_status = ?", "failed_sync").Count(&failed).Error; err != nil {
		return nil, err
	}
	if err := s.db.Model(&models.TimingRecord{}).Where("sync_status = ?", "synced").Count(&synced).Error; err != nil {
		return nil, err
	}

	return &SyncStatusResponse{
		PendingCount: pending,
		FailedCount:  failed,
		SyncedCount:  synced,
	}, nil
}

func (s *RFIDService) SyncPending() (int64, error) {
	result := s.db.Model(&models.TimingRecord{}).
		Where("sync_status = ?", "pending_sync").
		Update("sync_status", "synced")
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}
