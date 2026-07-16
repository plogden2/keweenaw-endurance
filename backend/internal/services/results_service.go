package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/cache"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/keweenaw-endurance/backend/internal/uuidutil"
	"gorm.io/gorm"
)

const leaderboardCacheTTL = 30 * time.Second

type LeaderboardEntry struct {
	Position         int                 `json:"position"`
	ParticipantID    uuidutil.PublicUUID `json:"participant_id"`
	BibNumber        string              `json:"bib_number"`
	FirstName        string              `json:"first_name"`
	LastName         string              `json:"last_name"`
	Location         string              `json:"location"`
	TotalTimeSeconds float64             `json:"total_time_seconds"`
	Laps             int                 `json:"laps,omitempty"`
	Status           string              `json:"status"`
	CategoryKey      string              `json:"category_key,omitempty"`
	LastLapAt        *time.Time          `json:"last_lap_at,omitempty"`
}

type LiveTimingData struct {
	RaceID  uuidutil.PublicUUID   `json:"race_id"`
	Records []models.TimingRecord `json:"records"`
}

// LiveOverallEntry is a row on the public event live leaderboard.
type LiveOverallEntry struct {
	Place         int                 `json:"place"`
	ParticipantID uuidutil.PublicUUID `json:"participant_id"`
	BibNumber     string              `json:"bib_number"`
	Name          string              `json:"name"`
	CategoryKey   string              `json:"category_key"`
	Laps          int                 `json:"laps"`
	LastLapAt     *time.Time          `json:"last_lap_at,omitempty"`
}

// CategoryLegendEntry maps category keys to display labels and colors.
type CategoryLegendEntry struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Color string `json:"color"`
}

// LiveRaceView is one race block inside GET /api/events/:id/live.
type LiveRaceView struct {
	ID                 uuidutil.PublicUUID `json:"id"`
	Name               string              `json:"name"`
	RaceType           string              `json:"race_type"`
	Status             string              `json:"status"`
	StartTime          time.Time           `json:"start_time"`
	CountdownSeconds   int                 `json:"countdown_seconds"`
	LeaderboardOverall []LiveOverallEntry  `json:"leaderboard_overall"`
	FlowSeries         []interface{}       `json:"flow_series"`
}

// EventLiveView is the public live board payload.
type EventLiveView struct {
	Event          EventLiveSummary       `json:"event"`
	CategoryLegend []CategoryLegendEntry  `json:"category_legend"`
	Races          []LiveRaceView         `json:"races"`
}

type EventLiveSummary struct {
	ID   uuidutil.PublicUUID `json:"id"`
	Name string              `json:"name"`
}

type ResultsService struct {
	db    *gorm.DB
	cache cache.LeaderboardCache
}

func NewResultsService(db *gorm.DB, leaderboardCache cache.LeaderboardCache) *ResultsService {
	return &ResultsService{db: db, cache: leaderboardCache}
}

func (s *ResultsService) GetRaceResults(raceID uuid.UUID) ([]LeaderboardEntry, error) {
	race, err := s.loadRace(raceID)
	if err != nil {
		return nil, err
	}

	switch race.RaceType {
	case "lap_based":
		return s.calculateLapResults(raceID)
	default:
		return s.calculateTimeBasedResults(raceID)
	}
}

func (s *ResultsService) GetLeaderboard(raceID uuid.UUID, categoryID *uuid.UUID) ([]LeaderboardEntry, error) {
	if s.cache != nil {
		cacheKey := leaderboardCacheKey(raceID, categoryID)
		ctx := context.Background()
		if cached, ok := s.cache.Get(ctx, cacheKey); ok {
			var entries []LeaderboardEntry
			if err := cache.UnmarshalJSON(cached, &entries); err == nil {
				return entries, nil
			}
		}
	}

	entries, err := s.computeLeaderboard(raceID, categoryID)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		cacheKey := leaderboardCacheKey(raceID, categoryID)
		if payload, err := cache.MarshalJSON(entries); err == nil {
			_ = s.cache.Set(context.Background(), cacheKey, payload, leaderboardCacheTTL)
		}
	}

	return entries, nil
}

func leaderboardCacheKey(raceID uuid.UUID, categoryID *uuid.UUID) string {
	if categoryID == nil {
		return fmt.Sprintf("leaderboard:%s:all", raceID)
	}
	return fmt.Sprintf("leaderboard:%s:%s", raceID, *categoryID)
}

func (s *ResultsService) computeLeaderboard(raceID uuid.UUID, categoryID *uuid.UUID) ([]LeaderboardEntry, error) {
	results, err := s.GetRaceResults(raceID)
	if err != nil {
		return nil, err
	}
	if categoryID == nil {
		return results, nil
	}

	category, err := NewCategoryService(s.db).GetCategory(*categoryID)
	if err != nil {
		return nil, err
	}
	if category.RaceID.UUID() != raceID {
		return nil, ErrInvalidCategoryInput
	}

	return filterResultsByCategory(s.db, results, category)
}

func (s *ResultsService) GetLiveTiming(raceID uuid.UUID) (*LiveTimingData, error) {
	if _, err := s.loadRace(raceID); err != nil {
		return nil, err
	}

	records, err := NewTimingService(s.db).ListRecordsByRace(raceID)
	if err != nil {
		return nil, err
	}

	return &LiveTimingData{
		RaceID:  uuidutil.NewPublicUUID(raceID),
		Records: records,
	}, nil
}

// GetEventLive builds the public live view for an event (countdown, overall boards, legend).
func (s *ResultsService) GetEventLive(eventID uuid.UUID, categoryID *uuid.UUID) (*EventLiveView, error) {
	var event models.Event
	if err := s.db.First(&event, "id = ?", eventID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEventNotFound
		}
		return nil, err
	}

	var races []models.Race
	if err := s.db.Where("event_id = ? AND status != ?", eventID, "cancelled").
		Order("start_time ASC").Find(&races).Error; err != nil {
		return nil, err
	}

	now := time.Now()
	legendMap := map[string]CategoryLegendEntry{}
	views := make([]LiveRaceView, 0, len(races))

	for _, race := range races {
		countdown := 0
		if race.Status == "scheduled" && !race.StartTime.IsZero() {
			secs := int(race.StartTime.Sub(now).Seconds())
			if secs > 0 {
				countdown = secs
			}
		}

		board, err := s.buildOverallLeaderboard(race.ID.UUID(), categoryID, legendMap)
		if err != nil {
			return nil, err
		}

		views = append(views, LiveRaceView{
			ID:                 race.ID,
			Name:               race.Name,
			RaceType:           race.RaceType,
			Status:             race.Status,
			StartTime:          race.StartTime,
			CountdownSeconds:   countdown,
			LeaderboardOverall: board,
			FlowSeries:         []interface{}{},
		})
	}

	legend := make([]CategoryLegendEntry, 0, len(legendMap))
	for _, e := range legendMap {
		legend = append(legend, e)
	}
	sort.Slice(legend, func(i, j int) bool { return legend[i].Label < legend[j].Label })

	return &EventLiveView{
		Event: EventLiveSummary{
			ID:   event.ID,
			Name: event.Name,
		},
		CategoryLegend: legend,
		Races:          views,
	}, nil
}

func (s *ResultsService) buildOverallLeaderboard(raceID uuid.UUID, categoryFilter *uuid.UUID, legend map[string]CategoryLegendEntry) ([]LiveOverallEntry, error) {
	var participants []models.Participant
	q := s.db.Preload("Category").Where("race_id = ?", raceID)
	if categoryFilter != nil {
		q = q.Where("category_id = ?", *categoryFilter)
	}
	if err := q.Find(&participants).Error; err != nil {
		return nil, err
	}

	type scored struct {
		entry LiveOverallEntry
		laps  int
		last  time.Time
	}
	var scoredResults []scored

	for _, p := range participants {
		var records []models.TimingRecord
		if err := s.db.Where(
			"participant_id = ? AND record_type IN ?",
			p.ID,
			[]string{"rfid_lap", "karaoke_bonus"},
		).Order("timestamp ASC").Find(&records).Error; err != nil {
			return nil, err
		}
		if len(records) == 0 {
			continue
		}

		lastLapAt := records[0].Timestamp
		for _, r := range records {
			if r.RecordType == "rfid_lap" && r.Timestamp.After(lastLapAt) {
				lastLapAt = r.Timestamp
			}
		}
		last := lastLapAt
		key := categoryKey(p.Category)
		if p.Category != nil {
			legend[key] = CategoryLegendEntry{
				Key:   key,
				Label: p.Category.Name,
				Color: categoryColor(key),
			}
		}

		scoredResults = append(scoredResults, scored{
			entry: LiveOverallEntry{
				ParticipantID: p.ID,
				BibNumber:     p.BibNumber,
				Name:          strings.TrimSpace(p.FirstName + " " + p.LastName),
				CategoryKey:   key,
				Laps:          len(records),
				LastLapAt:     &last,
			},
			laps: len(records),
			last: lastLapAt,
		})
	}

	sort.Slice(scoredResults, func(i, j int) bool {
		if scoredResults[i].laps != scoredResults[j].laps {
			return scoredResults[i].laps > scoredResults[j].laps
		}
		return scoredResults[i].last.Before(scoredResults[j].last)
	})

	out := make([]LiveOverallEntry, len(scoredResults))
	for i, item := range scoredResults {
		item.entry.Place = i + 1
		out[i] = item.entry
	}
	return out, nil
}

func categoryKey(cat *models.Category) string {
	if cat == nil {
		return "uncategorized"
	}
	name := strings.ToLower(strings.TrimSpace(cat.Name))
	replacer := strings.NewReplacer(" ", "_", "-", "_", "/", "_")
	return replacer.Replace(name)
}

var categoryColorPalette = []string{
	"#1a5276", "#117a65", "#9a7d0a", "#922b21", "#6c3483", "#1b4f72", "#196f3d", "#b9770e",
}

func categoryColor(key string) string {
	if key == "" || key == "uncategorized" {
		return "#566573"
	}
	h := 0
	for _, c := range key {
		h = (h*31 + int(c)) & 0xffff
	}
	return categoryColorPalette[h%len(categoryColorPalette)]
}

func (s *ResultsService) loadRace(raceID uuid.UUID) (*models.Race, error) {
	var race models.Race
	if err := s.db.First(&race, "id = ?", raceID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRaceNotFound
		}
		return nil, err
	}
	return &race, nil
}

func (s *ResultsService) calculateTimeBasedResults(raceID uuid.UUID) ([]LeaderboardEntry, error) {
	var startCheckpoint, finishCheckpoint models.TimingCheckpoint
	if err := s.db.Where("race_id = ? AND checkpoint_type = ?", raceID, "start").First(&startCheckpoint).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []LeaderboardEntry{}, nil
		}
		return nil, err
	}
	if err := s.db.Where("race_id = ? AND checkpoint_type = ?", raceID, "finish").First(&finishCheckpoint).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []LeaderboardEntry{}, nil
		}
		return nil, err
	}

	var participants []models.Participant
	if err := s.db.Where("race_id = ?", raceID).Find(&participants).Error; err != nil {
		return nil, err
	}

	type scored struct {
		entry LeaderboardEntry
		sort  float64
	}
	var scoredResults []scored

	for _, participant := range participants {
		var startRecord, finishRecord models.TimingRecord
		startErr := s.db.Where("participant_id = ? AND checkpoint_id = ?", participant.ID, startCheckpoint.ID).
			Order("timestamp ASC").First(&startRecord).Error
		finishErr := s.db.Where("participant_id = ? AND checkpoint_id = ?", participant.ID, finishCheckpoint.ID).
			Order("timestamp DESC").First(&finishRecord).Error

		entry := LeaderboardEntry{
			ParticipantID: participant.ID,
			BibNumber:     participant.BibNumber,
			FirstName:     participant.FirstName,
			LastName:      participant.LastName,
			Location:      participant.Location,
			Status:        participant.Status,
		}

		if startErr == nil && finishErr == nil && finishRecord.Timestamp.After(startRecord.Timestamp) {
			entry.TotalTimeSeconds = finishRecord.Timestamp.Sub(startRecord.Timestamp).Seconds()
			scoredResults = append(scoredResults, scored{entry: entry, sort: entry.TotalTimeSeconds})
		}
	}

	sort.Slice(scoredResults, func(i, j int) bool {
		return scoredResults[i].sort < scoredResults[j].sort
	})

	results := make([]LeaderboardEntry, len(scoredResults))
	for i, item := range scoredResults {
		item.entry.Position = i + 1
		results[i] = item.entry
	}

	return results, nil
}

func (s *ResultsService) calculateLapResults(raceID uuid.UUID) ([]LeaderboardEntry, error) {
	var participants []models.Participant
	if err := s.db.Where("race_id = ?", raceID).Find(&participants).Error; err != nil {
		return nil, err
	}

	type scored struct {
		entry LeaderboardEntry
		laps  int
		time  time.Time
	}
	var scoredResults []scored

	for _, participant := range participants {
		var records []models.TimingRecord
		if err := s.db.Where(
			"participant_id = ? AND record_type IN ?",
			participant.ID,
			[]string{"rfid_lap", "karaoke_bonus"},
		).Order("timestamp ASC").Find(&records).Error; err != nil {
			return nil, err
		}
		if len(records) == 0 {
			continue
		}

		lastLapAt := records[0].Timestamp
		firstRFID := time.Time{}
		for _, r := range records {
			if r.RecordType == "rfid_lap" {
				if firstRFID.IsZero() {
					firstRFID = r.Timestamp
				}
				if r.Timestamp.After(lastLapAt) {
					lastLapAt = r.Timestamp
				}
			}
		}
		entry := LeaderboardEntry{
			ParticipantID: participant.ID,
			BibNumber:     participant.BibNumber,
			FirstName:     participant.FirstName,
			LastName:      participant.LastName,
			Status:        participant.Status,
			Laps:          len(records),
		}
		if !firstRFID.IsZero() && lastLapAt.After(firstRFID) {
			entry.TotalTimeSeconds = lastLapAt.Sub(firstRFID).Seconds()
		}
		scoredResults = append(scoredResults, scored{entry: entry, laps: len(records), time: lastLapAt})
	}

	sort.Slice(scoredResults, func(i, j int) bool {
		if scoredResults[i].laps != scoredResults[j].laps {
			return scoredResults[i].laps > scoredResults[j].laps
		}
		return scoredResults[i].time.Before(scoredResults[j].time)
	})

	results := make([]LeaderboardEntry, len(scoredResults))
	for i, item := range scoredResults {
		item.entry.Position = i + 1
		results[i] = item.entry
	}

	return results, nil
}

func filterResultsByCategory(db *gorm.DB, results []LeaderboardEntry, category *models.Category) ([]LeaderboardEntry, error) {
	var filtered []LeaderboardEntry
	position := 1

	for _, entry := range results {
		var participant models.Participant
		if err := db.First(&participant, "id = ?", entry.ParticipantID.UUID()).Error; err != nil {
			return nil, err
		}
		if !participantMatchesCategory(&participant, category) {
			continue
		}
		entry.Position = position
		position++
		filtered = append(filtered, entry)
	}

	return filtered, nil
}

func participantMatchesCategory(participant *models.Participant, category *models.Category) bool {
	switch category.CategoryType {
	case "male":
		return participant.Gender == "male"
	case "female":
		return participant.Gender == "female"
	case "age_group":
		if participant.Age < category.AgeMin || participant.Age > category.AgeMax {
			return false
		}
		if category.GenderFilter != "" && participant.Gender != category.GenderFilter {
			return false
		}
		return true
	case "overall", "custom":
		return true
	default:
		return false
	}
}
