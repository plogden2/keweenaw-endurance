package services

import (
	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/cache"
	"github.com/keweenaw-endurance/backend/internal/config"
	"github.com/keweenaw-endurance/backend/internal/rfid"
	"github.com/keweenaw-endurance/backend/internal/services/scan"
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
	Bridge       *BridgeHub
	LiveStream   *LiveStreamHub
	Scan         *scan.ScanService
	Stations     *StationService
	Sync         *SyncService
	CSV          *CSVExportService
}

func NewServices(db *gorm.DB, cfg *config.Config) *Services {
	return NewServicesWithReader(db, cfg, rfid.DefaultReader())
}

func NewServicesWithReader(db *gorm.DB, cfg *config.Config, reader rfid.Reader) *Services {
	syncSvc := NewSyncService(db, cfg)
	dataDir := "data"
	if cfg != nil && cfg.DataDir != "" {
		dataDir = cfg.DataDir
	}
	mirrorDir := ""
	if cfg != nil {
		mirrorDir = cfg.LiveCSVMirrorDir
	}
	csvSvc := NewCSVExportService(db, dataDir, mirrorDir)
	resultsSvc := NewResultsService(db, cache.NewLeaderboardCache(cfg.Redis))
	scanSvc := scan.NewScanService(db, syncSvc)
	scanSvc.SetOnEventChange(func(eventID uuid.UUID) {
		csvSvc.RefreshEvent(eventID)
		resultsSvc.InvalidateLeaderboardForEvent(eventID)
	})

	bridgeHub := NewBridgeHub()
	rfidSvc := NewRFIDService(db, reader)
	rfidSvc.ConfigureBridge(cfg, bridgeHub)

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
		Results:      resultsSvc,
		RFID:         rfidSvc,
		Bridge:       bridgeHub,
		LiveStream:   NewLiveStreamHub(),
		Scan:         scanSvc,
		Stations:     NewStationService(db),
		Sync:         syncSvc,
		CSV:          csvSvc,
	}
}
