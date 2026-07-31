package database

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/config"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/keweenaw-endurance/backend/internal/uuidutil"
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
	if err := db.AutoMigrate(
		&models.Event{},
		&models.Race{},
		&models.Team{},
		&models.Participant{},
		&models.TimingCheckpoint{},
		&models.TimingRecord{},
		&models.Category{},
		&models.Bib{},
		&models.RFIDTagAssociation{},
		&models.ReaderStation{},
	); err != nil {
		return err
	}
	return migrateTagAssociationsToBibs(db)
}

// migrateTagAssociationsToBibs backfills Bib rows from participants and
// repoints rfid_tag_associations from participant_id to bib_id when upgrading.
func migrateTagAssociationsToBibs(db *gorm.DB) error {
	hasParticipantCol, err := columnExists(db, "rfid_tag_associations", "participant_id")
	if err != nil {
		return err
	}
	if hasParticipantCol {
		if err := backfillAssociationsFromParticipants(db); err != nil {
			return err
		}
	}

	if err := ensureBibsForParticipants(db); err != nil {
		return err
	}

	if hasParticipantCol {
		if err := dropParticipantIDColumn(db); err != nil {
			return err
		}
	}
	return nil
}

func columnExists(db *gorm.DB, table, column string) (bool, error) {
	switch db.Dialector.Name() {
	case "postgres":
		var exists bool
		err := db.Raw(`
			SELECT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = ? AND column_name = ?
			)`, table, column).Scan(&exists).Error
		return exists, err
	case "sqlite":
		rows, err := db.Raw(fmt.Sprintf("PRAGMA table_info(%s)", table)).Rows()
		if err != nil {
			// Table may not exist yet
			if strings.Contains(err.Error(), "no such table") {
				return false, nil
			}
			return false, err
		}
		defer rows.Close()
		for rows.Next() {
			var cid int
			var name, colType string
			var notnull, pk int
			var dflt interface{}
			if err := rows.Scan(&cid, &name, &colType, &notnull, &dflt, &pk); err != nil {
				return false, err
			}
			if name == column {
				return true, nil
			}
		}
		return false, nil
	default:
		return false, nil
	}
}

type assocParticipantRow struct {
	AssocID   string
	EventID   string
	BibNumber string
}

func backfillAssociationsFromParticipants(db *gorm.DB) error {
	var rows []assocParticipantRow
	err := db.Raw(`
		SELECT a.id AS assoc_id, r.event_id AS event_id, p.bib_number AS bib_number
		FROM rfid_tag_associations a
		JOIN participants p ON p.id = a.participant_id
		JOIN races r ON r.id = p.race_id
		WHERE a.participant_id IS NOT NULL
	`).Scan(&rows).Error
	if err != nil {
		return fmt.Errorf("list associations for bib backfill: %w", err)
	}

	for _, row := range rows {
		if strings.TrimSpace(row.BibNumber) == "" {
			continue
		}
		eventID, err := uuid.Parse(row.EventID)
		if err != nil {
			return fmt.Errorf("parse event_id %q: %w", row.EventID, err)
		}
		bib, err := ensureBib(db, uuidutil.NewPublicUUID(eventID), row.BibNumber)
		if err != nil {
			return err
		}
		if err := db.Exec(
			`UPDATE rfid_tag_associations SET bib_id = ? WHERE id = ?`,
			bib.ID, row.AssocID,
		).Error; err != nil {
			return fmt.Errorf("set bib_id on association %s: %w", row.AssocID, err)
		}
	}
	return nil
}

func ensureBibsForParticipants(db *gorm.DB) error {
	type partRow struct {
		EventID   string
		BibNumber string
	}
	var rows []partRow
	err := db.Raw(`
		SELECT DISTINCT r.event_id AS event_id, p.bib_number AS bib_number
		FROM participants p
		JOIN races r ON r.id = p.race_id
		WHERE p.bib_number IS NOT NULL AND TRIM(p.bib_number) != ''
	`).Scan(&rows).Error
	if err != nil {
		return fmt.Errorf("list participants for bib ensure: %w", err)
	}

	for _, row := range rows {
		eventID, err := uuid.Parse(row.EventID)
		if err != nil {
			return fmt.Errorf("parse event_id %q: %w", row.EventID, err)
		}
		if _, err := ensureBib(db, uuidutil.NewPublicUUID(eventID), row.BibNumber); err != nil {
			return err
		}
	}
	return nil
}

func ensureBib(db *gorm.DB, eventID uuidutil.PublicUUID, bibNumber string) (*models.Bib, error) {
	var bib models.Bib
	err := db.Where("event_id = ? AND bib_number = ?", eventID, bibNumber).First(&bib).Error
	if err == nil {
		return &bib, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("lookup bib %s/%s: %w", eventID, bibNumber, err)
	}
	bib = models.Bib{
		EventID:   eventID,
		BibNumber: bibNumber,
	}
	if err := db.Create(&bib).Error; err != nil {
		// Concurrent create: re-fetch
		var existing models.Bib
		if findErr := db.Where("event_id = ? AND bib_number = ?", eventID, bibNumber).First(&existing).Error; findErr == nil {
			return &existing, nil
		}
		return nil, fmt.Errorf("create bib %s/%s: %w", eventID, bibNumber, err)
	}
	return &bib, nil
}

func dropParticipantIDColumn(db *gorm.DB) error {
	switch db.Dialector.Name() {
	case "postgres":
		return db.Exec(`ALTER TABLE rfid_tag_associations DROP COLUMN IF EXISTS participant_id`).Error
	case "sqlite":
		// Fresh AutoMigrate test DBs already have the new shape; orphan column
		// on upgraded SQLite is acceptable. Prefer drop when SQLite supports it.
		if err := db.Exec(`ALTER TABLE rfid_tag_associations DROP COLUMN participant_id`).Error; err != nil {
			log.Printf("sqlite: leaving participant_id column (drop unsupported or failed): %v", err)
		}
		return nil
	default:
		return nil
	}
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
