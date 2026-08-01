package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/keweenaw-endurance/backend/internal/services"
	"github.com/keweenaw-endurance/backend/internal/uuidutil"
)

// ListEventTaps handles public GET /api/events/:id/taps.
func (h *Handlers) ListEventTaps(c *gin.Context) {
	eventID, err := h.resolveEventID(c.Param("id"))
	if err != nil {
		respondServiceError(c, err)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	var raceID *uuid.UUID
	if rawRaceID := c.Query("race_id"); rawRaceID != "" {
		id, err := h.resolveRaceID(rawRaceID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid race_id"})
			return
		}
		raceID = &id
	}

	records, total, err := h.services.Timing.ListRecordsByEvent(eventID, page, limit, raceID, c.Query("q"))
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":  records,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// ListEventParticipants handles public GET /api/events/:id/participants.
func (h *Handlers) ListEventParticipants(c *gin.Context) {
	eventID, err := h.resolveEventID(c.Param("id"))
	if err != nil {
		respondServiceError(c, err)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	participants, total, err := h.services.Participants.ListParticipantsByEvent(eventID, page, limit, c.Query("q"))
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":  participants,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// CreateEventTap handles POST /api/events/:id/taps (PIN / timerWrite).
func (h *Handlers) CreateEventTap(c *gin.Context) {
	eventID, err := h.resolveEventID(c.Param("id"))
	if err != nil {
		respondServiceError(c, err)
		return
	}

	var req createEventTapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	participantID, err := h.resolveParticipantID(req.ParticipantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid participant_id"})
		return
	}

	var timestamp *time.Time
	if req.Timestamp != "" {
		parsed, err := parseTimestamp(req.Timestamp)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid timestamp format, use RFC3339"})
			return
		}
		timestamp = &parsed
	}

	record, err := h.services.Timing.CreateEventTap(services.CreateEventTapInput{
		EventID:       eventID,
		ParticipantID: participantID,
		KaraokeBonus:  req.KaraokeBonus,
		Timestamp:     timestamp,
	})
	if err != nil {
		respondServiceError(c, err)
		return
	}

	h.refreshLiveCSV(eventID)
	h.publishEventTapRecorded(eventID, record)
	c.JSON(http.StatusCreated, record)
}

func (h *Handlers) publishEventTapRecorded(eventID uuid.UUID, record *models.TimingRecord) {
	if h.services.LiveStream == nil || record == nil {
		return
	}

	participant := record.Participant
	if participant.ID.UUID() == uuid.Nil {
		if err := h.services.DB.Preload("Race").First(&participant, "id = ?", record.ParticipantID).Error; err != nil {
			return
		}
	}

	lapCount := 1
	if counted, _, _, _, err := h.services.Scan.ScoreSnapshot(record.ParticipantID.UUID()); err == nil {
		lapCount = counted
	}

	h.services.LiveStream.Publish(eventID, services.LapRecordedEvent{
		Type:            "lap_recorded",
		EventID:         uuidutil.Suffix(eventID),
		RaceID:          participant.RaceID.Short(),
		ParticipantID:   participant.ID.Short(),
		ParticipantName: strings.TrimSpace(participant.FirstName + " " + participant.LastName),
		BibNumber:       participant.BibNumber,
		LapCount:        lapCount,
		RecordedAt:      time.Now().UTC(),
	})
}
