package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/uuidutil"
	"gorm.io/gorm"
)

// Event represents a race event
type Event struct {
	ID          uuidutil.PublicUUID `gorm:"type:uuid;primary_key" json:"id"`
	Name        string    `gorm:"type:varchar(255);not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	EventDate   time.Time `gorm:"type:date;not null" json:"event_date"`
	Location    string    `gorm:"type:varchar(500)" json:"location"`
	WebsiteURL  string    `gorm:"type:varchar(500)" json:"website_url"`
	LogoURL     string    `gorm:"type:varchar(500)" json:"logo_url"`
	Status      string    `gorm:"type:varchar(50);not null;check:status IN ('upcoming','active','completed','cancelled')" json:"status"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	
	// Relationships
	Races          []Race          `gorm:"foreignKey:EventID" json:"races,omitempty"`
	ReaderStations []ReaderStation `gorm:"foreignKey:EventID" json:"reader_stations,omitempty"`
}

// Race represents a race within an event
type Race struct {
	ID             uuidutil.PublicUUID `gorm:"type:uuid;primary_key" json:"id"`
	EventID        uuidutil.PublicUUID `gorm:"type:uuid;not null" json:"event_id"`
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
	ID         uuidutil.PublicUUID  `gorm:"type:uuid;primary_key" json:"id"`
	RaceID     uuidutil.PublicUUID  `gorm:"type:uuid;not null" json:"race_id"`
	CategoryID *uuidutil.PublicUUID `gorm:"type:uuid" json:"category_id,omitempty"`
	BibNumber  string               `gorm:"type:varchar(20);not null" json:"bib_number"`
	FirstName  string               `gorm:"type:varchar(100);not null" json:"first_name"`
	LastName   string               `gorm:"type:varchar(100);not null" json:"last_name"`
	Gender     string               `gorm:"type:varchar(10)" json:"gender"`
	Age        int                  `gorm:"type:integer" json:"age"`
	Location   string               `gorm:"type:varchar(500)" json:"location"`
	RFIDTagUID string               `gorm:"column:rfid_tag_uid;type:varchar(100)" json:"rfid_tag_uid"`
	Status     string               `gorm:"type:varchar(50);not null;check:status IN ('registered','started','finished','dnf','dns')" json:"status"`
	CreatedAt  time.Time            `gorm:"autoCreateTime" json:"created_at"`

	// TagUIDs is populated from rfid_tag_associations for list/detail JSON (not a column).
	TagUIDs []string `gorm:"-" json:"tag_uids,omitempty"`

	// Relationships
	Race              Race                 `gorm:"foreignKey:RaceID" json:"race,omitempty"`
	Category          *Category            `gorm:"foreignKey:CategoryID" json:"category,omitempty"`
	TimingRecords     []TimingRecord       `gorm:"foreignKey:ParticipantID" json:"timing_records,omitempty"`
	TagAssociations   []RFIDTagAssociation `gorm:"foreignKey:ParticipantID" json:"tag_associations,omitempty"`
}

// TimingCheckpoint represents a timing checkpoint in a race
type TimingCheckpoint struct {
	ID                uuidutil.PublicUUID `gorm:"type:uuid;primary_key" json:"id"`
	RaceID            uuidutil.PublicUUID `gorm:"type:uuid;not null" json:"race_id"`
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
	ID             uuidutil.PublicUUID  `gorm:"type:uuid;primary_key" json:"id"`
	ParticipantID  uuidutil.PublicUUID  `gorm:"type:uuid;not null" json:"participant_id"`
	CheckpointID   uuidutil.PublicUUID  `gorm:"type:uuid;not null" json:"checkpoint_id"`
	Timestamp      time.Time            `gorm:"type:timestamp;not null" json:"timestamp"`
	LocalTimestamp time.Time            `gorm:"type:timestamp;not null" json:"local_timestamp"`
	DeviceID       string               `gorm:"type:varchar(100)" json:"device_id"`
	SyncStatus     string               `gorm:"type:varchar(50);default:'synced';check:sync_status IN ('synced','pending_sync','failed_sync')" json:"sync_status"`
	RecordType     string               `gorm:"type:varchar(50);not null;default:'rfid_lap';check:record_type IN ('rfid_lap','karaoke_bonus','checkpoint_pass')" json:"record_type"`
	SourceLapID    *uuidutil.PublicUUID `gorm:"type:uuid" json:"source_lap_id,omitempty"`
	StationID      *uuidutil.PublicUUID `gorm:"type:uuid" json:"station_id,omitempty"`
	CreatedAt      time.Time            `gorm:"autoCreateTime" json:"created_at"`

	// Relationships
	Participant Participant       `gorm:"foreignKey:ParticipantID" json:"participant,omitempty"`
	Checkpoint  TimingCheckpoint  `gorm:"foreignKey:CheckpointID" json:"checkpoint,omitempty"`
	SourceLap   *TimingRecord     `gorm:"foreignKey:SourceLapID" json:"source_lap,omitempty"`
	Station     *ReaderStation    `gorm:"foreignKey:StationID" json:"station,omitempty"`
}

// RFIDTagAssociation links an RFID tag UID to a participant
type RFIDTagAssociation struct {
	ID            uuidutil.PublicUUID `gorm:"type:uuid;primary_key" json:"id"`
	ParticipantID uuidutil.PublicUUID `gorm:"type:uuid;not null" json:"participant_id"`
	TagUID        string              `gorm:"type:varchar(100);not null;uniqueIndex" json:"tag_uid"`
	CreatedAt     time.Time           `gorm:"autoCreateTime" json:"created_at"`
	Active        bool                `gorm:"type:boolean;not null;default:true" json:"active"`

	// Relationships
	Participant Participant `gorm:"foreignKey:ParticipantID" json:"participant,omitempty"`
}

func (RFIDTagAssociation) TableName() string { return "rfid_tag_associations" }

// ReaderStation represents a logical RFID reader configuration for an event
type ReaderStation struct {
	ID            uuidutil.PublicUUID  `gorm:"type:uuid;primary_key" json:"id"`
	EventID       uuidutil.PublicUUID  `gorm:"type:uuid;not null;index" json:"event_id"`
	Mode          string               `gorm:"type:varchar(50);not null;default:'finish';check:mode IN ('finish','checkpoint')" json:"mode"`
	CheckpointID  *uuidutil.PublicUUID `gorm:"type:uuid" json:"checkpoint_id,omitempty"`
	SequenceOrder int                  `gorm:"type:integer;not null;default:0" json:"sequence_order"`
	Name          string               `gorm:"type:varchar(255);not null" json:"name"`
	DeviceID      string               `gorm:"column:device_id;type:varchar(100)" json:"device_id"`
	LastSeenAt    *time.Time           `gorm:"type:timestamp" json:"last_seen_at,omitempty"`
	CreatedAt     time.Time            `gorm:"autoCreateTime" json:"created_at"`

	// Relationships
	Event      Event             `gorm:"foreignKey:EventID" json:"event,omitempty"`
	Checkpoint *TimingCheckpoint `gorm:"foreignKey:CheckpointID" json:"checkpoint,omitempty"`
}

func (ReaderStation) TableName() string { return "reader_stations" }

// Category represents a participant category for race results
type Category struct {
	ID            uuidutil.PublicUUID `gorm:"type:uuid;primary_key" json:"id"`
	RaceID        uuidutil.PublicUUID `gorm:"type:uuid;not null" json:"race_id"`
	Name          string    `gorm:"type:varchar(255);not null" json:"name"`
	CategoryType  string    `gorm:"type:varchar(50);not null;check:category_type IN ('overall','male','female','age_group','custom')" json:"category_type"`
	AgeMin        int       `gorm:"type:integer" json:"age_min"`
	AgeMax        int       `gorm:"type:integer" json:"age_max"`
	GenderFilter  string    `gorm:"type:varchar(10)" json:"gender_filter"`
	DisplayOrder  int       `gorm:"type:integer;default:0" json:"display_order"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
	
	// Relationships
	Race         Race          `gorm:"foreignKey:RaceID" json:"race,omitempty"`
	Participants []Participant `gorm:"foreignKey:CategoryID" json:"participants,omitempty"`
}

// BeforeCreate hooks for UUID generation
func (e *Event) BeforeCreate(tx *gorm.DB) error {
	if e.ID.IsZero() {
		e.ID = uuidutil.PublicUUID(uuid.New())
	}
	return nil
}

func (r *Race) BeforeCreate(tx *gorm.DB) error {
	if r.ID.IsZero() {
		r.ID = uuidutil.PublicUUID(uuid.New())
	}
	return nil
}

func (p *Participant) BeforeCreate(tx *gorm.DB) error {
	if p.ID.IsZero() {
		p.ID = uuidutil.PublicUUID(uuid.New())
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
	if tc.ID.IsZero() {
		tc.ID = uuidutil.PublicUUID(uuid.New())
	}
	return nil
}

func (tr *TimingRecord) BeforeCreate(tx *gorm.DB) error {
	if tr.ID.IsZero() {
		tr.ID = uuidutil.PublicUUID(uuid.New())
	}
	if tr.RecordType == "" {
		tr.RecordType = "rfid_lap"
	}
	return nil
}

func (c *Category) BeforeCreate(tx *gorm.DB) error {
	if c.ID.IsZero() {
		c.ID = uuidutil.PublicUUID(uuid.New())
	}
	return nil
}

func (a *RFIDTagAssociation) BeforeCreate(tx *gorm.DB) error {
	if a.ID.IsZero() {
		a.ID = uuidutil.PublicUUID(uuid.New())
	}
	return nil
}

func (s *ReaderStation) BeforeCreate(tx *gorm.DB) error {
	if s.ID.IsZero() {
		s.ID = uuidutil.PublicUUID(uuid.New())
	}
	if s.Mode == "" {
		s.Mode = "finish"
	}
	return nil
}