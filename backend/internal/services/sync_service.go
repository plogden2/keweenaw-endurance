package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/keweenaw-endurance/backend/internal/config"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/keweenaw-endurance/backend/internal/uuidutil"
	"gorm.io/gorm"
)

const syncCooldownWindow = 60 * time.Second

// SyncService replicates local timing records to/from HOSTED_API_URL with
// cooldown-window dedupe (R5): keep earliest RFID lap when two fall within 60s.
type SyncService struct {
	db     *gorm.DB
	cfg    *config.Config
	client *http.Client
}

type SyncPushResult struct {
	Pushed     int64 `json:"pushed"`
	Duplicates int64 `json:"duplicates"`
}

type SyncPullResult struct {
	Imported   int64 `json:"imported"`
	Duplicates int64 `json:"duplicates"`
}

type syncIngestResponse struct {
	Accepted   int64 `json:"accepted"`
	Duplicates int64 `json:"duplicates"`
}

// SyncRecordDTO is the wire format for timing records exchanged during sync.
type SyncRecordDTO struct {
	ID             string `json:"id"`
	ParticipantID  string `json:"participant_id"`
	CheckpointID   string `json:"checkpoint_id"`
	Timestamp      string `json:"timestamp"`
	LocalTimestamp string `json:"local_timestamp,omitempty"`
	DeviceID       string `json:"device_id,omitempty"`
	RecordType     string `json:"record_type,omitempty"`
	SyncStatus     string `json:"sync_status,omitempty"`
	SourceLapID    string `json:"source_lap_id,omitempty"`
	StationID      string `json:"station_id,omitempty"`
}

// SyncPushPayload is the body for push/ingest.
type SyncPushPayload struct {
	Records []SyncRecordDTO `json:"records"`
}

// SyncPullPayload is the body for pull/export.
type SyncPullPayload struct {
	Records []SyncRecordDTO `json:"records"`
}

func NewSyncService(db *gorm.DB, cfg *config.Config) *SyncService {
	if cfg == nil {
		cfg = &config.Config{}
	}
	return &SyncService{
		db:  db,
		cfg: cfg,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (s *SyncService) hostedURL() string {
	if s.cfg == nil {
		return ""
	}
	return strings.TrimRight(strings.TrimSpace(s.cfg.RFID.HostedAPIURL), "/")
}

// ResolveSyncStatus returns the sync_status to stamp on newly written local
// records. When no hosted URL is configured, local-only mode uses "synced".
// When hosted is configured, records start as pending_sync until Push succeeds.
func (s *SyncService) ResolveSyncStatus() string {
	if s.hostedURL() == "" {
		return "synced"
	}
	return "pending_sync"
}

// HostedReachable probes the hosted health endpoint (best-effort).
func (s *SyncService) HostedReachable() bool {
	base := s.hostedURL()
	if base == "" {
		return false
	}
	req, err := http.NewRequest(http.MethodGet, base+"/health", nil)
	if err != nil {
		return false
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 500
}

// Push sends pending_sync timing records to hosted /api/sync/ingest.
// With no hosted URL, marks pending records synced (local-only).
func (s *SyncService) Push() (*SyncPushResult, error) {
	var pending []models.TimingRecord
	if err := s.db.Where("sync_status = ?", "pending_sync").Find(&pending).Error; err != nil {
		return nil, err
	}
	if len(pending) == 0 {
		return &SyncPushResult{}, nil
	}

	base := s.hostedURL()
	if base == "" {
		if err := s.db.Model(&models.TimingRecord{}).
			Where("sync_status = ?", "pending_sync").
			Update("sync_status", "synced").Error; err != nil {
			return nil, err
		}
		return &SyncPushResult{Pushed: int64(len(pending))}, nil
	}

	payload := SyncPushPayload{Records: make([]SyncRecordDTO, 0, len(pending))}
	for _, r := range pending {
		payload.Records = append(payload.Records, timingRecordToDTO(r))
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Post(base+"/api/sync/ingest", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("hosted push failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("hosted push status %d: %s", resp.StatusCode, string(raw))
	}

	var ingest syncIngestResponse
	_ = json.NewDecoder(resp.Body).Decode(&ingest)

	ids := make([]uuidutil.PublicUUID, 0, len(pending))
	for _, r := range pending {
		ids = append(ids, r.ID)
	}
	if err := s.db.Model(&models.TimingRecord{}).
		Where("id IN ?", ids).
		Update("sync_status", "synced").Error; err != nil {
		return nil, err
	}

	return &SyncPushResult{
		Pushed:     int64(len(pending)),
		Duplicates: ingest.Duplicates,
	}, nil
}

// Pull fetches hosted timing records and merges them locally with cooldown dedupe.
func (s *SyncService) Pull() (*SyncPullResult, error) {
	base := s.hostedURL()
	if base == "" {
		return &SyncPullResult{}, nil
	}

	resp, err := s.client.Get(base + "/api/sync/export")
	if err != nil {
		return nil, fmt.Errorf("hosted pull failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("hosted pull status %d: %s", resp.StatusCode, string(raw))
	}

	var payload SyncPullPayload
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return s.Ingest(SyncPushPayload{Records: payload.Records})
}

// Ingest merges remote timing records into the local DB with cooldown dedupe.
// Records are applied earliest-first so the first RFID lap in a 60s window wins.
func (s *SyncService) Ingest(payload SyncPushPayload) (*SyncPullResult, error) {
	type pending struct {
		rec *models.TimingRecord
	}
	items := make([]pending, 0, len(payload.Records))
	for _, dto := range payload.Records {
		rec, err := dtoToTimingRecord(dto)
		if err != nil {
			return nil, err
		}
		if rec.RecordType == "" {
			rec.RecordType = "rfid_lap"
		}
		if rec.SyncStatus == "" {
			rec.SyncStatus = "synced"
		}
		items = append(items, pending{rec: rec})
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].rec.Timestamp.Before(items[j].rec.Timestamp)
	})

	var imported, duplicates int64
	for _, item := range items {
		rec := item.rec
		if rec.RecordType == "rfid_lap" && s.hasCooldownConflict(rec) {
			duplicates++
			continue
		}

		var existing models.TimingRecord
		err := s.db.First(&existing, "id = ?", rec.ID).Error
		if err == nil {
			continue
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}

		if err := s.db.Create(rec).Error; err != nil {
			return nil, err
		}
		imported++
	}
	return &SyncPullResult{Imported: imported, Duplicates: duplicates}, nil
}

func (s *SyncService) hasCooldownConflict(incoming *models.TimingRecord) bool {
	var count int64
	windowStart := incoming.Timestamp.Add(-syncCooldownWindow)
	windowEnd := incoming.Timestamp.Add(syncCooldownWindow)
	err := s.db.Model(&models.TimingRecord{}).
		Where(
			"participant_id = ? AND record_type = ? AND timestamp >= ? AND timestamp <= ? AND id <> ?",
			incoming.ParticipantID,
			"rfid_lap",
			windowStart,
			windowEnd,
			incoming.ID,
		).
		Count(&count).Error
	if err != nil {
		return false
	}
	return count > 0
}

func timingRecordToDTO(r models.TimingRecord) SyncRecordDTO {
	dto := SyncRecordDTO{
		ID:             r.ID.String(),
		ParticipantID:  r.ParticipantID.String(),
		CheckpointID:   r.CheckpointID.String(),
		Timestamp:      r.Timestamp.UTC().Format(time.RFC3339Nano),
		LocalTimestamp: r.LocalTimestamp.UTC().Format(time.RFC3339Nano),
		DeviceID:       r.DeviceID,
		RecordType:     r.RecordType,
		SyncStatus:     r.SyncStatus,
	}
	if r.SourceLapID != nil {
		dto.SourceLapID = r.SourceLapID.String()
	}
	if r.StationID != nil {
		dto.StationID = r.StationID.String()
	}
	return dto
}

func dtoToTimingRecord(dto SyncRecordDTO) (*models.TimingRecord, error) {
	id, err := parseUUID(dto.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid record id: %w", err)
	}
	pid, err := parseUUID(dto.ParticipantID)
	if err != nil {
		return nil, fmt.Errorf("invalid participant_id: %w", err)
	}
	cid, err := parseUUID(dto.CheckpointID)
	if err != nil {
		return nil, fmt.Errorf("invalid checkpoint_id: %w", err)
	}
	ts, err := time.Parse(time.RFC3339Nano, dto.Timestamp)
	if err != nil {
		ts, err = time.Parse(time.RFC3339, dto.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("invalid timestamp: %w", err)
		}
	}
	local := ts
	if dto.LocalTimestamp != "" {
		if lt, err := time.Parse(time.RFC3339Nano, dto.LocalTimestamp); err == nil {
			local = lt
		} else if lt, err := time.Parse(time.RFC3339, dto.LocalTimestamp); err == nil {
			local = lt
		}
	}
	rec := &models.TimingRecord{
		ID:             uuidutil.NewPublicUUID(id),
		ParticipantID:  uuidutil.NewPublicUUID(pid),
		CheckpointID:   uuidutil.NewPublicUUID(cid),
		Timestamp:      ts.UTC(),
		LocalTimestamp: local.UTC(),
		DeviceID:       dto.DeviceID,
		RecordType:     dto.RecordType,
		SyncStatus:     dto.SyncStatus,
	}
	if dto.SourceLapID != "" {
		sid, err := parseUUID(dto.SourceLapID)
		if err == nil {
			u := uuidutil.NewPublicUUID(sid)
			rec.SourceLapID = &u
		}
	}
	if dto.StationID != "" {
		sid, err := parseUUID(dto.StationID)
		if err == nil {
			u := uuidutil.NewPublicUUID(sid)
			rec.StationID = &u
		}
	}
	return rec, nil
}

// ExportPendingPayload builds a pull/export payload of all local timing records
// (used when this station acts as hosted ingest target in tests / peer sync).
func (s *SyncService) ExportRecords() (*SyncPullPayload, error) {
	var records []models.TimingRecord
	if err := s.db.Find(&records).Error; err != nil {
		return nil, err
	}
	out := &SyncPullPayload{Records: make([]SyncRecordDTO, 0, len(records))}
	for _, r := range records {
		out.Records = append(out.Records, timingRecordToDTO(r))
	}
	return out, nil
}
