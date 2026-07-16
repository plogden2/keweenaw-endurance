package services

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/keweenaw-endurance/backend/internal/uuidutil"
	"gorm.io/gorm"
)

var (
	ErrStationNotFound    = errors.New("station not found")
	ErrInvalidStationInput = errors.New("invalid station input")
)

// StationConfigInput configures the current reader station for an event.
type StationConfigInput struct {
	EventID      uuid.UUID
	Mode         string
	CheckpointID *uuid.UUID
	DeviceID     string
	Name         string
}

// CurrentStationResponse is returned by GET /api/stations/current.
type CurrentStationResponse struct {
	Station           *models.ReaderStation `json:"station"`
	Online            bool                  `json:"online"`
	PendingSyncCount  int64                 `json:"pending_sync_count"`
	FailedSyncCount   int64                 `json:"failed_sync_count"`
	LiveCSVLastWrite  *time.Time            `json:"live_csv_last_written_at,omitempty"`
}

// StationService persists ReaderStation config and tracks the "current" station.
type StationService struct {
	db *gorm.DB

	mu        sync.RWMutex
	currentID *uuid.UUID
}

func NewStationService(db *gorm.DB) *StationService {
	return &StationService{db: db}
}

func (s *StationService) PutCurrent(input *StationConfigInput) (*models.ReaderStation, error) {
	if input == nil {
		return nil, fmt.Errorf("%w: input is required", ErrInvalidStationInput)
	}
	if input.EventID == uuid.Nil {
		return nil, fmt.Errorf("%w: event_id is required", ErrInvalidStationInput)
	}

	var event models.Event
	if err := s.db.First(&event, "id = ?", input.EventID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEventNotFound
		}
		return nil, err
	}

	mode := strings.TrimSpace(input.Mode)
	if mode == "" {
		mode = "finish"
	}
	if mode != "finish" && mode != "checkpoint" {
		return nil, fmt.Errorf("%w: mode must be finish or checkpoint", ErrInvalidStationInput)
	}
	if mode == "checkpoint" && (input.CheckpointID == nil || *input.CheckpointID == uuid.Nil) {
		return nil, fmt.Errorf("%w: checkpoint_id is required when mode is checkpoint", ErrInvalidStationInput)
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = "Reader Station"
	}
	deviceID := strings.TrimSpace(input.DeviceID)

	now := time.Now().UTC()
	var station models.ReaderStation

	// Prefer updating an existing station for this device_id on the event.
	found := false
	if deviceID != "" {
		err := s.db.Where("event_id = ? AND device_id = ?", input.EventID, deviceID).First(&station).Error
		if err == nil {
			found = true
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	if !found {
		station = models.ReaderStation{
			EventID: uuidutil.NewPublicUUID(input.EventID),
		}
	}

	station.Mode = mode
	station.Name = name
	station.DeviceID = deviceID
	station.LastSeenAt = &now
	if input.CheckpointID != nil && *input.CheckpointID != uuid.Nil {
		cp := uuidutil.NewPublicUUID(*input.CheckpointID)
		station.CheckpointID = &cp
	} else {
		station.CheckpointID = nil
	}

	if found {
		if err := s.db.Save(&station).Error; err != nil {
			return nil, err
		}
	} else {
		if err := s.db.Create(&station).Error; err != nil {
			return nil, err
		}
	}

	id := station.ID.UUID()
	s.mu.Lock()
	s.currentID = &id
	s.mu.Unlock()

	return &station, nil
}

func (s *StationService) GetCurrent() (*CurrentStationResponse, error) {
	station, err := s.loadCurrentStation()
	if err != nil {
		return nil, err
	}

	var pending, failed int64
	_ = s.db.Model(&models.TimingRecord{}).Where("sync_status = ?", "pending_sync").Count(&pending).Error
	_ = s.db.Model(&models.TimingRecord{}).Where("sync_status = ?", "failed_sync").Count(&failed).Error

	return &CurrentStationResponse{
		Station:          station,
		Online:           s.hostedOnline(),
		PendingSyncCount: pending,
		FailedSyncCount:  failed,
	}, nil
}

func (s *StationService) hostedOnline() bool {
	// Browser/station "online" for local ops is always true at the service
	// layer; hosted reachability is exposed via Sync when wired. Default true
	// so local Postgres authority continues while offline from hosted.
	return true
}

func (s *StationService) CurrentDeviceID() string {
	station, err := s.loadCurrentStation()
	if err != nil || station == nil {
		return ""
	}
	return station.DeviceID
}

// EventIDForDevice returns the event bound to the most recently seen station for deviceID.
func (s *StationService) EventIDForDevice(deviceID string) (uuid.UUID, error) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return uuid.Nil, fmt.Errorf("%w: device_id is required", ErrInvalidStationInput)
	}

	var station models.ReaderStation
	err := s.db.Where("device_id = ?", deviceID).
		Order("last_seen_at DESC").
		Order("created_at DESC").
		First(&station).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return uuid.Nil, ErrStationNotFound
	}
	if err != nil {
		return uuid.Nil, err
	}
	return station.EventID.UUID(), nil
}

func (s *StationService) loadCurrentStation() (*models.ReaderStation, error) {
	s.mu.RLock()
	currentID := s.currentID
	s.mu.RUnlock()

	var station models.ReaderStation
	if currentID != nil {
		if err := s.db.First(&station, "id = ?", *currentID).Error; err == nil {
			return &station, nil
		}
	}

	// Fall back to most recently seen station.
	err := s.db.Order("last_seen_at DESC").Order("created_at DESC").First(&station).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrStationNotFound
	}
	if err != nil {
		return nil, err
	}

	id := station.ID.UUID()
	s.mu.Lock()
	s.currentID = &id
	s.mu.Unlock()
	return &station, nil
}
