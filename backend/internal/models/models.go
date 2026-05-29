package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Event represents a race event
type Event struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	Name        string    `gorm:"type:varchar(255);not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	EventDate   time.Time `gorm:"type:date;not null" json:"event_date"`
	Location    string    `gorm:"type:varchar(500)" json:"location"`
	WebsiteURL  string    `gorm:"type:varchar(500)" json:"website_url"`
	Status      string    `gorm:"type:varchar(50);not null;check:status IN ('upcoming','active','completed','cancelled')" json:"status"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	
	// Relationships
	Races []Race `gorm:"foreignKey:EventID" json:"races,omitempty"`
}

// Race represents a race within an event
type Race struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	EventID        uuid.UUID `gorm:"type:uuid;not null" json:"event_id"`
	Name           string    `gorm:"type:varchar(255);not null" json:"name"`
	RaceType       string    `gorm:"type:varchar(50);not null;check:race_type IN ('time_based','lap_based')" json:"race_type"`
	DistanceKm     float64   `gorm:"type:decimal(10,2)" json:"distance_km"`
	DurationMinutes int       `gorm:"type:integer" json:"duration_minutes"`
	StartTime      time.Time `gorm:"type:timestamp" json:"start_time"`
	Status         string    `gorm:"type:varchar(50);not null;check:status IN ('scheduled','active','finished','cancelled')" json:"status"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
	
	// Relationships
	Event        Event           `gorm:"foreignKey:EventID" json:"event,omitempty"`
	Participants []Participant   `gorm:"foreignKey:RaceID" json:"participants,omitempty"`
	Checkpoints  []TimingCheckpoint `gorm:"foreignKey:RaceID" json:"checkpoints,omitempty"`
}

// Participant represents a race participant
type Participant struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	RaceID     uuid.UUID `gorm:"type:uuid;not null" json:"race_id"`
	BibNumber  string    `gorm:"type:varchar(20);not null" json:"bib_number"`
	FirstName  string    `gorm:"type:varchar(100);not null" json:"first_name"`
	LastName   string    `gorm:"type:varchar(100);not null" json:"last_name"`
	Gender     string    `gorm:"type:varchar(10)" json:"gender"`
	Age        int       `gorm:"type:integer" json:"age"`
	RFIDTagUID string    `gorm:"type:varchar(100)" json:"rfid_tag_uid"`
	Status     string    `gorm:"type:varchar(50);not null;check:status IN ('registered','started','finished','dnf','dns')" json:"status"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
	
	// Relationships
	Race        Race          `gorm:"foreignKey:RaceID" json:"race,omitempty"`
	TimingRecords []TimingRecord `gorm:"foreignKey:ParticipantID" json:"timing_records,omitempty"`
}

// TimingCheckpoint represents a timing checkpoint in a race
type TimingCheckpoint struct {
	ID                uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	RaceID            uuid.UUID `gorm:"type:uuid;not null" json:"race_id"`
	Name              string    `gorm:"type:varchar(255);not null" json:"name"`
	CheckpointType    string    `gorm:"type:varchar(50);not null;check:checkpoint_type IN ('start','finish','intermediate')" json:"checkpoint_type"`
	DistanceFromStartKm float64 `gorm:"type:decimal(10,2)" json:"distance_from_start_km"`
	LocationDescription string  `gorm:"type:varchar(500)" json:"location_description"`
	IsActive          bool      `gorm:"type:boolean;default:true" json:"is_active"`
	CreatedAt         time.Time `gorm:"autoCreateTime" json:"created_at"`
	
	// Relationships
	Race         Race           `gorm:"foreignKey:RaceID" json:"race,omitempty"`
	TimingRecords []TimingRecord `gorm:"foreignKey:CheckpointID" json:"timing_records,omitempty"`
}

// TimingRecord represents a timing record for a participant at a checkpoint
type TimingRecord struct {
	ID              uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	ParticipantID   uuid.UUID `gorm:"type:uuid;not null" json:"participant_id"`
	CheckpointID    uuid.UUID `gorm:"type:uuid;not null" json:"checkpoint_id"`
	Timestamp       time.Time `gorm:"type:timestamp;not null" json:"timestamp"`
	LocalTimestamp  time.Time `gorm:"type:timestamp;not null" json:"local_timestamp"`
	DeviceID        string    `gorm:"type:varchar(100)" json:"device_id"`
	SyncStatus      string    `gorm:"type:varchar(50);default:'synced';check:sync_status IN ('synced','pending_sync','failed_sync')" json:"sync_status"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`
	
	// Relationships
	Participant Participant      `gorm:"foreignKey:ParticipantID" json:"participant,omitempty"`
	Checkpoint  TimingCheckpoint `gorm:"foreignKey:CheckpointID" json:"checkpoint,omitempty"`
}

// Category represents a participant category for race results
type Category struct {
	ID            uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	RaceID        uuid.UUID `gorm:"type:uuid;not null" json:"race_id"`
	Name          string    `gorm:"type:varchar(255);not null" json:"name"`
	CategoryType  string    `gorm:"type:varchar(50);not null;check:category_type IN ('overall','male','female','age_group','custom')" json:"category_type"`
	AgeMin        int       `gorm:"type:integer" json:"age_min"`
	AgeMax        int       `gorm:"type:integer" json:"age_max"`
	GenderFilter  string    `gorm:"type:varchar(10)" json:"gender_filter"`
	DisplayOrder  int       `gorm:"type:integer;default:0" json:"display_order"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
	
	// Relationships
	Race Race `gorm:"foreignKey:RaceID" json:"race,omitempty"`
}

// BeforeCreate hooks for UUID generation
func (e *Event) BeforeCreate(tx *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return nil
}

func (r *Race) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

func (p *Participant) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return p.validate()
}

func (p *Participant) BeforeSave(tx *gorm.DB) error {
	return p.validate()
}

func (p *Participant) validate() error {
	if p.Gender != "" {
		switch p.Gender {
		case "male", "female", "other":
		default:
			return errors.New("invalid gender")
		}
	}
	return nil
}

func (tc *TimingCheckpoint) BeforeCreate(tx *gorm.DB) error {
	if tc.ID == uuid.Nil {
		tc.ID = uuid.New()
	}
	return nil
}

func (tr *TimingRecord) BeforeCreate(tx *gorm.DB) error {
	if tr.ID == uuid.Nil {
		tr.ID = uuid.New()
	}
	return nil
}

func (c *Category) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}