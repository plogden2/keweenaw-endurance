package services

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/keweenaw-endurance/backend/internal/uuidutil"
	"gorm.io/gorm"
)

const maxBulkBibSpan = 500

var (
	ErrBibNotFound     = errors.New("bib not found")
	ErrInvalidBibInput = errors.New("invalid bib input")
)

// BibListItem is an event bib with tag inventory and optional assigned racer.
type BibListItem struct {
	ID              uuidutil.PublicUUID  `json:"id"`
	BibNumber       string               `json:"bib_number"`
	// LogicalUUID is the full bib UUID written to chips (PublicUUID JSON is short).
	LogicalUUID     string               `json:"logical_uuid"`
	TagCount        int                  `json:"tag_count"`
	TagUIDs         []string             `json:"tag_uids,omitempty"`
	ParticipantID   *uuidutil.PublicUUID `json:"participant_id,omitempty"`
	ParticipantName string               `json:"participant_name,omitempty"`
	RaceID          *uuidutil.PublicUUID `json:"race_id,omitempty"`
}

// BibService manages event-scoped bibs.
type BibService struct {
	db *gorm.DB
}

func NewBibService(db *gorm.DB) *BibService {
	return &BibService{db: db}
}

// EnsureBib finds or creates the bib for (eventID, bibNumber).
func (s *BibService) EnsureBib(eventID uuid.UUID, bibNumber string) (*models.Bib, error) {
	bibNumber = strings.TrimSpace(bibNumber)
	if bibNumber == "" {
		return nil, fmt.Errorf("%w: bib_number is required", ErrInvalidBibInput)
	}
	if eventID == uuid.Nil {
		return nil, fmt.Errorf("%w: event_id is required", ErrInvalidBibInput)
	}

	eventPub := uuidutil.NewPublicUUID(eventID)
	var bib models.Bib
	err := s.db.Where("event_id = ? AND bib_number = ?", eventPub, bibNumber).First(&bib).Error
	if err == nil {
		return &bib, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	bib = models.Bib{EventID: eventPub, BibNumber: bibNumber}
	if err := s.db.Create(&bib).Error; err != nil {
		var existing models.Bib
		if findErr := s.db.Where("event_id = ? AND bib_number = ?", eventPub, bibNumber).First(&existing).Error; findErr == nil {
			return &existing, nil
		}
		return nil, err
	}
	return &bib, nil
}

// BulkCreateBibs ensures bibs for every integer number in [from, to] inclusive.
func (s *BibService) BulkCreateBibs(eventID uuid.UUID, from, to int) ([]models.Bib, error) {
	if eventID == uuid.Nil {
		return nil, fmt.Errorf("%w: event_id is required", ErrInvalidBibInput)
	}
	if from < 0 || to < 0 {
		return nil, fmt.Errorf("%w: range must be non-negative", ErrInvalidBibInput)
	}
	if from > to {
		return nil, fmt.Errorf("%w: from must be <= to", ErrInvalidBibInput)
	}
	span := to - from + 1
	if span > maxBulkBibSpan {
		return nil, fmt.Errorf("%w: range span must be <= %d", ErrInvalidBibInput, maxBulkBibSpan)
	}

	out := make([]models.Bib, 0, span)
	for n := from; n <= to; n++ {
		bib, err := s.EnsureBib(eventID, strconv.Itoa(n))
		if err != nil {
			return nil, err
		}
		out = append(out, *bib)
	}
	return out, nil
}

// ListBibs returns all bibs for an event with tag counts and optional participant assignment.
func (s *BibService) ListBibs(eventID uuid.UUID) ([]BibListItem, error) {
	if eventID == uuid.Nil {
		return nil, fmt.Errorf("%w: event_id is required", ErrInvalidBibInput)
	}

	var bibs []models.Bib
	if err := s.db.Where("event_id = ?", eventID).Order("bib_number ASC").Find(&bibs).Error; err != nil {
		return nil, err
	}
	if len(bibs) == 0 {
		return []BibListItem{}, nil
	}

	bibIDs := make([]uuidutil.PublicUUID, len(bibs))
	for i, b := range bibs {
		bibIDs[i] = b.ID
	}

	var tags []models.RFIDTagAssociation
	if err := s.db.Where("bib_id IN ? AND active = ?", bibIDs, true).
		Order("created_at ASC").
		Find(&tags).Error; err != nil {
		return nil, err
	}
	tagsByBib := map[uuid.UUID][]string{}
	for _, t := range tags {
		id := t.BibID.UUID()
		tagsByBib[id] = append(tagsByBib[id], t.TagUID)
	}

	type assignedRow struct {
		BibNumber   string
		ID          uuidutil.PublicUUID
		FirstName   string
		LastName    string
		RaceID      uuidutil.PublicUUID
	}
	var assigned []assignedRow
	if err := s.db.Table("participants").
		Select("participants.bib_number, participants.id, participants.first_name, participants.last_name, participants.race_id").
		Joins("JOIN races ON races.id = participants.race_id").
		Where("races.event_id = ?", eventID).
		Find(&assigned).Error; err != nil {
		return nil, err
	}
	byBibNumber := map[string]assignedRow{}
	for _, row := range assigned {
		byBibNumber[row.BibNumber] = row
	}

	items := make([]BibListItem, 0, len(bibs))
	for _, b := range bibs {
		uids := tagsByBib[b.ID.UUID()]
		if uids == nil {
			uids = []string{}
		}
		item := BibListItem{
			ID:          b.ID,
			BibNumber:   b.BibNumber,
			LogicalUUID: strings.ToLower(b.ID.UUID().String()),
			TagCount:    len(uids),
			TagUIDs:     uids,
		}
		if p, ok := byBibNumber[b.BibNumber]; ok {
			pid := p.ID
			rid := p.RaceID
			item.ParticipantID = &pid
			item.ParticipantName = strings.TrimSpace(p.FirstName + " " + p.LastName)
			item.RaceID = &rid
		}
		items = append(items, item)
	}
	return items, nil
}

// GetBib returns a bib scoped to the event.
func (s *BibService) GetBib(eventID, bibID uuid.UUID) (*models.Bib, error) {
	if eventID == uuid.Nil || bibID == uuid.Nil {
		return nil, fmt.Errorf("%w: event_id and bib_id are required", ErrInvalidBibInput)
	}
	var bib models.Bib
	err := s.db.Where("id = ? AND event_id = ?", bibID, eventID).First(&bib).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBibNotFound
		}
		return nil, err
	}
	return &bib, nil
}

// ListBibTags returns active RFID tag associations for a bib.
func (s *BibService) ListBibTags(bibID uuid.UUID) ([]models.RFIDTagAssociation, error) {
	if bibID == uuid.Nil {
		return nil, fmt.Errorf("%w: bib_id is required", ErrInvalidBibInput)
	}
	var tags []models.RFIDTagAssociation
	if err := s.db.Where("bib_id = ? AND active = ?", bibID, true).
		Order("created_at ASC").
		Find(&tags).Error; err != nil {
		return nil, err
	}
	return tags, nil
}
