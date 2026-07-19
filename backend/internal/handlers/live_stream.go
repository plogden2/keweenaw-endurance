package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/keweenaw-endurance/backend/internal/services"
	"github.com/keweenaw-endurance/backend/internal/services/scan"
)

var liveStreamUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (h *Handlers) publishLapRecorded(eventID uuid.UUID, result *scan.ScanResult) {
	if h.services.LiveStream == nil || result == nil || result.Result != "lap" {
		return
	}
	if result.RaceID == nil || result.Participant == nil {
		return
	}
	h.services.LiveStream.Publish(eventID, services.LapRecordedEvent{
		Type:            "lap_recorded",
		EventID:         eventID.String(),
		RaceID:          result.RaceID.UUID().String(),
		ParticipantID:   result.Participant.ID.UUID().String(),
		ParticipantName: result.ParticipantName,
		BibNumber:       result.BibNumber,
		LapCount:        result.LapCount,
		RecordedAt:      time.Now().UTC(),
	})
}

// StreamEventLive upgrades to WebSocket and fans out lap_recorded events for an event.
func (h *Handlers) StreamEventLive(c *gin.Context) {
	eventID, err := h.resolveEventID(c.Param("id"))
	if err != nil {
		respondServiceError(c, err)
		return
	}
	conn, err := liveStreamUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	sub, cancel := h.services.LiveStream.Subscribe(eventID, 32)
	defer cancel()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case ev, ok := <-sub:
			if !ok {
				return
			}
			if err := conn.WriteJSON(ev); err != nil {
				return
			}
		}
	}
}
