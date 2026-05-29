package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/keweenaw-endurance/backend/internal/services"
)

func (h *Handlers) WriteRFIDTag(c *gin.Context) {
	var req writeRFIDTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	participantID, err := h.resolveParticipantID(req.ParticipantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid participant_id"})
		return
	}

	participant, err := h.services.RFID.WriteTag(participantID, req.TagUID)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, participant)
}

func (h *Handlers) ScanRFIDTag(c *gin.Context) {
	uid := c.Param("uid")
	if uid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "uid is required"})
		return
	}

	participant, err := h.services.RFID.LookupParticipantByUID(uid)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, participant)
}

func (h *Handlers) ManualTimingEntry(c *gin.Context) {
	var req manualTimingEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	raceID, err := h.resolveRaceID(req.RaceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid race_id"})
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

	record, err := h.services.RFID.ManualEntry(&services.ManualEntryInput{
		RaceID:       raceID,
		CheckpointID: checkpointID,
		BibNumber:    req.BibNumber,
		RFIDTagUID:   req.RFIDTagUID,
		Timestamp:    timestamp,
		DeviceID:     req.DeviceID,
		SyncStatus:   req.SyncStatus,
	})
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, record)
}

func (h *Handlers) GetSyncStatus(c *gin.Context) {
	status, err := h.services.RFID.GetSyncStatus()
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, status)
}

func (h *Handlers) SyncPendingRecords(c *gin.Context) {
	synced, err := h.services.RFID.SyncPending()
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"synced_count": synced})
}
