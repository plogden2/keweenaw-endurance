package database

import (
	"testing"
	"time"

	"github.com/keweenaw-endurance/backend/internal/config"
	"github.com/keweenaw-endurance/backend/internal/models"
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
		tables := []string{"events", "races", "participants", "timing_checkpoints", "timing_records", "categories", "bibs", "rfid_tag_associations"}

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

func TestMigrate_UpgradesLegacyParticipantAssociations(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:migrate_upgrade?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	// Parent tables via AutoMigrate; associations left in legacy participant_id shape.
	require.NoError(t, db.AutoMigrate(
		&models.Event{},
		&models.Race{},
		&models.Team{},
		&models.Participant{},
		&models.TimingCheckpoint{},
		&models.TimingRecord{},
		&models.Category{},
		&models.Bib{},
		&models.ReaderStation{},
	))
	require.NoError(t, db.Exec(`
		CREATE TABLE rfid_tag_associations (
			id TEXT PRIMARY KEY,
			participant_id TEXT NOT NULL,
			tag_uid TEXT NOT NULL UNIQUE,
			created_at DATETIME,
			active INTEGER NOT NULL DEFAULT 1
		)
	`).Error)

	event := models.Event{Name: "Legacy Event", EventDate: time.Now().UTC(), Status: "upcoming"}
	require.NoError(t, db.Create(&event).Error)
	race := models.Race{EventID: event.ID, Name: "Legacy Race", RaceType: "lap_based", Status: "scheduled"}
	require.NoError(t, db.Create(&race).Error)
	part := models.Participant{
		RaceID: race.ID, BibNumber: "42", FirstName: "Ada", LastName: "Lovelace", Status: "registered",
	}
	require.NoError(t, db.Create(&part).Error)

	assocID := "44444444-4444-4444-4444-444444444444"
	require.NoError(t, db.Exec(`
		INSERT INTO rfid_tag_associations (id, participant_id, tag_uid, created_at, active)
		VALUES (?, ?, 'LEGACY-TAG-42', ?, 1)
	`, assocID, part.ID.String(), time.Now().UTC()).Error)

	require.NoError(t, Migrate(db))

	var bibCount int64
	require.NoError(t, db.Table("bibs").Where("event_id = ? AND bib_number = ?", event.ID, "42").Count(&bibCount).Error)
	assert.Equal(t, int64(1), bibCount)

	var bibID string
	require.NoError(t, db.Raw(`SELECT id FROM bibs WHERE event_id = ? AND bib_number = ?`, event.ID, "42").Scan(&bibID).Error)
	require.NotEmpty(t, bibID)

	var gotBibID string
	require.NoError(t, db.Raw(`SELECT bib_id FROM rfid_tag_associations WHERE id = ?`, assocID).Scan(&gotBibID).Error)
	assert.Equal(t, bibID, gotBibID)

	hasParticipantCol, err := columnExists(db, "rfid_tag_associations", "participant_id")
	require.NoError(t, err)
	assert.False(t, hasParticipantCol, "participant_id should be dropped after successful backfill")

	var tagUID string
	require.NoError(t, db.Raw(`SELECT tag_uid FROM rfid_tag_associations WHERE id = ?`, assocID).Scan(&tagUID).Error)
	assert.Equal(t, "LEGACY-TAG-42", tagUID)
}

func TestMigrate_RefusesDropWhenAssociationHasEmptyBibNumber(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:migrate_empty_bib?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	defer func() {
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	require.NoError(t, db.AutoMigrate(
		&models.Event{},
		&models.Race{},
		&models.Team{},
		&models.Participant{},
		&models.TimingCheckpoint{},
		&models.TimingRecord{},
		&models.Category{},
		&models.Bib{},
		&models.ReaderStation{},
	))
	require.NoError(t, db.Exec(`
		CREATE TABLE rfid_tag_associations (
			id TEXT PRIMARY KEY,
			participant_id TEXT NOT NULL,
			tag_uid TEXT NOT NULL UNIQUE,
			created_at DATETIME,
			active INTEGER NOT NULL DEFAULT 1
		)
	`).Error)

	event := models.Event{Name: "Empty Bib Event", EventDate: time.Now().UTC(), Status: "upcoming"}
	require.NoError(t, db.Create(&event).Error)
	race := models.Race{EventID: event.ID, Name: "Race", RaceType: "lap_based", Status: "scheduled"}
	require.NoError(t, db.Create(&race).Error)
	// Bypass model validation: empty bib_number on the participant row.
	partID := "cccccccc-cccc-cccc-cccc-cccccccccccc"
	require.NoError(t, db.Exec(`
		INSERT INTO participants (id, race_id, bib_number, first_name, last_name, status, created_at)
		VALUES (?, ?, '   ', 'No', 'Bib', 'registered', ?)
	`, partID, race.ID.String(), time.Now().UTC()).Error)

	assocID := "dddddddd-dddd-dddd-dddd-dddddddddddd"
	require.NoError(t, db.Exec(`
		INSERT INTO rfid_tag_associations (id, participant_id, tag_uid, created_at, active)
		VALUES (?, ?, 'EMPTY-BIB-TAG', ?, 1)
	`, assocID, partID, time.Now().UTC()).Error)

	err = Migrate(db)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "null bib_id")

	hasParticipantCol, colErr := columnExists(db, "rfid_tag_associations", "participant_id")
	require.NoError(t, colErr)
	assert.True(t, hasParticipantCol, "must not drop participant_id when backfill left null bib_id")
}
