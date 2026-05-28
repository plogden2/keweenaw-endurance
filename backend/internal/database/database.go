package database

import (
	"fmt"
	"log"
	"time"

	"github.com/keweenaw-endurance/backend/internal/config"
	"github.com/keweenaw-endurance/backend/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

func Initialize(cfg config.DatabaseConfig) (*gorm.DB, error) {
	var err error
	
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Name)
	
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}
	
	// Configure connection pool
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	
	// Test connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	
	// Run migrations
	if err := Migrate(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}
	
	log.Println("Database initialized successfully")
	return db, nil
}

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.Event{},
		&models.Race{},
		&models.Participant{},
		&models.TimingCheckpoint{},
		&models.TimingRecord{},
		&models.Category{},
	)
}

func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func GetDB() *gorm.DB {
	return db
}