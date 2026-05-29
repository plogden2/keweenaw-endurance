package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/config"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/keweenaw-endurance/backend/internal/rfid"
	"github.com/keweenaw-endurance/backend/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupHandlerTest(t *testing.T) (*gin.Engine, *services.Services) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&models.Event{},
		&models.Race{},
		&models.Participant{},
		&models.TimingCheckpoint{},
		&models.TimingRecord{},
		&models.Category{},
	))

	svc := services.NewServicesWithReader(db, &config.Config{Environment: "test"}, rfid.NewMockReader())
	h := NewHandlers(svc)

	router := gin.New()
	api := router.Group("/api")
	{
		api.GET("/events", h.GetEvents)
		api.POST("/events", h.CreateEvent)
		api.GET("/events/:id", h.GetEvent)
		api.PUT("/events/:id", h.UpdateEvent)
		api.DELETE("/events/:id", h.DeleteEvent)

		api.GET("/races", h.GetRaces)
		api.POST("/races", h.CreateRace)
		api.GET("/races/:id/checkpoints", h.GetCheckpointsByRace)
		api.POST("/races/:id/checkpoints", h.CreateCheckpoint)
		api.GET("/races/:id/categories", h.GetCategoriesByRace)
		api.POST("/races/:id/categories", h.CreateCategory)
		api.GET("/races/:id", h.GetRace)
		api.PUT("/races/:id", h.UpdateRace)
		api.DELETE("/races/:id", h.DeleteRace)

		api.GET("/participants", h.GetParticipants)
		api.POST("/participants", h.CreateParticipant)
		api.GET("/participants/:id", h.GetParticipant)
		api.PUT("/participants/:id", h.UpdateParticipant)
		api.DELETE("/participants/:id", h.DeleteParticipant)

		api.GET("/checkpoints/:id", h.GetCheckpoint)
		api.PUT("/checkpoints/:id", h.UpdateCheckpoint)
		api.DELETE("/checkpoints/:id", h.DeleteCheckpoint)

		api.GET("/categories/:id", h.GetCategory)
		api.PUT("/categories/:id", h.UpdateCategory)
		api.DELETE("/categories/:id", h.DeleteCategory)

		api.GET("/timing/live/:raceId", h.GetLiveTiming)
		api.POST("/timing/record", h.CreateTimingRecord)
		api.PUT("/timing/records/:id", h.UpdateTimingRecord)
		api.GET("/timing/results/:raceId", h.GetRaceResults)
		api.GET("/timing/leaderboard/:raceId", h.GetLeaderboard)

		api.POST("/rfid/write-tag", h.WriteRFIDTag)
		api.GET("/rfid/scan/:uid", h.ScanRFIDTag)
		api.POST("/rfid/manual-entry", h.ManualTimingEntry)
		api.GET("/rfid/sync-status", h.GetSyncStatus)
		api.POST("/rfid/sync-pending", h.SyncPendingRecords)
	}

	return router, svc
}

func TestEventHandlers_CRUD(t *testing.T) {
	router, _ := setupHandlerTest(t)

	body := map[string]string{
		"name":       "Keweenaw Trail Fest",
		"event_date": time.Now().AddDate(0, 2, 0).Format("2006-01-02"),
		"location":   "Houghton, MI",
		"status":     "upcoming",
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/events", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var created models.Event
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	assert.Equal(t, "Keweenaw Trail Fest", created.Name)

	req = httptest.NewRequest(http.MethodGet, "/api/events/"+created.ID.String(), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/api/events", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	updateBody := map[string]string{"name": "Updated Fest"}
	payload, _ = json.Marshal(updateBody)
	req = httptest.NewRequest(http.MethodPut, "/api/events/"+created.ID.String(), bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var updated models.Event
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &updated))
	assert.Equal(t, "Updated Fest", updated.Name)
	assert.Equal(t, "Houghton, MI", updated.Location)

	req = httptest.NewRequest(http.MethodDelete, "/api/events/"+created.ID.String(), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/api/events/"+created.ID.String(), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestEventHandlers_PartialUpdate(t *testing.T) {
	router, _ := setupHandlerTest(t)

	body := map[string]string{
		"name":       "Original Name",
		"event_date": time.Now().AddDate(0, 2, 0).Format("2006-01-02"),
		"location":   "Calumet, MI",
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/events", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var created models.Event
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))

	statusOnly := map[string]string{"status": "active"}
	payload, _ = json.Marshal(statusOnly)
	req = httptest.NewRequest(http.MethodPut, "/api/events/"+created.ID.String(), bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var updated models.Event
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &updated))
	assert.Equal(t, "active", updated.Status)
	assert.Equal(t, "Original Name", updated.Name)
	assert.Equal(t, "Calumet, MI", updated.Location)
}

func TestEventHandlers_InvalidInput(t *testing.T) {
	router, _ := setupHandlerTest(t)

	req := httptest.NewRequest(http.MethodPost, "/api/events", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/api/events/not-a-uuid", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRaceHandlers_CRUD(t *testing.T) {
	router, svc := setupHandlerTest(t)

	event, err := svc.Events.CreateEvent(&models.Event{
		Name: "Parent Event", EventDate: time.Now().AddDate(0, 1, 0),
	})
	require.NoError(t, err)

	body := map[string]interface{}{
		"event_id":    event.ID.String(),
		"name":        "Marathon",
		"race_type":   "time_based",
		"distance_km": 42.195,
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/races", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var created models.Race
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))

	req = httptest.NewRequest(http.MethodGet, "/api/races?event_id="+event.ID.String(), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestParticipantHandlers_CRUD(t *testing.T) {
	router, svc := setupHandlerTest(t)

	event, err := svc.Events.CreateEvent(&models.Event{
		Name: "Event", EventDate: time.Now().AddDate(0, 1, 0),
	})
	require.NoError(t, err)
	race, err := svc.Races.CreateRace(&models.Race{
		EventID: event.ID, Name: "Race", RaceType: "time_based", DistanceKm: 10,
	})
	require.NoError(t, err)

	body := map[string]string{
		"race_id":    race.ID.String(),
		"bib_number": "007",
		"first_name": "James",
		"last_name":  "Bond",
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/participants", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var created models.Participant
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	assert.Equal(t, "007", created.BibNumber)

	req = httptest.NewRequest(http.MethodGet, "/api/participants/"+created.ID.String(), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestTimingHandlers_Results(t *testing.T) {
	router, svc := setupHandlerTest(t)

	event, err := svc.Events.CreateEvent(&models.Event{
		Name: "Event", EventDate: time.Now().AddDate(0, 1, 0),
	})
	require.NoError(t, err)
	race, err := svc.Races.CreateRace(&models.Race{
		EventID: event.ID, Name: "Race", RaceType: "time_based", DistanceKm: 10,
	})
	require.NoError(t, err)

	_, err = svc.Checkpoints.CreateCheckpoint(&models.TimingCheckpoint{
		RaceID: race.ID, Name: "Start", CheckpointType: "start",
	})
	require.NoError(t, err)
	_, err = svc.Checkpoints.CreateCheckpoint(&models.TimingCheckpoint{
		RaceID: race.ID, Name: "Finish", CheckpointType: "finish",
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/timing/results/"+race.ID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/api/timing/live/"+race.ID.String(), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/api/timing/leaderboard/"+race.ID.String(), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestTimingHandlers_CreateRecord(t *testing.T) {
	router, svc := setupHandlerTest(t)

	event, err := svc.Events.CreateEvent(&models.Event{
		Name: "Event", EventDate: time.Now().AddDate(0, 1, 0),
	})
	require.NoError(t, err)
	race, err := svc.Races.CreateRace(&models.Race{
		EventID: event.ID, Name: "Race", RaceType: "time_based", DistanceKm: 10,
	})
	require.NoError(t, err)
	checkpoint, err := svc.Checkpoints.CreateCheckpoint(&models.TimingCheckpoint{
		RaceID: race.ID, Name: "Start", CheckpointType: "start",
	})
	require.NoError(t, err)
	participant, err := svc.Participants.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "42", FirstName: "Test", LastName: "Runner",
	})
	require.NoError(t, err)

	now := time.Now().UTC().Format(time.RFC3339)
	body := map[string]string{
		"participant_id": participant.ID.String(),
		"checkpoint_id":  checkpoint.ID.String(),
		"timestamp":      now,
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/timing/record", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
}

func TestCheckpointHandlers_CRUD(t *testing.T) {
	router, svc := setupHandlerTest(t)

	event, err := svc.Events.CreateEvent(&models.Event{
		Name: "Event", EventDate: time.Now().AddDate(0, 1, 0),
	})
	require.NoError(t, err)
	race, err := svc.Races.CreateRace(&models.Race{
		EventID: event.ID, Name: "Race", RaceType: "time_based", DistanceKm: 10,
	})
	require.NoError(t, err)

	body := map[string]interface{}{
		"name":            "Start",
		"checkpoint_type": "start",
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/races/"+race.ID.String()+"/checkpoints", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var created models.TimingCheckpoint
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	assert.Equal(t, "Start", created.Name)

	req = httptest.NewRequest(http.MethodGet, "/api/races/"+race.ID.String()+"/checkpoints", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/api/checkpoints/"+created.ID.String(), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	updateBody := map[string]interface{}{"name": "Start Line", "distance_from_start_km": 0.0}
	payload, _ = json.Marshal(updateBody)
	req = httptest.NewRequest(http.MethodPut, "/api/checkpoints/"+created.ID.String(), bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequest(http.MethodDelete, "/api/checkpoints/"+created.ID.String(), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestCategoryHandlers_CRUD(t *testing.T) {
	router, svc := setupHandlerTest(t)

	event, err := svc.Events.CreateEvent(&models.Event{
		Name: "Event", EventDate: time.Now().AddDate(0, 1, 0),
	})
	require.NoError(t, err)
	race, err := svc.Races.CreateRace(&models.Race{
		EventID: event.ID, Name: "Race", RaceType: "time_based", DistanceKm: 10,
	})
	require.NoError(t, err)

	body := map[string]string{
		"name":          "Overall",
		"category_type": "overall",
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/races/"+race.ID.String()+"/categories", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var created models.Category
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	assert.Equal(t, "Overall", created.Name)

	req = httptest.NewRequest(http.MethodGet, "/api/races/"+race.ID.String()+"/categories", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/api/categories/"+created.ID.String(), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequest(http.MethodDelete, "/api/categories/"+created.ID.String(), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestCheckpointHandlers_InvalidInput(t *testing.T) {
	router, _ := setupHandlerTest(t)

	req := httptest.NewRequest(http.MethodPost, "/api/races/not-a-uuid/checkpoints", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodPost, "/api/races/"+uuid.New().String()+"/checkpoints", bytes.NewReader([]byte(`{"name":"X","checkpoint_type":"invalid"}`)))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRFIDHandlers_Scan(t *testing.T) {
	router, svc := setupHandlerTest(t)

	event, err := svc.Events.CreateEvent(&models.Event{
		Name: "Event", EventDate: time.Now().AddDate(0, 1, 0),
	})
	require.NoError(t, err)
	race, err := svc.Races.CreateRace(&models.Race{
		EventID: event.ID, Name: "Race", RaceType: "time_based", DistanceKm: 10,
	})
	require.NoError(t, err)
	participant, err := svc.Participants.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "42", FirstName: "Scan", LastName: "Test",
		RFIDTagUID: "TAG-SCAN-001",
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/rfid/scan/TAG-SCAN-001", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var found models.Participant
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &found))
	assert.Equal(t, participant.ID, found.ID)

	req = httptest.NewRequest(http.MethodGet, "/api/rfid/scan/UNKNOWN", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRFIDHandlers_ManualEntry(t *testing.T) {
	router, svc := setupHandlerTest(t)

	event, err := svc.Events.CreateEvent(&models.Event{
		Name: "Event", EventDate: time.Now().AddDate(0, 1, 0),
	})
	require.NoError(t, err)
	race, err := svc.Races.CreateRace(&models.Race{
		EventID: event.ID, Name: "Race", RaceType: "time_based", DistanceKm: 10,
	})
	require.NoError(t, err)
	checkpoint, err := svc.Checkpoints.CreateCheckpoint(&models.TimingCheckpoint{
		RaceID: race.ID, Name: "Start", CheckpointType: "start",
	})
	require.NoError(t, err)
	_, err = svc.Participants.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "99", FirstName: "Manual", LastName: "Entry",
	})
	require.NoError(t, err)

	now := time.Now().UTC().Format(time.RFC3339)
	body := map[string]string{
		"race_id":       race.ID.String(),
		"checkpoint_id": checkpoint.ID.String(),
		"bib_number":    "99",
		"timestamp":     now,
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/rfid/manual-entry", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
}

func TestRFIDHandlers_GetSyncStatus(t *testing.T) {
	router, _ := setupHandlerTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/rfid/sync-status", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var status map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &status))
	assert.Contains(t, status, "pending_count")
}

func TestRFIDHandlers_SyncPending(t *testing.T) {
	router, svc := setupHandlerTest(t)

	event, err := svc.Events.CreateEvent(&models.Event{
		Name: "Event", EventDate: time.Now().AddDate(0, 1, 0),
	})
	require.NoError(t, err)
	race, err := svc.Races.CreateRace(&models.Race{
		EventID: event.ID, Name: "Race", RaceType: "time_based", DistanceKm: 10,
	})
	require.NoError(t, err)
	checkpoint, err := svc.Checkpoints.CreateCheckpoint(&models.TimingCheckpoint{
		RaceID: race.ID, Name: "Start", CheckpointType: "start",
	})
	require.NoError(t, err)
	participant, err := svc.Participants.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "1", FirstName: "A", LastName: "B",
	})
	require.NoError(t, err)

	now := time.Now().UTC()
	_, err = svc.Timing.CreateRecord(&models.TimingRecord{
		ParticipantID: participant.ID, CheckpointID: checkpoint.ID,
		Timestamp: now, LocalTimestamp: now, SyncStatus: "pending_sync",
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/rfid/sync-pending", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &result))
	assert.Equal(t, float64(1), result["synced_count"])
}

func TestRFIDHandlers_WriteTag(t *testing.T) {
	router, svc := setupHandlerTest(t)

	event, err := svc.Events.CreateEvent(&models.Event{
		Name: "Event", EventDate: time.Now().AddDate(0, 1, 0),
	})
	require.NoError(t, err)
	race, err := svc.Races.CreateRace(&models.Race{
		EventID: event.ID, Name: "Race", RaceType: "time_based", DistanceKm: 10,
	})
	require.NoError(t, err)
	participant, err := svc.Participants.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "10", FirstName: "Write", LastName: "Tag",
	})
	require.NoError(t, err)

	body := map[string]string{
		"participant_id": participant.ID.String(),
		"tag_uid":        "NEW-HW-TAG",
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/rfid/write-tag", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var updated models.Participant
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &updated))
	assert.Equal(t, "NEW-HW-TAG", updated.RFIDTagUID)
}
