package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/keweenaw-endurance/backend/internal/config"
	"github.com/keweenaw-endurance/backend/internal/middleware"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/keweenaw-endurance/backend/internal/rfid"
	"github.com/keweenaw-endurance/backend/internal/services"
	"github.com/keweenaw-endurance/backend/internal/uuidutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testHandlerConfig() *config.Config {
	return &config.Config{
		Environment: "test",
		JWT: config.JWTConfig{
			Secret:          "test-secret-key",
			AccessTokenTTL:  time.Hour,
			RefreshTokenTTL: time.Hour * 24,
		},
		Auth: config.AuthConfig{
			Users:        "admin:admin123:admin,timer:timer123:timer,viewer:viewer123:viewer",
			OrganizerPIN: "1738",
		},
		RFID: config.RFIDConfig{
			InjectEnabled: true,
			Hardware:      true,
		},
	}
}

func bearerTokenForRole(t *testing.T, svc *services.Services, role string) string {
	t.Helper()
	var username, password string
	switch role {
	case services.RoleAdmin:
		username, password = "admin", "admin123"
	case services.RoleTimer:
		username, password = "timer", "timer123"
	default:
		username, password = "viewer", "viewer123"
	}
	resp, err := svc.Auth.Login(username, password)
	require.NoError(t, err)
	return "Bearer " + resp.Token
}

func setupHandlerTest(t *testing.T) (*gin.Engine, *services.Services) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
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
	))

	svc := services.NewServicesWithReader(db, testHandlerConfig(), rfid.NewMockReader())
	h := NewHandlers(svc)

	auth := middleware.JWTAuth(svc.Auth)
	adminOnly := []gin.HandlerFunc{auth, middleware.RequireRoles(services.RoleAdmin, services.RoleOwner)}
	timerWrite := []gin.HandlerFunc{auth, middleware.RequireRoles(services.RoleAdmin, services.RoleOwner, services.RoleTimer)}

	router := gin.New()
	api := router.Group("/api")
	{
		api.POST("/auth/login", h.Login)
		// Public PIN → admin JWT for management routes.
		api.POST("/auth/pin", h.ExchangePIN)

		api.GET("/events", h.GetEvents)
		api.POST("/events", append(adminOnly, h.CreateEvent)...)
		api.GET("/events/:id", h.GetEvent)
		api.PUT("/events/:id", append(adminOnly, h.UpdateEvent)...)
		api.DELETE("/events/:id", append(adminOnly, h.DeleteEvent)...)
		api.GET("/events/:id/live", h.GetEventLive)
		api.GET("/events/:id/live/stream", h.StreamEventLive)
		api.POST("/events/:id/scans", h.ProcessEventScan)
		api.GET("/events/:id/taps", h.ListEventTaps)
		api.GET("/events/:id/participants", h.ListEventParticipants)
		api.POST("/events/:id/taps", append(timerWrite, h.CreateEventTap)...)
		api.GET("/events/:id/bibs", h.ListEventBibs)
		api.POST("/events/:id/bibs/bulk", append(adminOnly, h.BulkCreateEventBibs)...)
		api.GET("/events/:id/bibs/:bibId/tags", h.ListBibTags)
		api.POST("/events/:id/bibs/:bibId/tags", append(adminOnly, h.PostBibTag)...)
		api.GET("/events/:id/live-csv", append(adminOnly, h.GetLiveCSV)...)
		api.GET("/events/:id/live-csv/status", append(adminOnly, h.GetLiveCSVStatus)...)
		api.POST("/events/:id/import.csv", append(adminOnly, h.ImportCSV)...)
		api.GET("/events/:id/results.xlsx", append(adminOnly, h.GetEventResultsExcel)...)

		api.GET("/stations/current", h.GetCurrentStation)
		api.PUT("/stations/current", append(adminOnly, h.PutCurrentStation)...)

		api.GET("/races", h.GetRaces)
		api.POST("/races", append(adminOnly, h.CreateRace)...)
		api.GET("/races/:id/checkpoints", h.GetCheckpointsByRace)
		api.POST("/races/:id/checkpoints", append(adminOnly, h.CreateCheckpoint)...)
		api.GET("/races/:id/categories", h.GetCategoriesByRace)
		api.POST("/races/:id/categories", append(adminOnly, h.CreateCategory)...)
		api.GET("/races/:id/teams", h.GetTeamsByRace)
		api.POST("/races/:id/teams", append(adminOnly, h.CreateTeam)...)
		api.GET("/races/:id/participants", h.GetRaceParticipants)
		api.POST("/races/:id/participants", append(adminOnly, h.CreateRaceParticipant)...)
		api.GET("/races/:id/participants/:participantId/tags", h.GetParticipantTags)
		api.POST("/races/:id/participants/:participantId/tags", append(adminOnly, h.PostParticipantTag)...)
		api.POST("/races/:id/start", append(adminOnly, h.StartRace)...)
		api.POST("/races/:id/finish", append(adminOnly, h.FinishRace)...)
		api.GET("/races/:id", h.GetRace)
		api.PUT("/races/:id", append(adminOnly, h.UpdateRace)...)
		api.DELETE("/races/:id", append(adminOnly, h.DeleteRace)...)

		api.GET("/teams/:id", h.GetTeam)
		api.PUT("/teams/:id", append(adminOnly, h.UpdateTeam)...)
		api.DELETE("/teams/:id", append(adminOnly, h.DeleteTeam)...)
		api.PUT("/teams/:id/members", append(adminOnly, h.SetTeamMembers)...)

		api.GET("/participants", h.GetParticipants)
		api.POST("/participants", append(adminOnly, h.CreateParticipant)...)
		api.GET("/participants/:id", h.GetParticipant)
		api.PUT("/participants/:id", append(adminOnly, h.UpdateParticipant)...)
		api.DELETE("/participants/:id", append(adminOnly, h.DeleteParticipant)...)

		api.GET("/checkpoints/:id", h.GetCheckpoint)
		api.PUT("/checkpoints/:id", append(adminOnly, h.UpdateCheckpoint)...)
		api.DELETE("/checkpoints/:id", append(adminOnly, h.DeleteCheckpoint)...)

		api.GET("/categories/:id", h.GetCategory)
		api.PUT("/categories/:id", append(adminOnly, h.UpdateCategory)...)
		api.DELETE("/categories/:id", append(adminOnly, h.DeleteCategory)...)

		api.GET("/timing/live/:raceId", h.GetLiveTiming)
		api.POST("/timing/record", append(timerWrite, h.CreateTimingRecord)...)
		api.PUT("/timing/records/:id", append(timerWrite, h.UpdateTimingRecord)...)
		api.POST("/timing/records/:id/void", append(timerWrite, h.VoidTimingRecord)...)
		api.POST("/timing/records/:id/restore", append(timerWrite, h.RestoreTimingRecord)...)
		api.GET("/timing/results/:raceId", h.GetRaceResults)
		api.GET("/timing/leaderboard/:raceId", h.GetLeaderboard)
		api.GET("/timing/team-leaderboard/:raceId", h.GetTeamLeaderboard)
		api.POST("/timing-records/:id/karaoke-bonus", h.CreateKaraokeBonus)

		api.POST("/rfid/write-tag", append(adminOnly, h.WriteRFIDTag)...)
		api.GET("/rfid/read-payload", append(adminOnly, h.ReadRFIDPayload)...)
		api.GET("/rfid/scan/:uid", h.ScanRFIDTag)
		api.GET("/rfid/stream", h.StreamRFIDTags)
		api.POST("/rfid/inject", h.InjectRFIDTag)
		api.POST("/rfid/manual-entry", append(timerWrite, h.ManualTimingEntry)...)
		api.GET("/rfid/sync-status", h.GetSyncStatus)
		api.POST("/rfid/sync-pending", append(timerWrite, h.SyncPendingRecords)...)
		api.GET("/rfid/bridge", h.BridgeWebSocket)
		api.GET("/rfid/bridge/status", h.GetBridgeStatus)

		api.POST("/sync/push", h.PushSync)
		api.POST("/sync/pull", h.PullSync)
		api.POST("/sync/ingest", h.IngestSync)
		api.GET("/sync/export", h.ExportSync)
	}

	return router, svc
}

func adminAuthHeader(t *testing.T, svc *services.Services) string {
	return bearerTokenForRole(t, svc, services.RoleAdmin)
}

func timerAuthHeader(t *testing.T, svc *services.Services) string {
	return bearerTokenForRole(t, svc, services.RoleTimer)
}

func TestEventHandlers_CRUD(t *testing.T) {
	router, svc := setupHandlerTest(t)
	auth := adminAuthHeader(t, svc)

	body := map[string]string{
		"name":       "Keweenaw Trail Fest",
		"event_date": time.Now().AddDate(0, 2, 0).Format("2006-01-02"),
		"location":   "Houghton, MI",
		"status":     "upcoming",
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/events", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", auth)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var created models.Event
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	assert.Equal(t, "Keweenaw Trail Fest", created.Name)

	req = httptest.NewRequest(http.MethodGet, "/api/events/"+created.ID.Short(), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/api/events", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	updateBody := map[string]string{"name": "Updated Fest"}
	payload, _ = json.Marshal(updateBody)
	req = httptest.NewRequest(http.MethodPut, "/api/events/"+created.ID.Short(), bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", auth)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var updated models.Event
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &updated))
	assert.Equal(t, "Updated Fest", updated.Name)
	assert.Equal(t, "Houghton, MI", updated.Location)

	req = httptest.NewRequest(http.MethodDelete, "/api/events/"+created.ID.Short(), nil)
	req.Header.Set("Authorization", auth)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/api/events/"+created.ID.Short(), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestEventHandlers_PartialUpdate(t *testing.T) {
	router, svc := setupHandlerTest(t)
	auth := adminAuthHeader(t, svc)

	body := map[string]string{
		"name":       "Original Name",
		"event_date": time.Now().AddDate(0, 2, 0).Format("2006-01-02"),
		"location":   "Calumet, MI",
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/events", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", auth)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var created models.Event
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))

	statusOnly := map[string]string{"status": "active"}
	payload, _ = json.Marshal(statusOnly)
	req = httptest.NewRequest(http.MethodPut, "/api/events/"+created.ID.Short(), bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", auth)
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
	router, svc := setupHandlerTest(t)

	req := httptest.NewRequest(http.MethodPost, "/api/events", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", adminAuthHeader(t, svc))
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
		"event_id":    event.ID.Short(),
		"name":        "Marathon",
		"race_type":   "time_based",
		"distance_km": 42.195,
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/races", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", adminAuthHeader(t, svc))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var created models.Race
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))

	req = httptest.NewRequest(http.MethodGet, "/api/races?event_id="+event.ID.Short(), nil)
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
		"race_id":    race.ID.Short(),
		"bib_number": "007",
		"first_name": "James",
		"last_name":  "Bond",
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/participants", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", adminAuthHeader(t, svc))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var created models.Participant
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	assert.Equal(t, "007", created.BibNumber)

	req = httptest.NewRequest(http.MethodGet, "/api/participants/"+created.ID.Short(), nil)
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

	req := httptest.NewRequest(http.MethodGet, "/api/timing/results/"+race.ID.Short(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/api/timing/live/"+race.ID.Short(), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/api/timing/leaderboard/"+race.ID.Short(), nil)
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
		"participant_id": participant.ID.Short(),
		"checkpoint_id":  checkpoint.ID.Short(),
		"timestamp":      now,
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/timing/record", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", timerAuthHeader(t, svc))
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

	req := httptest.NewRequest(http.MethodPost, "/api/races/"+race.ID.Short()+"/checkpoints", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", adminAuthHeader(t, svc))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var created models.TimingCheckpoint
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	assert.Equal(t, "Start", created.Name)

	req = httptest.NewRequest(http.MethodGet, "/api/races/"+race.ID.Short()+"/checkpoints", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/api/checkpoints/"+created.ID.Short(), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	updateBody := map[string]interface{}{"name": "Start Line", "distance_from_start_km": 0.0}
	payload, _ = json.Marshal(updateBody)
	req = httptest.NewRequest(http.MethodPut, "/api/checkpoints/"+created.ID.Short(), bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", adminAuthHeader(t, svc))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequest(http.MethodDelete, "/api/checkpoints/"+created.ID.Short(), nil)
	req.Header.Set("Authorization", adminAuthHeader(t, svc))
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

	req := httptest.NewRequest(http.MethodPost, "/api/races/"+race.ID.Short()+"/categories", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", adminAuthHeader(t, svc))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var created models.Category
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	assert.Equal(t, "Overall", created.Name)

	req = httptest.NewRequest(http.MethodGet, "/api/races/"+race.ID.Short()+"/categories", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/api/categories/"+created.ID.Short(), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequest(http.MethodDelete, "/api/categories/"+created.ID.Short(), nil)
	req.Header.Set("Authorization", adminAuthHeader(t, svc))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestCheckpointHandlers_InvalidInput(t *testing.T) {
	router, svc := setupHandlerTest(t)
	auth := adminAuthHeader(t, svc)

	req := httptest.NewRequest(http.MethodPost, "/api/races/not-a-uuid/checkpoints", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", auth)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	req = httptest.NewRequest(http.MethodPost, "/api/races/"+uuid.New().String()+"/checkpoints", bytes.NewReader([]byte(`{"name":"X","checkpoint_type":"invalid"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", auth)
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
	assert.Equal(t, participant.ID.Short(), found.ID.Short())
	assert.Equal(t, participant.RFIDTagUID, found.RFIDTagUID)

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
		"race_id":       race.ID.Short(),
		"checkpoint_id": checkpoint.ID.Short(),
		"bib_number":    "99",
		"timestamp":     now,
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/rfid/manual-entry", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", timerAuthHeader(t, svc))
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
	req.Header.Set("Authorization", timerAuthHeader(t, svc))
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
		"participant_id": participant.ID.Short(),
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/rfid/write-tag", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", adminAuthHeader(t, svc))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var updated models.Participant
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &updated))
	require.NotEmpty(t, updated.RFIDTagUID)
	_, err = uuid.Parse(updated.RFIDTagUID)
	require.NoError(t, err)
}

func TestAuthHandlers_ExchangePIN(t *testing.T) {
	router, _ := setupHandlerTest(t)

	payload, _ := json.Marshal(map[string]string{"pin": "1738"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/pin", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["token"])
	assert.Equal(t, "admin", resp["role"])
	assert.NotNil(t, resp["expires_at"])

	payload, _ = json.Marshal(map[string]string{"pin": "9999"})
	req = httptest.NewRequest(http.MethodPost, "/api/auth/pin", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthHandlers_PINJWTAccessesAdminRoute(t *testing.T) {
	router, _ := setupHandlerTest(t)

	payload, _ := json.Marshal(map[string]string{"pin": "1738"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/pin", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var pinResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &pinResp))
	token := pinResp["token"].(string)

	body := map[string]string{
		"name":       "PIN Managed Event",
		"event_date": time.Now().AddDate(0, 2, 0).Format("2006-01-02"),
		"location":   "Calumet",
		"status":     "upcoming",
	}
	payload, _ = json.Marshal(body)
	req = httptest.NewRequest(http.MethodPost, "/api/events", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
}

func pinBearerToken(t *testing.T, router *gin.Engine) string {
	t.Helper()
	payload, _ := json.Marshal(map[string]string{"pin": "1738"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/pin", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var pinResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &pinResp))
	return "Bearer " + pinResp["token"].(string)
}

func TestGetEventResultsExcel_RequiresPINJWT(t *testing.T) {
	router, svc := setupHandlerTest(t)
	event, err := svc.Events.CreateEvent(&models.Event{
		Name: "Results Export Event", EventDate: time.Now().AddDate(0, 0, 1),
	})
	require.NoError(t, err)

	path := "/api/events/" + event.ID.Short() + "/results.xlsx"

	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.True(t, w.Code == http.StatusUnauthorized || w.Code == http.StatusForbidden)

	req = httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", pinBearerToken(t, router))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Header().Get("Content-Disposition"), ".xlsx")
	assert.True(t, bytes.HasPrefix(w.Body.Bytes(), []byte("PK")))
}

// T059 — race create/delete require PIN JWT (adminOnly).
func TestRaceHandlers_CreateDeleteRequirePINJWT(t *testing.T) {
	router, svc := setupHandlerTest(t)

	event, err := svc.Events.CreateEvent(&models.Event{
		Name: "PIN Race Event", EventDate: time.Now().AddDate(0, 1, 0), Status: "upcoming",
	})
	require.NoError(t, err)

	createBody := map[string]interface{}{
		"event_id":         event.ID.String(),
		"name":             "PIN Lap Race",
		"race_type":        "lap_based",
		"duration_minutes": 360,
		"start_time":       time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339),
	}
	payload, _ := json.Marshal(createBody)

	// Unauthenticated create → 401
	req := httptest.NewRequest(http.MethodPost, "/api/races", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// PIN JWT create → 201
	req = httptest.NewRequest(http.MethodPost, "/api/races", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", pinBearerToken(t, router))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var created models.Race
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	raceShort := created.ID.Short()
	require.Len(t, raceShort, 6)

	// Unauthenticated delete → 401
	req = httptest.NewRequest(http.MethodDelete, "/api/races/"+raceShort, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// PIN JWT delete → 200 (soft-cancel)
	req = httptest.NewRequest(http.MethodDelete, "/api/races/"+raceShort, nil)
	req.Header.Set("Authorization", pinBearerToken(t, router))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	raceID, err := uuidutil.Resolve(svc.DB, &models.Race{}, raceShort)
	require.NoError(t, err)
	race, err := svc.Races.GetRace(raceID)
	require.NoError(t, err)
	assert.Equal(t, "cancelled", race.Status)
}

func TestKaraokeBonus_CreatesAndConflicts(t *testing.T) {
	router, svc := setupHandlerTest(t)
	eventID, tagUID := seedScanHandlerFixture(t, svc, "active")

	body := map[string]string{
		"tag_uid":         tagUID,
		"device_id":       "laptop-finish-1",
		"local_timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/events/"+eventID+"/scans", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var lapResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &lapResp))
	timingID, ok := lapResp["timing_record_id"].(string)
	require.True(t, ok && timingID != "")

	// No auth required (armed-station / open like scans).
	req = httptest.NewRequest(http.MethodPost, "/api/timing-records/"+timingID+"/karaoke-bonus", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var bonusResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &bonusResp))
	assert.Equal(t, float64(2), bonusResp["lap_count"])
	assert.Equal(t, "karaoke_bonus", bonusResp["record"].(map[string]interface{})["record_type"])

	req = httptest.NewRequest(http.MethodPost, "/api/timing-records/"+timingID+"/karaoke-bonus", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestVoidTimingRecord_RequiresPINAndCascades(t *testing.T) {
	router, svc := setupHandlerTest(t)
	eventID, tagUID := seedScanHandlerFixture(t, svc, "active")

	body := map[string]string{
		"tag_uid":         tagUID,
		"device_id":       "laptop-finish-1",
		"local_timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/events/"+eventID+"/scans", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var lapResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &lapResp))
	timingID := lapResp["timing_record_id"].(string)

	req = httptest.NewRequest(http.MethodPost, "/api/timing-records/"+timingID+"/karaoke-bonus", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	// Unauthenticated void rejected
	req = httptest.NewRequest(http.MethodPost, "/api/timing/records/"+timingID+"/void", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.True(t, w.Code == http.StatusUnauthorized || w.Code == http.StatusForbidden)

	req = httptest.NewRequest(http.MethodPost, "/api/timing/records/"+timingID+"/void", nil)
	req.Header.Set("Authorization", pinBearerToken(t, router))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var voidResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &voidResp))
	assert.Equal(t, float64(0), voidResp["lap_count"])
	cascaded, ok := voidResp["cascaded_ids"].([]interface{})
	require.True(t, ok)
	assert.Len(t, cascaded, 1)
	rec := voidResp["record"].(map[string]interface{})
	assert.NotNil(t, rec["voided_at"])

	req = httptest.NewRequest(http.MethodPost, "/api/timing/records/"+timingID+"/restore", nil)
	req.Header.Set("Authorization", pinBearerToken(t, router))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &voidResp))
	assert.Equal(t, float64(1), voidResp["lap_count"])
}

func TestRFIDHandlers_ReadPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
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
	))

	mock := rfid.NewMockReader()
	mock.Enqueue("a1b2c3d4-e5f6-7890-abcd-ef1234567890")
	svc := services.NewServicesWithReader(db, testHandlerConfig(), mock)
	h := NewHandlers(svc)

	auth := middleware.JWTAuth(svc.Auth)
	adminOnly := []gin.HandlerFunc{auth, middleware.RequireRoles(services.RoleAdmin, services.RoleOwner)}
	router := gin.New()
	router.GET("/api/rfid/read-payload", append(adminOnly, h.ReadRFIDPayload)...)

	req := httptest.NewRequest(http.MethodGet, "/api/rfid/read-payload", nil)
	req.Header.Set("Authorization", adminAuthHeader(t, svc))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "a1b2c3d4-e5f6-7890-abcd-ef1234567890", resp["logical_uuid"])

	// Empty poll returns 200 with empty string.
	req = httptest.NewRequest(http.MethodGet, "/api/rfid/read-payload", nil)
	req.Header.Set("Authorization", adminAuthHeader(t, svc))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "", resp["logical_uuid"])

	// Unauthenticated → 401
	req = httptest.NewRequest(http.MethodGet, "/api/rfid/read-payload", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRFIDHandlers_Inject(t *testing.T) {
	router, svc := setupHandlerTest(t)

	sub := svc.RFID.SubscribeTagReads(4)
	payload, _ := json.Marshal(map[string]string{"tag_uid": "DEMO-TAG-0001"})
	req := httptest.NewRequest(http.MethodPost, "/api/rfid/inject", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	uid, err := svc.RFID.Poll()
	require.NoError(t, err)
	assert.Equal(t, "demo-tag-0001", uid)

	select {
	case ev := <-sub:
		assert.Equal(t, "DEMO-TAG-0001", ev.TagUID)
	default:
		t.Fatal("expected tag fan-out event")
	}
}

func TestRFIDHandlers_InjectDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
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
	))

	cfg := testHandlerConfig()
	cfg.RFID.InjectEnabled = false
	svc := services.NewServicesWithReader(db, cfg, rfid.NewMockReader())
	h := NewHandlers(svc)
	router := gin.New()
	router.POST("/api/rfid/inject", h.InjectRFIDTag)

	payload, _ := json.Marshal(map[string]string{"tag_uid": "X"})
	req := httptest.NewRequest(http.MethodPost, "/api/rfid/inject", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func seedScanHandlerFixture(t *testing.T, svc *services.Services, raceStatus string) (eventID, tagUID string) {
	t.Helper()
	db := svc.DB

	event := &models.Event{
		Name:      "Live Board Event",
		EventDate: time.Now().AddDate(0, 0, 1),
		Status:    "upcoming",
	}
	require.NoError(t, db.Create(event).Error)

	start := time.Now().Add(time.Hour)
	if raceStatus == "active" {
		start = time.Now().Add(-time.Hour)
	}
	race := &models.Race{
		EventID:         event.ID,
		Name:            "12 Hour",
		RaceType:        "lap_based",
		DurationMinutes: 720,
		StartTime:       start,
		Status:          raceStatus,
	}
	require.NoError(t, db.Create(race).Error)

	cat := &models.Category{
		RaceID:       race.ID,
		Name:         "Expert Men",
		CategoryType: "custom",
		GenderFilter: "male",
	}
	require.NoError(t, db.Create(cat).Error)
	catID := cat.ID

	part := &models.Participant{
		RaceID:     race.ID,
		CategoryID: &catID,
		BibNumber:  "12",
		FirstName:  "Alex",
		LastName:   "Rivera",
		Gender:     "male",
		Status:     "started",
	}
	require.NoError(t, db.Create(part).Error)

	finish := &models.TimingCheckpoint{
		RaceID:         race.ID,
		Name:           "Finish",
		CheckpointType: "finish",
		IsActive:       true,
	}
	require.NoError(t, db.Create(finish).Error)

	tagUID = "DEMO-TAG-0001"
	bib := models.Bib{EventID: event.ID, BibNumber: part.BibNumber}
	require.NoError(t, db.Create(&bib).Error)
	require.NoError(t, db.Create(&models.RFIDTagAssociation{
		BibID:  bib.ID,
		TagUID: tagUID,
		Active: true,
	}).Error)

	return event.ID.Short(), tagUID
}

func TestProcessEventScan_LapAndUnknown(t *testing.T) {
	router, svc := setupHandlerTest(t)
	eventID, tagUID := seedScanHandlerFixture(t, svc, "active")

	body := map[string]string{
		"tag_uid":         tagUID,
		"device_id":       "laptop-finish-1",
		"local_timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/events/"+eventID+"/scans", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var lapResp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &lapResp))
	assert.Equal(t, "lap", lapResp["result"])
	assert.Equal(t, float64(1), lapResp["lap_count"])
	assert.Equal(t, true, lapResp["karaoke_available"])

	payload, _ = json.Marshal(map[string]string{"tag_uid": "NOPE"})
	req = httptest.NewRequest(http.MethodPost, "/api/events/"+eventID+"/scans", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	var unknown map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &unknown))
	assert.Equal(t, "unknown_tag", unknown["result"])
}

func TestGetEventLive_CountdownAndLegend(t *testing.T) {
	router, svc := setupHandlerTest(t)
	eventID, tagUID := seedScanHandlerFixture(t, svc, "scheduled")

	req := httptest.NewRequest(http.MethodGet, "/api/events/"+eventID+"/live", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var live map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &live))
	require.Contains(t, live, "event")
	require.Contains(t, live, "races")
	require.Contains(t, live, "category_legend")

	races := live["races"].([]interface{})
	require.Len(t, races, 1)
	race0 := races[0].(map[string]interface{})
	assert.Equal(t, "scheduled", race0["status"])
	assert.Equal(t, "lap_based", race0["race_type"])
	assert.Greater(t, race0["countdown_seconds"].(float64), float64(0))
	assert.NotNil(t, race0["flow_series"])
	assert.NotNil(t, race0["leaderboard_overall"])

	// Score a lap after activating race via direct DB, then live board should include legend + place.
	var race models.Race
	require.NoError(t, svc.DB.First(&race).Error)
	require.NoError(t, svc.DB.Model(&race).Update("status", "active").Error)

	scanBody, _ := json.Marshal(map[string]string{
		"tag_uid":   tagUID,
		"device_id": "laptop-finish-1",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/events/"+eventID+"/scans", bytes.NewReader(scanBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/api/events/"+eventID+"/live", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &live))

	legend := live["category_legend"].([]interface{})
	require.NotEmpty(t, legend)
	entry := legend[0].(map[string]interface{})
	assert.Equal(t, "expert_men", entry["key"])
	assert.Equal(t, "Expert Men", entry["label"])
	assert.NotEmpty(t, entry["color"])

	races = live["races"].([]interface{})
	race0 = races[0].(map[string]interface{})
	board := race0["leaderboard_overall"].([]interface{})
	require.Len(t, board, 1)
	row := board[0].(map[string]interface{})
	assert.Equal(t, float64(1), row["place"])
	assert.Equal(t, "Alex Rivera", row["name"])
}

func TestStationsCurrent_PutAndGet(t *testing.T) {
	router, svc := setupHandlerTest(t)
	auth := adminAuthHeader(t, svc)
	eventID, _ := seedScanHandlerFixture(t, svc, "scheduled")

	body := map[string]interface{}{
		"event_id":  eventID,
		"mode":      "finish",
		"device_id": "laptop-finish-1",
		"name":      "Finish Mat A",
	}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/api/stations/current", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", auth)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var station models.ReaderStation
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &station))
	assert.Equal(t, "finish", station.Mode)
	assert.Equal(t, "laptop-finish-1", station.DeviceID)

	req = httptest.NewRequest(http.MethodGet, "/api/stations/current", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var current map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &current))
	assert.Equal(t, true, current["online"])
	st := current["station"].(map[string]interface{})
	assert.Equal(t, "Finish Mat A", st["name"])
}

func TestRFIDStream_WebSocketReceivesInject(t *testing.T) {
	router, _ := setupHandlerTest(t)
	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + server.URL[len("http"):] + "/api/rfid/stream"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	payload, _ := json.Marshal(map[string]string{"tag_uid": "WS-TAG-1"})
	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/rfid/inject", bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var msg map[string]interface{}
	require.NoError(t, conn.ReadJSON(&msg))
	assert.Equal(t, "tag_read", msg["type"])
	assert.Equal(t, "WS-TAG-1", msg["tag_uid"])
}

func TestEventTaps_ListAndParticipantsArePaginated(t *testing.T) {
	router, svc := setupHandlerTest(t)
	event, err := svc.Events.CreateEvent(&models.Event{
		Name: "Event taps", EventDate: time.Now().AddDate(0, 0, 1),
	})
	require.NoError(t, err)
	race, err := svc.Races.CreateRace(&models.Race{
		EventID: event.ID, Name: "12 Hour", RaceType: "lap_based", DurationMinutes: 720,
	})
	require.NoError(t, err)
	_, err = svc.Checkpoints.CreateCheckpoint(&models.TimingCheckpoint{
		RaceID: race.ID, Name: "Finish", CheckpointType: "finish",
	})
	require.NoError(t, err)
	ada, err := svc.Participants.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "101", FirstName: "Ada", LastName: "Rider",
	})
	require.NoError(t, err)
	ben, err := svc.Participants.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "102", FirstName: "Ben", LastName: "Rider",
	})
	require.NoError(t, err)
	now := time.Now().UTC()
	_, err = svc.Timing.CreateEventTap(services.CreateEventTapInput{
		EventID: event.ID.UUID(), ParticipantID: ada.ID.UUID(), Timestamp: &now,
	})
	require.NoError(t, err)
	older := now.Add(-time.Minute)
	_, err = svc.Timing.CreateEventTap(services.CreateEventTapInput{
		EventID: event.ID.UUID(), ParticipantID: ben.ID.UUID(), Timestamp: &older,
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/events/"+event.ID.Short()+"/taps?page=1&limit=1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var taps map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &taps))
	assert.Equal(t, float64(2), taps["total"])
	assert.Equal(t, float64(1), taps["page"])
	assert.Equal(t, float64(1), taps["limit"])
	assert.Len(t, taps["data"], 1)

	req = httptest.NewRequest(http.MethodGet, "/api/events/"+event.ID.Short()+"/participants?page=1&limit=1&q=ada", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var participants map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &participants))
	assert.Equal(t, float64(1), participants["total"])
	assert.Len(t, participants["data"], 1)
}

func TestEventTaps_CreateRequiresTimerJWTAndSupportsKaraoke(t *testing.T) {
	router, svc := setupHandlerTest(t)
	eventID, _ := seedScanHandlerFixture(t, svc, "active")
	var participant models.Participant
	require.NoError(t, svc.DB.First(&participant).Error)

	payload, err := json.Marshal(map[string]interface{}{
		"participant_id": participant.ID.Short(),
		"karaoke_bonus":  false,
	})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/events/"+eventID+"/taps", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.True(t, w.Code == http.StatusUnauthorized || w.Code == http.StatusForbidden)

	req = httptest.NewRequest(http.MethodPost, "/api/events/"+eventID+"/taps", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", timerAuthHeader(t, svc))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	var lap map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &lap))
	assert.Equal(t, "rfid_lap", lap["record_type"])

	payload, err = json.Marshal(map[string]interface{}{
		"participant_id": participant.ID.Short(),
		"karaoke_bonus":  true,
	})
	require.NoError(t, err)
	req = httptest.NewRequest(http.MethodPost, "/api/events/"+eventID+"/taps", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", pinBearerToken(t, router))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)
	var bonus map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &bonus))
	assert.Equal(t, "karaoke_bonus", bonus["record_type"])
	assert.NotContains(t, bonus, "source_lap_id")
}

func TestBibHandlers_BulkCreateAndListShowsAssignment(t *testing.T) {
	router, svc := setupHandlerTest(t)
	auth := adminAuthHeader(t, svc)

	event, err := svc.Events.CreateEvent(&models.Event{
		Name: "Bib Event", EventDate: time.Now().AddDate(0, 1, 0),
	})
	require.NoError(t, err)
	race, err := svc.Races.CreateRace(&models.Race{
		EventID: event.ID, Name: "Race", RaceType: "time_based", DistanceKm: 10,
	})
	require.NoError(t, err)
	participant, err := svc.Participants.CreateParticipant(&models.Participant{
		RaceID: race.ID, BibNumber: "2", FirstName: "Ada", LastName: "Lovelace",
	})
	require.NoError(t, err)

	payload, err := json.Marshal(map[string]int{"from": 1, "to": 3})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/events/"+event.ID.Short()+"/bibs/bulk", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", auth)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var created []map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	require.Len(t, created, 3)

	req = httptest.NewRequest(http.MethodGet, "/api/events/"+event.ID.Short()+"/bibs", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var listed struct {
		Data []services.BibListItem `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &listed))
	require.Len(t, listed.Data, 3)

	byNumber := map[string]services.BibListItem{}
	for _, item := range listed.Data {
		byNumber[item.BibNumber] = item
	}
	assigned := byNumber["2"]
	require.NotNil(t, assigned.ParticipantID)
	assert.Equal(t, participant.ID.Short(), assigned.ParticipantID.Short())
	assert.Equal(t, "Ada Lovelace", assigned.ParticipantName)
	assert.Nil(t, byNumber["1"].ParticipantID)
}

func TestBibHandlers_WriteTagForBibReturnsBibLogical(t *testing.T) {
	router, svc := setupHandlerTest(t)
	auth := adminAuthHeader(t, svc)

	event, err := svc.Events.CreateEvent(&models.Event{
		Name: "Write Bib Event", EventDate: time.Now().AddDate(0, 1, 0),
	})
	require.NoError(t, err)
	bib, err := svc.Bibs.EnsureBib(event.ID.UUID(), "7")
	require.NoError(t, err)
	logical := strings.ToLower(bib.ID.String())

	req := httptest.NewRequest(http.MethodPost, "/api/events/"+event.ID.Short()+"/bibs/"+bib.ID.Short()+"/tags", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", auth)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, logical, resp["tag_uid"])
	assert.Equal(t, bib.ID.Short(), resp["bib_id"])

	req = httptest.NewRequest(http.MethodGet, "/api/events/"+event.ID.Short()+"/bibs/"+bib.ID.Short()+"/tags", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	var tags struct {
		Data []models.RFIDTagAssociation `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &tags))
	require.Len(t, tags.Data, 1)
	assert.Equal(t, logical, tags.Data[0].TagUID)

	// write-tag with bib_id also programs bib logical UUID
	payload, err := json.Marshal(map[string]string{"bib_id": bib.ID.Short()})
	require.NoError(t, err)
	req = httptest.NewRequest(http.MethodPost, "/api/rfid/write-tag", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", auth)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, logical, resp["tag_uid"])
	assert.Equal(t, bib.ID.Short(), resp["bib_id"])
}

func TestRFIDHandlers_WriteTag_ParticipantStillWorks(t *testing.T) {
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

	body := map[string]string{"participant_id": participant.ID.Short()}
	payload, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/rfid/write-tag", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", adminAuthHeader(t, svc))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var updated models.Participant
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &updated))
	require.NotEmpty(t, updated.RFIDTagUID)
	_, err = uuid.Parse(updated.RFIDTagUID)
	require.NoError(t, err)

	bib, err := svc.Bibs.EnsureBib(event.ID.UUID(), "10")
	require.NoError(t, err)
	assert.Equal(t, strings.ToLower(bib.ID.String()), updated.RFIDTagUID)
}

func TestParticipantHandlers_EventWideBibClashReturns400(t *testing.T) {
	router, svc := setupHandlerTest(t)
	auth := adminAuthHeader(t, svc)

	event, err := svc.Events.CreateEvent(&models.Event{
		Name: "Clash Event", EventDate: time.Now().AddDate(0, 1, 0),
	})
	require.NoError(t, err)
	raceA, err := svc.Races.CreateRace(&models.Race{
		EventID: event.ID, Name: "Race A", RaceType: "time_based", DistanceKm: 5,
	})
	require.NoError(t, err)
	raceB, err := svc.Races.CreateRace(&models.Race{
		EventID: event.ID, Name: "Race B", RaceType: "time_based", DistanceKm: 10,
	})
	require.NoError(t, err)
	catA, err := svc.Categories.CreateCategory(&models.Category{
		RaceID: raceA.ID, Name: "Open", CategoryType: "overall",
	})
	require.NoError(t, err)
	catB, err := svc.Categories.CreateCategory(&models.Category{
		RaceID: raceB.ID, Name: "Open", CategoryType: "overall",
	})
	require.NoError(t, err)

	payload, err := json.Marshal(map[string]string{
		"first_name":  "A",
		"last_name":   "One",
		"bib_number":  "7",
		"category_id": catA.ID.Short(),
	})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/races/"+raceA.ID.Short()+"/participants", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", auth)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	payload, err = json.Marshal(map[string]string{
		"first_name":  "B",
		"last_name":   "Two",
		"bib_number":  "7",
		"category_id": catB.ID.Short(),
	})
	require.NoError(t, err)
	req = httptest.NewRequest(http.MethodPost, "/api/races/"+raceB.ID.Short()+"/participants", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", auth)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}
