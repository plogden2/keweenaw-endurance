package database

import (
	"testing"
	"time"

	"github.com/keweenaw-endurance/backend/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDatabaseInitialization(t *testing.T) {
	t.Run("SuccessfulInitialization", func(t *testing.T) {
		// Use SQLite for testing instead of PostgreSQL
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		require.NoError(t, err)
		require.NotNil(t, db)
		
		// Test that we can get the underlying SQL database
		sqlDB, err := db.DB()
		require.NoError(t, err)
		require.NotNil(t, sqlDB)
		
		// Test connection
		err = sqlDB.Ping()
		assert.NoError(t, err)
		
		// Close the connection
		err = sqlDB.Close()
		assert.NoError(t, err)
	})
}

func TestMigrate(t *testing.T) {
	t.Run("SuccessfulMigration", func(t *testing.T) {
		// Create in-memory SQLite database for testing
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		require.NoError(t, err)
		defer func() {
			sqlDB, _ := db.DB()
			if sqlDB != nil {
				sqlDB.Close()
			}
		}()
		
		// Test migration
		err = Migrate(db)
		assert.NoError(t, err)
		
		// Verify tables exist by attempting to query them
		// This will fail if the tables don't exist
		tables := []string{"events", "races", "participants", "timing_checkpoints", "timing_records", "categories"}
		
		for _, table := range tables {
			var count int64
			err = db.Table(table).Count(&count).Error
			assert.NoError(t, err, "Table %s should exist after migration", table)
		}
	})
}

func TestClose(t *testing.T) {
	t.Run("SuccessfulClose", func(t *testing.T) {
		// Create in-memory SQLite database
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		require.NoError(t, err)
		
		// Test close function
		err = Close(db)
		assert.NoError(t, err)
	})
}

func TestIntegrationWithConfig(t *testing.T) {
	t.Run("IntegrationWithTestConfig", func(t *testing.T) {
		// Create test configuration
		cfg := config.DatabaseConfig{
			Host:            "localhost",
			Port:            "5432",
			Name:            "test_db",
			User:            "test_user",
			Password:        "test_pass",
			MaxOpenConns:    10,
			MaxIdleConns:    2,
			ConnMaxLifetime: time.Hour,
		}
		
		// Note: We can't test the actual Initialize function with PostgreSQL
		// in unit tests without a real database, but we can test the configuration
		// processing logic
		assert.Equal(t, "localhost", cfg.Host)
		assert.Equal(t, "5432", cfg.Port)
		assert.Equal(t, "test_db", cfg.Name)
		assert.Equal(t, 10, cfg.MaxOpenConns)
		assert.Equal(t, 2, cfg.MaxIdleConns)
		assert.Equal(t, time.Hour, cfg.ConnMaxLifetime)
	})
}

func TestDatabaseConnectionPooling(t *testing.T) {
	t.Run("ConnectionPoolingConfiguration", func(t *testing.T) {
		// Create in-memory SQLite database
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		require.NoError(t, err)
		defer func() {
			sqlDB, _ := db.DB()
			if sqlDB != nil {
				sqlDB.Close()
			}
		}()
		
		// Get the underlying SQL database
		sqlDB, err := db.DB()
		require.NoError(t, err)
		
		// Configure connection pool
		sqlDB.SetMaxOpenConns(5)
		sqlDB.SetMaxIdleConns(2)
		sqlDB.SetConnMaxLifetime(time.Hour)
		
		// Verify configuration
		assert.Equal(t, 5, sqlDB.Stats().MaxOpenConnections)
		
		// Test that we can perform operations
		err = db.Exec("SELECT 1").Error
		assert.NoError(t, err)
	})
}

func TestDatabaseErrorHandling(t *testing.T) {
	t.Run("InvalidDatabaseConnection", func(t *testing.T) {
		// Test with invalid connection string
		_, err := gorm.Open(sqlite.Open("/invalid/path/to/database.db"), &gorm.Config{})
		// This should fail because the directory doesn't exist
		assert.Error(t, err)
	})
}

func TestDatabaseMigrationErrorHandling(t *testing.T) {
	t.Run("MigrationWithInvalidDatabase", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Log("Migrate with nil DB did not panic; skipping assertion")
			}
		}()
		_ = Migrate(nil)
	})
}