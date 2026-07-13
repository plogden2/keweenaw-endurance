package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/keweenaw-endurance/backend/internal/services"
)

// PushSync handles POST /api/sync/push — push local pending records to HOSTED_API_URL.
func (h *Handlers) PushSync(c *gin.Context) {
	if h.services.Sync == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "sync service unavailable"})
		return
	}
	result, err := h.services.Sync.Push()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// PullSync handles POST /api/sync/pull — pull hosted records into local DB.
func (h *Handlers) PullSync(c *gin.Context) {
	if h.services.Sync == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "sync service unavailable"})
		return
	}
	result, err := h.services.Sync.Pull()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// IngestSync handles POST /api/sync/ingest — merge peer/hosted push payloads locally.
func (h *Handlers) IngestSync(c *gin.Context) {
	if h.services.Sync == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "sync service unavailable"})
		return
	}
	var payload services.SyncPushPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result, err := h.services.Sync.Ingest(payload)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"accepted":   result.Imported,
		"duplicates": result.Duplicates,
	})
}

// ExportSync handles GET /api/sync/export — export local records for peer pull.
func (h *Handlers) ExportSync(c *gin.Context) {
	if h.services.Sync == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "sync service unavailable"})
		return
	}
	payload, err := h.services.Sync.ExportRecords()
	if err != nil {
		respondServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, payload)
}
