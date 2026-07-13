package scan

import (
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/keweenaw-endurance/backend/internal/uuidutil"
	"gorm.io/gorm"
)

const (
	ResultOutOfOrder     = "out_of_order"
	ResultCheckpointPass = "checkpoint_pass"
)

// processCheckpointMode advances ordered checkpoint progress for a station tap.
// Out-of-order taps do not complete a lap. Completing the full sequence creates
// an rfid_lap (karaoke offered only then).
func (s *ScanService) processCheckpointMode(
	station *models.ReaderStation,
	participant *models.Participant,
	race *models.Race,
	deviceID string,
	localTimestamp time.Time,
) (*ScanResult, error) {
	part := *participant
	raceID := race.ID

	if station.CheckpointID == nil || station.CheckpointID.IsZero() {
		return withScanDisplay(&ScanResult{
			Result:      ResultOutOfOrder,
			Participant: &part,
			RaceID:      &raceID,
			RaceStatus:  race.Status,
			Message:     "Checkpoint station is not bound to a checkpoint",
		}), nil
	}

	sequence, err := s.orderedCheckpoints(race.ID.UUID())
	if err != nil {
		return nil, err
	}
	if len(sequence) == 0 {
		return nil, errors.New("no checkpoints configured for race")
	}

	expected, err := s.expectedCheckpoint(participant.ID.UUID(), sequence)
	if err != nil {
		return nil, err
	}

	tappedID := station.CheckpointID.UUID()
	if expected == nil || expected.ID.UUID() != tappedID {
		return withScanDisplay(&ScanResult{
			Result:      ResultOutOfOrder,
			Participant: &part,
			RaceID:      &raceID,
			RaceStatus:  race.Status,
			Message:     "Out of sequence — this checkpoint is not next for a completed lap",
		}), nil
	}

	syncStatus := "synced"
	if s.sync != nil {
		syncStatus = s.sync.ResolveSyncStatus()
	}

	pass := &models.TimingRecord{
		ParticipantID:  participant.ID,
		CheckpointID:   expected.ID,
		Timestamp:      localTimestamp,
		LocalTimestamp: localTimestamp,
		DeviceID:       deviceID,
		SyncStatus:     syncStatus,
		RecordType:     "checkpoint_pass",
		StationID:      ptrStationID(station),
	}
	if err := s.db.Create(pass).Error; err != nil {
		return nil, err
	}

	// Completing the last checkpoint in the ordered sequence yields one scored lap.
	if expected.ID.UUID() == sequence[len(sequence)-1].ID.UUID() {
		if retry := s.cooldownRemaining(participant.ID.UUID(), localTimestamp); retry > 0 {
			return withScanDisplay(&ScanResult{
				Result:            ResultCooldown,
				Participant:       &part,
				RaceID:            &raceID,
				RaceStatus:        race.Status,
				RetryAfterSeconds: retry,
				Message:           "Checkpoint sequence complete but lap cooldown still active",
			}), nil
		}

		lap := &models.TimingRecord{
			ParticipantID:  participant.ID,
			CheckpointID:   expected.ID,
			Timestamp:      localTimestamp,
			LocalTimestamp: localTimestamp,
			DeviceID:       deviceID,
			SyncStatus:     syncStatus,
			RecordType:     "rfid_lap",
			StationID:      ptrStationID(station),
		}
		if err := s.db.Create(lap).Error; err != nil {
			return nil, err
		}

		lapCount, _ := s.scoredLapCount(participant.ID.UUID())
		placement, placementCat, _ := s.placements(race.ID.UUID(), participant)
		recID := lap.ID
		s.notifyChange(race.EventID.UUID())
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

	passID := pass.ID
	s.notifyChange(race.EventID.UUID())
	return withScanDisplay(&ScanResult{
		Result:         ResultCheckpointPass,
		Participant:    &part,
		RaceID:         &raceID,
		RaceStatus:     race.Status,
		TimingRecordID: &passID,
		Message:        "Checkpoint recorded — continue the sequence to complete a lap",
		KaraokeAvailable: false,
	}), nil
}

func ptrStationID(station *models.ReaderStation) *uuidutil.PublicUUID {
	if station == nil || station.ID.IsZero() {
		return nil
	}
	id := station.ID
	return &id
}

// stationMode returns the configured station mode (default finish). Used by tests.
func (s *ScanService) stationMode(eventID uuid.UUID, deviceID string) string {
	st := s.loadStation(eventID, deviceID)
	if st == nil || strings.TrimSpace(st.Mode) == "" {
		return "finish"
	}
	return st.Mode
}

func (s *ScanService) loadStation(eventID uuid.UUID, deviceID string) *models.ReaderStation {
	deviceID = strings.TrimSpace(deviceID)
	var station models.ReaderStation
	q := s.db.Where("event_id = ?", eventID)
	if deviceID != "" {
		q = q.Where("device_id = ?", deviceID)
	}
	if err := q.Order("created_at DESC").First(&station).Error; err != nil {
		return nil
	}
	return &station
}

// orderedCheckpoints returns active race checkpoints in course order:
// start → intermediate (by distance) → finish.
func (s *ScanService) orderedCheckpoints(raceID uuid.UUID) ([]models.TimingCheckpoint, error) {
	var cps []models.TimingCheckpoint
	if err := s.db.Where("race_id = ? AND is_active = ?", raceID, true).Find(&cps).Error; err != nil {
		return nil, err
	}
	sort.SliceStable(cps, func(i, j int) bool {
		pi, pj := checkpointSequencePriority(cps[i]), checkpointSequencePriority(cps[j])
		if pi != pj {
			return pi < pj
		}
		return cps[i].DistanceFromStartKm < cps[j].DistanceFromStartKm
	})
	return cps, nil
}

func checkpointSequencePriority(cp models.TimingCheckpoint) int {
	switch cp.CheckpointType {
	case "start":
		return 0
	case "intermediate":
		return 1
	case "finish":
		return 2
	default:
		return 9
	}
}

// expectedCheckpoint is the next checkpoint in sequence for the current lap attempt
// (passes since the participant's last scored rfid_lap).
func (s *ScanService) expectedCheckpoint(
	participantID uuid.UUID,
	sequence []models.TimingCheckpoint,
) (*models.TimingCheckpoint, error) {
	if len(sequence) == 0 {
		return nil, nil
	}

	var lastLap models.TimingRecord
	lapErr := s.db.Where("participant_id = ? AND record_type = ?", participantID, "rfid_lap").
		Order("timestamp DESC").
		First(&lastLap).Error

	q := s.db.Where("participant_id = ? AND record_type = ?", participantID, "checkpoint_pass")
	if lapErr == nil {
		q = q.Where("timestamp > ?", lastLap.Timestamp)
	} else if !errors.Is(lapErr, gorm.ErrRecordNotFound) {
		return nil, lapErr
	}

	var passes []models.TimingRecord
	if err := q.Order("timestamp ASC").Find(&passes).Error; err != nil {
		return nil, err
	}

	passed := make(map[uuid.UUID]bool, len(passes))
	for _, p := range passes {
		passed[p.CheckpointID.UUID()] = true
	}

	for i := range sequence {
		if !passed[sequence[i].ID.UUID()] {
			return &sequence[i], nil
		}
	}
	// All checkpoints already passed without a lap (should not happen) — restart at first.
	return &sequence[0], nil
}
