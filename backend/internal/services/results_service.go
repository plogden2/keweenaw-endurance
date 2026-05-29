package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/cache"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/keweenaw-endurance/backend/internal/uuidutil"
	"gorm.io/gorm"
)

const leaderboardCacheTTL = 30 * time.Second

type LeaderboardEntry struct {
	Position         int       `json:"position"`
	ParticipantID    uuidutil.PublicUUID `json:"participant_id"`
	BibNumber        string    `json:"bib_number"`
	FirstName        string    `json:"first_name"`
	LastName         string    `json:"last_name"`
	Location         string    `json:"location"`
	TotalTimeSeconds float64   `json:"total_time_seconds"`
	Laps             int       `json:"laps,omitempty"`
	Status           string    `json:"status"`
}

type LiveTimingData struct {
	RaceID  uuidutil.PublicUUID    `json:"race_id"`
	Records []models.TimingRecord  `json:"records"`
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
	if category.RaceID != raceID {
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
	var finishCheckpoint models.TimingCheckpoint
	if err := s.db.Where("race_id = ? AND checkpoint_type = ?", raceID, "finish").First(&finishCheckpoint).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := s.db.Where("race_id = ? AND checkpoint_type = ?", raceID, "start").First(&finishCheckpoint).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return []LeaderboardEntry{}, nil
				}
				return nil, err
			}
		} else {
			return nil, err
		}
	}

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
		if err := s.db.Where("participant_id = ? AND checkpoint_id = ?", participant.ID, finishCheckpoint.ID).
			Order("timestamp ASC").Find(&records).Error; err != nil {
			return nil, err
		}
		if len(records) == 0 {
			continue
		}

		last := records[len(records)-1]
		entry := LeaderboardEntry{
			ParticipantID: participant.ID,
			BibNumber:     participant.BibNumber,
			FirstName:     participant.FirstName,
			LastName:      participant.LastName,
			Status:        participant.Status,
			Laps:          len(records),
		}
		if len(records) > 1 {
			entry.TotalTimeSeconds = last.Timestamp.Sub(records[0].Timestamp).Seconds()
		}
		scoredResults = append(scoredResults, scored{entry: entry, laps: len(records), time: last.Timestamp})
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
		if err := db.First(&participant, "id = ?", entry.ParticipantID).Error; err != nil {
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
