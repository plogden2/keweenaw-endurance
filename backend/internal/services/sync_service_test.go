package services

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/config"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/keweenaw-endurance/backend/internal/uuidutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestSyncService_PushMarksPendingSyncedWhenNoHostedURL(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewSyncService(db, &config.Config{})

	race := createTestRace(t, db)
	cp := createCheckpoint(t, db, race.ID, "Finish", "finish")
	part := createTestParticipant(t, db, race.ID, "1")
	now := time.Now().UTC().Truncate(time.Second)
	rec := &models.TimingRecord{
		ParticipantID:  part.ID,
		CheckpointID:   cp.ID,
		Timestamp:      now,
		LocalTimestamp: now,
		DeviceID:       "station-a",
		SyncStatus:     "pending_sync",
		RecordType:     "rfid_lap",
	}
	require.NoError(t, db.Create(rec).Error)

	result, err := svc.Push()
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Pushed)
	assert.Equal(t, int64(0), result.Duplicates)

	var updated models.TimingRecord
	require.NoError(t, db.First(&updated, "id = ?", rec.ID).Error)
	assert.Equal(t, "synced", updated.SyncStatus)
}

func TestSyncService_PushSendsPendingToHosted(t *testing.T) {
	db := setupServiceTestDB(t)
	var received SyncPushPayload
	hosted := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/sync/ingest", r.URL.Path)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&received))
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(syncIngestResponse{Accepted: 1, Duplicates: 0})
	}))
	defer hosted.Close()

	svc := NewSyncService(db, &config.Config{
		RFID: config.RFIDConfig{HostedAPIURL: hosted.URL},
	})

	race := createTestRace(t, db)
	cp := createCheckpoint(t, db, race.ID, "Finish", "finish")
	part := createTestParticipant(t, db, race.ID, "2")
	now := time.Now().UTC().Truncate(time.Second)
	rec := &models.TimingRecord{
		ParticipantID:  part.ID,
		CheckpointID:   cp.ID,
		Timestamp:      now,
		LocalTimestamp: now,
		DeviceID:       "station-a",
		SyncStatus:     "pending_sync",
		RecordType:     "rfid_lap",
	}
	require.NoError(t, db.Create(rec).Error)

	result, err := svc.Push()
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Pushed)
	require.Len(t, received.Records, 1)
	assert.Equal(t, rec.ID.String(), received.Records[0].ID)

	var updated models.TimingRecord
	require.NoError(t, db.First(&updated, "id = ?", rec.ID).Error)
	assert.Equal(t, "synced", updated.SyncStatus)
}

func TestSyncService_PushLeavesPendingWhenHostedUnreachable(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewSyncService(db, &config.Config{
		RFID: config.RFIDConfig{HostedAPIURL: "http://127.0.0.1:1"},
	})
	svc.client = &http.Client{Timeout: 200 * time.Millisecond}

	race := createTestRace(t, db)
	cp := createCheckpoint(t, db, race.ID, "Finish", "finish")
	part := createTestParticipant(t, db, race.ID, "3")
	now := time.Now().UTC().Truncate(time.Second)
	rec := &models.TimingRecord{
		ParticipantID:  part.ID,
		CheckpointID:   cp.ID,
		Timestamp:      now,
		LocalTimestamp: now,
		DeviceID:       "station-a",
		SyncStatus:     "pending_sync",
		RecordType:     "rfid_lap",
	}
	require.NoError(t, db.Create(rec).Error)

	_, err := svc.Push()
	require.Error(t, err)

	var updated models.TimingRecord
	require.NoError(t, db.First(&updated, "id = ?", rec.ID).Error)
	assert.Equal(t, "pending_sync", updated.SyncStatus)
}

func TestSyncService_PullMergesAndCooldownDedupes(t *testing.T) {
	db := setupServiceTestDB(t)
	race := createTestRace(t, db)
	cp := createCheckpoint(t, db, race.ID, "Finish", "finish")
	part := createTestParticipant(t, db, race.ID, "4")

	t0 := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	local := &models.TimingRecord{
		ParticipantID:  part.ID,
		CheckpointID:   cp.ID,
		Timestamp:      t0,
		LocalTimestamp: t0,
		DeviceID:       "station-a",
		SyncStatus:     "synced",
		RecordType:     "rfid_lap",
	}
	require.NoError(t, db.Create(local).Error)

	dupID := uuid.New()
	laterID := uuid.New()
	hosted := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/sync/export", r.URL.Path)
		_ = json.NewEncoder(w).Encode(SyncPullPayload{
			Records: []SyncRecordDTO{
				{
					ID:            dupID.String(),
					ParticipantID: part.ID.String(),
					CheckpointID:  cp.ID.String(),
					Timestamp:     t0.Add(30 * time.Second).Format(time.RFC3339Nano),
					DeviceID:      "station-b",
					RecordType:    "rfid_lap",
					SyncStatus:    "synced",
				},
				{
					ID:            laterID.String(),
					ParticipantID: part.ID.String(),
					CheckpointID:  cp.ID.String(),
					Timestamp:     t0.Add(2 * time.Minute).Format(time.RFC3339Nano),
					DeviceID:      "station-b",
					RecordType:    "rfid_lap",
					SyncStatus:    "synced",
				},
			},
		})
	}))
	defer hosted.Close()

	svc := NewSyncService(db, &config.Config{
		RFID: config.RFIDConfig{HostedAPIURL: hosted.URL},
	})

	result, err := svc.Pull()
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Imported)
	assert.Equal(t, int64(1), result.Duplicates)

	var count int64
	require.NoError(t, db.Model(&models.TimingRecord{}).Where("participant_id = ?", part.ID).Count(&count).Error)
	assert.Equal(t, int64(2), count) // original + later; 30s duplicate discarded

	var later models.TimingRecord
	require.NoError(t, db.First(&later, "id = ?", uuidutil.NewPublicUUID(laterID)).Error)
	assert.Equal(t, "rfid_lap", later.RecordType)
}

func TestSyncService_IngestCooldownDedupesKeepEarliest(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewSyncService(db, &config.Config{})

	race := createTestRace(t, db)
	cp := createCheckpoint(t, db, race.ID, "Finish", "finish")
	part := createTestParticipant(t, db, race.ID, "5")
	t0 := time.Date(2026, 8, 1, 13, 0, 0, 0, time.UTC)

	earlyID := uuid.New()
	lateID := uuid.New()
	result, err := svc.Ingest(SyncPushPayload{
		Records: []SyncRecordDTO{
			{
				ID:            lateID.String(),
				ParticipantID: part.ID.String(),
				CheckpointID:  cp.ID.String(),
				Timestamp:     t0.Add(20 * time.Second).Format(time.RFC3339Nano),
				DeviceID:      "b",
				RecordType:    "rfid_lap",
				SyncStatus:    "synced",
			},
			{
				ID:            earlyID.String(),
				ParticipantID: part.ID.String(),
				CheckpointID:  cp.ID.String(),
				Timestamp:     t0.Format(time.RFC3339Nano),
				DeviceID:      "a",
				RecordType:    "rfid_lap",
				SyncStatus:    "synced",
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Imported)
	assert.Equal(t, int64(1), result.Duplicates)

	var kept models.TimingRecord
	require.NoError(t, db.First(&kept, "participant_id = ?", part.ID).Error)
	assert.Equal(t, earlyID, kept.ID.UUID())
}

func TestSyncService_MultiStationCooldownDedupeAcrossDevices(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewSyncService(db, &config.Config{})

	race := createTestRace(t, db)
	cp := createCheckpoint(t, db, race.ID, "Finish", "finish")
	part := createTestParticipant(t, db, race.ID, "ms-1")
	t0 := time.Date(2026, 8, 1, 14, 0, 0, 0, time.UTC)

	// Station A already scored a lap locally.
	local := &models.TimingRecord{
		ParticipantID:  part.ID,
		CheckpointID:   cp.ID,
		Timestamp:      t0,
		LocalTimestamp: t0,
		DeviceID:       "laptop-finish-1",
		SyncStatus:     "synced",
		RecordType:     "rfid_lap",
	}
	require.NoError(t, db.Create(local).Error)

	// Station B / C push overlapping RFID laps within 60s — keep earliest only.
	bID := uuid.New()
	cID := uuid.New()
	okID := uuid.New()
	result, err := svc.Ingest(SyncPushPayload{
		Records: []SyncRecordDTO{
			{
				ID:            bID.String(),
				ParticipantID: part.ID.String(),
				CheckpointID:  cp.ID.String(),
				Timestamp:     t0.Add(15 * time.Second).Format(time.RFC3339Nano),
				DeviceID:      "laptop-finish-2",
				RecordType:    "rfid_lap",
				SyncStatus:    "synced",
			},
			{
				ID:            cID.String(),
				ParticipantID: part.ID.String(),
				CheckpointID:  cp.ID.String(),
				Timestamp:     t0.Add(45 * time.Second).Format(time.RFC3339Nano),
				DeviceID:      "laptop-finish-3",
				RecordType:    "rfid_lap",
				SyncStatus:    "synced",
			},
			{
				ID:            okID.String(),
				ParticipantID: part.ID.String(),
				CheckpointID:  cp.ID.String(),
				Timestamp:     t0.Add(90 * time.Second).Format(time.RFC3339Nano),
				DeviceID:      "laptop-finish-2",
				RecordType:    "rfid_lap",
				SyncStatus:    "synced",
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Imported)
	assert.Equal(t, int64(2), result.Duplicates)

	var count int64
	require.NoError(t, db.Model(&models.TimingRecord{}).
		Where("participant_id = ? AND record_type = ?", part.ID, "rfid_lap").
		Count(&count).Error)
	assert.Equal(t, int64(2), count) // original + 90s later

	var kept models.TimingRecord
	require.NoError(t, db.First(&kept, "id = ?", uuidutil.NewPublicUUID(okID)).Error)
	assert.Equal(t, "laptop-finish-2", kept.DeviceID)
}

func TestSyncService_ResolveStatusPendingWhenHostedConfigured(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewSyncService(db, &config.Config{
		RFID: config.RFIDConfig{HostedAPIURL: "http://hosted.example"},
	})
	assert.Equal(t, "pending_sync", svc.ResolveSyncStatus())
}

func TestSyncService_ResolveStatusSyncedWhenNoHosted(t *testing.T) {
	db := setupServiceTestDB(t)
	svc := NewSyncService(db, &config.Config{})
	assert.Equal(t, "synced", svc.ResolveSyncStatus())
}

func createTestParticipant(t *testing.T, db *gorm.DB, raceID uuidutil.PublicUUID, bib string) *models.Participant {
	t.Helper()
	partSvc := NewParticipantService(db)
	part, err := partSvc.CreateParticipant(&models.Participant{
		RaceID:    raceID,
		BibNumber: bib,
		FirstName: "Test",
		LastName:  "Racer",
	})
	require.NoError(t, err)
	return part
}
