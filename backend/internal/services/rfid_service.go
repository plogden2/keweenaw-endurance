package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/config"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/keweenaw-endurance/backend/internal/rfid"
	"github.com/keweenaw-endurance/backend/internal/uuidutil"
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

// TagReadEvent is a fan-out payload for the RFID WebSocket stream.
type TagReadEvent struct {
	Type     string    `json:"type"`
	TagUID   string    `json:"tag_uid"`
	ReadAt   time.Time `json:"read_at"`
	DeviceID string    `json:"device_id"`
	// Scan is set when Type is "scan_result" — reader UI applies it without POST /scans.
	Scan any `json:"scan,omitempty"`
}

type RFIDService struct {
	db     *gorm.DB
	timing *TimingService
	reader rfid.Reader
	cfg    *config.Config
	bridge *BridgeHub

	mu          sync.Mutex
	subscribers []chan TagReadEvent
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

func (s *RFIDService) ConfigureBridge(cfg *config.Config, hub *BridgeHub) {
	s.cfg = cfg
	s.bridge = hub
}

func (s *RFIDService) LookupParticipantByUID(uid string) (*models.Participant, error) {
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return nil, fmt.Errorf("%w: uid is required", ErrInvalidRFIDInput)
	}

	var assoc models.RFIDTagAssociation
	err := s.db.Where("tag_uid = ? AND active = ?", uid, true).First(&assoc).Error
	if err == nil {
		return NewParticipantService(s.db).GetParticipant(assoc.ParticipantID.UUID())
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	var participant models.Participant
	if err := s.db.Where(&models.Participant{RFIDTagUID: uid}).First(&participant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRFIDTagNotFound
		}
		return nil, err
	}
	partSvc := NewParticipantService(s.db)
	return partSvc.GetParticipant(participant.ID.UUID())
}

// ListParticipantTags returns active RFID tag associations for a participant.
func (s *RFIDService) ListParticipantTags(participantID uuid.UUID) ([]models.RFIDTagAssociation, error) {
	if participantID == uuid.Nil {
		return nil, fmt.Errorf("%w: participant_id is required", ErrInvalidRFIDInput)
	}
	if _, err := NewParticipantService(s.db).GetParticipant(participantID); err != nil {
		return nil, err
	}
	var assocs []models.RFIDTagAssociation
	if err := s.db.Where("participant_id = ? AND active = ?", participantID, true).
		Order("created_at ASC").
		Find(&assocs).Error; err != nil {
		return nil, err
	}
	return assocs, nil
}

// AssociateTag links a tag UID to a participant via rfid_tag_associations (no revoke).
// Also mirrors the UID onto participants.rfid_tag_uid for legacy lookup compatibility.
func (s *RFIDService) AssociateTag(participantID uuid.UUID, tagUID string) (*models.RFIDTagAssociation, error) {
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

	var existing models.RFIDTagAssociation
	err = s.db.Where("tag_uid = ?", tagUID).First(&existing).Error
	if err == nil {
		if existing.ParticipantID.UUID() != participantID {
			return nil, fmt.Errorf("%w: tag_uid already associated with another participant", ErrInvalidRFIDInput)
		}
		if !existing.Active {
			existing.Active = true
			if err := s.db.Save(&existing).Error; err != nil {
				return nil, err
			}
		}
		if participant.RFIDTagUID == "" {
			_, _ = partSvc.UpdateParticipant(participantID, &models.Participant{RFIDTagUID: tagUID})
		}
		return &existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	assoc := models.RFIDTagAssociation{
		ParticipantID: participant.ID,
		TagUID:        tagUID,
		Active:        true,
	}
	if err := s.db.Create(&assoc).Error; err != nil {
		return nil, err
	}

	if _, err := partSvc.UpdateParticipant(participantID, &models.Participant{RFIDTagUID: tagUID}); err != nil {
		return nil, err
	}
	return &assoc, nil
}

// WriteTag programs the chip on the antenna with this participant's logical RFID UUID.
// Ensures an association exists (creates one with a new UUID if the racer has none).
func (s *RFIDService) WriteTag(participantID uuid.UUID) (*models.Participant, error) {
	if participantID == uuid.Nil {
		return nil, fmt.Errorf("%w: participant_id is required", ErrInvalidRFIDInput)
	}

	partSvc := NewParticipantService(s.db)
	if _, err := partSvc.GetParticipant(participantID); err != nil {
		return nil, err
	}

	logical, err := s.ensureLogicalTagUUID(participantID)
	if err != nil {
		return nil, err
	}

	if s.cfg != nil && s.cfg.RFID.Hardware {
		device := rfid.NewProxmark3(s.reader)
		if err := device.WriteLogicalUUID(logical); err != nil {
			return nil, err
		}
		return partSvc.GetParticipant(participantID)
	}

	if s.bridge == nil {
		return nil, ErrBridgeUnavailable
	}
	deviceID := s.bridgeDeviceID()
	deviceID = s.bridge.TargetDeviceID(deviceID)
	if err := s.bridge.RequestWrite(deviceID, logical, 0); err != nil {
		return nil, err
	}
	return partSvc.GetParticipant(participantID)
}

func (s *RFIDService) bridgeDeviceID() string {
	if s.cfg != nil && strings.TrimSpace(s.cfg.RFID.BridgeDeviceID) != "" {
		return strings.TrimSpace(s.cfg.RFID.BridgeDeviceID)
	}
	return DefaultBridgeDeviceID
}

func (s *RFIDService) ensureLogicalTagUUID(participantID uuid.UUID) (string, error) {
	var assoc models.RFIDTagAssociation
	err := s.db.Where("participant_id = ? AND active = ?", participantID, true).
		Order("created_at ASC").First(&assoc).Error
	if err == nil {
		return assoc.TagUID, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}

	partSvc := NewParticipantService(s.db)
	participant, err := partSvc.GetParticipant(participantID)
	if err != nil {
		return "", err
	}
	if legacy := strings.TrimSpace(participant.RFIDTagUID); legacy != "" {
		if _, err := s.AssociateTag(participantID, legacy); err != nil {
			return "", err
		}
		return legacy, nil
	}

	logical := uuid.New().String()
	if _, err := s.AssociateTag(participantID, logical); err != nil {
		return "", err
	}
	return logical, nil
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
		found, lookupErr := s.LookupParticipantByUID(uid)
		if lookupErr != nil {
			return nil, lookupErr
		}
		participant = *found
		if participant.RaceID.UUID() != input.RaceID {
			return nil, fmt.Errorf("%w: participant is not registered for this race", ErrInvalidRFIDInput)
		}
	}

	record := &models.TimingRecord{
		ParticipantID:  participant.ID,
		CheckpointID:   uuidutil.NewPublicUUID(input.CheckpointID),
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

// InjectTag pushes a UID into the mock reader queue (when present) and
// broadcasts on the stub tag-stream fan-out channel.
func (s *RFIDService) InjectTag(tagUID string) error {
	tagUID = strings.TrimSpace(tagUID)
	if tagUID == "" {
		return fmt.Errorf("%w: tag_uid is required", ErrInvalidRFIDInput)
	}

	if mock, ok := s.reader.(*rfid.MockReader); ok {
		mock.PushUID(tagUID)
	}

	s.broadcastTag(TagReadEvent{
		Type:   "tag_read",
		TagUID: tagUID,
		ReadAt: time.Now(),
	})
	return nil
}

// PublishScanResult fans out an already-scored scan to reader UIs.
// Use this after bridge ProcessScan so the popup updates without a second POST /scans.
func (s *RFIDService) PublishScanResult(tagUID string, scan any) {
	tagUID = strings.TrimSpace(tagUID)
	if tagUID == "" || scan == nil {
		return
	}
	s.broadcastTag(TagReadEvent{
		Type:   "scan_result",
		TagUID: tagUID,
		ReadAt: time.Now(),
		Scan:   scan,
	})
}

// StartPolling continuously polls the reader and fans out tag_read events.
func (s *RFIDService) StartPolling(ctx context.Context, interval time.Duration, deviceIDFn func() string) {
	if interval <= 0 {
		interval = 200 * time.Millisecond
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				uid, err := s.Poll()
				if err != nil || uid == "" {
					continue
				}
				deviceID := ""
				if deviceIDFn != nil {
					deviceID = deviceIDFn()
				}
				s.broadcastTag(TagReadEvent{
					Type:     "tag_read",
					TagUID:   uid,
					ReadAt:   time.Now(),
					DeviceID: deviceID,
				})
			}
		}
	}()
}

// Poll reads the next tag UID from the underlying reader (empty string if none).
func (s *RFIDService) Poll() (string, error) {
	return s.reader.Poll()
}

// SubscribeTagReads returns a buffered channel of injected/read tag events.
// Callers should not close the channel; it is for stub stream fan-out.
func (s *RFIDService) SubscribeTagReads(buffer int) <-chan TagReadEvent {
	if buffer < 1 {
		buffer = 8
	}
	ch := make(chan TagReadEvent, buffer)
	s.mu.Lock()
	s.subscribers = append(s.subscribers, ch)
	s.mu.Unlock()
	return ch
}

func (s *RFIDService) broadcastTag(ev TagReadEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ch := range s.subscribers {
		select {
		case ch <- ev:
		default:
		}
	}
}
