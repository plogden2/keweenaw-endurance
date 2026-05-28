package services

import (
	"testing"

	"github.com/keweenaw-endurance/backend/internal/config"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestNewServices(t *testing.T) {
	t.Run("CreateServices", func(t *testing.T) {
		// Create a test database
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		assert.NoError(t, err)
		defer func() {
			sqlDB, _ := db.DB()
			if sqlDB != nil {
				sqlDB.Close()
			}
		}()
		
		// Create test config
		cfg := &config.Config{
			Environment: "test",
			Port:        "8080",
		}
		
		// Create services
		services := NewServices(db, cfg)
		
		assert.NotNil(t, services)
		assert.Equal(t, db, services.DB)
		assert.Equal(t, cfg, services.Config)
	})
	
	t.Run("ServicesWithNilDB", func(t *testing.T) {
		cfg := &config.Config{
			Environment: "test",
			Port:        "8080",
		}
		
		services := NewServices(nil, cfg)
		
		assert.NotNil(t, services)
		assert.Nil(t, services.DB)
		assert.Equal(t, cfg, services.Config)
	})
	
	t.Run("ServicesWithNilConfig", func(t *testing.T) {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		assert.NoError(t, err)
		defer func() {
			sqlDB, _ := db.DB()
			if sqlDB != nil {
				sqlDB.Close()
			}
		}()
		
		services := NewServices(db, nil)
		
		assert.NotNil(t, services)
		assert.Equal(t, db, services.DB)
		assert.Nil(t, services.Config)
	})
	
	t.Run("ServicesWithBothNil", func(t *testing.T) {
		services := NewServices(nil, nil)
		
		assert.NotNil(t, services)
		assert.Nil(t, services.DB)
		assert.Nil(t, services.Config)
	})
}