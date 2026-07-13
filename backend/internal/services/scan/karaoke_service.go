package scan

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/keweenaw-endurance/backend/internal/uuidutil"
	"gorm.io/gorm"
)

var (
	// ErrAlreadyExists is returned when a karaoke_bonus already exists for the source lap.
	ErrAlreadyExists = errors.New("karaoke bonus already exists for this lap")
	// ErrInvalidSourceLap is returned when the source timing record is missing or not an RFID lap.
	ErrInvalidSourceLap = errors.New("invalid source lap for karaoke bonus")
	// ErrSourceLapNotFound is returned when the timing record id does not exist.
	ErrSourceLapNotFound = errors.New("timing record not found")
)

// KaraokeBonusResult is the outcome of recording a karaoke bonus lap.
type KaraokeBonusResult struct {
	Record            *models.TimingRecord  `json:"record"`
	LapCount          int                   `json:"lap_count"`
	Placement         int                   `json:"placement"`
	PlacementCategory int                   `json:"placement_category,omitempty"`
	TimingRecordID    *uuidutil.PublicUUID  `json:"timing_record_id,omitempty"`
}

// AddKaraokeBonus creates one karaoke_bonus timing record for the given RFID lap.
// At most one bonus is allowed per source_lap_id; duplicates return ErrAlreadyExists.
func (s *ScanService) AddKaraokeBonus(sourceLapID uuid.UUID) (*KaraokeBonusResult, error) {
	var source models.TimingRecord
	if err := s.db.First(&source, "id = ?", sourceLapID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSourceLapNotFound
		}
		return nil, err
	}
	// Karaoke is only for completed RFID laps — never checkpoint_pass or other types.
	if source.RecordType != "rfid_lap" {
		return nil, ErrInvalidSourceLap
	}

	var existing models.TimingRecord
	err := s.db.Where("record_type = ? AND source_lap_id = ?", "karaoke_bonus", sourceLapID).
		First(&existing).Error
	if err == nil {
		return nil, ErrAlreadyExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	now := time.Now().UTC()
	syncStatus := "synced"
	if s.sync != nil {
		syncStatus = s.sync.ResolveSyncStatus()
	}
	sourceID := source.ID
	bonus := &models.TimingRecord{
		ParticipantID:  source.ParticipantID,
		CheckpointID:   source.CheckpointID,
		Timestamp:      now,
		LocalTimestamp: now,
		DeviceID:       source.DeviceID,
		SyncStatus:     syncStatus,
		RecordType:     "karaoke_bonus",
		SourceLapID:    &sourceID,
		StationID:      source.StationID,
	}
	if err := s.db.Create(bonus).Error; err != nil {
		// Unique index race: treat as already exists.
		if isUniqueViolation(err) {
			return nil, ErrAlreadyExists
		}
		return nil, err
	}

	var participant models.Participant
	if err := s.db.Preload("Category").Preload("Race").First(&participant, "id = ?", source.ParticipantID).Error; err != nil {
		return nil, err
	}

	lapCount, _ := s.scoredLapCount(participant.ID.UUID())
	placement, placementCat, _ := s.placements(participant.RaceID.UUID(), &participant)

	s.notifyChange(participant.Race.EventID.UUID())

	bonusID := bonus.ID
	return &KaraokeBonusResult{
		Record:            bonus,
		LapCount:          lapCount,
		Placement:         placement,
		PlacementCategory: placementCat,
		TimingRecordID:    &bonusID,
	}, nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE") ||
		strings.Contains(msg, "unique") ||
		strings.Contains(msg, "constraint failed")
}
