package services

import (
	"github.com/keweenaw-endurance/backend/internal/config"
	"gorm.io/gorm"
)

type Services struct {
	DB     *gorm.DB
	Config *config.Config
}

func NewServices(db *gorm.DB, cfg *config.Config) *Services {
	return &Services{
		DB:     db,
		Config: cfg,
	}
}