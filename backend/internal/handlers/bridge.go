package handlers

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/models"
	"github.com/keweenaw-endurance/backend/internal/services"
	"github.com/keweenaw-endurance/backend/internal/services/scan"
	"gorm.io/gorm"
)

func (h *Handlers) BridgeWebSocket(c *gin.Context) {
	deviceID := strings.TrimSpace(c.Query("device_id"))
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required"})
		return
	}
	if !h.authorizeBridge(c) {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	if h.services.Bridge == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "bridge unavailable"})
		return
	}

	conn, err := rfidUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	h.services.Bridge.Register(deviceID, conn)
	defer h.services.Bridge.Unregister(deviceID, conn)

	for {
		var msg services.BridgeMessage
		if err := conn.ReadJSON(&msg); err != nil {
			return
		}
		switch msg.Type {
		case "read":
			readMsg := msg
			go h.dispatchBridgeRead(c, deviceID, &readMsg)
		default:
			_ = h.services.Bridge.HandleMessage(deviceID, &msg)
		}
	}
}

func (h *Handlers) dispatchBridgeRead(c *gin.Context, deviceID string, msg *services.BridgeMessage) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("bridge read panic device_id=%s: %v", deviceID, r)
		}
	}()
	h.handleBridgeRead(c, deviceID, msg)
}

func (h *Handlers) GetBridgeStatus(c *gin.Context) {
	deviceID := strings.TrimSpace(c.Query("device_id"))
	if deviceID == "" {
		deviceID = services.DefaultBridgeDeviceID
		if h.services.Config != nil && strings.TrimSpace(h.services.Config.RFID.BridgeDeviceID) != "" {
			deviceID = strings.TrimSpace(h.services.Config.RFID.BridgeDeviceID)
		}
	}

	if h.services.Bridge == nil {
		c.JSON(http.StatusOK, services.BridgeStatus{})
		return
	}

	status := h.services.Bridge.Status(deviceID)
	if pending := h.services.Bridge.PendingWriteCount(deviceID); pending > status.PendingCount {
		status.PendingCount = pending
	}
	c.JSON(http.StatusOK, status)
}

func (h *Handlers) authorizeBridge(c *gin.Context) bool {
	if h.services.Config != nil {
		bridgeToken := strings.TrimSpace(c.GetHeader("X-Bridge-Token"))
		expected := strings.TrimSpace(h.services.Config.RFID.BridgeToken)
		if bridgeToken != "" && expected != "" && bridgeToken == expected {
			return true
		}
	}

	auth := c.GetHeader("Authorization")
	const prefix = "Bearer "
	if strings.HasPrefix(auth, prefix) && h.services.Auth != nil {
		token := strings.TrimSpace(auth[len(prefix):])
		if _, err := h.services.Auth.ValidateToken(token); err == nil {
			return true
		}
	}
	return false
}

func (h *Handlers) handleBridgeRead(c *gin.Context, deviceID string, msg *services.BridgeMessage) {
	logicalUUID := strings.TrimSpace(msg.LogicalUUID)
	if logicalUUID == "" {
		return
	}

	ts := time.Now().UTC()
	if msg.TS != "" {
		if parsed, err := time.Parse(time.RFC3339, msg.TS); err == nil {
			ts = parsed.UTC()
		}
	}

	sourceLapID := strings.TrimSpace(msg.SourceLapID)
	if sourceLapID == "" {
		sourceLapID = strings.TrimSpace(msg.RequestID)
	}
	// Bridge flush replays use source_lap_id as timing_records.id so reconnect
	// replay does not double-score. Dedupe is durable across process restarts.
	if sourceLapID != "" {
		if id, err := uuid.Parse(sourceLapID); err == nil {
			var existing models.TimingRecord
			err := h.services.DB.First(&existing, "id = ?", id).Error
			if err == nil {
				log.Printf("bridge read duplicate skipped source_lap_id=%s logical_uuid=%s", sourceLapID, logicalUUID)
				return
			}
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				log.Printf("bridge read dedupe lookup failed source_lap_id=%s: %v", sourceLapID, err)
			}
		}
	}

	eventID, err := h.services.Stations.EventIDForDevice(deviceID)
	if err != nil {
		log.Printf("bridge read: event lookup failed device_id=%s logical_uuid=%s: %v", deviceID, logicalUUID, err)
		c.Error(err)
		return
	}

	var scanOpts []scan.ScanOptions
	if sourceLapID != "" {
		scanOpts = append(scanOpts, scan.ScanOptions{BridgeRecordID: sourceLapID})
	}
	result, err := h.services.Scan.ProcessScan(eventID, logicalUUID, deviceID, ts, scanOpts...)
	if err != nil {
		c.Error(err)
		return
	}
	// Do NOT InjectTag here — that fans out tag_read and the reader UI would
	// POST /scans again, double-scoring within the cooldown race window.
	// Publish the already-scored result for ScanPopup feedback instead.
	if result != nil && h.services.RFID != nil {
		h.services.RFID.PublishScanResult(logicalUUID, result)
	}
	h.publishLapRecorded(eventID, result)
	if result != nil && result.Result != scan.ResultUnknownTag {
		h.refreshLiveCSV(eventID)
	}
}
