package scan

import (
	"errors"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/keweenaw-endurance/backend/internal/uuidutil"
	"gorm.io/gorm"
)

const (
	ResultLap        = "lap"
	ResultTestRead   = "test_read"
	ResultCooldown   = "cooldown"
	ResultUnknownTag = "unknown_tag"

	CooldownDuration = 60 * time.Second
)

// ScanResult is the outcome of processing an RFID tap for an event.
type ScanResult struct {
	Result            string               `json:"result"`
	Participant       *models.Participant  `json:"participant,omitempty"`
	ParticipantName   string               `json:"participant_name,omitempty"`
	RaceName          string               `json:"race_name,omitempty"`
	BibNumber         string               `json:"bib_number,omitempty"`
	CategoryLabel     string               `json:"category_label,omitempty"`
	RaceID            *uuidutil.PublicUUID `json:"race_id,omitempty"`
	RaceStatus        string               `json:"race_status,omitempty"`
	LapCount          int                  `json:"lap_count,omitempty"`
	Placement         int                  `json:"placement,omitempty"`
	PlacementCategory int                  `json:"placement_category,omitempty"`
	TimingRecordID    *uuidutil.PublicUUID `json:"timing_record_id,omitempty"`
	KaraokeAvailable  bool                 `json:"karaoke_available,omitempty"`
	RetryAfterSeconds int                  `json:"retry_after_seconds,omitempty"`
	Message           string               `json:"message,omitempty"`
}

// SyncStatusResolver stamps local timing rows when hosted sync is unavailable.
type SyncStatusResolver interface {
	ResolveSyncStatus() string
}

// EventChangeHook is called after mutations that should refresh live CSV.
type EventChangeHook func(eventID uuid.UUID)

// ScanService resolves tags and scores finish-mode RFID laps.
type ScanService struct {
	db       *gorm.DB
	sync     SyncStatusResolver
	onChange EventChangeHook
}

func NewScanService(db *gorm.DB, sync SyncStatusResolver) *ScanService {
	return &ScanService{db: db, sync: sync}
}

func (s *ScanService) SetOnEventChange(hook EventChangeHook) {
	s.onChange = hook
}

func (s *ScanService) notifyChange(eventID uuid.UUID) {
	if s.onChange != nil && eventID != uuid.Nil {
		s.onChange(eventID)
	}
}

// ProcessScan handles a tag read for eventID.
func (s *ScanService) ProcessScan(eventID uuid.UUID, tagUID, deviceID string, localTimestamp time.Time) (*ScanResult, error) {
	tagUID = strings.TrimSpace(tagUID)
	if tagUID == "" {
		return &ScanResult{Result: ResultUnknownTag}, nil
	}
	if localTimestamp.IsZero() {
		localTimestamp = time.Now().UTC()
	}

	participant, err := s.resolveParticipant(eventID, tagUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &ScanResult{Result: ResultUnknownTag}, nil
		}
		return nil, err
	}

	race := participant.Race
	raceID := race.ID
	part := *participant

	if race.Status != "active" {
		s.notifyChange(eventID)
		return withScanDisplay(&ScanResult{
			Result:      ResultTestRead,
			Participant: &part,
			RaceID:      &raceID,
			RaceStatus:  race.Status,
		}), nil
	}

	station := s.loadStation(eventID, deviceID)
	mode := "finish"
	if station != nil && station.Mode != "" {
		mode = station.Mode
	}

	if mode == "checkpoint" {
		return s.processCheckpointMode(station, participant, &race, deviceID, localTimestamp)
	}

	if retry := s.cooldownRemaining(participant.ID.UUID(), localTimestamp); retry > 0 {
		s.notifyChange(eventID)
		return withScanDisplay(&ScanResult{
			Result:            ResultCooldown,
			Participant:       &part,
			RaceID:            &raceID,
			RaceStatus:        race.Status,
			RetryAfterSeconds: retry,
		}), nil
	}

	finish, err := s.finishCheckpoint(race.ID.UUID())
	if err != nil {
		return nil, err
	}

	syncStatus := "synced"
	if s.sync != nil {
		syncStatus = s.sync.ResolveSyncStatus()
	}

	record := &models.TimingRecord{
		ParticipantID:  participant.ID,
		CheckpointID:   finish.ID,
		Timestamp:      localTimestamp,
		LocalTimestamp: localTimestamp,
		DeviceID:       deviceID,
		SyncStatus:     syncStatus,
		RecordType:     "rfid_lap",
		StationID:      ptrStationID(station),
	}
	if err := s.db.Create(record).Error; err != nil {
		return nil, err
	}

	lapCount, _ := s.scoredLapCount(participant.ID.UUID())
	placement, placementCat, _ := s.placements(race.ID.UUID(), participant)

	s.notifyChange(eventID)

	recID := record.ID
	return withScanDisplay(&ScanResult{
		Result:            ResultLap,
		Participant:       &part,
		RaceID:            &raceID,
		RaceStatus:        race.Status,
		LapCount:          lapCount,
		Placement:         placement,
		PlacementCategory: placementCat,
		TimingRecordID:    &recID,
		KaraokeAvailable:  true,
	}), nil
}

func withScanDisplay(r *ScanResult) *ScanResult {
	if r == nil || r.Participant == nil {
		return r
	}
	p := r.Participant
	r.ParticipantName = strings.TrimSpace(p.FirstName + " " + p.LastName)
	r.BibNumber = p.BibNumber
	if p.Race.Name != "" {
		r.RaceName = p.Race.Name
	}
	if p.Category != nil {
		r.CategoryLabel = p.Category.Name
	}
	return r
}

func (s *ScanService) resolveParticipant(eventID uuid.UUID, tagUID string) (*models.Participant, error) {
	var assoc models.RFIDTagAssociation
	err := s.db.Where("tag_uid = ? AND active = ?", tagUID, true).First(&assoc).Error
	if err == nil {
		var p models.Participant
		if err := s.db.Preload("Category").Preload("Race").First(&p, "id = ?", assoc.ParticipantID).Error; err != nil {
			return nil, err
		}
		if !s.participantInEvent(&p, eventID) {
			return nil, gorm.ErrRecordNotFound
		}
		return &p, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	var p models.Participant
	if err := s.db.Preload("Category").Preload("Race").Where("rfid_tag_uid = ?", tagUID).First(&p).Error; err != nil {
		return nil, err
	}
	if !s.participantInEvent(&p, eventID) {
		return nil, gorm.ErrRecordNotFound
	}
	return &p, nil
}

func (s *ScanService) participantInEvent(p *models.Participant, eventID uuid.UUID) bool {
	var race models.Race
	if err := s.db.Select("id", "event_id").First(&race, "id = ?", p.RaceID).Error; err != nil {
		return false
	}
	return race.EventID.UUID() == eventID
}

func (s *ScanService) cooldownRemaining(participantID uuid.UUID, at time.Time) int {
	var last models.TimingRecord
	err := s.db.Where("participant_id = ? AND record_type = ?", participantID, "rfid_lap").
		Order("timestamp DESC").
		First(&last).Error
	if err != nil {
		return 0
	}
	elapsed := at.Sub(last.Timestamp)
	if elapsed >= CooldownDuration {
		return 0
	}
	secs := int(math.Ceil((CooldownDuration - elapsed).Seconds()))
	return secs
}

func (s *ScanService) finishCheckpoint(raceID uuid.UUID) (*models.TimingCheckpoint, error) {
	var finish models.TimingCheckpoint
	err := s.db.Where("race_id = ? AND checkpoint_type = ?", raceID, "finish").First(&finish).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = s.db.Where("race_id = ? AND checkpoint_type = ?", raceID, "start").First(&finish).Error
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("no finish checkpoint for race")
		}
		return nil, err
	}
	return &finish, nil
}

func (s *ScanService) scoredLapCount(participantID uuid.UUID) (int, error) {
	var count int64
	err := s.db.Model(&models.TimingRecord{}).
		Where("participant_id = ? AND record_type IN ?", participantID, []string{"rfid_lap", "karaoke_bonus"}).
		Count(&count).Error
	return int(count), err
}

type scoredEntry struct {
	participantID uuidutil.PublicUUID
	categoryID    *uuidutil.PublicUUID
	laps          int
	lastLapAt     time.Time
}

func (s *ScanService) placements(raceID uuid.UUID, participant *models.Participant) (overall, category int, err error) {
	entries, err := s.scoreRace(raceID)
	if err != nil {
		return 0, 0, err
	}

	for i, e := range entries {
		if e.participantID == participant.ID {
			overall = i + 1
			break
		}
	}

	if participant.CategoryID == nil {
		return overall, 0, nil
	}

	catPos := 0
	for _, e := range entries {
		if e.categoryID == nil || e.categoryID.UUID() != participant.CategoryID.UUID() {
			continue
		}
		catPos++
		if e.participantID == participant.ID {
			return overall, catPos, nil
		}
	}
	return overall, 0, nil
}

func (s *ScanService) scoreRace(raceID uuid.UUID) ([]scoredEntry, error) {
	var participants []models.Participant
	if err := s.db.Where("race_id = ?", raceID).Find(&participants).Error; err != nil {
		return nil, err
	}

	var entries []scoredEntry
	for _, p := range participants {
		var records []models.TimingRecord
		_ = s.db.Where(
			"participant_id = ? AND record_type IN ?",
			p.ID,
			[]string{"rfid_lap", "karaoke_bonus"},
		).Order("timestamp ASC").Find(&records).Error
		if len(records) == 0 {
			continue
		}
		lastLap := records[0].Timestamp
		for _, r := range records {
			if r.RecordType == "rfid_lap" && r.Timestamp.After(lastLap) {
				lastLap = r.Timestamp
			}
		}
		entries = append(entries, scoredEntry{
			participantID: p.ID,
			categoryID:    p.CategoryID,
			laps:          len(records),
			lastLapAt:     lastLap,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].laps != entries[j].laps {
			return entries[i].laps > entries[j].laps
		}
		return entries[i].lastLapAt.Before(entries[j].lastLapAt)
	})
	return entries, nil
}
