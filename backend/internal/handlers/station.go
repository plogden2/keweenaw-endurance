package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/keweenaw-endurance/backend/internal/services"
)

func (h *Handlers) PutCurrentStation(c *gin.Context) {
	var req putStationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	eventID, err := h.resolveEventID(req.EventID)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	var checkpointID *uuid.UUID
	if req.CheckpointID != nil && *req.CheckpointID != "" {
		id, err := h.resolveCheckpointID(*req.CheckpointID)
		if err != nil {
			respondServiceError(c, err)
			return
		}
		checkpointID = &id
	}

	station, err := h.services.Stations.PutCurrent(&services.StationConfigInput{
		EventID:      eventID,
		Mode:         req.Mode,
		CheckpointID: checkpointID,
		DeviceID:     req.DeviceID,
		Name:         req.Name,
	})
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, station)
}

func (h *Handlers) GetCurrentStation(c *gin.Context) {
	resp, err := h.services.Stations.GetCurrent()
	if err != nil {
		respondServiceError(c, err)
		return
	}
	if resp.Station != nil && h.services.CSV != nil {
		if status, err := h.services.CSV.LiveSnapshotStatus(resp.Station.EventID.UUID()); err == nil && status.Exists {
			t := status.UpdatedAt
			resp.LiveCSVLastWrite = &t
		}
	}
	c.JSON(http.StatusOK, resp)
}
