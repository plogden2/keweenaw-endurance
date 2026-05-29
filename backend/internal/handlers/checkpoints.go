package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/keweenaw-endurance/backend/internal/models"
)

func (h *Handlers) GetCheckpointsByRace(c *gin.Context) {
	raceID, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid race id"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	checkpoints, total, err := h.services.Checkpoints.ListCheckpointsByRace(raceID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list checkpoints"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  checkpoints,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *Handlers) CreateCheckpoint(c *gin.Context) {
	raceID, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid race id"})
		return
	}

	var req createCheckpointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	checkpoint := &models.TimingCheckpoint{
		RaceID:              raceID,
		Name:                req.Name,
		CheckpointType:      req.CheckpointType,
		DistanceFromStartKm: req.DistanceFromStartKm,
		LocationDescription: req.LocationDescription,
		IsActive:            true,
	}
	if req.IsActive != nil {
		checkpoint.IsActive = *req.IsActive
	}

	created, err := h.services.Checkpoints.CreateCheckpoint(checkpoint)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

func (h *Handlers) GetCheckpoint(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid checkpoint id"})
		return
	}

	checkpoint, err := h.services.Checkpoints.GetCheckpoint(id)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, checkpoint)
}

func (h *Handlers) UpdateCheckpoint(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid checkpoint id"})
		return
	}

	var req updateCheckpointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	update := &models.TimingCheckpoint{}
	if req.Name != nil {
		update.Name = *req.Name
	}
	if req.CheckpointType != nil {
		update.CheckpointType = *req.CheckpointType
	}
	if req.DistanceFromStartKm != nil {
		update.DistanceFromStartKm = *req.DistanceFromStartKm
	}
	if req.LocationDescription != nil {
		update.LocationDescription = *req.LocationDescription
	}

	checkpoint, err := h.services.Checkpoints.UpdateCheckpoint(id, update)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	if req.IsActive != nil {
		checkpoint.IsActive = *req.IsActive
		checkpoint, err = h.services.Checkpoints.UpdateCheckpoint(id, checkpoint)
		if err != nil {
			respondServiceError(c, err)
			return
		}
	}

	c.JSON(http.StatusOK, checkpoint)
}

func (h *Handlers) DeleteCheckpoint(c *gin.Context) {
	id, err := parseUUID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid checkpoint id"})
		return
	}

	if err := h.services.Checkpoints.DeleteCheckpoint(id); err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "checkpoint deleted"})
}
