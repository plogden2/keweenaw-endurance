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
}

func NewServices(db *gorm.DB, cfg *config.Config) *Services {
	return &Services{
		DB:           db,
		Config:       cfg,
		Events:       NewEventService(db),
		Races:        NewRaceService(db),
		Participants: NewParticipantService(db),
	}
}
