package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/keweenaw-endurance/backend/internal/services"
	"github.com/keweenaw-endurance/backend/internal/services/scan"
	"github.com/keweenaw-endurance/backend/internal/uuidutil"
)

// CreateKaraokeBonus handles POST /api/timing-records/:id/karaoke-bonus.
// Open like scan ingest (no re-PIN on an armed station).
func (h *Handlers) CreateKaraokeBonus(c *gin.Context) {
	id, err := h.resolveTimingRecordID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid timing record id"})
		return
	}

	result, err := h.services.Scan.AddKaraokeBonus(id)
	if err != nil {
		switch {
		case errors.Is(err, scan.ErrAlreadyExists):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case errors.Is(err, scan.ErrSourceLapNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case errors.Is(err, scan.ErrInvalidSourceLap):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			respondServiceError(c, err)
		}
		return
	}

	if h.services.LiveStream != nil && result != nil && result.Record != nil {
		var participant models.Participant
		if err := h.services.DB.Preload("Race").First(&participant, "id = ?", result.Record.ParticipantID).Error; err == nil {
			eventID := participant.Race.EventID.UUID()
			h.services.LiveStream.Publish(eventID, services.LapRecordedEvent{
				Type:            "lap_recorded",
				EventID:         eventID.String(),
				RaceID:          participant.RaceID.UUID().String(),
				ParticipantID:   participant.ID.UUID().String(),
				ParticipantName: strings.TrimSpace(participant.FirstName + " " + participant.LastName),
				BibNumber:       participant.BibNumber,
				LapCount:        result.LapCount,
				RecordedAt:      time.Now().UTC(),
			})
		}
	}

	c.JSON(http.StatusCreated, result)
}

func (h *Handlers) GetLiveTiming(c *gin.Context) {
	raceID, err := h.resolveRaceID(c.Param("raceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid race id"})
		return
	}

	live, err := h.services.Results.GetLiveTiming(raceID)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, live)
}

func (h *Handlers) CreateTimingRecord(c *gin.Context) {
	var req createTimingRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	participantID, err := h.resolveParticipantID(req.ParticipantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid participant_id"})
		return
	}
	checkpointID, err := h.resolveCheckpointID(req.CheckpointID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid checkpoint_id"})
		return
	}
	timestamp, err := parseTimestamp(req.Timestamp)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid timestamp format, use RFC3339"})
		return
	}

	record := &models.TimingRecord{
		ParticipantID: uuidutil.NewPublicUUID(participantID),
		CheckpointID:  uuidutil.NewPublicUUID(checkpointID),
		Timestamp:     timestamp,
		DeviceID:      req.DeviceID,
		SyncStatus:    req.SyncStatus,
	}
	if req.LocalTimestamp != "" {
		localTimestamp, err := parseTimestamp(req.LocalTimestamp)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid local_timestamp format, use RFC3339"})
			return
		}
		record.LocalTimestamp = localTimestamp
	}

	created, err := h.services.Timing.CreateRecord(record)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

func (h *Handlers) UpdateTimingRecord(c *gin.Context) {
	id, err := h.resolveTimingRecordID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid timing record id"})
		return
	}

	var req updateTimingRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	update := &models.TimingRecord{}
	if req.Timestamp != nil {
		timestamp, err := parseTimestamp(*req.Timestamp)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid timestamp format, use RFC3339"})
			return
		}
		update.Timestamp = timestamp
	}
	if req.LocalTimestamp != nil {
		localTimestamp, err := parseTimestamp(*req.LocalTimestamp)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid local_timestamp format, use RFC3339"})
			return
		}
		update.LocalTimestamp = localTimestamp
	}
	if req.DeviceID != nil {
		update.DeviceID = *req.DeviceID
	}
	if req.SyncStatus != nil {
		update.SyncStatus = *req.SyncStatus
	}

	record, err := h.services.Timing.UpdateRecord(id, update)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, record)
}

// VoidTimingRecord handles POST /api/timing/records/:id/void (PIN / timerWrite).
func (h *Handlers) VoidTimingRecord(c *gin.Context) {
	h.mutateTimingVoidState(c, true)
}

// RestoreTimingRecord handles POST /api/timing/records/:id/restore (PIN / timerWrite).
func (h *Handlers) RestoreTimingRecord(c *gin.Context) {
	h.mutateTimingVoidState(c, false)
}

func (h *Handlers) mutateTimingVoidState(c *gin.Context, voiding bool) {
	id, err := h.resolveTimingRecordID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid timing record id"})
		return
	}

	var (
		record   *models.TimingRecord
		cascaded []uuid.UUID
	)
	if voiding {
		record, cascaded, err = h.services.Timing.VoidRecord(id)
	} else {
		record, cascaded, err = h.services.Timing.RestoreRecord(id)
	}
	if err != nil {
		respondServiceError(c, err)
		return
	}

	cascadedIDs := make([]string, 0, len(cascaded))
	for _, cid := range cascaded {
		cascadedIDs = append(cascadedIDs, cid.String())
	}

	lapCount, placement, placementCat, eventID, scoreErr := h.services.Scan.ScoreSnapshot(record.ParticipantID.UUID())
	if scoreErr != nil {
		respondServiceError(c, scoreErr)
		return
	}
	h.refreshLiveCSV(eventID)

	eventType := "lap_restored"
	if voiding {
		eventType = "lap_voided"
	}
	if h.services.LiveStream != nil {
		var participant models.Participant
		if err := h.services.DB.Preload("Race").First(&participant, "id = ?", record.ParticipantID).Error; err == nil {
			h.services.LiveStream.Publish(eventID, services.LapRecordedEvent{
				Type:            eventType,
				EventID:         eventID.String(),
				RaceID:          participant.RaceID.UUID().String(),
				ParticipantID:   participant.ID.UUID().String(),
				ParticipantName: strings.TrimSpace(participant.FirstName + " " + participant.LastName),
				BibNumber:       participant.BibNumber,
				LapCount:        lapCount,
				RecordedAt:      time.Now().UTC(),
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"record":             record,
		"cascaded_ids":       cascadedIDs,
		"lap_count":          lapCount,
		"placement":          placement,
		"placement_category": placementCat,
	})
}

func (h *Handlers) GetRaceResults(c *gin.Context) {
	raceID, err := h.resolveRaceID(c.Param("raceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid race id"})
		return
	}

	results, err := h.services.Results.GetRaceResults(raceID)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": results})
}

func (h *Handlers) GetLeaderboard(c *gin.Context) {
	raceID, err := h.resolveRaceID(c.Param("raceId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid race id"})
		return
	}

	var categoryID *uuid.UUID
	if categoryIDStr := c.Query("category_id"); categoryIDStr != "" {
		id, err := h.resolveCategoryID(categoryIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category_id"})
			return
		}
		categoryID = &id
	}

	leaderboard, err := h.services.Results.GetLeaderboard(raceID, categoryID)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": leaderboard})
}
