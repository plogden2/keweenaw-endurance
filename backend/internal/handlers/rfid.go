package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handlers) WriteRFIDTag(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "RFID write not yet implemented"})
}

func (h *Handlers) ScanRFIDTag(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "RFID scan not yet implemented"})
}

func (h *Handlers) ManualTimingEntry(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "manual timing entry not yet implemented"})
}

func (h *Handlers) GetSyncStatus(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "sync status not yet implemented"})
}
