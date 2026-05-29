package services

import (
	"github.com/keweenaw-endurance/backend/internal/config"
	"gorm.io/gorm"
)

type Services struct {
	DB           *gorm.DB
	Config       *config.Config
	Events       *EventService
	Races        *RaceService
	Participants *ParticipantService
	Checkpoints  *CheckpointService
	Categories   *CategoryService
	Timing       *TimingService
	Results      *ResultsService
}

func NewServices(db *gorm.DB, cfg *config.Config) *Services {
	return &Services{
		DB:           db,
		Config:       cfg,
		Events:       NewEventService(db),
		Races:        NewRaceService(db),
		Participants: NewParticipantService(db),
		Checkpoints:  NewCheckpointService(db),
		Categories:   NewCategoryService(db),
		Timing:       NewTimingService(db),
		Results:      NewResultsService(db),
	}
}
