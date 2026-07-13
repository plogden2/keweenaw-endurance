package services

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/keweenaw-endurance/backend/internal/uuidutil"
	"gorm.io/gorm"
)

var (
	ErrLiveCSVNotFound = errors.New("live csv snapshot not found")
	ErrInvalidCSV      = errors.New("invalid csv")
)

// LiveSnapshotStatus describes the on-disk live CSV for an event.
type LiveSnapshotStatus struct {
	Path      string    `json:"path"`
	Exists    bool      `json:"exists"`
	UpdatedAt time.Time `json:"updated_at"`
	SizeBytes int64     `json:"size_bytes"`
}

// CSVImportSummary is returned after a replace-semantics import.
type CSVImportSummary struct {
	EventID         string    `json:"event_id"`
	EventName       string    `json:"event_name"`
	Races           int       `json:"races"`
	Racers          int       `json:"racers"`
	TagAssociations int       `json:"tag_associations"`
	TimingRecords   int       `json:"timing_records"`
	Categories      int       `json:"categories"`
	Checkpoints     int       `json:"checkpoints"`
	ImportedAt      time.Time `json:"imported_at"`
}

// CSVExportService maintains live CSV snapshots and import/restore.
type CSVExportService struct {
	db      *gorm.DB
	dataDir string
	mu      sync.Mutex
}

func NewCSVExportService(db *gorm.DB, dataDir string) *CSVExportService {
	if dataDir == "" {
		dataDir = "data"
	}
	return &CSVExportService{db: db, dataDir: dataDir}
}

func (s *CSVExportService) LiveSnapshotPath(eventID uuid.UUID) (string, error) {
	if eventID == uuid.Nil {
		return "", fmt.Errorf("%w: event id required", ErrInvalidCSV)
	}
	return filepath.Join(s.dataDir, "events", eventID.String(), "live-snapshot.csv"), nil
}

func (s *CSVExportService) LiveSnapshotStatus(eventID uuid.UUID) (*LiveSnapshotStatus, error) {
	path, err := s.LiveSnapshotPath(eventID)
	if err != nil {
		return nil, err
	}
	st := &LiveSnapshotStatus{Path: path}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return st, nil
		}
		return nil, err
	}
	st.Exists = true
	st.UpdatedAt = info.ModTime().UTC()
	st.SizeBytes = info.Size()
	return st, nil
}

func (s *CSVExportService) ReadLiveSnapshot(eventID uuid.UUID) ([]byte, time.Time, error) {
	path, err := s.LiveSnapshotPath(eventID)
	if err != nil {
		return nil, time.Time{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Build on demand if missing so GET always works after seed/load.
			if _, werr := s.WriteLiveSnapshot(eventID); werr != nil {
				return nil, time.Time{}, ErrLiveCSVNotFound
			}
			info, err = os.Stat(path)
			if err != nil {
				return nil, time.Time{}, err
			}
		} else {
			return nil, time.Time{}, err
		}
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, time.Time{}, err
	}
	return body, info.ModTime().UTC(), nil
}

// WriteLiveSnapshot rewrites the live CSV for the event and returns the path.
func (s *CSVExportService) WriteLiveSnapshot(eventID uuid.UUID) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	body, err := s.BuildCSV(eventID)
	if err != nil {
		return "", err
	}
	path, err := s.LiveSnapshotPath(eventID)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, body, 0o644); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	return path, nil
}

// RefreshEvent is a convenience hook for mutation sites.
func (s *CSVExportService) RefreshEvent(eventID uuid.UUID) {
	if s == nil || eventID == uuid.Nil {
		return
	}
	_, _ = s.WriteLiveSnapshot(eventID)
}

// BuildCSV renders the multi-section CSV for an event (does not write disk).
func (s *CSVExportService) BuildCSV(eventID uuid.UUID) ([]byte, error) {
	var event models.Event
	if err := s.db.First(&event, "id = ?", eventID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEventNotFound
		}
		return nil, err
	}

	var races []models.Race
	if err := s.db.Where("event_id = ?", eventID).Order("created_at ASC").Find(&races).Error; err != nil {
		return nil, err
	}
	raceIDs := make([]uuidutil.PublicUUID, 0, len(races))
	for _, r := range races {
		raceIDs = append(raceIDs, r.ID)
	}

	var categories []models.Category
	var checkpoints []models.TimingCheckpoint
	var participants []models.Participant
	if len(raceIDs) > 0 {
		if err := s.db.Where("race_id IN ?", raceIDs).Order("display_order ASC, created_at ASC").Find(&categories).Error; err != nil {
			return nil, err
		}
		if err := s.db.Where("race_id IN ?", raceIDs).Order("created_at ASC").Find(&checkpoints).Error; err != nil {
			return nil, err
		}
		if err := s.db.Where("race_id IN ?", raceIDs).Order("bib_number ASC").Find(&participants).Error; err != nil {
			return nil, err
		}
	}

	partIDs := make([]uuidutil.PublicUUID, 0, len(participants))
	for _, p := range participants {
		partIDs = append(partIDs, p.ID)
	}

	var tags []models.RFIDTagAssociation
	var timing []models.TimingRecord
	if len(partIDs) > 0 {
		if err := s.db.Where("participant_id IN ?", partIDs).Order("created_at ASC").Find(&tags).Error; err != nil {
			return nil, err
		}
		if err := s.db.Where("participant_id IN ?", partIDs).Order("timestamp ASC").Find(&timing).Error; err != nil {
			return nil, err
		}
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	writeSection := func(name string, header []string, rows [][]string) error {
		if err := w.Write([]string{"#SECTION", name}); err != nil {
			return err
		}
		if err := w.Write(header); err != nil {
			return err
		}
		for _, row := range rows {
			if err := w.Write(row); err != nil {
				return err
			}
		}
		return nil
	}

	if err := writeSection("event",
		[]string{"id", "name", "event_date", "location", "status"},
		[][]string{{
			event.ID.String(),
			event.Name,
			event.EventDate.Format("2006-01-02"),
			event.Location,
			event.Status,
		}},
	); err != nil {
		return nil, err
	}

	raceRows := make([][]string, 0, len(races))
	for _, r := range races {
		start := ""
		if !r.StartTime.IsZero() {
			start = r.StartTime.Format(time.RFC3339)
		}
		raceRows = append(raceRows, []string{
			r.ID.String(),
			r.EventID.String(),
			r.Name,
			r.RaceType,
			strconv.Itoa(r.DurationMinutes),
			start,
			r.Status,
		})
	}
	if err := writeSection("races",
		[]string{"id", "event_id", "name", "race_type", "duration_minutes", "start_time", "status"},
		raceRows,
	); err != nil {
		return nil, err
	}

	catRows := make([][]string, 0, len(categories))
	for _, c := range categories {
		catRows = append(catRows, []string{
			c.ID.String(),
			c.RaceID.String(),
			c.Name,
			c.CategoryType,
			c.GenderFilter,
			strconv.Itoa(c.DisplayOrder),
		})
	}
	if err := writeSection("categories",
		[]string{"id", "race_id", "name", "category_type", "gender_filter", "display_order"},
		catRows,
	); err != nil {
		return nil, err
	}

	partRows := make([][]string, 0, len(participants))
	for _, p := range participants {
		catID := ""
		if p.CategoryID != nil {
			catID = p.CategoryID.String()
		}
		partRows = append(partRows, []string{
			p.ID.String(),
			p.RaceID.String(),
			p.BibNumber,
			p.FirstName,
			p.LastName,
			p.Gender,
			p.Status,
			catID,
		})
	}
	if err := writeSection("participants",
		[]string{"id", "race_id", "bib_number", "first_name", "last_name", "gender", "status", "category_id"},
		partRows,
	); err != nil {
		return nil, err
	}

	tagRows := make([][]string, 0, len(tags))
	for _, tg := range tags {
		tagRows = append(tagRows, []string{
			tg.ID.String(),
			tg.ParticipantID.String(),
			tg.TagUID,
			tg.CreatedAt.Format(time.RFC3339),
		})
	}
	if err := writeSection("tags",
		[]string{"id", "participant_id", "tag_uid", "created_at"},
		tagRows,
	); err != nil {
		return nil, err
	}

	cpRows := make([][]string, 0, len(checkpoints))
	for _, cp := range checkpoints {
		cpRows = append(cpRows, []string{
			cp.ID.String(),
			cp.RaceID.String(),
			cp.Name,
			cp.CheckpointType,
			formatFloat(cp.DistanceFromStartKm),
			strconv.FormatBool(cp.IsActive),
		})
	}
	if err := writeSection("checkpoints",
		[]string{"id", "race_id", "name", "checkpoint_type", "distance_from_start_km", "is_active"},
		cpRows,
	); err != nil {
		return nil, err
	}

	trRows := make([][]string, 0, len(timing))
	for _, tr := range timing {
		source := ""
		if tr.SourceLapID != nil {
			source = tr.SourceLapID.String()
		}
		station := ""
		if tr.StationID != nil {
			station = tr.StationID.String()
		}
		trRows = append(trRows, []string{
			tr.ID.String(),
			tr.ParticipantID.String(),
			tr.CheckpointID.String(),
			tr.Timestamp.Format(time.RFC3339),
			tr.LocalTimestamp.Format(time.RFC3339),
			tr.DeviceID,
			tr.SyncStatus,
			tr.RecordType,
			source,
			station,
		})
	}
	if err := writeSection("timing_records",
		[]string{"id", "participant_id", "checkpoint_id", "timestamp", "local_timestamp", "device_id", "sync_status", "record_type", "source_lap_id", "station_id"},
		trRows,
	); err != nil {
		return nil, err
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// ImportCSV replaces event-scoped local data with CSV contents (authoritative restore).
func (s *CSVExportService) ImportCSV(eventID uuid.UUID, data []byte) (*CSVImportSummary, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sections, err := parseCSVSections(data)
	if err != nil {
		return nil, err
	}

	eventRows := sections["event"]
	if len(eventRows) == 0 {
		return nil, fmt.Errorf("%w: missing event section", ErrInvalidCSV)
	}

	parsedEvent, err := parseEventRow(eventRows[0])
	if err != nil {
		return nil, err
	}
	// Target station event id wins; remap CSV event id if needed.
	parsedEvent.ID = uuidutil.NewPublicUUID(eventID)

	races, err := parseRaceRows(sections["races"], eventID)
	if err != nil {
		return nil, err
	}
	categories, err := parseCategoryRows(sections["categories"])
	if err != nil {
		return nil, err
	}
	participants, err := parseParticipantRows(sections["participants"])
	if err != nil {
		return nil, err
	}
	tags, err := parseTagRows(sections["tags"])
	if err != nil {
		return nil, err
	}
	checkpoints, err := parseCheckpointRows(sections["checkpoints"])
	if err != nil {
		return nil, err
	}
	timing, err := parseTimingRows(sections["timing_records"])
	if err != nil {
		return nil, err
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := deleteEventScopedData(tx, eventID); err != nil {
			return err
		}

		var existing models.Event
		err := tx.First(&existing, "id = ?", eventID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := tx.Create(parsedEvent).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			existing.Name = parsedEvent.Name
			existing.EventDate = parsedEvent.EventDate
			existing.Location = parsedEvent.Location
			existing.Status = parsedEvent.Status
			if err := tx.Save(&existing).Error; err != nil {
				return err
			}
		}

		if len(races) > 0 {
			if err := tx.Create(&races).Error; err != nil {
				return err
			}
		}
		if len(categories) > 0 {
			if err := tx.Create(&categories).Error; err != nil {
				return err
			}
		}
		if len(checkpoints) > 0 {
			if err := tx.Create(&checkpoints).Error; err != nil {
				return err
			}
		}
		if len(participants) > 0 {
			if err := tx.Create(&participants).Error; err != nil {
				return err
			}
		}
		if len(tags) > 0 {
			if err := tx.Create(&tags).Error; err != nil {
				return err
			}
		}
		if len(timing) > 0 {
			var bases, deps []models.TimingRecord
			for _, tr := range timing {
				if tr.SourceLapID == nil {
					bases = append(bases, tr)
				} else {
					deps = append(deps, tr)
				}
			}
			if len(bases) > 0 {
				if err := tx.Create(&bases).Error; err != nil {
					return err
				}
			}
			if len(deps) > 0 {
				if err := tx.Create(&deps).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Resume live CSV writing after import.
	body, err := s.BuildCSV(eventID)
	if err != nil {
		return nil, err
	}
	path, err := s.LiveSnapshotPath(eventID)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return nil, err
	}

	return &CSVImportSummary{
		EventID:         eventID.String(),
		EventName:       parsedEvent.Name,
		Races:           len(races),
		Racers:          len(participants),
		TagAssociations: len(tags),
		TimingRecords:   len(timing),
		Categories:      len(categories),
		Checkpoints:     len(checkpoints),
		ImportedAt:      time.Now().UTC(),
	}, nil
}

func deleteEventScopedData(tx *gorm.DB, eventID uuid.UUID) error {
	var raceIDs []uuidutil.PublicUUID
	if err := tx.Model(&models.Race{}).Where("event_id = ?", eventID).Pluck("id", &raceIDs).Error; err != nil {
		return err
	}
	if len(raceIDs) == 0 {
		return nil
	}

	var partIDs []uuidutil.PublicUUID
	if err := tx.Model(&models.Participant{}).Where("race_id IN ?", raceIDs).Pluck("id", &partIDs).Error; err != nil {
		return err
	}

	if len(partIDs) > 0 {
		if err := tx.Where("participant_id IN ?", partIDs).Delete(&models.TimingRecord{}).Error; err != nil {
			return err
		}
		if err := tx.Where("participant_id IN ?", partIDs).Delete(&models.RFIDTagAssociation{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id IN ?", partIDs).Delete(&models.Participant{}).Error; err != nil {
			return err
		}
	}
	if err := tx.Where("race_id IN ?", raceIDs).Delete(&models.Category{}).Error; err != nil {
		return err
	}
	// Detach station checkpoint refs before deleting checkpoints.
	if err := tx.Model(&models.ReaderStation{}).
		Where("checkpoint_id IN (?)", tx.Model(&models.TimingCheckpoint{}).Select("id").Where("race_id IN ?", raceIDs)).
		Update("checkpoint_id", nil).Error; err != nil {
		return err
	}
	if err := tx.Where("race_id IN ?", raceIDs).Delete(&models.TimingCheckpoint{}).Error; err != nil {
		return err
	}
	if err := tx.Where("id IN ?", raceIDs).Delete(&models.Race{}).Error; err != nil {
		return err
	}
	return nil
}

func parseCSVSections(data []byte) (map[string][]map[string]string, error) {
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1
	r.TrimLeadingSpace = true

	sections := map[string][]map[string]string{}
	var current string
	var header []string

	for {
		row, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidCSV, err)
		}
		if len(row) == 0 {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(row[0]), "#SECTION") {
			if len(row) < 2 {
				return nil, fmt.Errorf("%w: malformed section sentinel", ErrInvalidCSV)
			}
			current = strings.TrimSpace(strings.ToLower(row[1]))
			header = nil
			if _, ok := sections[current]; !ok {
				sections[current] = nil
			}
			continue
		}
		if current == "" {
			continue
		}
		if header == nil {
			header = normalizeHeader(row)
			continue
		}
		rec := map[string]string{}
		for i, key := range header {
			if i < len(row) {
				rec[key] = row[i]
			} else {
				rec[key] = ""
			}
		}
		sections[current] = append(sections[current], rec)
	}
	return sections, nil
}

func normalizeHeader(row []string) []string {
	out := make([]string, len(row))
	for i, h := range row {
		out[i] = strings.TrimSpace(strings.ToLower(h))
	}
	return out
}

func parseEventRow(row map[string]string) (*models.Event, error) {
	id, err := parseUUID(row["id"])
	if err != nil {
		return nil, fmt.Errorf("%w: event.id: %v", ErrInvalidCSV, err)
	}
	date, err := time.Parse("2006-01-02", strings.TrimSpace(row["event_date"]))
	if err != nil {
		return nil, fmt.Errorf("%w: event.event_date: %v", ErrInvalidCSV, err)
	}
	status := strings.TrimSpace(row["status"])
	if status == "" {
		status = "upcoming"
	}
	return &models.Event{
		ID:        uuidutil.NewPublicUUID(id),
		Name:      strings.TrimSpace(row["name"]),
		EventDate: date,
		Location:  strings.TrimSpace(row["location"]),
		Status:    status,
	}, nil
}

func parseRaceRows(rows []map[string]string, eventID uuid.UUID) ([]models.Race, error) {
	out := make([]models.Race, 0, len(rows))
	for _, row := range rows {
		id, err := parseUUID(row["id"])
		if err != nil {
			return nil, fmt.Errorf("%w: races.id: %v", ErrInvalidCSV, err)
		}
		duration, _ := strconv.Atoi(strings.TrimSpace(row["duration_minutes"]))
		var start time.Time
		if ts := strings.TrimSpace(row["start_time"]); ts != "" {
			start, err = time.Parse(time.RFC3339, ts)
			if err != nil {
				return nil, fmt.Errorf("%w: races.start_time: %v", ErrInvalidCSV, err)
			}
		}
		status := strings.TrimSpace(row["status"])
		if status == "" {
			status = "scheduled"
		}
		raceType := strings.TrimSpace(row["race_type"])
		if raceType == "" {
			raceType = "time_based"
		}
		out = append(out, models.Race{
			ID:              uuidutil.NewPublicUUID(id),
			EventID:         uuidutil.NewPublicUUID(eventID),
			Name:            strings.TrimSpace(row["name"]),
			RaceType:        raceType,
			DurationMinutes: duration,
			StartTime:       start,
			Status:          status,
		})
	}
	return out, nil
}

func parseCategoryRows(rows []map[string]string) ([]models.Category, error) {
	out := make([]models.Category, 0, len(rows))
	for _, row := range rows {
		id, err := parseUUID(row["id"])
		if err != nil {
			return nil, fmt.Errorf("%w: categories.id: %v", ErrInvalidCSV, err)
		}
		raceID, err := parseUUID(row["race_id"])
		if err != nil {
			return nil, fmt.Errorf("%w: categories.race_id: %v", ErrInvalidCSV, err)
		}
		order, _ := strconv.Atoi(strings.TrimSpace(row["display_order"]))
		catType := strings.TrimSpace(row["category_type"])
		if catType == "" {
			catType = "custom"
		}
		out = append(out, models.Category{
			ID:           uuidutil.NewPublicUUID(id),
			RaceID:       uuidutil.NewPublicUUID(raceID),
			Name:         strings.TrimSpace(row["name"]),
			CategoryType: catType,
			GenderFilter: strings.TrimSpace(row["gender_filter"]),
			DisplayOrder: order,
		})
	}
	return out, nil
}

func parseParticipantRows(rows []map[string]string) ([]models.Participant, error) {
	out := make([]models.Participant, 0, len(rows))
	for _, row := range rows {
		id, err := parseUUID(row["id"])
		if err != nil {
			return nil, fmt.Errorf("%w: participants.id: %v", ErrInvalidCSV, err)
		}
		raceID, err := parseUUID(row["race_id"])
		if err != nil {
			return nil, fmt.Errorf("%w: participants.race_id: %v", ErrInvalidCSV, err)
		}
		status := strings.TrimSpace(row["status"])
		if status == "" {
			status = "registered"
		}
		p := models.Participant{
			ID:        uuidutil.NewPublicUUID(id),
			RaceID:    uuidutil.NewPublicUUID(raceID),
			BibNumber: strings.TrimSpace(row["bib_number"]),
			FirstName: strings.TrimSpace(row["first_name"]),
			LastName:  strings.TrimSpace(row["last_name"]),
			Gender:    strings.TrimSpace(row["gender"]),
			Status:    status,
		}
		if cat := strings.TrimSpace(row["category_id"]); cat != "" {
			catID, err := parseUUID(cat)
			if err != nil {
				return nil, fmt.Errorf("%w: participants.category_id: %v", ErrInvalidCSV, err)
			}
			cid := uuidutil.NewPublicUUID(catID)
			p.CategoryID = &cid
		}
		out = append(out, p)
	}
	return out, nil
}

func parseTagRows(rows []map[string]string) ([]models.RFIDTagAssociation, error) {
	out := make([]models.RFIDTagAssociation, 0, len(rows))
	for _, row := range rows {
		id, err := parseUUID(row["id"])
		if err != nil {
			return nil, fmt.Errorf("%w: tags.id: %v", ErrInvalidCSV, err)
		}
		partID, err := parseUUID(row["participant_id"])
		if err != nil {
			return nil, fmt.Errorf("%w: tags.participant_id: %v", ErrInvalidCSV, err)
		}
		created := time.Now().UTC()
		if ts := strings.TrimSpace(row["created_at"]); ts != "" {
			created, err = time.Parse(time.RFC3339, ts)
			if err != nil {
				return nil, fmt.Errorf("%w: tags.created_at: %v", ErrInvalidCSV, err)
			}
		}
		out = append(out, models.RFIDTagAssociation{
			ID:            uuidutil.NewPublicUUID(id),
			ParticipantID: uuidutil.NewPublicUUID(partID),
			TagUID:        strings.TrimSpace(row["tag_uid"]),
			CreatedAt:     created,
			Active:        true,
		})
	}
	return out, nil
}

func parseCheckpointRows(rows []map[string]string) ([]models.TimingCheckpoint, error) {
	out := make([]models.TimingCheckpoint, 0, len(rows))
	for _, row := range rows {
		id, err := parseUUID(row["id"])
		if err != nil {
			return nil, fmt.Errorf("%w: checkpoints.id: %v", ErrInvalidCSV, err)
		}
		raceID, err := parseUUID(row["race_id"])
		if err != nil {
			return nil, fmt.Errorf("%w: checkpoints.race_id: %v", ErrInvalidCSV, err)
		}
		dist, _ := strconv.ParseFloat(strings.TrimSpace(row["distance_from_start_km"]), 64)
		active := true
		if v := strings.TrimSpace(row["is_active"]); v != "" {
			active, _ = strconv.ParseBool(v)
		}
		cpType := strings.TrimSpace(row["checkpoint_type"])
		if cpType == "" {
			cpType = "finish"
		}
		out = append(out, models.TimingCheckpoint{
			ID:                  uuidutil.NewPublicUUID(id),
			RaceID:              uuidutil.NewPublicUUID(raceID),
			Name:                strings.TrimSpace(row["name"]),
			CheckpointType:      cpType,
			DistanceFromStartKm: dist,
			IsActive:            active,
		})
	}
	return out, nil
}

func parseTimingRows(rows []map[string]string) ([]models.TimingRecord, error) {
	out := make([]models.TimingRecord, 0, len(rows))
	for _, row := range rows {
		id, err := parseUUID(row["id"])
		if err != nil {
			return nil, fmt.Errorf("%w: timing_records.id: %v", ErrInvalidCSV, err)
		}
		partID, err := parseUUID(row["participant_id"])
		if err != nil {
			return nil, fmt.Errorf("%w: timing_records.participant_id: %v", ErrInvalidCSV, err)
		}
		cpID, err := parseUUID(row["checkpoint_id"])
		if err != nil {
			return nil, fmt.Errorf("%w: timing_records.checkpoint_id: %v", ErrInvalidCSV, err)
		}
		ts, err := time.Parse(time.RFC3339, strings.TrimSpace(row["timestamp"]))
		if err != nil {
			return nil, fmt.Errorf("%w: timing_records.timestamp: %v", ErrInvalidCSV, err)
		}
		local := ts
		if v := strings.TrimSpace(row["local_timestamp"]); v != "" {
			local, err = time.Parse(time.RFC3339, v)
			if err != nil {
				return nil, fmt.Errorf("%w: timing_records.local_timestamp: %v", ErrInvalidCSV, err)
			}
		}
		syncStatus := strings.TrimSpace(row["sync_status"])
		if syncStatus == "" {
			syncStatus = "synced"
		}
		recordType := strings.TrimSpace(row["record_type"])
		if recordType == "" {
			recordType = "rfid_lap"
		}
		rec := models.TimingRecord{
			ID:             uuidutil.NewPublicUUID(id),
			ParticipantID:  uuidutil.NewPublicUUID(partID),
			CheckpointID:   uuidutil.NewPublicUUID(cpID),
			Timestamp:      ts,
			LocalTimestamp: local,
			DeviceID:       strings.TrimSpace(row["device_id"]),
			SyncStatus:     syncStatus,
			RecordType:     recordType,
		}
		if v := strings.TrimSpace(row["source_lap_id"]); v != "" {
			sid, err := parseUUID(v)
			if err != nil {
				return nil, fmt.Errorf("%w: timing_records.source_lap_id: %v", ErrInvalidCSV, err)
			}
			pu := uuidutil.NewPublicUUID(sid)
			rec.SourceLapID = &pu
		}
		if v := strings.TrimSpace(row["station_id"]); v != "" {
			sid, err := parseUUID(v)
			if err != nil {
				return nil, fmt.Errorf("%w: timing_records.station_id: %v", ErrInvalidCSV, err)
			}
			pu := uuidutil.NewPublicUUID(sid)
			rec.StationID = &pu
		}
		out = append(out, rec)
	}
	return out, nil
}

func parseUUID(raw string) (uuid.UUID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return uuid.Nil, errors.New("empty uuid")
	}
	return uuid.Parse(raw)
}
