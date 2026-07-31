package scan

import (
	"errors"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/eventpolicy"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/keweenaw-endurance/backend/internal/uuidutil"
	"gorm.io/gorm"
)

const (
	ResultLap           = "lap"
	ResultTestRead      = "test_read"
	ResultCooldown      = "cooldown"
	ResultUnknownTag    = "unknown_tag"
	ResultUnassignedBib = "unassigned_bib"

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
	TeamID            *uuidutil.PublicUUID `json:"team_id,omitempty"`
	TeamName          string               `json:"team_name,omitempty"`
	TeamPlacement     int                  `json:"team_placement,omitempty"`
	TeamAvgLaps       float64              `json:"team_avg_laps,omitempty"`
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

// ScanOptions carries optional behavior for ProcessScan.
type ScanOptions struct {
	// BridgeRecordID, when set, is used as the timing_records primary key so
	// bridge flush replays are idempotent across reconnects.
	BridgeRecordID string
}

// ProcessScan handles a tag read for eventID.
func (s *ScanService) ProcessScan(eventID uuid.UUID, tagUID, deviceID string, localTimestamp time.Time, opts ...ScanOptions) (*ScanResult, error) {
	var opt ScanOptions
	if len(opts) > 0 {
		opt = opts[0]
	}
	tagUID = strings.TrimSpace(tagUID)
	if tagUID == "" {
		return &ScanResult{Result: ResultUnknownTag}, nil
	}
	if localTimestamp.IsZero() {
		localTimestamp = time.Now().UTC()
	}

	participant, bib, err := s.resolveParticipant(eventID, tagUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &ScanResult{Result: ResultUnknownTag}, nil
		}
		return nil, err
	}
	if participant == nil {
		bibNumber := ""
		if bib != nil {
			bibNumber = bib.BibNumber
		}
		s.notifyChange(eventID)
		return &ScanResult{
			Result:    ResultUnassignedBib,
			BibNumber: bibNumber,
			Message:   "Bib has no assigned racer",
		}, nil
	}

	race := participant.Race
	raceID := race.ID
	part := *participant
	bridgeReplay := strings.TrimSpace(opt.BridgeRecordID) != ""

	// Live taps only score while the race is active. Offline bridge flush
	// replays (source_lap_id set) may arrive after AutoFinish — still score
	// when the original tap timestamp falls inside the race window.
	if race.Status != "active" {
		if !bridgeReplay || !timestampInRaceWindow(race, localTimestamp) {
			s.notifyChange(eventID)
			return withScanDisplay(&ScanResult{
				Result:      ResultTestRead,
				Participant: &part,
				RaceID:      &raceID,
				RaceStatus:  race.Status,
			}), nil
		}
	}

	station := s.loadStation(eventID, deviceID)
	mode := "finish"
	if station != nil && station.Mode != "" {
		mode = station.Mode
	}
	// Bluffet is finish-only mid-event safety: an already-armed checkpoint
	// station must not keep mis-scoring, even if the DB still says checkpoint.
	if eventpolicy.IsBluffetEventID(eventID.String()) {
		mode = "finish"
	}

	if mode == "checkpoint" {
		return s.processCheckpointMode(station, participant, &race, deviceID, localTimestamp)
	}

	// Cooldown is for live double-taps. Bridge flush uses source_lap_id for
	// idempotency; skipping cooldown avoids dropping valid offline laps that
	// land within 60s of a pre-outage lap (or of each other after A→B→A).
	if !bridgeReplay {
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
	if bridgeID := strings.TrimSpace(opt.BridgeRecordID); bridgeID != "" {
		if id, err := uuid.Parse(bridgeID); err == nil {
			record.ID = uuidutil.NewPublicUUID(id)
		}
	}
	if err := s.db.Create(record).Error; err != nil {
		return nil, err
	}

	lapCount, _ := s.scoredLapCount(participant.ID.UUID())
	placement, placementCat, _ := s.placements(race.ID.UUID(), participant)
	teamPlace, teamAvg, _ := s.teamPlacement(race.ID.UUID(), participant)

	s.notifyChange(eventID)

	recID := record.ID
	result := &ScanResult{
		Result:            ResultLap,
		Participant:       &part,
		RaceID:            &raceID,
		RaceStatus:        race.Status,
		LapCount:          lapCount,
		Placement:         placement,
		PlacementCategory: placementCat,
		TimingRecordID:    &recID,
		KaraokeAvailable:  true,
	}
	if participant.TeamID != nil && !participant.TeamID.IsZero() {
		tid := *participant.TeamID
		result.TeamID = &tid
		result.TeamPlacement = teamPlace
		result.TeamAvgLaps = teamAvg
	}
	return withScanDisplay(result), nil
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
	if p.Team != nil {
		r.TeamName = p.Team.Name
		if r.TeamID == nil {
			tid := p.Team.ID
			r.TeamID = &tid
		}
	}
	return r
}

// resolveParticipant maps a chip UID to a participant for the event.
// Order: active association → bib → participant by bib_number; else legacy
// rfid_tag_uid; else chip UUID = participants.id (scoped to event).
// Association/bib wins when both association and legacy id would match.
// When a bib is found in the event with no assigned participant, returns
// (nil, bib, nil) so ProcessScan can emit ResultUnassignedBib.
func (s *ScanService) resolveParticipant(eventID uuid.UUID, tagUID string) (*models.Participant, *models.Bib, error) {
	var assoc models.RFIDTagAssociation
	err := s.db.Where("tag_uid = ? AND active = ?", tagUID, true).First(&assoc).Error
	if err == nil {
		var bib models.Bib
		if err := s.db.First(&bib, "id = ?", assoc.BibID).Error; err != nil {
			return nil, nil, err
		}
		if bib.EventID.UUID() != eventID {
			return nil, nil, gorm.ErrRecordNotFound
		}
		var p models.Participant
		err := s.db.Preload("Category").Preload("Team").Preload("Race").
			Joins("JOIN races ON races.id = participants.race_id").
			Where("races.event_id = ? AND participants.bib_number = ?", eventID, bib.BibNumber).
			First(&p).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, &bib, nil
			}
			return nil, nil, err
		}
		return &p, &bib, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, err
	}

	var p models.Participant
	err = s.db.Preload("Category").Preload("Team").Preload("Race").
		Where("rfid_tag_uid = ?", tagUID).First(&p).Error
	if err == nil {
		if !s.participantInEvent(&p, eventID) {
			return nil, nil, gorm.ErrRecordNotFound
		}
		return &p, nil, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, err
	}

	parsed, parseErr := uuid.Parse(tagUID)
	if parseErr != nil {
		return nil, nil, gorm.ErrRecordNotFound
	}
	err = s.db.Preload("Category").Preload("Team").Preload("Race").
		Joins("JOIN races ON races.id = participants.race_id").
		Where("races.event_id = ? AND participants.id = ?", eventID, parsed).
		First(&p).Error
	if err != nil {
		return nil, nil, err
	}
	return &p, nil, nil
}

func (s *ScanService) participantInEvent(p *models.Participant, eventID uuid.UUID) bool {
	var race models.Race
	if err := s.db.Select("id", "event_id").First(&race, "id = ?", p.RaceID).Error; err != nil {
		return false
	}
	return race.EventID.UUID() == eventID
}

// timestampInRaceWindow reports whether at falls in
// [start_time, start_time+duration] for a timed lap race.
func timestampInRaceWindow(race models.Race, at time.Time) bool {
	if race.StartTime.IsZero() || race.DurationMinutes <= 0 {
		return false
	}
	start := race.StartTime.UTC()
	end := start.Add(time.Duration(race.DurationMinutes) * time.Minute)
	ts := at.UTC()
	return !ts.Before(start) && !ts.After(end)
}

func (s *ScanService) cooldownRemaining(participantID uuid.UUID, at time.Time) int {
	var last models.TimingRecord
	err := s.db.Where("participant_id = ? AND record_type = ? AND voided_at IS NULL", participantID, "rfid_lap").
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
		Where("participant_id = ? AND record_type IN ? AND voided_at IS NULL", participantID, []string{"rfid_lap", "karaoke_bonus"}).
		Count(&count).Error
	return int(count), err
}

// ScoreSnapshot returns current scored lap count and placements after a timing mutation,
// and refreshes the live CSV via onChange.
func (s *ScanService) ScoreSnapshot(participantID uuid.UUID) (lapCount, placement, placementCategory int, eventID uuid.UUID, err error) {
	var participant models.Participant
	if err := s.db.Preload("Category").Preload("Team").Preload("Race").First(&participant, "id = ?", participantID).Error; err != nil {
		return 0, 0, 0, uuid.Nil, err
	}
	lapCount, err = s.scoredLapCount(participantID)
	if err != nil {
		return 0, 0, 0, uuid.Nil, err
	}
	placement, placementCategory, err = s.placements(participant.RaceID.UUID(), &participant)
	if err != nil {
		return 0, 0, 0, uuid.Nil, err
	}
	eventID = participant.Race.EventID.UUID()
	s.notifyChange(eventID)
	return lapCount, placement, placementCategory, eventID, nil
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

func (s *ScanService) teamPlacement(raceID uuid.UUID, participant *models.Participant) (place int, avg float64, err error) {
	if participant == nil || participant.TeamID == nil || participant.TeamID.IsZero() {
		return 0, 0, nil
	}
	var race models.Race
	if err := s.db.First(&race, "id = ?", raceID).Error; err != nil {
		return 0, 0, err
	}
	raceEnd := race.StartTime.Add(time.Duration(race.DurationMinutes) * time.Minute)

	var teams []models.Team
	if err := s.db.Where("race_id = ?", raceID).Find(&teams).Error; err != nil {
		return 0, 0, err
	}

	type scored struct {
		teamID   uuidutil.PublicUUID
		avg      float64
		meanLast time.Time
		nameKey  string
	}
	var scoredResults []scored
	for _, team := range teams {
		var members []models.Participant
		if err := s.db.Where("team_id = ?", team.ID).Find(&members).Error; err != nil {
			return 0, 0, err
		}
		if len(members) < 2 {
			continue
		}
		sumLaps := 0
		var lastSumNs int64
		for _, m := range members {
			var records []models.TimingRecord
			_ = s.db.Where(
				"participant_id = ? AND record_type IN ? AND voided_at IS NULL",
				m.ID,
				[]string{"rfid_lap", "karaoke_bonus"},
			).Find(&records).Error
			sumLaps += len(records)
			memberLast := raceEnd
			hasRFID := false
			for _, r := range records {
				if r.RecordType != "rfid_lap" {
					continue
				}
				if !hasRFID || r.Timestamp.After(memberLast) {
					memberLast = r.Timestamp
					hasRFID = true
				}
			}
			lastSumNs += memberLast.UnixNano()
		}
		n := len(members)
		scoredResults = append(scoredResults, scored{
			teamID:   team.ID,
			avg:      float64(sumLaps) / float64(n),
			meanLast: time.Unix(0, lastSumNs/int64(n)).UTC(),
			nameKey:  strings.ToLower(team.Name),
		})
	}
	sort.Slice(scoredResults, func(i, j int) bool {
		if scoredResults[i].avg != scoredResults[j].avg {
			return scoredResults[i].avg > scoredResults[j].avg
		}
		if !scoredResults[i].meanLast.Equal(scoredResults[j].meanLast) {
			return scoredResults[i].meanLast.Before(scoredResults[j].meanLast)
		}
		return scoredResults[i].nameKey < scoredResults[j].nameKey
	})
	for i, item := range scoredResults {
		if item.teamID == *participant.TeamID {
			rounded := float64(int(item.avg*10+0.5)) / 10
			return i + 1, rounded, nil
		}
	}
	return 0, 0, nil
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
			"participant_id = ? AND record_type IN ? AND voided_at IS NULL",
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
