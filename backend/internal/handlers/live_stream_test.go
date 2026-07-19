package handlers

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/keweenaw-endurance/backend/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamEventLive_WebSocketReceivesPublishedLap(t *testing.T) {
	router, svc := setupHandlerTest(t)
	eventID, _ := seedScanHandlerFixture(t, svc, "active")

	var event models.Event
	require.NoError(t, svc.DB.First(&event).Error)

	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + server.URL[len("http"):] + "/api/events/" + eventID + "/live/stream"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	svc.LiveStream.Publish(event.ID.UUID(), services.LapRecordedEvent{
		Type:            "lap_recorded",
		EventID:         event.ID.UUID().String(),
		RaceID:          "race-uuid",
		ParticipantID:   "participant-uuid",
		ParticipantName: "Alex Rivera",
		BibNumber:       "12",
		LapCount:        3,
		RecordedAt:      time.Now().UTC(),
	})

	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var msg map[string]interface{}
	require.NoError(t, conn.ReadJSON(&msg))
	assert.Equal(t, "lap_recorded", msg["type"])
	assert.Equal(t, "Alex Rivera", msg["participant_name"])
	assert.Equal(t, float64(3), msg["lap_count"])
}
