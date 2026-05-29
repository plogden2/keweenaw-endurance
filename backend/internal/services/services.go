package services

import (
	"github.com/keweenaw-endurance/backend/internal/cache"
	"github.com/keweenaw-endurance/backend/internal/config"
	"github.com/keweenaw-endurance/backend/internal/rfid"
	"gorm.io/gorm"
)

type Services struct {
	DB           *gorm.DB
	Config       *config.Config
	Auth         *AuthService
	Events       *EventService
	Races        *RaceService
	Participants *ParticipantService
	Checkpoints  *CheckpointService
	Categories   *CategoryService
	Timing       *TimingService
	Results      *ResultsService
	RFID         *RFIDService
}

func NewServices(db *gorm.DB, cfg *config.Config) *Services {
	return NewServicesWithReader(db, cfg, rfid.DefaultReader())
}

func NewServicesWithReader(db *gorm.DB, cfg *config.Config, reader rfid.Reader) *Services {
	return &Services{
		DB:           db,
		Config:       cfg,
		Auth:         NewAuthService(cfg),
		Events:       NewEventService(db),
		Races:        NewRaceService(db),
		Participants: NewParticipantService(db),
		Checkpoints:  NewCheckpointService(db),
		Categories:   NewCategoryService(db),
		Timing:       NewTimingService(db),
		Results:      NewResultsService(db, cache.NewLeaderboardCache(cfg.Redis)),
		RFID:         NewRFIDService(db, reader),
	}
}
