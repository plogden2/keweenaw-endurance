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

	svc := services.NewServices(db, &config.Config{Environment: "test"})
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
		api.GET("/races/:id", h.GetRace)
		api.PUT("/races/:id", h.UpdateRace)
		api.DELETE("/races/:id", h.DeleteRace)

		api.GET("/participants", h.GetParticipants)
		api.POST("/participants", h.CreateParticipant)
		api.GET("/participants/:id", h.GetParticipant)
		api.PUT("/participants/:id", h.UpdateParticipant)
		api.DELETE("/participants/:id", h.DeleteParticipant)
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

	updateBody := map[string]string{"name": "Updated Fest", "event_date": created.EventDate.Format("2006-01-02")}
	payload, _ = json.Marshal(updateBody)
	req = httptest.NewRequest(http.MethodPut, "/api/events/"+created.ID.String(), bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequest(http.MethodDelete, "/api/events/"+created.ID.String(), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/api/events/"+created.ID.String(), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
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

func TestTimingHandlers_NotImplemented(t *testing.T) {
	router, _ := setupHandlerTest(t)

	gin.SetMode(gin.TestMode)
	h := NewHandlers(&services.Services{})
	r := gin.New()
	r.GET("/api/timing/live/:raceId", h.GetLiveTiming)

	req := httptest.NewRequest(http.MethodGet, "/api/timing/live/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotImplemented, w.Code)

	_ = router
}
