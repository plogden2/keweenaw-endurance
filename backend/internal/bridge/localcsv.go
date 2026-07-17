package bridge

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const pendingFileName = "pending.jsonl"

// PendingLap is a durable offline tag read waiting for hosted sync.
type PendingLap struct {
	ID          string    `json:"id"`
	LogicalUUID string    `json:"logical_uuid"`
	TS          time.Time `json:"ts"`
	DeviceID    string    `json:"device_id"`
}

// LocalStore maintains append-only live CSV rows and a pending lap queue on disk.
type LocalStore struct {
	mu        sync.Mutex
	dataDir   string
	eventID   string
	eventDir  string
	csvPath   string
	pendingPath string
}

// NewLocalStore opens (or creates) bridge data under dataDir/events/{eventID}/.
func NewLocalStore(dataDir, eventID string) (*LocalStore, error) {
	dataDir = strings.TrimSpace(dataDir)
	eventID = strings.TrimSpace(eventID)
	if dataDir == "" {
		return nil, errors.New("data dir is required")
	}
	if eventID == "" {
		return nil, errors.New("event id is required")
	}
	if _, err := uuid.Parse(eventID); err != nil {
		return nil, fmt.Errorf("invalid event id: %w", err)
	}

	eventDir := filepath.Join(dataDir, "events", eventID)
	if err := os.MkdirAll(eventDir, 0o755); err != nil {
		return nil, err
	}

	return &LocalStore{
		dataDir:     dataDir,
		eventID:     eventID,
		eventDir:    eventDir,
		csvPath:     filepath.Join(eventDir, "live-snapshot.csv"),
		pendingPath: filepath.Join(eventDir, pendingFileName),
	}, nil
}

// PendingCount returns the number of laps waiting for sync.
func (s *LocalStore) PendingCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pendingCountLocked()
}

func (s *LocalStore) pendingCountLocked() int {
	pending, err := s.listPendingLocked()
	if err != nil {
		log.Printf("bridge local store: list pending failed: %v", err)
		return 0
	}
	return len(pending)
}

// ListPending returns all queued laps in enqueue order.
func (s *LocalStore) ListPending() ([]PendingLap, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.listPendingLocked()
}

func (s *LocalStore) listPendingLocked() ([]PendingLap, error) {
	f, err := os.Open(s.pendingPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var out []PendingLap
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var lap PendingLap
		if err := json.Unmarshal([]byte(line), &lap); err != nil {
			return nil, fmt.Errorf("decode pending lap: %w", err)
		}
		out = append(out, lap)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// EnqueueLap appends an audit row to live-snapshot.csv, then pending.jsonl.
// If pending append fails, the CSV row is rolled back so the queue stays authoritative.
func (s *LocalStore) EnqueueLap(lap PendingLap) error {
	lap.LogicalUUID = strings.TrimSpace(strings.ToLower(lap.LogicalUUID))
	lap.DeviceID = strings.TrimSpace(lap.DeviceID)
	if lap.LogicalUUID == "" {
		return errors.New("logical_uuid is required")
	}
	if lap.DeviceID == "" {
		return errors.New("device_id is required")
	}
	if lap.TS.IsZero() {
		lap.TS = time.Now().UTC()
	} else {
		lap.TS = lap.TS.UTC()
	}
	if lap.ID == "" {
		lap.ID = uuid.New().String()
	}
	if _, err := uuid.Parse(lap.ID); err != nil {
		return fmt.Errorf("invalid lap id: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(s.csvPath), 0o755); err != nil {
		return err
	}

	rollback, err := s.appendCSVRowLocked(lap)
	if err != nil {
		return err
	}
	if err := s.appendPendingLocked(lap); err != nil {
		rollback()
		return err
	}
	return nil
}

func (s *LocalStore) appendPendingLocked(lap PendingLap) error {
	f, err := os.OpenFile(s.pendingPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	payload, err := json.Marshal(lap)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(payload, '\n')); err != nil {
		return err
	}
	return f.Sync()
}

func (s *LocalStore) appendCSVRowLocked(lap PendingLap) (func(), error) {
	if err := s.ensureCSVHeaderLocked(); err != nil {
		return nil, err
	}

	sizeBefore, err := s.csvSizeLocked()
	if err != nil {
		return nil, err
	}

	f, err := os.OpenFile(s.csvPath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	ts := lap.TS.UTC().Format(time.RFC3339)
	row := []string{
		lap.ID,
		lap.LogicalUUID,
		"",
		ts,
		ts,
		lap.DeviceID,
		"pending_sync",
		"bridge_read",
		lap.ID,
		"",
	}
	if err := w.Write(row); err != nil {
		return nil, err
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	if err := f.Sync(); err != nil {
		return nil, err
	}

	rollback := func() {
		if err := os.Truncate(s.csvPath, sizeBefore); err != nil {
			log.Printf("bridge local store: rollback csv failed: %v", err)
		}
	}
	return rollback, nil
}

func (s *LocalStore) csvSizeLocked() (int64, error) {
	info, err := os.Stat(s.csvPath)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	return info.Size(), nil
}

func (s *LocalStore) ensureCSVHeaderLocked() error {
	if _, err := os.Stat(s.csvPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	header := strings.Join([]string{
		"#SECTION,timing_records",
		"id,participant_id,checkpoint_id,timestamp,local_timestamp,device_id,sync_status,record_type,source_lap_id,station_id",
	}, "\n") + "\n"
	return os.WriteFile(s.csvPath, []byte(header), 0o644)
}

// RemovePending drops a single lap from the queue by id.
func (s *LocalStore) RemovePending(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("lap id is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pending, err := s.listPendingLocked()
	if err != nil {
		return err
	}
	filtered := pending[:0]
	for _, lap := range pending {
		if lap.ID != id {
			filtered = append(filtered, lap)
		}
	}
	return s.rewritePendingLocked(filtered)
}

// ClearPending removes all queued laps.
func (s *LocalStore) ClearPending() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rewritePendingLocked(nil)
}

func (s *LocalStore) rewritePendingLocked(pending []PendingLap) error {
	if len(pending) == 0 {
		if err := os.Remove(s.pendingPath); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}

	f, err := os.OpenFile(s.pendingPath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, lap := range pending {
		payload, err := json.Marshal(lap)
		if err != nil {
			return err
		}
		if _, err := f.Write(append(payload, '\n')); err != nil {
			return err
		}
	}
	return f.Sync()
}

// CSVPath returns the live snapshot path for diagnostics.
func (s *LocalStore) CSVPath() string {
	return s.csvPath
}

// PendingPath returns the pending queue path for diagnostics.
func (s *LocalStore) PendingPath() string {
	return s.pendingPath
}
