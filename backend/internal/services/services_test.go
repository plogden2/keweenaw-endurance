package services

import (
	"testing"

	"github.com/keweenaw-endurance/backend/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNewServices(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	cfg := &config.Config{
		Environment: "test",
		Port:        "8080",
	}

	services := NewServices(db, cfg)

	assert.NotNil(t, services)
	assert.Equal(t, db, services.DB)
	assert.Equal(t, cfg, services.Config)
	assert.NotNil(t, services.Events)
	assert.NotNil(t, services.Races)
	assert.NotNil(t, services.Participants)
	assert.NotNil(t, services.Checkpoints)
	assert.NotNil(t, services.Categories)
	assert.NotNil(t, services.Timing)
	assert.NotNil(t, services.Results)
	assert.NotNil(t, services.RFID)
	assert.NotNil(t, services.CSV)
	assert.NotNil(t, services.Scan)
	assert.NotNil(t, services.Stations)
	assert.NotNil(t, services.Bridge)
}
