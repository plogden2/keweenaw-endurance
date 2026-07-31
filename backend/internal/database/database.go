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

// Migrate applies schema in a safe order for bib-associated tags:
// 1) AutoMigrate core tables + bibs (not associations final shape)
// 2) If upgrading from participant_id: add nullable bib_id, backfill, drop participant_id, SET NOT NULL
// 3) Create final associations table (fresh) or ensure indexes (upgrade / existing)
// 4) Ensure Bib rows for all participants with bib numbers
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
		&models.ReaderStation{},
	); err != nil {
		return err
	}

	assocExisted, err := tableExists(db, "rfid_tag_associations")
	if err != nil {
		return err
	}

	if err := upgradeTagAssociationsToBibs(db); err != nil {
		return err
	}

	if !assocExisted {
		if err := db.AutoMigrate(&models.RFIDTagAssociation{}); err != nil {
			return err
		}
	} else {
		// After an in-place upgrade (or on an already-migrated DB), avoid SQLite
		// AutoMigrate table rebuilds that can drop/omit columns. Ensure indexes;
		// on Postgres AutoMigrate is safe for constraints/indexes.
		if err := ensureAssociationIndexes(db); err != nil {
			return err
		}
		if db.Dialector.Name() == "postgres" {
			if err := db.AutoMigrate(&models.RFIDTagAssociation{}); err != nil {
				return err
			}
		}
	}

	return ensureBibsForParticipants(db)
}

// upgradeTagAssociationsToBibs follows database/migrations/06-bib-tag-association.sql:
// add nullable bib_id → backfill → refuse drop if nulls remain → drop participant_id → NOT NULL.
func upgradeTagAssociationsToBibs(db *gorm.DB) error {
	exists, err := tableExists(db, "rfid_tag_associations")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	hasParticipantCol, err := columnExists(db, "rfid_tag_associations", "participant_id")
	if err != nil {
		return err
	}
	if !hasParticipantCol {
		return nil
	}

	run := func(tx *gorm.DB) error {
		hasBibCol, err := columnExists(tx, "rfid_tag_associations", "bib_id")
		if err != nil {
			return err
		}
		if !hasBibCol {
			if err := addNullableBibIDColumn(tx); err != nil {
				return fmt.Errorf("add nullable bib_id: %w", err)
			}
		}

		if err := backfillAssociationsFromParticipants(tx); err != nil {
			return err
		}

		nullCount, err := countNullBibIDs(tx)
		if err != nil {
			return err
		}
		if nullCount > 0 {
			return fmt.Errorf(
				"cannot drop participant_id: %d association(s) still have null bib_id after backfill (empty bib_number or missing participant)",
				nullCount,
			)
		}

		if err := dropParticipantIDColumn(tx); err != nil {
			return err
		}
		if err := setBibIDNotNull(tx); err != nil {
			return err
		}
		return nil
	}

	if db.Dialector.Name() == "postgres" {
		return db.Transaction(run)
	}
	return run(db)
}

func tableExists(db *gorm.DB, table string) (bool, error) {
	switch db.Dialector.Name() {
	case "postgres":
		var exists bool
		err := db.Raw(`
			SELECT EXISTS (
				SELECT 1 FROM information_schema.tables
				WHERE table_schema = 'public' AND table_name = ?
			)`, table).Scan(&exists).Error
		return exists, err
	case "sqlite":
		var name string
		err := db.Raw(`SELECT name FROM sqlite_master WHERE type='table' AND name = ?`, table).Scan(&name).Error
		if err != nil {
			return false, err
		}
		return name == table, nil
	default:
		return db.Migrator().HasTable(table), nil
	}
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

func ensureAssociationIndexes(db *gorm.DB) error {
	if err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_rfid_tag_associations_tag_uid ON rfid_tag_associations(tag_uid)`).Error; err != nil {
		return fmt.Errorf("ensure tag_uid unique index: %w", err)
	}
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_rfid_tag_associations_bib_id ON rfid_tag_associations(bib_id)`).Error; err != nil {
		return fmt.Errorf("ensure bib_id index: %w", err)
	}
	return nil
}

func addNullableBibIDColumn(db *gorm.DB) error {
	switch db.Dialector.Name() {
	case "postgres":
		return db.Exec(`ALTER TABLE rfid_tag_associations ADD COLUMN IF NOT EXISTS bib_id UUID REFERENCES bibs(id)`).Error
	case "sqlite":
		return db.Exec(`ALTER TABLE rfid_tag_associations ADD COLUMN bib_id TEXT`).Error
	default:
		return db.Exec(`ALTER TABLE rfid_tag_associations ADD COLUMN bib_id UUID`).Error
	}
}

func setBibIDNotNull(db *gorm.DB) error {
	switch db.Dialector.Name() {
	case "postgres":
		return db.Exec(`ALTER TABLE rfid_tag_associations ALTER COLUMN bib_id SET NOT NULL`).Error
	case "sqlite":
		// SQLite cannot ALTER COLUMN to NOT NULL in place; final AutoMigrate covers fresh DBs.
		return nil
	default:
		return nil
	}
}

func countNullBibIDs(db *gorm.DB) (int64, error) {
	var count int64
	err := db.Raw(`SELECT COUNT(*) FROM rfid_tag_associations WHERE bib_id IS NULL`).Scan(&count).Error
	return count, err
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

	skippedEmpty := 0
	for _, row := range rows {
		if strings.TrimSpace(row.BibNumber) == "" {
			skippedEmpty++
			log.Printf("bib backfill: skipping association %s — participant has empty bib_number", row.AssocID)
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
	if skippedEmpty > 0 {
		log.Printf("bib backfill: skipped %d association(s) with empty bib_number", skippedEmpty)
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
		if err := db.Exec(`ALTER TABLE rfid_tag_associations DROP COLUMN participant_id`).Error; err != nil {
			return fmt.Errorf("sqlite drop participant_id: %w", err)
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
