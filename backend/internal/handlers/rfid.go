package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/keweenaw-endurance/backend/internal/services"
)

var rfidUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

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

	h.refreshLiveCSVForRace(participant.RaceID.UUID())
	c.JSON(http.StatusOK, participant)
}

// InjectRFIDTag is test/dev only (cfg.RFID.InjectEnabled, including GO_ENV=test).
func (h *Handlers) InjectRFIDTag(c *gin.Context) {
	if h.services.Config == nil || !h.services.Config.RFID.InjectEnabled {
		c.JSON(http.StatusForbidden, gin.H{"error": "rfid inject disabled"})
		return
	}

	var req injectRFIDTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.services.RFID.InjectTag(req.TagUID); err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"injected": true, "tag_uid": req.TagUID})
}

// StreamRFIDTags upgrades to WebSocket and fans out tag_read events.
func (h *Handlers) StreamRFIDTags(c *gin.Context) {
	conn, err := rfidUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	sub := h.services.RFID.SubscribeTagReads(32)
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case ev, ok := <-sub:
			if !ok {
				return
			}
			if ev.Type == "" {
				ev.Type = "tag_read"
			}
			if ev.DeviceID == "" && h.services.Stations != nil {
				ev.DeviceID = h.services.Stations.CurrentDeviceID()
			}
			if err := conn.WriteJSON(ev); err != nil {
				return
			}
		}
	}
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

// ProcessEventScan handles POST /api/events/:id/scans.
func (h *Handlers) ProcessEventScan(c *gin.Context) {
	eventID, err := h.resolveEventID(c.Param("id"))
	if err != nil {
		respondServiceError(c, err)
		return
	}

	var req processScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ts := time.Now().UTC()
	if req.LocalTimestamp != "" {
		parsed, err := parseTimestamp(req.LocalTimestamp)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid local_timestamp format, use RFC3339"})
			return
		}
		ts = parsed
	}

	result, err := h.services.Scan.ProcessScan(eventID, req.TagUID, req.DeviceID, ts)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	if result.Result == "unknown_tag" {
		c.JSON(http.StatusNotFound, result)
		return
	}
	c.JSON(http.StatusOK, result)
}
