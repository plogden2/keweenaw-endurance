package handlers

import (
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// GetLiveCSVStatus returns metadata for the continuously maintained live snapshot.
func (h *Handlers) GetLiveCSVStatus(c *gin.Context) {
	eventID, err := h.resolveEventID(c.Param("id"))
	if err != nil {
		respondServiceError(c, err)
		return
	}

	status, err := h.services.CSV.LiveSnapshotStatus(eventID)
	if err != nil {
		respondServiceError(c, err)
		return
	}
	// Ensure a snapshot exists so status reflects current DB state.
	if !status.Exists {
		if _, err := h.services.CSV.WriteLiveSnapshot(eventID); err != nil {
			respondServiceError(c, err)
			return
		}
		status, err = h.services.CSV.LiveSnapshotStatus(eventID)
		if err != nil {
			respondServiceError(c, err)
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"path":       status.Path,
		"exists":     status.Exists,
		"updated_at": status.UpdatedAt.UTC().Format(time.RFC3339Nano),
		"size_bytes": status.SizeBytes,
	})
}

// GetLiveCSV returns the current live CSV snapshot bytes for the event.
func (h *Handlers) GetLiveCSV(c *gin.Context) {
	eventID, err := h.resolveEventID(c.Param("id"))
	if err != nil {
		respondServiceError(c, err)
		return
	}

	body, updatedAt, err := h.services.CSV.ReadLiveSnapshot(eventID)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="live-snapshot.csv"`)
	c.Header("Last-Modified", updatedAt.UTC().Format(http.TimeFormat))
	c.Header("X-Live-CSV-Updated-At", updatedAt.UTC().Format(time.RFC3339Nano))
	c.Data(http.StatusOK, "text/csv; charset=utf-8", body)
}

// ImportCSV handles PIN-gated multipart CSV restore with replace semantics.
func (h *Handlers) ImportCSV(c *gin.Context) {
	eventID, err := h.resolveEventID(c.Param("id"))
	if err != nil {
		respondServiceError(c, err)
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "multipart file field 'file' is required"})
		return
	}
	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to open uploaded file"})
		return
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read uploaded file"})
		return
	}

	summary, err := h.services.CSV.ImportCSV(eventID, data)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, summary)
}
